package repository

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/application/service"
	"github.com/fishimei/NovelOS/internal/config"
	"github.com/fishimei/NovelOS/internal/domain"
	gormstore "github.com/fishimei/NovelOS/internal/infrastructure/persistence/gorm"
	persistencemodels "github.com/fishimei/NovelOS/internal/infrastructure/persistence/gorm/models"
	"github.com/fishimei/NovelOS/internal/pkgerr"
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
		&persistencemodels.FactionInfluence{},
		&persistencemodels.LocationState{},
		&persistencemodels.MapArea{},
		&persistencemodels.MapTile{},
		&persistencemodels.WorldMap{},
		&persistencemodels.ChapterEventSpan{},
		&persistencemodels.StorySnapshot{},
		&persistencemodels.StoryEvent{},
		&persistencemodels.StoryBranch{},
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
		nil,
		nil,
		service.WorldInitializationSettings{},
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
	run, err := repos.StorySessions.CreateRun(context.Background(), session.ID, model.AdvanceStorySessionInput{})
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
	after, err := repos.Audit.ListRunEventsAfter(context.Background(), "story", run.RunID, 1)
	if err != nil {
		t.Fatalf("list run events after sequence: %v", err)
	}
	if len(after) != 1 || after[0].Sequence != 2 {
		t.Fatalf("expected only sequence 2 after cursor, got %+v", after)
	}
}

func TestStoryRunResultRoundTripsEventFields(t *testing.T) {
	_, repos, _, _, _ := testRepos(t)
	project := createProject(t, repos)
	session, err := repos.StorySessions.CreateSession(context.Background(), project.ID, model.CreateStorySessionInput{Title: "event run"})
	if err != nil {
		t.Fatalf("create story session: %v", err)
	}
	run, err := repos.StorySessions.CreateRun(context.Background(), session.ID, model.AdvanceStorySessionInput{})
	if err != nil {
		t.Fatalf("create story run: %v", err)
	}
	collisionAt := time.Date(2026, 5, 14, 13, 0, 0, 0, time.UTC)
	if err := repos.StorySessions.SaveRunResult(context.Background(), run.RunID, model.StoryRunResult{
		RunID:       run.RunID,
		SessionID:   session.ID,
		Status:      "completed",
		BranchID:    "branch_1",
		BaseEventID: "event_base",
		HeadEventID: "event_head",
		EventPlan: []model.StoryEventPlan{
			{ID: "plan_1", TimeIndex: 1, CharacterID: "character_1", LocationKey: "old_dock", ActionType: "action", Summary: "arrive", TargetActorIDs: []string{"character_2"}},
		},
		Turns: []model.StoryTurn{
			{TurnIndex: 1, ActorID: "character_1", ActionType: "speak", Speech: "arrived", LocationKey: "old_dock"},
		},
		SceneSummary: "scene summary",
		Draft:        model.Draft{ID: "draft_1", Title: "event run", ChapterNumber: 1, Content: "body", Summary: "summary"},
		MemoryPatch:  model.MemoryPatch{ID: "patch_1"},
		Events: []model.StoryEvent{
			{ID: "event_head", Kind: model.EventKindSceneResolved, Summary: "summary"},
		},
		CompletedActions:  []model.OngoingAction{{ID: "action_done", CharacterID: "character_1", Status: "completed"}},
		SupersededActions: []model.OngoingAction{{ID: "action_superseded", CharacterID: "character_2", Status: "superseded"}},
		CollisionAt:       &collisionAt,
	}); err != nil {
		t.Fatalf("save story result: %v", err)
	}

	got, err := repos.StorySessions.GetRunResultByID(context.Background(), run.RunID)
	if err != nil {
		t.Fatalf("get story result: %v", err)
	}
	if got.BaseEventID != "event_base" || got.HeadEventID != "event_head" {
		t.Fatalf("unexpected event ids: %#v", got)
	}
	if len(got.EventPlan) != 1 || got.EventPlan[0].TargetActorIDs[0] != "character_2" {
		t.Fatalf("unexpected event plan: %#v", got.EventPlan)
	}
	if got.SceneSummary != "scene summary" || len(got.Turns) != 1 || got.Turns[0].Speech != "arrived" {
		t.Fatalf("unexpected scene fields: %#v", got)
	}
	if len(got.Events) != 1 || got.Events[0].ID != "event_head" {
		t.Fatalf("unexpected events: %#v", got.Events)
	}
	if len(got.CompletedActions) != 1 || got.CompletedActions[0].ID != "action_done" {
		t.Fatalf("unexpected completed actions: %#v", got.CompletedActions)
	}
	if len(got.SupersededActions) != 1 || got.SupersededActions[0].ID != "action_superseded" {
		t.Fatalf("unexpected superseded actions: %#v", got.SupersededActions)
	}
	if got.CollisionAt == nil || !got.CollisionAt.Equal(collisionAt) {
		t.Fatalf("unexpected collision_at: %#v", got.CollisionAt)
	}
}

