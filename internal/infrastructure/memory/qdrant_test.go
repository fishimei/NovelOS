package memory

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/config"
)

func TestQdrantRecallEmbedsAndSearchesWithScopedFilter(t *testing.T) {
	embeddingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" {
			t.Fatalf("unexpected embedding path %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer embed-key" {
			t.Fatalf("unexpected embedding auth %q", got)
		}
		_, _ = w.Write([]byte(`{"data":[{"embedding":[0.1,0.2,0.3]}]}`))
	}))
	defer embeddingServer.Close()

	var searchRequest map[string]any
	qdrantServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/collections/memories/points/search" {
			t.Fatalf("unexpected qdrant path %s", r.URL.Path)
		}
		if got := r.Header.Get("api-key"); got != "qdrant-key" {
			t.Fatalf("unexpected qdrant api key %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&searchRequest); err != nil {
			t.Fatalf("decode search request: %v", err)
		}
		_, _ = w.Write([]byte(`{"result":[{"id":"memory_1","payload":{"content":"雨夜背叛留下旧伤","memory_id":"memory_1","importance":9,"status":"active"}}]}`))
	}))
	defer qdrantServer.Close()

	service := NewQdrantService(
		config.QdrantConfig{URL: qdrantServer.URL, APIKey: "qdrant-key", Collection: "memories"},
		config.EmbeddingConfig{BaseURL: embeddingServer.URL, APIKey: "embed-key", Model: "text-embedding-test"},
	)
	memories, err := service.Recall(context.Background(), port.CharacterMemoryRecallInput{ProjectID: "project_1", CharacterID: "character_1", Query: "雨夜"})
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if len(memories) != 1 || memories[0].Content != "雨夜背叛留下旧伤" || memories[0].Importance != 9 {
		t.Fatalf("unexpected memories: %#v", memories)
	}
	filter := searchRequest["filter"].(map[string]any)
	must := filter["must"].([]any)
	if len(must) != 3 {
		t.Fatalf("expected project/character/canon filters, got %#v", filter)
	}
}

func TestQdrantCommitEmbedsAndUpsertsCanonicalPoints(t *testing.T) {
	embeddingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"embedding":[0.4,0.5]}]}`))
	}))
	defer embeddingServer.Close()

	var upsertRequest map[string]any
	qdrantServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/collections/memories/points" || r.URL.Query().Get("wait") != "true" {
			t.Fatalf("unexpected qdrant request %s %s", r.Method, r.URL.String())
		}
		if err := json.NewDecoder(r.Body).Decode(&upsertRequest); err != nil {
			t.Fatalf("decode upsert request: %v", err)
		}
		_, _ = w.Write([]byte(`{"result":{"status":"completed"}}`))
	}))
	defer qdrantServer.Close()

	service := NewQdrantService(
		config.QdrantConfig{URL: qdrantServer.URL, Collection: "memories"},
		config.EmbeddingConfig{BaseURL: embeddingServer.URL, APIKey: "embed-key", Model: "text-embedding-test"},
	)
	err := service.Commit(context.Background(), port.CharacterMemoryCommitInput{
		ProjectID: "project_1",
		RunID:     "run_1",
		Chapter:   model.Chapter{ID: "chapter_1", ChapterNumber: 4},
		Memories:  []model.Memory{{ID: "memory_1", CharacterID: "character_1", Content: "记住暗门。", SourceChapterID: "chapter_1", Importance: 6, Status: "active"}},
	})
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	points := upsertRequest["points"].([]any)
	point := points[0].(map[string]any)
	payload := point["payload"].(map[string]any)
	if point["id"] != "memory_1" || payload["project_id"] != "project_1" || payload["canon_status"] != "committed" || payload["importance"].(float64) != 6 {
		t.Fatalf("unexpected point: %#v", point)
	}
}
