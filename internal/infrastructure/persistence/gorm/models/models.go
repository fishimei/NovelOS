// Package models 包含 GORM 数据库模型。
// 这些模型直接映射到数据库表结构，用于持久化存储。
package models

import "time"

// Project 是项目数据库模型。
type Project struct {
	ID          string    `gorm:"primaryKey;size:64"`
	Title       string    `gorm:"not null"`
	Genre       string    `gorm:"not null;default:''"`
	Description string    `gorm:"not null;default:''"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
}

func (Project) TableName() string { return "projects" }

type AuthorBible struct {
	ID                  string    `gorm:"primaryKey;size:64"`
	ProjectID           string    `gorm:"not null;uniqueIndex"`
	Theme               string    `gorm:"not null;default:''"`
	StyleGuide          string    `gorm:"not null;default:''"`
	WorldRulesJSON      JSONB     `gorm:"type:jsonb;not null"`
	AestheticJSON       JSONB     `gorm:"type:jsonb;not null"`
	HardConstraintsJSON JSONB     `gorm:"type:jsonb;not null"`
	SoftPreferencesJSON JSONB     `gorm:"type:jsonb;not null"`
	ForbiddenMovesJSON  JSONB     `gorm:"type:jsonb;not null"`
	Status              string    `gorm:"not null;default:''"`
	UpdatedAt           time.Time `gorm:"not null"`
}

func (AuthorBible) TableName() string { return "author_bibles" }

type WorldStateEntry struct {
	ID         string    `gorm:"primaryKey;size:64"`
	ProjectID  string    `gorm:"not null;index:idx_world_state_project_key,priority:1"`
	Key        string    `gorm:"not null;index:idx_world_state_project_key,priority:2,unique"`
	ValueJSON  JSONB     `gorm:"type:jsonb;not null"`
	Note       string    `gorm:"not null;default:''"`
	Status     string    `gorm:"not null;default:''"`
	Importance int       `gorm:"not null;default:0"`
	Volatility int       `gorm:"not null;default:0"`
	UpdatedAt  time.Time `gorm:"not null"`
}

func (WorldStateEntry) TableName() string { return "world_state_entries" }

type Character struct {
	ID                  string    `gorm:"primaryKey;size:64"`
	ProjectID           string    `gorm:"not null;index"`
	Name                string    `gorm:"not null"`
	Role                string    `gorm:"not null;default:''"`
	Profile             string    `gorm:"not null;default:''"`
	Personality         string    `gorm:"not null;default:''"`
	VoiceStyle          string    `gorm:"not null;default:''"`
	GoalsJSON           JSONB     `gorm:"type:jsonb;not null"`
	FearsJSON           JSONB     `gorm:"type:jsonb;not null"`
	SecretsJSON         JSONB     `gorm:"type:jsonb;not null"`
	ConstraintsJSON     JSONB     `gorm:"type:jsonb;not null"`
	RecentMemorySummary string    `gorm:"not null;default:''"`
	Status              string    `gorm:"not null;default:''"`
	CreatedAt           time.Time `gorm:"not null"`
	UpdatedAt           time.Time `gorm:"not null"`
}

func (Character) TableName() string { return "characters" }

type RelationshipPair struct {
	ID                string    `gorm:"primaryKey;size:64"`
	ProjectID         string    `gorm:"not null;index:idx_relationship_pair_unique,priority:1"`
	LeftCharacterID   string    `gorm:"not null;index:idx_relationship_pair_unique,priority:2,unique"`
	RightCharacterID  string    `gorm:"not null;index:idx_relationship_pair_unique,priority:3,unique"`
	Summary           string    `gorm:"not null;default:''"`
	AnchorsJSON       JSONB     `gorm:"type:jsonb;not null"`
	TensionJSON       JSONB     `gorm:"type:jsonb;not null"`
	SharedHistoryJSON JSONB     `gorm:"type:jsonb;not null"`
	Volatility        int       `gorm:"not null;default:0"`
	Status            string    `gorm:"not null;default:''"`
	CreatedAt         time.Time `gorm:"not null"`
	UpdatedAt         time.Time `gorm:"not null"`
}

func (RelationshipPair) TableName() string { return "relationship_pairs" }

type RelationshipView struct {
	ID                     string    `gorm:"primaryKey;size:64"`
	ProjectID              string    `gorm:"not null;index:idx_relationship_view_unique,priority:1"`
	PairID                 string    `gorm:"not null;index"`
	SourceCharacterID      string    `gorm:"not null;index:idx_relationship_view_unique,priority:2,unique"`
	TargetCharacterID      string    `gorm:"not null;index:idx_relationship_view_unique,priority:3,unique"`
	PublicAttitude         string    `gorm:"not null;default:''"`
	PrivateAttitude        string    `gorm:"not null;default:''"`
	BelievedTargetAttitude string    `gorm:"not null;default:''"`
	MaskingStrategy        string    `gorm:"not null;default:''"`
	Status                 string    `gorm:"not null;default:''"`
	CreatedAt              time.Time `gorm:"not null"`
	UpdatedAt              time.Time `gorm:"not null"`
}

func (RelationshipView) TableName() string { return "relationship_views" }

type RelationshipEvent struct {
	ID          string    `gorm:"primaryKey;size:64"`
	ProjectID   string    `gorm:"not null;index"`
	PairID      string    `gorm:"not null;index"`
	EventType   string    `gorm:"not null;default:''"`
	Summary     string    `gorm:"not null;default:''"`
	PayloadJSON JSONB     `gorm:"type:jsonb;not null"`
	CreatedAt   time.Time `gorm:"not null"`
}

func (RelationshipEvent) TableName() string { return "relationship_events" }

type CharacterMemory struct {
	ID              string    `gorm:"primaryKey;size:64"`
	CharacterID     string    `gorm:"not null;index"`
	Content         string    `gorm:"not null"`
	SourceChapterID string    `gorm:"not null;default:''"`
	Importance      int       `gorm:"not null;default:0"`
	Note            string    `gorm:"not null;default:''"`
	Status          string    `gorm:"not null;default:''"`
	CreatedAt       time.Time `gorm:"not null"`
}

func (CharacterMemory) TableName() string { return "character_memories" }

type Chapter struct {
	ID            string    `gorm:"primaryKey;size:64"`
	ProjectID     string    `gorm:"not null;index:idx_chapter_unique,priority:1"`
	ChapterNumber int       `gorm:"not null;index:idx_chapter_unique,priority:2,unique"`
	Title         string    `gorm:"not null;default:''"`
	Summary       string    `gorm:"not null;default:''"`
	Content       string    `gorm:"not null"`
	AuthorNote    string    `gorm:"not null;default:''"`
	Status        string    `gorm:"not null;default:''"`
	WordCount     int       `gorm:"not null;default:0"`
	CommittedAt   time.Time `gorm:"not null"`
}

func (Chapter) TableName() string { return "chapters" }

type SetupSession struct {
	ID              string    `gorm:"primaryKey;size:64"`
	ProjectID       string    `gorm:"not null;index"`
	SeedIdea        string    `gorm:"not null"`
	LastUserMessage string    `gorm:"not null;default:''"`
	Status          string    `gorm:"not null;default:''"`
	CreatedAt       time.Time `gorm:"not null"`
	UpdatedAt       time.Time `gorm:"not null"`
}

func (SetupSession) TableName() string { return "setup_sessions" }

type SetupMessage struct {
	ID        string    `gorm:"primaryKey;size:64"`
	SessionID string    `gorm:"not null;index"`
	Role      string    `gorm:"not null"`
	Content   string    `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null"`
}