func TestStoryRunStopRequestPersists(t *testing.T) {
	_, repos, _, _, _ := testRepos(t)
	project := createProject(t, repos)
	session, err := repos.StorySessions.CreateSession(context.Background(), project.ID, model.CreateStorySessionInput{Title: "stop request"})
	if err != nil {
		t.Fatalf("create story session: %v", err)
	}
	run, err := repos.StorySessions.CreateRun(context.Background(), session.ID, model.AdvanceStorySessionInput{})
	if err != nil {
		t.Fatalf("create story run: %v", err)
	}
	if run.StopRequested {
		t.Fatalf("new run should not have stop requested: %#v", run)
	}
	if err := repos.StorySessions.RequestRunStop(context.Background(), run.RunID); err != nil {
		t.Fatalf("request run stop: %v", err)
	}
	stopped, err := repos.StorySessions.GetRunByID(context.Background(), run.RunID)
	if err != nil {
		t.Fatalf("get stopped run: %v", err)
	}
	if !stopped.StopRequested {
		t.Fatalf("stop request was not persisted: %#v", stopped)
	}
	if stopped.Status != domain.RunStatusCancelled {
		t.Fatalf("queued stop should cancel run, got %#v", stopped)
	}
}

func TestStoryRunLeasePreventsStaleResultOverwrite(t *testing.T) {
	db, repos, _, ids, clock := testRepos(t)
	project := createProject(t, repos)
	session, err := repos.StorySessions.CreateSession(context.Background(), project.ID, model.CreateStorySessionInput{Title: "story"})
	if err != nil {
		t.Fatalf("create story session: %v", err)
	}
	run, err := repos.StorySessions.CreateRun(context.Background(), session.ID, model.AdvanceStorySessionInput{})
	if err != nil {
		t.Fatalf("create story run: %v", err)
	}

	execRepos := New(db, ids, clock)
	work := model.RunExecutionWork{RunKind: port.RunKindStory, RunID: run.RunID}
	lease1 := port.RunLease{Owner: "worker-1", Duration: time.Minute}
	claimed, err := execRepos.ClaimRun(context.Background(), work, lease1, clock.now.Add(-time.Hour))
	if err != nil {
		t.Fatalf("claim lease1: %v", err)
	}
	if !claimed {
		t.Fatal("expected first claim")
	}

	expired := clock.now.Add(-time.Second)
	if err := db.Model(&persistencemodels.StoryRun{}).Where("id = ?", run.RunID).Update("lease_expires_at", expired).Error; err != nil {
		t.Fatalf("expire lease: %v", err)
	}
	lease2 := port.RunLease{Owner: "worker-2", Duration: time.Minute}
	claimed, err = execRepos.ClaimRun(context.Background(), work, lease2, clock.now.Add(-time.Hour))
	if err != nil {
		t.Fatalf("claim lease2: %v", err)
	}
	if !claimed {
		t.Fatal("expected expired run to be reclaimed")
	}

	oldCtx := port.ContextWithRunLease(context.Background(), lease1)
	err = repos.StorySessions.SaveRunResult(oldCtx, run.RunID, model.StoryRunResult{
		RunID:     run.RunID,
		SessionID: session.ID,
		Status:    domain.RunStatusCompleted,
	})
	if !isRunLeaseLostForTest(err) {
		t.Fatalf("SaveRunResult with old lease error = %v, want run lease lost", err)
	}
	current, err := repos.StorySessions.GetRunByID(context.Background(), run.RunID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if current.Status == domain.RunStatusCompleted {
		t.Fatal("stale worker completed a run after losing lease")
	}
}

func isRunLeaseLostForTest(err error) bool {
	var appErr *pkgerr.Error
	return errors.As(err, &appErr) && appErr.Code == pkgerr.CodeConflict && appErr.Message == "run lease lost"
}

func TestStoryEventStoreGenesisForkAndResolveState(t *testing.T) {
	_, repos, _, _, clock := testRepos(t)
	project := createProject(t, repos)
	session, err := repos.StorySessions.CreateSession(context.Background(), project.ID, model.CreateStorySessionInput{Title: "events"})
	if err != nil {
		t.Fatalf("create story session: %v", err)
	}
	genesis, err := repos.StoryEvents.InitGenesis(context.Background(), project.ID, session.ID, model.WorldSnapshot{
		StoryTime: clock.Now(),
		WorldState: map[string]model.WorldStateEntry{
			"weather": {Key: "weather", Value: "clear", Note: "seed"},
		},
		Characters: map[string]model.CharacterRuntimeState{
			"character_1": {CharacterID: "character_1", LocationKey: "town", X: 1, Y: 2, Status: "active"},
		},
		Relationships: map[string]model.Relationship{},
	})
	if err != nil {
		t.Fatalf("init genesis: %v", err)
	}
	branches, err := repos.StoryEvents.ListBranchesBySession(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("list branches: %v", err)
	}
	if len(branches) != 1 || branches[0].HeadEventID != genesis.ID {
		t.Fatalf("unexpected branches: %#v", branches)
	}
	resolved, err := repos.StoryEvents.ResolveStateAt(context.Background(), genesis.ID)
	if err != nil {
		t.Fatalf("resolve genesis: %v", err)
	}
	if resolved.WorldState["weather"].Value != "clear" || resolved.Characters["character_1"].LocationKey != "town" {
		t.Fatalf("unexpected resolved state: %#v", resolved)
	}
	child, err := repos.StoryEvents.AppendEvent(context.Background(), model.StoryEvent{
		ProjectID:     project.ID,
		SessionID:     session.ID,
		BranchID:      branches[0].ID,
		ParentEventID: genesis.ID,
		StoryTime:     clock.Now().Add(time.Hour),
		Kind:          model.EventKindSceneResolved,
		Summary:       "rain starts",
		StateDelta: model.EventStateDelta{MemoryPatch: model.MemoryPatch{WorldStateUpdates: []model.WorldStateUpdate{
			{Key: "weather", Operation: "set", Value: "rain"},
		}}},
		CreatedAt: clock.Now(),
	})
	if err != nil {
		t.Fatalf("append event: %v", err)
	}
	if err := repos.StoryEvents.UpdateBranchHead(context.Background(), branches[0].ID, child.ID); err != nil {
		t.Fatalf("update branch head: %v", err)
	}
	resolved, err = repos.StoryEvents.ResolveStateAt(context.Background(), child.ID)
	if err != nil {
		t.Fatalf("resolve child: %v", err)
	}
	if resolved.WorldState["weather"].Value != "rain" {
		t.Fatalf("expected weather rain, got %#v", resolved.WorldState["weather"])
	}
	fork, err := repos.StoryEvents.CreateBranch(context.Background(), model.Branch{ProjectID: project.ID, SessionID: session.ID, Name: "fork", BaseEventID: genesis.ID, HeadEventID: genesis.ID, Status: "active", CreatedAt: clock.Now(), UpdatedAt: clock.Now()})
	if err != nil {
		t.Fatalf("fork branch: %v", err)
	}
	if fork.BaseEventID != genesis.ID || fork.HeadEventID != genesis.ID {
		t.Fatalf("unexpected fork: %#v", fork)
	}
}

func TestStoryEventStoreInFlightActionsStopAtCompletionEvent(t *testing.T) {
	_, repos, _, _, clock := testRepos(t)
	project := createProject(t, repos)
	session, err := repos.StorySessions.CreateSession(context.Background(), project.ID, model.CreateStorySessionInput{Title: "in-flight actions"})
	if err != nil {
		t.Fatalf("create story session: %v", err)
	}
	genesis, err := repos.StoryEvents.InitGenesis(context.Background(), project.ID, session.ID, model.WorldSnapshot{
		StoryTime:     clock.Now(),
		WorldState:    map[string]model.WorldStateEntry{},
		Characters:    map[string]model.CharacterRuntimeState{},
		Relationships: map[string]model.Relationship{},
	})
	if err != nil {
		t.Fatalf("init genesis: %v", err)
	}
	branches, err := repos.StoryEvents.ListBranchesBySession(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("list branches: %v", err)
	}
	branch := branches[0]
	start := clock.Now()
	action := model.OngoingAction{
		CharacterID:       "character_1",
		ActionType:        "observe",
		Description:       "Lin watches the dock",
		TargetLocationKey: "old_dock",
		StartAt:           start,
		ArriveAt:          start,
		EffectAt:          start,
		EndsAt:            start.Add(time.Hour),
		ResourceKeys:      []string{"character:character_1", "location:old_dock"},
		Status:            "ongoing",
	}
	scheduledInput := service.StoryEventFromAction(branch, action, genesis.ID)
	scheduledInput.CreatedAt = clock.Now()
	scheduled, err := repos.StoryEvents.AppendEvent(context.Background(), scheduledInput)
	if err != nil {
		t.Fatalf("append scheduled action: %v", err)
	}
	completionInput := service.StoryEventFromActionCompletion(branch, action, scheduled.ID)
	completionInput.CreatedAt = clock.Now()
	if _, err := repos.StoryEvents.AppendEvent(context.Background(), completionInput); err != nil {
		t.Fatalf("append completed action: %v", err)
	}

	inFlight, err := repos.StoryEvents.InFlightActionsAt(context.Background(), branch.ID, start.Add(30*time.Minute))
	if err != nil {
		t.Fatalf("in-flight before completion: %v", err)
	}
	if len(inFlight) != 1 || inFlight[0].CharacterID != "character_1" {
		t.Fatalf("expected action in flight before completion, got %#v", inFlight)
	}
	inFlight, err = repos.StoryEvents.InFlightActionsAt(context.Background(), branch.ID, start.Add(time.Hour))
	if err != nil {
		t.Fatalf("in-flight at completion: %v", err)
	}
	if len(inFlight) != 0 {
		t.Fatalf("expected action released at completion, got %#v", inFlight)
	}
}

func TestStoryChapterCutCreatesChapterSpanWithoutStateWrites(t *testing.T) {
	_, repos, txm, ids, clock := testRepos(t)
	project := createProject(t, repos)
	session, err := repos.StorySessions.CreateSession(context.Background(), project.ID, model.CreateStorySessionInput{Title: "cut"})
	if err != nil {
		t.Fatalf("create story session: %v", err)
	}
	genesis, err := repos.StoryEvents.InitGenesis(context.Background(), project.ID, session.ID, model.WorldSnapshot{StoryTime: clock.Now(), WorldState: map[string]model.WorldStateEntry{}, Characters: map[string]model.CharacterRuntimeState{}, Relationships: map[string]model.Relationship{}})
	if err != nil {
		t.Fatalf("init genesis: %v", err)
	}
	branches, err := repos.StoryEvents.ListBranchesBySession(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("list branches: %v", err)
	}
	run, err := repos.StorySessions.CreateRun(context.Background(), session.ID, model.AdvanceStorySessionInput{BranchID: branches[0].ID, BaseEventID: genesis.ID})
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	scene, err := repos.StoryEvents.AppendEvent(context.Background(), model.StoryEvent{
		ProjectID:     project.ID,
		SessionID:     session.ID,
		BranchID:      branches[0].ID,
		ParentEventID: genesis.ID,
		StoryTime:     clock.Now().Add(time.Hour),
		Kind:          model.EventKindSceneResolved,
		Summary:       "scene summary",
		Payload:       map[string]any{"draft": model.Draft{Title: "Cut title", ChapterNumber: 1, Content: "Scene body.", Summary: "scene summary", WordCount: 11}},
		StateDelta:    model.EventStateDelta{MemoryPatch: model.MemoryPatch{CharacterMemoryUpdates: []model.CharacterMemoryUpdate{{CharacterID: "character_1", Content: "saw rain", Importance: 5}}}},
		CreatedAt:     clock.Now(),
	})
	if err != nil {
		t.Fatalf("append scene: %v", err)
	}
	if err := repos.StoryEvents.UpdateBranchHead(context.Background(), branches[0].ID, scene.ID); err != nil {
		t.Fatalf("update branch head: %v", err)
	}
	if err := repos.StorySessions.UpdateRunHead(context.Background(), run.RunID, scene.ID); err != nil {
		t.Fatalf("update run head: %v", err)
	}
	cutter := service.NewStoryChapterCutter(repos.StorySessions, repos.StoryEvents, repos.Chapters, repos.Audit, nil, txm, clock, ids)
	cut, err := cutter.CutChapter(context.Background(), run.RunID, model.CutChapterInput{BranchID: branches[0].ID, FromEventID: genesis.ID, ToEventID: scene.ID, AuthorNote: "publish"})
	if err != nil {
		t.Fatalf("cut chapter: %v", err)
	}
	if cut.Chapter.Title != "Cut title" || cut.Span.ToEventID != scene.ID {
		t.Fatalf("unexpected cut result: %#v", cut)
	}
	if _, err := cutter.CutChapter(context.Background(), run.RunID, model.CutChapterInput{BranchID: branches[0].ID, FromEventID: genesis.ID, ToEventID: scene.ID}); err == nil {
		t.Fatalf("expected duplicate event span cut to fail")
	}
}
