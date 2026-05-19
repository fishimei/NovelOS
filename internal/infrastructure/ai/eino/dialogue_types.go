package eino

import "github.com/fishimei/NovelOS/internal/application/model"

type DialogueContextInput struct {
	ProjectID string `json:"project_id"`
	SessionID string `json:"session_id"`
}

type DialogueContextSnapshot struct {
	Project            model.ProjectDetail          `json:"project"`
	HasAuthorBible     bool                         `json:"has_author_bible"`
	AuthorBibleSummary string                       `json:"author_bible_summary"`
	WorldStateCount    int                          `json:"world_state_count"`
	CharacterCount     int                          `json:"character_count"`
	RelationshipCount  int                          `json:"relationship_count"`
	RecentChapters     []model.Chapter              `json:"recent_chapters"`
	SetupSessions      []model.SetupSession         `json:"setup_sessions"`
	StorySessions      []model.StorySession         `json:"story_sessions"`
	PendingOptions     []model.DialogueActionOption `json:"pending_options"`
}

type InspectSetupRunResultInput struct {
	RunID string `json:"run_id"`
}

type SetupRunResultInspection struct {
	RunID                      string `json:"run_id"`
	SessionID                  string `json:"session_id"`
	ProjectID                  string `json:"project_id"`
	Status                     string `json:"status"`
	AssistantSummary           string `json:"assistant_summary"`
	CharacterCount             int    `json:"character_count"`
	RelationshipCount          int    `json:"relationship_count"`
	WorldStateCount            int    `json:"world_state_count"`
	AcceptAuthorBibleDefault   bool   `json:"accept_author_bible_default"`
	AcceptCharactersDefault    bool   `json:"accept_characters_default"`
	AcceptRelationshipsDefault bool   `json:"accept_relationships_default"`
	AcceptWorldStateDefault    bool   `json:"accept_world_state_default"`
}

type InspectStoryRunResultInput struct {
	RunID string `json:"run_id"`
}

type StoryRunResultInspection struct {
	RunID         string `json:"run_id"`
	SessionID     string `json:"session_id"`
	ProjectID     string `json:"project_id"`
	Status        string `json:"status"`
	DraftID       string `json:"draft_id"`
	MemoryPatchID string `json:"memory_patch_id"`
	Title         string `json:"title"`
	Summary       string `json:"summary"`
	WordCount     int    `json:"word_count"`
	Committed     bool   `json:"committed"`
}

type ListPendingDialogueOptionsInput struct {
	SessionID string `json:"session_id"`
}

type ProposeSetupStartAndAdvanceInput struct {
	SeedIdea    string `json:"seed_idea"`
	UserMessage string `json:"user_message"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Rationale   string `json:"rationale"`
}

type ProposeSetupAdvanceInput struct {
	SetupSessionID string `json:"setup_session_id"`
	UserMessage    string `json:"user_message"`
	Label          string `json:"label"`
	Description    string `json:"description"`
	Rationale      string `json:"rationale"`
}

type ProposeSetupApplyInput struct {
	SetupSessionID      string `json:"setup_session_id"`
	SetupRunID          string `json:"setup_run_id"`
	AcceptAuthorBible   bool   `json:"accept_author_bible"`
	AcceptCharacters    bool   `json:"accept_characters"`
	AcceptRelationships bool   `json:"accept_relationships"`
	AcceptWorldState    bool   `json:"accept_world_state"`
	AuthorNote          string `json:"author_note"`
	Label               string `json:"label"`
	Description         string `json:"description"`
	Rationale           string `json:"rationale"`
}

type ProposeStoryCreateAndAdvanceInput struct {
	Title            string `json:"title"`
	OpeningSituation string `json:"opening_situation"`
	AuthorIntent     string `json:"author_intent"`
	AuthorMessage    string `json:"author_message"`
	Label            string `json:"label"`
	Description      string `json:"description"`
	Rationale        string `json:"rationale"`
}

type ProposeStoryAdvanceInput struct {
	StorySessionID string `json:"story_session_id"`
	AuthorMessage  string `json:"author_message"`
	Label          string `json:"label"`
	Description    string `json:"description"`
	Rationale      string `json:"rationale"`
}

type ProposeStoryCommitInput struct {
	StoryRunID    string `json:"story_run_id"`
	DraftID       string `json:"draft_id"`
	MemoryPatchID string `json:"memory_patch_id"`
	AuthorNote    string `json:"author_note"`
	Label         string `json:"label"`
	Description   string `json:"description"`
	Rationale     string `json:"rationale"`
}

type ExecuteConfirmedDialogueActionInput struct {
	OptionID string `json:"option_id"`
}

type FinalizeDialogueResponseInput struct {
	AssistantMessage    string                   `json:"assistant_message"`
	ClarifyingQuestions []model.DialogueQuestion `json:"clarifying_questions"`
	SuggestedReplies    []string                 `json:"suggested_replies"`
	ContextSummary      string                   `json:"context_summary"`
}

type dialogueToolResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	Option  any    `json:"option,omitempty"`
}
