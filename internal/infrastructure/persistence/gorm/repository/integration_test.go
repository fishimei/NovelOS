package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/application/service"
	"github.com/fishimei/NovelOS/internal/config"
	gormstore "github.com/fishimei/NovelOS/internal/infrastructure/persistence/gorm"
	persistencemodels "github.com/fishimei/NovelOS/internal/infrastructure/persistence/gorm/models"
	"gorm.io/gorm"
)

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

func testRepos(t *testing.T) (*gorm.DB, port.Repositories, port.TxManager, port.IDGenerator, fixedClock) {
	t.Helper()
	dsn := os.Getenv("NOVEL_OS_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("NOVEL_OS_TEST_POSTGRES_DSN not set")
	}
	clock := fixedClock{now: time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)}
	store, err := gormstore.New(context.Background(), config.PostgresConfig{
		DSN:         dsn,
		AutoMigrate: true,
	})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	db := store.DB()
	resetTables(t, db)
	ids := gormstore.NewIDGenerator()
	repos := New(db, ids, clock).Repositories
	txm := gormstore.NewTxManager(db)
	return db, repos, txm, ids, clock
}

func resetTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	tables := []any{
		&persistencemodels.StateRevision{},
		&persistencemodels.RunEvent{},
		&persistencemodels.StoryRunResult{},
		&persistencemodels.StoryRun{},
		&persistencemodels.StoryMessage{},
		&persistencemodels.StorySession{},
		&persistencemodels.SetupRunResult{},
		&persistencemodels.SetupRun{},
		&persistencemodels.SetupMessage{},
		&persistencemodels.SetupSession{},
		&persistencemodels.RelationshipEvent{},
		&persistencemodels.RelationshipView{},
		&persistencemodels.RelationshipPair{},
		&persistencemodels.Chapter{},
		&persistencemodels.CharacterMemory{},
		&persistencemodels.Character{},
		&persistencemodels.WorldStateEntry{},
		&persistencemodels.AuthorBible{},
		&persistencemodels.Project{},
	}
	for _, table := range tables {
		if err := db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(table).Error; err != nil {
			t.Fatalf("reset table: %v", err)
		}
	}
}

