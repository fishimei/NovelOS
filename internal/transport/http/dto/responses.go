package dto

import (
	"time"

	"github.com/fishimei/NovelOS/internal/application/model"
)

type WorldStateEntryResponse struct {
	ID         string    `json:"id"`
	ProjectID  string    `json:"project_id"`
	Key        string    `json:"key"`
	Value      any       `json:"value"`
	Note       string    `json:"note"`
	Status     string    `json:"status"`
	Importance int       `json:"importance"`
	Volatility int       `json:"volatility"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type AuthorBibleResponse struct {
	ID                  string                    `json:"id"`
	ProjectID           string                    `json:"project_id"`
	Theme               string                    `json:"theme"`
	StyleGuide          string                    `json:"style_guide"`
	WorldRules          []string                  `json:"world_rules"`
	AestheticPrinciples []string                  `json:"aesthetic_principles"`
	HardConstraints     []string                  `json:"hard_constraints"`
	SoftPreferences     []string                  `json:"soft_preferences"`
	ForbiddenMoves      []string                  `json:"forbidden_moves"`
	InitialWorldState   []WorldStateEntryResponse `json:"initial_world_state"`
	Status              string                    `json:"status"`
	UpdatedAt           time.Time                 `json:"updated_at"`
}

func AuthorBibleFromModel(bible model.AuthorBible) AuthorBibleResponse {
	worldState := make([]WorldStateEntryResponse, 0, len(bible.InitialWorldState))
	for _, entry := range bible.InitialWorldState {
		worldState = append(worldState, WorldStateEntryFromModel(entry))
	}

	return AuthorBibleResponse{
		ID:                  bible.ID,
		ProjectID:           bible.ProjectID,
		Theme:               bible.Theme,
		StyleGuide:          bible.StyleGuide,
		WorldRules:          bible.WorldRules,
		AestheticPrinciples: bible.AestheticPrinciples,
		HardConstraints:     bible.HardConstraints,
		SoftPreferences:     bible.SoftPreferences,
		ForbiddenMoves:      bible.ForbiddenMoves,
		InitialWorldState:   worldState,
		Status:              bible.Status,
		UpdatedAt:           bible.UpdatedAt,
	}
}

func WorldStateEntryFromModel(entry model.WorldStateEntry) WorldStateEntryResponse {
	return WorldStateEntryResponse{
		ID:         entry.ID,
		ProjectID:  entry.ProjectID,
		Key:        entry.Key,
		Value:      entry.Value,
		Note:       entry.Note,
		Status:     entry.Status,
		Importance: entry.Importance,
		Volatility: entry.Volatility,
		UpdatedAt:  entry.UpdatedAt,
	}
}
