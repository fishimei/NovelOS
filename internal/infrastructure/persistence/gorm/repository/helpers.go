package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	gormstore "github.com/fishimei/NovelOS/internal/infrastructure/persistence/gorm"
	persistencemodels "github.com/fishimei/NovelOS/internal/infrastructure/persistence/gorm/models"
	"github.com/fishimei/NovelOS/internal/pkgerr"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type container struct {
	db    *gorm.DB
	ids   port.IDGenerator
	clock port.Clock
}

func (c *container) dbFor(ctx context.Context) *gorm.DB {
	return gormstore.DBFromContext(ctx, c.db)
}

func (c *container) now() time.Time {
	if c.clock != nil {
		return c.clock.Now().UTC()
	}
	return time.Now().UTC()
}

func (c *container) nextID(prefix string) string {
	if c.ids != nil {
		return c.ids.New(prefix)
	}
	return prefix + "_" + strings.ReplaceAll(fmt.Sprintf("%d", time.Now().UTC().UnixNano()), "-", "")
}

func encodeJSON(value any) (persistencemodels.JSONB, error) {
	if value == nil {
		return persistencemodels.JSONB("null"), nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return persistencemodels.JSONB(raw), nil
}

func decodeJSON[T any](raw persistencemodels.JSONB) (T, error) {
	var out T
	if len(raw) == 0 || string(raw) == "null" {
		return out, nil
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, err
	}
	return out, nil
}

func normalizePair(a, b string) (string, string) {
	if strings.Compare(a, b) <= 0 {
		return a, b
	}
	return b, a
}

func mapDBError(err error, notFoundMessage string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return pkgerr.NotFound(notFoundMessage)
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return pkgerr.Conflict("", pgErr.Message)
		}
	}

	return pkgerr.Internal("", err)
}