func createProject(t *testing.T, repos port.Repositories) model.Project {
	t.Helper()
	project, err := repos.Projects.Create(context.Background(), model.CreateProjectInput{
		Title:       "Test",
		Genre:       "xuanhuan",
		Description: "desc",
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	return project
}

func TestProjectAuthorBibleCharacterCRUD(t *testing.T) {
	_, repos, _, _, _ := testRepos(t)
	project := createProject(t, repos)

	bible, err := repos.AuthorBibles.Upsert(context.Background(), model.AuthorBible{
		ID:         "bible_test",
		ProjectID:  project.ID,
		Theme:      "成长与代价",
		Status:     "active",
		WorldRules: []string{"魂力有代价"},
	})
	if err != nil {
		t.Fatalf("upsert bible: %v", err)
	}
	if bible.ProjectID != project.ID {
		t.Fatalf("unexpected bible project id: %s", bible.ProjectID)
	}

	character, err := repos.Characters.Create(context.Background(), project.ID, model.CreateCharacterInput{
		Name: "唐三",
		Role: "protagonist",
	})
	if err != nil {
		t.Fatalf("create character: %v", err)
	}
	got, err := repos.Characters.GetByID(context.Background(), character.ID)
	if err != nil {
		t.Fatalf("get character: %v", err)
	}
	if got.Name != "唐三" {
		t.Fatalf("unexpected character name: %s", got.Name)
	}
}

func TestRelationshipPairNormalizationAndViews(t *testing.T) {
	_, repos, _, _, _ := testRepos(t)
	project := createProject(t, repos)
	a, _ := repos.Characters.Create(context.Background(), project.ID, model.CreateCharacterInput{Name: "A", Role: "lead"})
	b, _ := repos.Characters.Create(context.Background(), project.ID, model.CreateCharacterInput{Name: "B", Role: "lead"})

	relationship, err := repos.Relationships.Create(context.Background(), project.ID, model.CreateRelationshipInput{
		CharacterAID: a.ID,
		CharacterBID: b.ID,
		Summary:      "表面盟友",
		CharacterAView: model.RelationshipViewInput{
			PublicAttitude:  "友好",
			PrivateAttitude: "利用",
		},
		CharacterBView: model.RelationshipViewInput{
			PublicAttitude:         "信任",
			BelievedTargetAttitude: "友好",
		},
	})
	if err != nil {
		t.Fatalf("create relationship: %v", err)
	}
	if relationship.Pair.LeftCharacterID > relationship.Pair.RightCharacterID {
		t.Fatalf("pair not normalized")
	}
	if len(relationship.Views) != 2 {
		t.Fatalf("expected 2 views, got %d", len(relationship.Views))
	}
}

func TestSetupApplyRunPersistsFormalState(t *testing.T) {
	_, repos, txm, ids, clock := testRepos(t)
	project := createProject(t, repos)
	applier := service.NewSetupRunApplier(
		repos.SetupSessions,
		repos.AuthorBibles,
		repos.WorldState,
		repos.Characters,
		repos.Relationships,
		repos.Audit,
		txm,
		clock,
		ids,
	)

	session, err := repos.SetupSessions.CreateSession(context.Background(), project.ID, model.CreateSetupSessionInput{SeedIdea: "雨夜重逢"})
	if err != nil {
		t.Fatalf("create setup session: %v", err)
	}
	run, err := repos.SetupSessions.CreateRun(context.Background(), session.ID, model.AdvanceSetupSessionInput{UserMessage: "补全设定"})
	if err != nil {
		t.Fatalf("create setup run: %v", err)
	}
	characterID := "character_existing"
	if err := repos.SetupSessions.SaveRunResult(context.Background(), run.RunID, model.SetupRunResult{
		RunID:     run.RunID,
		SessionID: session.ID,
		Status:    "review_required",
		SetupDraft: model.SetupDraft{
			AuthorBible: model.AuthorBible{
				ID:        "bible_apply",
				ProjectID: project.ID,
				Theme:     "命运与牺牲",
				Status:    "active",
			},
			Characters: []model.Character{
				{ID: characterID, Name: "小舞", Role: "heroine"},
			},
			WorldState: []model.WorldStateEntry{
				{ID: "world_apply", Key: "power_system", Value: map[string]any{"name": "魂力"}},
			},
		},
	}); err != nil {
		t.Fatalf("save setup result: %v", err)
	}

	if _, err := applier.Apply(context.Background(), session.ID, model.ApplySetupRunInput{
		RunID:             run.RunID,
		AcceptAuthorBible: true,
		AcceptCharacters:  true,
		AcceptWorldState:  true,
	}); err != nil {
		t.Fatalf("apply run: %v", err)
	}

	if _, err := repos.AuthorBibles.GetByProjectID(context.Background(), project.ID); err != nil {
		t.Fatalf("author bible missing: %v", err)
	}
	if _, err := repos.Characters.GetByID(context.Background(), characterID); err != nil {
		t.Fatalf("character missing: %v", err)
	}
	world, err := repos.WorldState.ListByProjectID(context.Background(), project.ID)
	if err != nil {
		t.Fatalf("world state list: %v", err)
	}
	if len(world) != 1 {
		t.Fatalf("expected 1 world state entry, got %d", len(world))
	}
	events, err := repos.Audit.ListRunEvents(context.Background(), "setup", run.RunID)
	if err != nil {
		t.Fatalf("list setup events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 setup event, got %d", len(events))
	}
}

func TestRunEventHistoryOrdersBySequence(t *testing.T) {
	_, repos, _, _, clock := testRepos(t)
	project := createProject(t, repos)
	session, err := repos.StorySessions.CreateSession(context.Background(), project.ID, model.CreateStorySessionInput{Title: "History"})
	if err != nil {
		t.Fatalf("create story session: %v", err)
	}
	run, err := repos.StorySessions.CreateRun(context.Background(), session.ID, model.AdvanceStorySessionInput{AuthorMessage: "advance"})
	if err != nil {
		t.Fatalf("create story run: %v", err)
	}
	for _, event := range []model.RunEvent{
		{ID: "event_2", RunKind: "story", RunID: run.RunID, EventName: "second", Sequence: 2, Payload: map[string]any{"step": "second"}, CreatedAt: clock.Now()},
		{ID: "event_1", RunKind: "story", RunID: run.RunID, EventName: "first", Sequence: 1, Payload: map[string]any{"step": "first"}, CreatedAt: clock.Now()},
	} {
		if _, err := repos.Audit.AppendRunEvent(context.Background(), event); err != nil {
			t.Fatalf("append run event: %v", err)
		}
	}

	events, err := repos.Audit.ListRunEvents(context.Background(), "story", run.RunID)
	if err != nil {
		t.Fatalf("list run events: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Sequence != 1 || events[1].Sequence != 2 {
		t.Fatalf("events not ordered by sequence: %+v", events)
	}
}

func TestStoryCommitRunPersistsChapterMemoryRelationshipAndWorldState(t *testing.T) {
	_, repos, txm, ids, clock := testRepos(t)
	project := createProject(t, repos)
	charA, _ := repos.Characters.Create(context.Background(), project.ID, model.CreateCharacterInput{Name: "唐三", Role: "lead"})
	charB, _ := repos.Characters.Create(context.Background(), project.ID, model.CreateCharacterInput{Name: "小舞", Role: "lead"})
	committer := service.NewStoryRunCommitter(
		repos.StorySessions,
		repos.Chapters,
		repos.Memories,
		repos.WorldState,
		repos.Relationships,
		repos.Audit,
		nil,
		txm,
		clock,
		ids,
	)

	if err := repos.WorldState.UpsertEntries(context.Background(), project.ID, []model.WorldStateEntry{
		{ID: "world_weather", Key: "station_weather", Value: "clear", Note: "旧天气", Importance: 5, Volatility: 4, Status: "active"},
	}); err != nil {
		t.Fatalf("seed world state: %v", err)
	}

	session, _ := repos.StorySessions.CreateSession(context.Background(), project.ID, model.CreateStorySessionInput{Title: "第一章"})
	run, _ := repos.StorySessions.CreateRun(context.Background(), session.ID, model.AdvanceStorySessionInput{AuthorMessage: "推进"})
	if err := repos.StorySessions.SaveRunResult(context.Background(), run.RunID, model.StoryRunResult{
		RunID:     run.RunID,
		SessionID: session.ID,
		Status:    "review_required",
		Draft: model.Draft{
			ID:            "draft_1",
			Title:         "第一章 雨夜",
			ChapterNumber: 1,
			Content:       "正文",
			Summary:       "摘要",
			WordCount:     1200,
		},
		MemoryPatch: model.MemoryPatch{
			ID: "patch_1",
			CharacterMemoryUpdates: []model.CharacterMemoryUpdate{
				{CharacterID: charA.ID, Content: "记住重逢", Importance: 8},
			},
			RelationshipUpdates: []model.RelationshipUpdate{
				{
					Pair: &model.RelationshipPair{
						ID:               "pair_story",
						ProjectID:        project.ID,
						LeftCharacterID:  charA.ID,
						RightCharacterID: charB.ID,
						Summary:          "暧昧试探",
						Status:           "active",
					},
					Views: []model.RelationshipViewUpdate{
						{
							ViewID:            "view_story_a",
							SourceCharacterID: charA.ID,
							TargetCharacterID: charB.ID,
							PublicAttitude:    "克制",
							PrivateAttitude:   "动摇",
							MaskingStrategy:   "压抑",
						},
					},
					Events: []model.RelationshipEvent{
						{ID: "event_story", EventType: "beat", Summary: "擦肩试探"},
					},
				},
			},
			WorldStateUpdates: []model.WorldStateUpdate{
				{Key: "station_weather", Operation: "upsert", Value: "storm"},
			},
		},
	}); err != nil {
		t.Fatalf("save story result: %v", err)
	}

	commitResult, err := committer.Commit(context.Background(), run.RunID, model.CommitStoryRunInput{
		DraftID:       "draft_1",
		MemoryPatchID: "patch_1",
		AuthorNote:    "通过",
	})
	if err != nil {
		t.Fatalf("commit run: %v", err)
	}
	if commitResult.Chapter.ChapterNumber != 1 {
		t.Fatalf("unexpected chapter number: %d", commitResult.Chapter.ChapterNumber)
	}
	memories, err := repos.Memories.ListByCharacterID(context.Background(), charA.ID, 10)
	if err != nil {
		t.Fatalf("list memories: %v", err)
	}
	if len(memories) != 1 {
		t.Fatalf("expected 1 memory, got %d", len(memories))
	}
	relationship, err := repos.Relationships.GetByID(context.Background(), "pair_story")
	if err != nil {
		t.Fatalf("get relationship after commit: %v", err)
	}
	if relationship.Pair.Summary != "暧昧试探" {
		t.Fatalf("unexpected relationship summary: %s", relationship.Pair.Summary)
	}
	world, err := repos.WorldState.ListByProjectID(context.Background(), project.ID)
	if err != nil {
		t.Fatalf("list world state: %v", err)
	}
	if len(world) != 1 || world[0].Key != "station_weather" {
		t.Fatalf("unexpected world state: %+v", world)
	}
	if world[0].Importance != 5 || world[0].Volatility != 4 {
		t.Fatalf("expected world state weight to be preserved, got importance=%d volatility=%d", world[0].Importance, world[0].Volatility)
	}
	if _, err := committer.Commit(context.Background(), run.RunID, model.CommitStoryRunInput{
		DraftID:       "draft_1",
		MemoryPatchID: "patch_1",
		AuthorNote:    "重复提交",
	}); err == nil {
		t.Fatalf("expected duplicate commit to be rejected")
	}
}

func TestStoryCommitRunRejectsMismatchedDraftOrPatchID(t *testing.T) {
	_, repos, txm, ids, clock := testRepos(t)
	project := createProject(t, repos)
	committer := service.NewStoryRunCommitter(
		repos.StorySessions,
		repos.Chapters,
		repos.Memories,
		repos.WorldState,
		repos.Relationships,
		repos.Audit,
		nil,
		txm,
		clock,
		ids,
	)

	session, err := repos.StorySessions.CreateSession(context.Background(), project.ID, model.CreateStorySessionInput{Title: "Mismatch"})
	if err != nil {
		t.Fatalf("create story session: %v", err)
	}
	run, err := repos.StorySessions.CreateRun(context.Background(), session.ID, model.AdvanceStorySessionInput{AuthorMessage: "advance"})
	if err != nil {
		t.Fatalf("create story run: %v", err)
	}
	if err := repos.StorySessions.SaveRunResult(context.Background(), run.RunID, model.StoryRunResult{
		RunID:     run.RunID,
		SessionID: session.ID,
		Status:    "review_required",
		Draft: model.Draft{
			ID:            "draft_1",
			Title:         "Mismatch",
			ChapterNumber: 1,
			Content:       "body",
			Summary:       "summary",
			WordCount:     10,
		},
		MemoryPatch: model.MemoryPatch{ID: "patch_1"},
	}); err != nil {
		t.Fatalf("save story result: %v", err)
	}

	// Commit validation must match the reviewed candidate result before any canon writes happen.
	if _, err := committer.Commit(context.Background(), run.RunID, model.CommitStoryRunInput{
		DraftID:       "draft_bad",
		MemoryPatchID: "patch_1",
	}); err == nil {
		t.Fatalf("expected draft id mismatch")
	}
	if _, err := committer.Commit(context.Background(), run.RunID, model.CommitStoryRunInput{
		DraftID:       "draft_1",
		MemoryPatchID: "patch_bad",
	}); err == nil {
		t.Fatalf("expected memory patch id mismatch")
	}

	chapters, err := repos.Chapters.ListByProjectID(context.Background(), project.ID, model.PageInput{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list chapters: %v", err)
	}
	if chapters.Total != 0 {
		t.Fatalf("expected no chapters after rejected commits, got %d", chapters.Total)
	}
}
