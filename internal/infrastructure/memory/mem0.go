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

type Mem0Service struct {
	cfg    config.Mem0Config
	client *http.Client
}

func NewMem0Service(cfg config.Mem0Config) *Mem0Service {
	return &Mem0Service{
		cfg:    cfg,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *Mem0Service) Recall(ctx context.Context, input port.CharacterMemoryRecallInput) ([]model.Memory, error) {
	if !s.enabled() || strings.TrimSpace(input.Query) == "" {
		return nil, nil
	}
	limit := input.Limit
	if limit <= 0 {
		limit = s.cfg.TopK
	}
	if limit <= 0 {
		limit = 12
	}
	body := map[string]any{
		"query": input.Query,
		"filters": map[string]any{
			"AND": []map[string]any{
				{"agent_id": input.CharacterID},
				{"app_id": appID(s.cfg)},
				{"project_id": input.ProjectID},
			},
		},
		"top_k":  limit,
		"rerank": s.cfg.Rerank,
	}
	var response mem0SearchResponse
	if err := s.do(ctx, http.MethodPost, "/v3/memories/search/", body, &response); err != nil {
		return nil, err
	}
	return response.memories(input.CharacterID), nil
}

func (s *Mem0Service) Commit(ctx context.Context, input port.CharacterMemoryCommitInput) error {
	if !s.enabled() || len(input.Memories) == 0 {
		return nil
	}
	for _, memory := range input.Memories {
		body := map[string]any{
			"agent_id": memory.CharacterID,
			"app_id":   appID(s.cfg),
			"messages": []map[string]string{{
				"role":    "system",
				"content": memory.Content,
			}},
			"metadata": map[string]any{
				"project_id":        input.ProjectID,
				"run_id":            input.RunID,
				"chapter_id":        input.Chapter.ID,
				"chapter_number":    input.Chapter.ChapterNumber,
				"memory_id":         memory.ID,
				"character_id":      memory.CharacterID,
				"source_chapter_id": memory.SourceChapterID,
				"importance":        memory.Importance,
				"status":            memory.Status,
				"canon_status":      "committed",
			},
			"infer": false,
		}
		if err := s.do(ctx, http.MethodPost, "/v3/memories/add/", body, nil); err != nil {
			return err
		}
	}
	return nil
}

func (s *Mem0Service) enabled() bool {
	return strings.TrimSpace(s.cfg.APIKey) != "" && strings.TrimSpace(s.cfg.BaseURL) != ""
}

func (s *Mem0Service) do(ctx context.Context, method string, path string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	baseURL := strings.TrimRight(s.cfg.BaseURL, "/")
	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+s.cfg.APIKey)
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("mem0 %s %s: %s", method, path, resp.Status)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

type mem0SearchResponse struct {
	Results  []mem0Memory `json:"results"`
	Memories []mem0Memory `json:"memories"`
}

type mem0Memory struct {
	ID        string         `json:"id"`
	Memory    string         `json:"memory"`
	Text      string         `json:"text"`
	Content   string         `json:"content"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt string         `json:"created_at"`
	UpdatedAt string         `json:"updated_at"`
}

func (r mem0SearchResponse) memories(characterID string) []model.Memory {
	items := r.Results
	if len(items) == 0 {
		items = r.Memories
	}
	out := make([]model.Memory, 0, len(items))
	for _, item := range items {
		content := firstNonEmpty(item.Memory, item.Text, item.Content)
		if content == "" {
			continue
		}
		memory := model.Memory{
			ID:          item.ID,
			CharacterID: characterID,
			Content:     content,
			Status:      stringValue(item.Metadata, "status"),
			Note:        stringValue(item.Metadata, "note"),
			Importance:  intValue(item.Metadata, "importance"),
		}
		memory.SourceChapterID = stringValue(item.Metadata, "source_chapter_id")
		if memory.SourceChapterID == "" {
			memory.SourceChapterID = stringValue(item.Metadata, "chapter_id")
		}
		if memory.Status == "" {
			memory.Status = "active"
		}
		memory.CreatedAt = parseTime(firstNonEmpty(item.CreatedAt, item.UpdatedAt))
		out = append(out, memory)
	}
	return out
}

func appID(cfg config.Mem0Config) string {
	if strings.TrimSpace(cfg.AppID) == "" {
		return "novelos"
	}
	return cfg.AppID
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func stringValue(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, ok := values[key]
	if !ok {
		return ""
	}
	s, ok := value.(string)
	if ok {
		return s
	}
	return fmt.Sprint(value)
}

func intValue(values map[string]any, key string) int {
	if values == nil {
		return 0
	}
	switch value := values[key].(type) {
	case float64:
		return int(value)
	case int:
		return value
	case json.Number:
		out, _ := value.Int64()
		return int(out)
	default:
		return 0
	}
}

func parseTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}