func toProject(row persistencemodels.Project) model.Project {
	return model.Project{
		ID:          row.ID,
		Title:       row.Title,
		Genre:       row.Genre,
		Description: row.Description,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func toWorldState(row persistencemodels.WorldStateEntry) (model.WorldStateEntry, error) {
	value, err := decodeJSON[any](row.ValueJSON)
	if err != nil {
		return model.WorldStateEntry{}, err
	}
	return model.WorldStateEntry{
		ID:         row.ID,
		ProjectID:  row.ProjectID,
		Key:        row.Key,
		Value:      value,
		Note:       row.Note,
		Status:     row.Status,
		Importance: row.Importance,
		Volatility: row.Volatility,
		UpdatedAt:  row.UpdatedAt,
	}, nil
}

func worldStateRowFromModel(entry model.WorldStateEntry, now time.Time, id string) (persistencemodels.WorldStateEntry, error) {
	valueJSON, err := encodeJSON(entry.Value)
	if err != nil {
		return persistencemodels.WorldStateEntry{}, err
	}
	if entry.UpdatedAt.IsZero() {
		entry.UpdatedAt = now
	}
	return persistencemodels.WorldStateEntry{
		ID:         id,
		ProjectID:  entry.ProjectID,
		Key:        entry.Key,
		ValueJSON:  valueJSON,
		Note:       entry.Note,
		Status:     entry.Status,
		Importance: entry.Importance,
		Volatility: entry.Volatility,
		UpdatedAt:  entry.UpdatedAt,
	}, nil
}

func toAuthorBible(row persistencemodels.AuthorBible, worldState []model.WorldStateEntry) (model.AuthorBible, error) {
	worldRules, err := decodeJSON[[]string](row.WorldRulesJSON)
	if err != nil {
		return model.AuthorBible{}, err
	}
	aesthetic, err := decodeJSON[[]string](row.AestheticJSON)
	if err != nil {
		return model.AuthorBible{}, err
	}
	hard, err := decodeJSON[[]string](row.HardConstraintsJSON)
	if err != nil {
		return model.AuthorBible{}, err
	}
	soft, err := decodeJSON[[]string](row.SoftPreferencesJSON)
	if err != nil {
		return model.AuthorBible{}, err
	}
	forbidden, err := decodeJSON[[]string](row.ForbiddenMovesJSON)
	if err != nil {
		return model.AuthorBible{}, err
	}
	return model.AuthorBible{
		ID:                  row.ID,
		ProjectID:           row.ProjectID,
		Theme:               row.Theme,
		StyleGuide:          row.StyleGuide,
		WorldRules:          worldRules,
		AestheticPrinciples: aesthetic,
		HardConstraints:     hard,
		SoftPreferences:     soft,
		ForbiddenMoves:      forbidden,
		InitialWorldState:   worldState,
		Status:              row.Status,
		UpdatedAt:           row.UpdatedAt,
	}, nil
}

func authorBibleRowFromModel(bible model.AuthorBible, now time.Time, id string) (persistencemodels.AuthorBible, error) {
	worldRulesJSON, err := encodeJSON(bible.WorldRules)
	if err != nil {
		return persistencemodels.AuthorBible{}, err
	}
	aestheticJSON, err := encodeJSON(bible.AestheticPrinciples)
	if err != nil {
		return persistencemodels.AuthorBible{}, err
	}
	hardJSON, err := encodeJSON(bible.HardConstraints)
	if err != nil {
		return persistencemodels.AuthorBible{}, err
	}
	softJSON, err := encodeJSON(bible.SoftPreferences)
	if err != nil {
		return persistencemodels.AuthorBible{}, err
	}
	forbiddenJSON, err := encodeJSON(bible.ForbiddenMoves)
	if err != nil {
		return persistencemodels.AuthorBible{}, err
	}
	if bible.UpdatedAt.IsZero() {
		bible.UpdatedAt = now
	}
	return persistencemodels.AuthorBible{
		ID:                  id,
		ProjectID:           bible.ProjectID,
		Theme:               bible.Theme,
		StyleGuide:          bible.StyleGuide,
		WorldRulesJSON:      worldRulesJSON,
		AestheticJSON:       aestheticJSON,
		HardConstraintsJSON: hardJSON,
		SoftPreferencesJSON: softJSON,
		ForbiddenMovesJSON:  forbiddenJSON,
		Status:              bible.Status,
		UpdatedAt:           bible.UpdatedAt,
	}, nil
}

func toCharacter(row persistencemodels.Character) (model.Character, error) {
	goals, err := decodeJSON[[]string](row.GoalsJSON)
	if err != nil {
		return model.Character{}, err
	}
	fears, err := decodeJSON[[]string](row.FearsJSON)
	if err != nil {
		return model.Character{}, err
	}
	secrets, err := decodeJSON[[]string](row.SecretsJSON)
	if err != nil {
		return model.Character{}, err
	}
	constraints, err := decodeJSON[[]string](row.ConstraintsJSON)
	if err != nil {
		return model.Character{}, err
	}
	return model.Character{
		ID:                  row.ID,
		ProjectID:           row.ProjectID,
		Name:                row.Name,
		Role:                row.Role,
		Profile:             row.Profile,
		Personality:         row.Personality,
		VoiceStyle:          row.VoiceStyle,
		Goals:               goals,
		Fears:               fears,
		Secrets:             secrets,
		Constraints:         constraints,
		RecentMemorySummary: row.RecentMemorySummary,
		Status:              row.Status,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}, nil
}

func characterRowFromModel(character model.Character, now time.Time, id string) (persistencemodels.Character, error) {
	goalsJSON, err := encodeJSON(character.Goals)
	if err != nil {
		return persistencemodels.Character{}, err
	}
	fearsJSON, err := encodeJSON(character.Fears)
	if err != nil {
		return persistencemodels.Character{}, err
	}
	secretsJSON, err := encodeJSON(character.Secrets)
	if err != nil {
		return persistencemodels.Character{}, err
	}
	constraintsJSON, err := encodeJSON(character.Constraints)
	if err != nil {
		return persistencemodels.Character{}, err
	}
	if character.CreatedAt.IsZero() {
		character.CreatedAt = now
	}
	if character.UpdatedAt.IsZero() {
		character.UpdatedAt = now
	}
	return persistencemodels.Character{
		ID:                  id,
		ProjectID:           character.ProjectID,
		Name:                character.Name,
		Role:                character.Role,
		Profile:             character.Profile,
		Personality:         character.Personality,
		VoiceStyle:          character.VoiceStyle,
		GoalsJSON:           goalsJSON,
		FearsJSON:           fearsJSON,
		SecretsJSON:         secretsJSON,
		ConstraintsJSON:     constraintsJSON,
		RecentMemorySummary: character.RecentMemorySummary,
		Status:              character.Status,
		CreatedAt:           character.CreatedAt,
		UpdatedAt:           character.UpdatedAt,
	}, nil
}

func toRelationshipPair(row persistencemodels.RelationshipPair) (model.RelationshipPair, error) {
	anchors, err := decodeJSON[[]string](row.AnchorsJSON)
	if err != nil {
		return model.RelationshipPair{}, err
	}
	tension, err := decodeJSON[[]string](row.TensionJSON)
	if err != nil {
		return model.RelationshipPair{}, err
	}
	sharedHistory, err := decodeJSON[[]string](row.SharedHistoryJSON)
	if err != nil {
		return model.RelationshipPair{}, err
	}
	return model.RelationshipPair{
		ID:               row.ID,
		ProjectID:        row.ProjectID,
		LeftCharacterID:  row.LeftCharacterID,
		RightCharacterID: row.RightCharacterID,
		Summary:          row.Summary,
		Anchors:          anchors,
		TensionPoints:    tension,
		SharedHistory:    sharedHistory,
		Volatility:       row.Volatility,
		Status:           row.Status,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}, nil
}

func relationshipPairRowFromModel(pair model.RelationshipPair, now time.Time, id string) (persistencemodels.RelationshipPair, error) {
	anchorsJSON, err := encodeJSON(pair.Anchors)
	if err != nil {
		return persistencemodels.RelationshipPair{}, err
	}
	tensionJSON, err := encodeJSON(pair.TensionPoints)
	if err != nil {
		return persistencemodels.RelationshipPair{}, err
	}
	sharedHistoryJSON, err := encodeJSON(pair.SharedHistory)
	if err != nil {
		return persistencemodels.RelationshipPair{}, err
	}
	if pair.CreatedAt.IsZero() {
		pair.CreatedAt = now
	}
	if pair.UpdatedAt.IsZero() {
		pair.UpdatedAt = now
	}
	return persistencemodels.RelationshipPair{
		ID:                id,
		ProjectID:         pair.ProjectID,
		LeftCharacterID:   pair.LeftCharacterID,
		RightCharacterID:  pair.RightCharacterID,
		Summary:           pair.Summary,
		AnchorsJSON:       anchorsJSON,
		TensionJSON:       tensionJSON,
		SharedHistoryJSON: sharedHistoryJSON,
		Volatility:        pair.Volatility,
		Status:            pair.Status,
		CreatedAt:         pair.CreatedAt,
		UpdatedAt:         pair.UpdatedAt,
	}, nil
}

func toRelationshipView(row persistencemodels.RelationshipView) model.RelationshipView {
	return model.RelationshipView{
		ID:                     row.ID,
		ProjectID:              row.ProjectID,
		PairID:                 row.PairID,
		SourceCharacterID:      row.SourceCharacterID,
		TargetCharacterID:      row.TargetCharacterID,
		PublicAttitude:         row.PublicAttitude,
		PrivateAttitude:        row.PrivateAttitude,
		BelievedTargetAttitude: row.BelievedTargetAttitude,
		MaskingStrategy:        row.MaskingStrategy,
		Status:                 row.Status,
		CreatedAt:              row.CreatedAt,
		UpdatedAt:              row.UpdatedAt,
	}
}

func relationshipViewRowFromModel(view model.RelationshipView, now time.Time, id string) persistencemodels.RelationshipView {
	if view.CreatedAt.IsZero() {
		view.CreatedAt = now
	}
	if view.UpdatedAt.IsZero() {
		view.UpdatedAt = now
	}
	return persistencemodels.RelationshipView{
		ID:                     id,
		ProjectID:              view.ProjectID,
		PairID:                 view.PairID,
		SourceCharacterID:      view.SourceCharacterID,
		TargetCharacterID:      view.TargetCharacterID,
		PublicAttitude:         view.PublicAttitude,
		PrivateAttitude:        view.PrivateAttitude,
		BelievedTargetAttitude: view.BelievedTargetAttitude,
		MaskingStrategy:        view.MaskingStrategy,
		Status:                 view.Status,
		CreatedAt:              view.CreatedAt,
		UpdatedAt:              view.UpdatedAt,
	}
}

func toRelationshipEvent(row persistencemodels.RelationshipEvent) (model.RelationshipEvent, error) {
	payload, err := decodeJSON[map[string]any](row.PayloadJSON)
	if err != nil {
		return model.RelationshipEvent{}, err
	}
	return model.RelationshipEvent{
		ID:        row.ID,
		ProjectID: row.ProjectID,
		PairID:    row.PairID,
		EventType: row.EventType,
		Summary:   row.Summary,
		Payload:   payload,
		CreatedAt: row.CreatedAt,
	}, nil
}

func relationshipEventRowFromModel(event model.RelationshipEvent, now time.Time, id string) (persistencemodels.RelationshipEvent, error) {
	payloadJSON, err := encodeJSON(event.Payload)
	if err != nil {
		return persistencemodels.RelationshipEvent{}, err
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = now
	}
	return persistencemodels.RelationshipEvent{
		ID:          id,
		ProjectID:   event.ProjectID,
		PairID:      event.PairID,
		EventType:   event.EventType,
		Summary:     event.Summary,
		PayloadJSON: payloadJSON,
		CreatedAt:   event.CreatedAt,
	}, nil
}

func toMessage(id, sessionID, role, content string, createdAt time.Time) model.ConversationMessage {
	return model.ConversationMessage{
		ID:        id,
		SessionID: sessionID,
		Role:      role,
		Content:   content,
		CreatedAt: createdAt,
	}
}

func toRunEvent(row persistencemodels.RunEvent) (model.RunEvent, error) {
	payload, err := decodeJSON[map[string]any](row.PayloadJSON)
	if err != nil {
		return model.RunEvent{}, err
	}
	return model.RunEvent{
		ID:        row.ID,
		RunKind:   row.RunKind,
		RunID:     row.RunID,
		EventName: row.EventName,
		Sequence:  row.Sequence,
		Payload:   payload,
		CreatedAt: row.CreatedAt,
	}, nil
}

func runEventRowFromModel(event model.RunEvent, now time.Time, id string) (persistencemodels.RunEvent, error) {
	payloadJSON, err := encodeJSON(event.Payload)
	if err != nil {
		return persistencemodels.RunEvent{}, err
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = now
	}
	return persistencemodels.RunEvent{
		ID:          id,
		RunKind:     event.RunKind,
		RunID:       event.RunID,
		EventName:   event.EventName,
		Sequence:    event.Sequence,
		PayloadJSON: payloadJSON,
		CreatedAt:   event.CreatedAt,
	}, nil
}

func toRevision(row persistencemodels.StateRevision) (model.StateRevision, error) {
	snapshot, err := decodeJSON[map[string]any](row.SnapshotJSON)
	if err != nil {
		return model.StateRevision{}, err
	}
	return model.StateRevision{
		ID:          row.ID,
		ProjectID:   row.ProjectID,
		EntityType:  row.EntityType,
		EntityID:    row.EntityID,
		SourceRunID: row.SourceRunID,
		Snapshot:    snapshot,
		CreatedAt:   row.CreatedAt,
	}, nil
}

func revisionRowFromModel(revision model.StateRevision, now time.Time, id string) (persistencemodels.StateRevision, error) {
	snapshotJSON, err := encodeJSON(revision.Snapshot)
	if err != nil {
		return persistencemodels.StateRevision{}, err
	}
	if revision.CreatedAt.IsZero() {
		revision.CreatedAt = now
	}
	return persistencemodels.StateRevision{
		ID:           id,
		ProjectID:    revision.ProjectID,
		EntityType:   revision.EntityType,
		EntityID:     revision.EntityID,
		SourceRunID:  revision.SourceRunID,
		SnapshotJSON: snapshotJSON,
		CreatedAt:    revision.CreatedAt,
	}, nil
}

func toChapter(row persistencemodels.Chapter) model.Chapter {
	return model.Chapter{
		ID:            row.ID,
		ProjectID:     row.ProjectID,
		ChapterNumber: row.ChapterNumber,
		Title:         row.Title,
		Summary:       row.Summary,
		Content:       row.Content,
		AuthorNote:    row.AuthorNote,
		Status:        row.Status,
		WordCount:     row.WordCount,
		CommittedAt:   row.CommittedAt,
	}
}

func toMemory(row persistencemodels.CharacterMemory) model.Memory {
	return model.Memory{
		ID:              row.ID,
		CharacterID:     row.CharacterID,
		Content:         row.Content,
		SourceChapterID: row.SourceChapterID,
		Importance:      row.Importance,
		Note:            row.Note,
		Status:          row.Status,
		CreatedAt:       row.CreatedAt,
	}
}

func payloadError(kind string, err error) error {
	return pkgerr.Internal(fmt.Sprintf("decode %s payload", kind), err)
}