func (SetupMessage) TableName() string { return "setup_messages" }

type SetupRun struct {
	ID          string    `gorm:"primaryKey;size:64"`
	SessionID   string    `gorm:"not null;index"`
	ProjectID   string    `gorm:"not null;index"`
	Status      string    `gorm:"not null;default:''"`
	CurrentStep string    `gorm:"not null;default:''"`
	Progress    int       `gorm:"not null;default:0"`
	Error       string    `gorm:"not null;default:''"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
}

func (SetupRun) TableName() string { return "setup_runs" }

type SetupRunResult struct {
	ID          string    `gorm:"primaryKey;size:64"`
	RunID       string    `gorm:"not null;uniqueIndex"`
	SessionID   string    `gorm:"not null;index"`
	Status      string    `gorm:"not null;default:''"`
	PayloadJSON JSONB     `gorm:"type:jsonb;not null"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
}

func (SetupRunResult) TableName() string { return "setup_run_results" }

type StorySession struct {
	ID                         string    `gorm:"primaryKey;size:64"`
	ProjectID                  string    `gorm:"not null;index"`
	Title                      string    `gorm:"not null;default:''"`
	OpeningSituation           string    `gorm:"not null;default:''"`
	AuthorIntent               string    `gorm:"not null;default:''"`
	LastAuthorMessage          string    `gorm:"not null;default:''"`
	Status                     string    `gorm:"not null;default:''"`
	CurrentPlotVariableSummary string    `gorm:"not null;default:''"`
	CreatedAt                  time.Time `gorm:"not null"`
	UpdatedAt                  time.Time `gorm:"not null"`
}

