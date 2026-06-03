package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/config"
)

type QdrantService struct {
	cfg       config.QdrantConfig
	embedding config.EmbeddingConfig
	client    *http.Client
}

func NewQdrantService(qdrant config.QdrantConfig, embedding config.EmbeddingConfig) *QdrantService {
	return &QdrantService{
		cfg:       qdrant,
		embedding: embedding,
		client:    &http.Client{Timeout: 20 * time.Second},
	}
}

func (s *QdrantService) Recall(ctx context.Context, input port.CharacterMemoryRecallInput) ([]model.Memory, error) {
	if !s.enabled() || strings.TrimSpace(input.Query) == "" {
		return nil, nil
	}
	vector, err := s.embed(ctx, input.Query)
	if err != nil {
		return nil, err
	}
	limit := input.Limit
	if limit <= 0 {
		limit = 12
	}
	body := map[string]any{
		"vector": vector,
		"limit":  limit,
		"filter": map[string]any{"must": []map[string]any{
			matchCondition("project_id", input.ProjectID),
			matchCondition("character_id", input.CharacterID),
			matchCondition("canon_status", "committed"),
		}},
		"with_payload": true,
	}
	var response qdrantSearchResponse
	if err := s.qdrant(ctx, http.MethodPost, fmt.Sprintf("/collections/%s/points/search", s.collection()), body, &response); err != nil {
		return nil, err
	}
	return response.memories(input.CharacterID), nil
}

func (s *QdrantService) Commit(ctx context.Context, input port.CharacterMemoryCommitInput) error {
	if !s.enabled() || len(input.Memories) == 0 {
		return nil
	}
	points := make([]map[string]any, 0, len(input.Memories))
	for _, memory := range input.Memories {
		vector, err := s.embed(ctx, memory.Content)
		if err != nil {
			return err
		}
		points = append(points, map[string]any{
			"id":     memory.ID,
			"vector": vector,
			"payload": map[string]any{
				"content":           memory.Content,
				"project_id":        input.ProjectID,
				"run_id":            input.RunID,
				"chapter_id":        input.Chapter.ID,
				"chapter_number":    input.Chapter.ChapterNumber,
				"memory_id":         memory.ID,
				"character_id":      memory.CharacterID,
				"source_chapter_id": memory.SourceChapterID,
				"source_run_id":     memory.SourceRunID,
				"branch_id":         memory.BranchID,
				"source_event_id":   memory.SourceEventID,
				"importance":        memory.Importance,
				"status":            memory.Status,
				"note":              memory.Note,
				"canon_status":      "committed",
				"created_at":        memory.CreatedAt.Format(time.RFC3339),
			},
		})
	}
	return s.qdrant(ctx, http.MethodPut, fmt.Sprintf("/collections/%s/points?wait=true", s.collection()), map[string]any{"points": points}, nil)
}

func (s *QdrantService) enabled() bool {
	return strings.TrimSpace(s.cfg.URL) != "" && strings.TrimSpace(s.embedding.BaseURL) != "" && strings.TrimSpace(s.embedding.APIKey) != "" && strings.TrimSpace(s.embedding.Model) != ""
}

func (s *QdrantService) embed(ctx context.Context, text string) ([]float64, error) {
	body := map[string]any{
		"model": s.embedding.Model,
		"input": text,
	}
	var response embeddingResponse
	if err := s.openAI(ctx, http.MethodPost, "/embeddings", body, &response); err != nil {
		return nil, err
	}
	if len(response.Data) == 0 || len(response.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("embedding response is empty")
	}
	return response.Data[0].Embedding, nil
}

func (s *QdrantService) openAI(ctx context.Context, method string, path string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	baseURL := strings.TrimRight(s.embedding.BaseURL, "/")
	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.embedding.APIKey)
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("embedding %s %s: %s", method, path, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (s *QdrantService) qdrant(ctx context.Context, method string, path string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	baseURL := strings.TrimRight(s.cfg.URL, "/")
	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(s.cfg.APIKey) != "" {
		req.Header.Set("api-key", s.cfg.APIKey)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("qdrant %s %s: %s", method, path, resp.Status)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (s *QdrantService) collection() string {
	if strings.TrimSpace(s.cfg.Collection) == "" {
		return "novelos_character_memories"
	}
	return s.cfg.Collection
}

type embeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

type qdrantSearchResponse struct {
	Result []qdrantPoint `json:"result"`
}

type qdrantPoint struct {
	ID      any            `json:"id"`
	Payload map[string]any `json:"payload"`
}

func (r qdrantSearchResponse) memories(characterID string) []model.Memory {
	out := make([]model.Memory, 0, len(r.Result))
	for _, point := range r.Result {
		content := stringValue(point.Payload, "content")
		if content == "" {
			continue
		}
		memory := model.Memory{
			ID:              firstNonEmpty(stringValue(point.Payload, "memory_id"), fmt.Sprint(point.ID)),
			CharacterID:     characterID,
			Content:         content,
			SourceChapterID: stringValue(point.Payload, "source_chapter_id"),
			SourceRunID:     stringValue(point.Payload, "source_run_id"),
			BranchID:        stringValue(point.Payload, "branch_id"),
			SourceEventID:   stringValue(point.Payload, "source_event_id"),
			Importance:      intValue(point.Payload, "importance"),
			Note:            stringValue(point.Payload, "note"),
			Status:          stringValue(point.Payload, "status"),
			CreatedAt:       parseTime(stringValue(point.Payload, "created_at")),
		}
		if memory.Status == "" {
			memory.Status = "active"
		}
		out = append(out, memory)
	}
	return out
}

func matchCondition(key string, value string) map[string]any {
	return map[string]any{
		"key": key,
		"match": map[string]any{
			"value": value,
		},
	}
}
