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
	ID          string `gorm:"primaryKey;size:64"`
	SessionID   string `gorm:"not null;index"`
	ProjectID   string `gorm:"not null;index"`
	Status      string `gorm:"not null;default:''"`
	CurrentStep string `gorm:"not null;default:''"`
	Progress    int    `gorm:"not null;default:0"`
	CommittedAt *time.Time
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
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