func (StorySession) TableName() string { return "story_sessions" }

type StoryMessage struct {
	ID        string    `gorm:"primaryKey;size:64"`
	SessionID string    `gorm:"not null;index"`
	Role      string    `gorm:"not null"`
	Content   string    `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null"`
}

func (StoryMessage) TableName() string { return "story_messages" }

type StoryRun struct {
	ID            string `gorm:"primaryKey;size:64"`
	SessionID     string `gorm:"not null;index"`
	ProjectID     string `gorm:"not null;index"`
	BranchID      string `gorm:"not null;default:'';index"`
	BaseEventID   string `gorm:"not null;default:'';index"`
	HeadEventID   string `gorm:"not null;default:'';index"`
	Status        string `gorm:"not null;default:''"`
	CurrentStep   string `gorm:"not null;default:''"`
	Progress      int    `gorm:"not null;default:0"`
	Error         string `gorm:"not null;default:''"`
	StopRequested bool   `gorm:"not null;default:false"`
	CutAt         *time.Time
	CreatedAt     time.Time `gorm:"not null"`
	UpdatedAt     time.Time `gorm:"not null"`
}

func (StoryRun) TableName() string { return "story_runs" }

type StoryRunResult struct {
	ID          string    `gorm:"primaryKey;size:64"`
	RunID       string    `gorm:"not null;uniqueIndex"`
	SessionID   string    `gorm:"not null;index"`
	Status      string    `gorm:"not null;default:''"`
	PayloadJSON JSONB     `gorm:"type:jsonb;not null"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
}

func (StoryRunResult) TableName() string { return "story_run_results" }

type StoryBranch struct {
	ID                       string    `gorm:"primaryKey;size:64"`
	ProjectID                string    `gorm:"not null;index"`
	SessionID                string    `gorm:"not null;index"`
	Name                     string    `gorm:"not null;default:''"`
	BaseEventID              string    `gorm:"not null;default:'';index"`
	HeadEventID              string    `gorm:"not null;default:'';index"`
	PublishedFrontierEventID string    `gorm:"not null;default:'';index"`
	Status                   string    `gorm:"not null;default:''"`
	CreatedAt                time.Time `gorm:"not null"`
	UpdatedAt                time.Time `gorm:"not null"`
}

func (StoryBranch) TableName() string { return "story_branches" }

type StoryEvent struct {
	ID             string    `gorm:"primaryKey;size:64"`
	ProjectID      string    `gorm:"not null;index"`
	SessionID      string    `gorm:"not null;index"`
	BranchID       string    `gorm:"not null;index:idx_story_event_branch_sequence,priority:1"`
	ParentEventID  string    `gorm:"not null;default:'';index"`
	Sequence       int       `gorm:"not null;index:idx_story_event_branch_sequence,priority:2"`
	StoryTime      time.Time `gorm:"not null;index"`
	Kind           string    `gorm:"not null;default:'';index"`
	ActorIDsJSON   JSONB     `gorm:"type:jsonb;not null"`
	LocationKey    string    `gorm:"not null;default:'';index"`
	ResourceJSON   JSONB     `gorm:"type:jsonb;not null"`
	Summary        string    `gorm:"not null;default:''"`
	PayloadJSON    JSONB     `gorm:"type:jsonb;not null"`
	StateDeltaJSON JSONB     `gorm:"type:jsonb;not null"`
	Published      bool      `gorm:"not null;default:false;index"`
	CreatedAt      time.Time `gorm:"not null"`
}

func (StoryEvent) TableName() string { return "story_events" }

type StorySnapshot struct {
	BranchID     string    `gorm:"primaryKey;size:64"`
	EventID      string    `gorm:"primaryKey;size:64"`
	SnapshotJSON JSONB     `gorm:"type:jsonb;not null"`
	UpdatedAt    time.Time `gorm:"not null"`
}

func (StorySnapshot) TableName() string { return "story_snapshots" }

type ChapterEventSpan struct {
	ID          string    `gorm:"primaryKey;size:64"`
	ProjectID   string    `gorm:"not null;index"`
	ChapterID   string    `gorm:"not null;index"`
	BranchID    string    `gorm:"not null;uniqueIndex:idx_chapter_event_span_unique,priority:1"`
	FromEventID string    `gorm:"not null;uniqueIndex:idx_chapter_event_span_unique,priority:2"`
	ToEventID   string    `gorm:"not null;uniqueIndex:idx_chapter_event_span_unique,priority:3"`
	CreatedAt   time.Time `gorm:"not null"`
}

func (ChapterEventSpan) TableName() string { return "chapter_event_spans" }

type WorldMap struct {
	ID             string    `gorm:"primaryKey;size:64"`
	ProjectID      string    `gorm:"not null;uniqueIndex"`
	Name           string    `gorm:"not null;default:''"`
	Seed           string    `gorm:"not null;default:''"`
	Width          int       `gorm:"not null;default:0"`
	Height         int       `gorm:"not null;default:0"`
	Status         string    `gorm:"not null;default:''"`
	PropertiesJSON JSONB     `gorm:"type:jsonb;not null"`
	CreatedAt      time.Time `gorm:"not null"`
	UpdatedAt      time.Time `gorm:"not null"`
}

func (WorldMap) TableName() string { return "world_maps" }

type MapTile struct {
	ID             string    `gorm:"primaryKey;size:64"`
	ProjectID      string    `gorm:"not null;index"`
	MapID          string    `gorm:"not null;uniqueIndex:idx_map_tile_unique,priority:1"`
	X              int       `gorm:"not null;uniqueIndex:idx_map_tile_unique,priority:2"`
	Y              int       `gorm:"not null;uniqueIndex:idx_map_tile_unique,priority:3"`
	Altitude       int       `gorm:"not null;default:0"`
	Temperature    int       `gorm:"not null;default:0"`
	Humidity       int       `gorm:"not null;default:0"`
	IsOcean        bool      `gorm:"not null;default:false"`
	Terrain        string    `gorm:"not null;default:''"`
	PropertiesJSON JSONB     `gorm:"type:jsonb;not null"`
	CreatedAt      time.Time `gorm:"not null"`
	UpdatedAt      time.Time `gorm:"not null"`
}

func (MapTile) TableName() string { return "map_tiles" }

type LocationState struct {
	ID             string    `gorm:"primaryKey;size:64"`
	ProjectID      string    `gorm:"not null;index"`
	MapID          string    `gorm:"not null;default:'';index"`
	RegionID       string    `gorm:"not null;default:'';index"`
	Name           string    `gorm:"not null;default:''"`
	Type           string    `gorm:"not null;default:''"`
	Description    string    `gorm:"not null;default:''"`
	X              int       `gorm:"not null;default:0;index"`
	Y              int       `gorm:"not null;default:0;index"`
	Radius         int       `gorm:"not null;default:0"`
	Status         string    `gorm:"not null;default:''"`
	PropertiesJSON JSONB     `gorm:"type:jsonb;not null"`
	CreatedAt      time.Time `gorm:"not null"`
	UpdatedAt      time.Time `gorm:"not null"`
}

func (LocationState) TableName() string { return "location_states" }

type FactionInfluence struct {
	ID          string    `gorm:"primaryKey;size:64"`
	ProjectID   string    `gorm:"not null;index"`
	LocationID  string    `gorm:"not null;index"`
	FactionName string    `gorm:"not null;default:''"`
	Influence   int       `gorm:"not null;default:0"`
	Attitude    string    `gorm:"not null;default:''"`
	Description string    `gorm:"not null;default:''"`
	Status      string    `gorm:"not null;default:''"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
}

