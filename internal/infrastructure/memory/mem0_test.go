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

func TestMem0RecallSendsScopedSearchRequest(t *testing.T) {
	var request map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/memories/search/" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Token test-key" {
			t.Fatalf("unexpected authorization header %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"results":[{"id":"remote_1","memory":"记得雨夜背叛","metadata":{"importance":8,"source_chapter_id":"chapter_1"}}]}`))
	}))
	defer server.Close()

	service := NewMem0Service(config.Mem0Config{BaseURL: server.URL, APIKey: "test-key", AppID: "novelos-test", TopK: 9, Rerank: true})
	memories, err := service.Recall(context.Background(), port.CharacterMemoryRecallInput{ProjectID: "project_1", CharacterID: "character_1", Query: "雨夜重逢"})
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if len(memories) != 1 || memories[0].Content != "记得雨夜背叛" || memories[0].Importance != 8 {
		t.Fatalf("unexpected memories: %#v", memories)
	}
	if request["query"] != "雨夜重逢" || request["top_k"].(float64) != 9 || request["rerank"] != true {
		t.Fatalf("unexpected request: %#v", request)
	}
	filters := request["filters"].(map[string]any)
	and := filters["AND"].([]any)
	if len(and) != 3 {
		t.Fatalf("expected scoped filters, got %#v", filters)
	}
}

func TestMem0CommitSendsCanonicalMemoryAddRequest(t *testing.T) {
	var request map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/memories/add/" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"event_id":"event_1","status":"PENDING"}`))
	}))
	defer server.Close()

	service := NewMem0Service(config.Mem0Config{BaseURL: server.URL, APIKey: "test-key", AppID: "novelos-test"})
	err := service.Commit(context.Background(), port.CharacterMemoryCommitInput{
		ProjectID: "project_1",
		RunID:     "run_1",
		Chapter:   model.Chapter{ID: "chapter_1", ChapterNumber: 3},
		Memories: []model.Memory{{
			ID:              "memory_1",
			CharacterID:     "character_1",
			Content:         "林澈记住了雨夜试探。",
			SourceChapterID: "chapter_1",
			Importance:      7,
			Status:          "active",
		}},
	})
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	if request["agent_id"] != "character_1" || request["app_id"] != "novelos-test" || request["infer"] != false {
		t.Fatalf("unexpected request identity: %#v", request)
	}
	messages := request["messages"].([]any)
	message := messages[0].(map[string]any)
	if message["role"] != "system" || message["content"] != "林澈记住了雨夜试探。" {
		t.Fatalf("unexpected messages: %#v", messages)
	}
	metadata := request["metadata"].(map[string]any)
	if metadata["project_id"] != "project_1" || metadata["canon_status"] != "committed" || metadata["importance"].(float64) != 7 {
		t.Fatalf("unexpected metadata: %#v", metadata)
	}
}