func (FactionInfluence) TableName() string { return "faction_influences" }

type DialogueSession struct {
	ID              string    `gorm:"primaryKey;size:64"`
	ProjectID       string    `gorm:"not null;index"`
	Title           string    `gorm:"not null;default:''"`
	LastUserMessage string    `gorm:"not null;default:''"`
	Status          string    `gorm:"not null;default:''"`
	CreatedAt       time.Time `gorm:"not null"`
	UpdatedAt       time.Time `gorm:"not null"`
}

func (DialogueSession) TableName() string { return "dialogue_sessions" }

type DialogueMessage struct {
	ID           string    `gorm:"primaryKey;size:64"`
	SessionID    string    `gorm:"not null;index"`
	Role         string    `gorm:"not null"`
	Content      string    `gorm:"not null"`
	MetadataJSON JSONB     `gorm:"type:jsonb;not null"`
	CreatedAt    time.Time `gorm:"not null"`
}

func (DialogueMessage) TableName() string { return "dialogue_messages" }

type DialogueRun struct {
	ID          string    `gorm:"primaryKey;size:64"`
	SessionID   string    `gorm:"not null;index"`
	ProjectID   string    `gorm:"not null;index"`
	Status      string    `gorm:"not null;default:''"`
	CurrentStep string    `gorm:"not null;default:''"`
	Progress    int       `gorm:"not null;default:0"`
	Error       string    `gorm:"not null;default:''"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
}

func (DialogueRun) TableName() string { return "dialogue_runs" }

type DialogueRunResult struct {
	ID          string    `gorm:"primaryKey;size:64"`
	RunID       string    `gorm:"not null;uniqueIndex"`
	SessionID   string    `gorm:"not null;index"`
	Status      string    `gorm:"not null;default:''"`
	PayloadJSON JSONB     `gorm:"type:jsonb;not null"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
}

func (DialogueRunResult) TableName() string { return "dialogue_run_results" }

type DialogueActionOption struct {
	ID                   string     `gorm:"primaryKey;size:64"`
	SessionID            string     `gorm:"not null;index"`
	RunID                string     `gorm:"not null;index"`
	ProjectID            string     `gorm:"not null;index"`
	ActionType           string     `gorm:"not null;index"`
	Label                string     `gorm:"not null;default:''"`
	Description          string     `gorm:"not null;default:''"`
	Rationale            string     `gorm:"not null;default:''"`
	ConfirmationRequired bool       `gorm:"not null;default:true"`
	PayloadJSON          JSONB      `gorm:"type:jsonb;not null"`
	Status               string     `gorm:"not null;default:'';index"`
	ResultJSON           JSONB      `gorm:"type:jsonb;not null"`
	Error                string     `gorm:"not null;default:''"`
	ExpiresAt            *time.Time `gorm:"index"`
	CreatedAt            time.Time  `gorm:"not null"`
	UpdatedAt            time.Time  `gorm:"not null"`
}

func (DialogueActionOption) TableName() string { return "dialogue_action_options" }

type RunEvent struct {
	ID          string    `gorm:"primaryKey;size:64"`
	RunKind     string    `gorm:"not null;index:idx_run_event_lookup,priority:1"`
	RunID       string    `gorm:"not null;index:idx_run_event_lookup,priority:2"`
	EventName   string    `gorm:"not null;default:''"`
	Sequence    int       `gorm:"not null;index:idx_run_event_lookup,priority:3"`
	PayloadJSON JSONB     `gorm:"type:jsonb;not null"`
	CreatedAt   time.Time `gorm:"not null"`
}

func (RunEvent) TableName() string { return "run_events" }

type StateRevision struct {
	ID           string    `gorm:"primaryKey;size:64"`
	ProjectID    string    `gorm:"not null;index"`
	EntityType   string    `gorm:"not null;index"`
	EntityID     string    `gorm:"not null;index"`
	SourceRunID  string    `gorm:"not null;default:'';index"`
	SnapshotJSON JSONB     `gorm:"type:jsonb;not null"`
	CreatedAt    time.Time `gorm:"not null"`
}

func (StateRevision) TableName() string { return "state_revisions" }
