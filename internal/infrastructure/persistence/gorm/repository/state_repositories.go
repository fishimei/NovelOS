package repository

import (
	"context"
	"errors"

	"github.com/fishimei/NovelOS/internal/application/model"
	persistencemodels "github.com/fishimei/NovelOS/internal/infrastructure/persistence/gorm/models"
	"github.com/fishimei/NovelOS/internal/pkgerr"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type projectRepository struct {
	*container
}

func (r *projectRepository) Create(ctx context.Context, input model.CreateProjectInput) (model.Project, error) {
	now := r.now()
	row := persistencemodels.Project{
		ID:          r.nextID("project"),
		Title:       input.Title,
		Genre:       input.Genre,
		Description: input.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := r.dbFor(ctx).Create(&row).Error; err != nil {
		return model.Project{}, mapDBError(err, "project not found")
	}
	return toProject(row), nil
}

func (r *projectRepository) GetByID(ctx context.Context, id string) (model.Project, error) {
	var row persistencemodels.Project
	if err := r.dbFor(ctx).First(&row, "id = ?", id).Error; err != nil {
		return model.Project{}, mapDBError(err, "project not found")
	}
	return toProject(row), nil
}

func (r *projectRepository) Update(ctx context.Context, id string, input model.UpdateProjectInput) (model.Project, error) {
	db := r.dbFor(ctx)
	var row persistencemodels.Project
	if err := db.First(&row, "id = ?", id).Error; err != nil {
		return model.Project{}, mapDBError(err, "project not found")
	}
	row.Title = input.Title
	row.Genre = input.Genre
	row.Description = input.Description
	row.UpdatedAt = r.now()
	if err := db.Save(&row).Error; err != nil {
		return model.Project{}, mapDBError(err, "project not found")
	}
	return toProject(row), nil
}

func (r *projectRepository) GetDetail(ctx context.Context, id string) (model.ProjectDetail, error) {
	project, err := r.GetByID(ctx, id)
	if err != nil {
		return model.ProjectDetail{}, err
	}

	db := r.dbFor(ctx)
	var characterCount int64
	if err := db.Model(&persistencemodels.Character{}).Where("project_id = ?", id).Count(&characterCount).Error; err != nil {
		return model.ProjectDetail{}, mapDBError(err, "project not found")
	}
	var relationshipCount int64
	if err := db.Model(&persistencemodels.RelationshipPair{}).Where("project_id = ?", id).Count(&relationshipCount).Error; err != nil {
		return model.ProjectDetail{}, mapDBError(err, "project not found")
	}
	var storySessionCount int64
	if err := db.Model(&persistencemodels.StorySession{}).Where("project_id = ?", id).Count(&storySessionCount).Error; err != nil {
		return model.ProjectDetail{}, mapDBError(err, "project not found")
	}
	var chapter persistencemodels.Chapter
	lastChapterNumber := 0
	if err := db.Where("project_id = ?", id).Order("chapter_number desc").Take(&chapter).Error; err == nil {
		lastChapterNumber = chapter.ChapterNumber
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return model.ProjectDetail{}, mapDBError(err, "project not found")
	}

	return model.ProjectDetail{
		Project:                    project,
		CharacterCount:             int(characterCount),
		RelationshipCount:          int(relationshipCount),
		StorySessionCount:          int(storySessionCount),
		LastCommittedChapterNumber: lastChapterNumber,
	}, nil
}

type authorBibleRepository struct {
	*container
}

func (r *authorBibleRepository) GetByProjectID(ctx context.Context, projectID string) (model.AuthorBible, error) {
	db := r.dbFor(ctx)
	var row persistencemodels.AuthorBible
	if err := db.First(&row, "project_id = ?", projectID).Error; err != nil {
		return model.AuthorBible{}, mapDBError(err, "author bible not found")
	}
	worldState, err := (&worldStateRepository{container: r.container}).ListByProjectID(ctx, projectID)
	if err != nil {
		return model.AuthorBible{}, err
	}
	bible, err := toAuthorBible(row, worldState)
	if err != nil {
		return model.AuthorBible{}, payloadError("author bible", err)
	}
	return bible, nil
}

func (r *authorBibleRepository) UpdateByProjectID(ctx context.Context, projectID string, input model.UpdateAuthorBibleInput) (model.AuthorBible, error) {
	current, err := r.GetByProjectID(ctx, projectID)
	if err != nil {
		if appErr, ok := err.(*pkgerr.Error); !ok || appErr.Code != pkgerr.CodeNotFound {
			return model.AuthorBible{}, err
		}
		current = model.AuthorBible{ID: r.nextID("bible"), ProjectID: projectID}
	}
	current.ProjectID = projectID
	current.Theme = input.Theme
	current.StyleGuide = input.StyleGuide
	current.WorldRules = input.WorldRules
	current.AestheticPrinciples = input.AestheticPrinciples
	current.HardConstraints = input.HardConstraints
	current.SoftPreferences = input.SoftPreferences
	current.ForbiddenMoves = input.ForbiddenMoves
	current.InitialWorldState = input.InitialWorldState
	current.UpdatedAt = r.now()
	return r.Upsert(ctx, current)
}

func (r *authorBibleRepository) Upsert(ctx context.Context, bible model.AuthorBible) (model.AuthorBible, error) {
	if bible.ID == "" {
		bible.ID = r.nextID("bible")
	}
	if bible.Status == "" {
		bible.Status = "active"
	}
	row, err := authorBibleRowFromModel(bible, r.now(), bible.ID)
	if err != nil {
		return model.AuthorBible{}, payloadError("author bible", err)
	}
	db := r.dbFor(ctx)
	if err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "project_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"theme", "style_guide", "world_rules_json", "aesthetic_json", "hard_constraints_json", "soft_preferences_json", "forbidden_moves_json", "status", "updated_at"}),
	}).Create(&row).Error; err != nil {
		return model.AuthorBible{}, mapDBError(err, "author bible not found")
	}
	if len(bible.InitialWorldState) > 0 {
		if err := (&worldStateRepository{container: r.container}).UpsertEntries(ctx, bible.ProjectID, bible.InitialWorldState); err != nil {
			return model.AuthorBible{}, err
		}
	}
	return r.GetByProjectID(ctx, bible.ProjectID)
}

type worldStateRepository struct {
	*container
}

func (r *worldStateRepository) ListByProjectID(ctx context.Context, projectID string) ([]model.WorldStateEntry, error) {
	var rows []persistencemodels.WorldStateEntry
	if err := r.dbFor(ctx).Where("project_id = ?", projectID).Order("\"key\" asc").Find(&rows).Error; err != nil {
		return nil, mapDBError(err, "world state not found")
	}
	result := make([]model.WorldStateEntry, 0, len(rows))
	for _, row := range rows {
		entry, err := toWorldState(row)
		if err != nil {
			return nil, payloadError("world state", err)
		}
		result = append(result, entry)
	}
	return result, nil
}

func (r *worldStateRepository) UpsertEntries(ctx context.Context, projectID string, entries []model.WorldStateEntry) error {
	if len(entries) == 0 {
		return nil
	}
	now := r.now()
	rows := make([]persistencemodels.WorldStateEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.ID == "" {
			entry.ID = r.nextID("world")
		}
		entry.ProjectID = projectID
		if entry.Status == "" {
			entry.Status = "active"
		}
		row, err := worldStateRowFromModel(entry, now, entry.ID)
		if err != nil {
			return payloadError("world state", err)
		}
		rows = append(rows, row)
	}
	return mapDBError(r.dbFor(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "project_id"}, {Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value_json", "note", "status", "importance", "volatility", "updated_at"}),
	}).Create(&rows).Error, "world state not found")
}

func (r *worldStateRepository) DeleteKeys(ctx context.Context, projectID string, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	return mapDBError(r.dbFor(ctx).Where("project_id = ? AND \"key\" IN ?", projectID, keys).Delete(&persistencemodels.WorldStateEntry{}).Error, "world state not found")
}

type characterRepository struct {
	*container
}

func (r *characterRepository) Create(ctx context.Context, projectID string, input model.CreateCharacterInput) (model.Character, error) {
	now := r.now()
	character := model.Character{
		ID:          r.nextID("character"),
		ProjectID:   projectID,
		Name:        input.Name,
		Role:        input.Role,
		Profile:     input.Profile,
		Personality: input.Personality,
		VoiceStyle:  input.VoiceStyle,
		Goals:       input.Goals,
		Fears:       input.Fears,
		Secrets:     input.Secrets,
		Constraints: input.Constraints,
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	return r.Upsert(ctx, character)
}

func (r *characterRepository) ListByProjectID(ctx context.Context, projectID string, pageInput model.PageInput) (model.ListResult[model.Character], error) {
	db := r.dbFor(ctx)
	var total int64
	if err := db.Model(&persistencemodels.Character{}).Where("project_id = ?", projectID).Count(&total).Error; err != nil {
		return model.ListResult[model.Character]{}, mapDBError(err, "character not found")
	}
	if pageInput.Page <= 0 {
		pageInput.Page = 1
	}
	if pageInput.PageSize <= 0 {
		pageInput.PageSize = 20
	}
	var rows []persistencemodels.Character
	if err := db.Where("project_id = ?", projectID).Order("created_at asc").Limit(pageInput.PageSize).Offset((pageInput.Page - 1) * pageInput.PageSize).Find(&rows).Error; err != nil {
		return model.ListResult[model.Character]{}, mapDBError(err, "character not found")
	}
	items := make([]model.Character, 0, len(rows))
	for _, row := range rows {
		character, err := toCharacter(row)
		if err != nil {
			return model.ListResult[model.Character]{}, payloadError("character", err)
		}
		items = append(items, character)
	}
	return model.ListResult[model.Character]{Items: items, Total: int(total)}, nil
}

func (r *characterRepository) GetByID(ctx context.Context, id string) (model.Character, error) {
	var row persistencemodels.Character
	if err := r.dbFor(ctx).First(&row, "id = ?", id).Error; err != nil {
		return model.Character{}, mapDBError(err, "character not found")
	}
	character, err := toCharacter(row)
	if err != nil {
		return model.Character{}, payloadError("character", err)
	}
	return character, nil
}

func (r *characterRepository) Update(ctx context.Context, id string, input model.UpdateCharacterInput) (model.Character, error) {
	current, err := r.GetByID(ctx, id)
	if err != nil {
		return model.Character{}, err
	}
	current.Name = input.Name
	current.Role = input.Role
	current.Profile = input.Profile
	current.Personality = input.Personality
	current.VoiceStyle = input.VoiceStyle
	current.Goals = input.Goals
	current.Fears = input.Fears
	current.Secrets = input.Secrets
	current.Constraints = input.Constraints
	current.UpdatedAt = r.now()
	return r.Upsert(ctx, current)
}

func (r *characterRepository) Upsert(ctx context.Context, character model.Character) (model.Character, error) {
	if character.ID == "" {
		character.ID = r.nextID("character")
	}
	if character.Status == "" {
		character.Status = "active"
	}
	row, err := characterRowFromModel(character, r.now(), character.ID)
	if err != nil {
		return model.Character{}, payloadError("character", err)
	}
	if err := r.dbFor(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"project_id", "name", "role", "profile", "personality", "voice_style", "goals_json", "fears_json", "secrets_json", "constraints_json", "recent_memory_summary", "status", "updated_at"}),
	}).Create(&row).Error; err != nil {
		return model.Character{}, mapDBError(err, "character not found")
	}
	return r.GetByID(ctx, character.ID)
}

type chapterRepository struct {
	*container
}

func (r *chapterRepository) ListByProjectID(ctx context.Context, projectID string, pageInput model.PageInput) (model.ListResult[model.Chapter], error) {
	db := r.dbFor(ctx)
	var total int64
	if err := db.Model(&persistencemodels.Chapter{}).Where("project_id = ?", projectID).Count(&total).Error; err != nil {
		return model.ListResult[model.Chapter]{}, mapDBError(err, "chapter not found")
	}
	if pageInput.Page <= 0 {
		pageInput.Page = 1
	}
	if pageInput.PageSize <= 0 {
		pageInput.PageSize = 20
	}
	var rows []persistencemodels.Chapter
	if err := db.Where("project_id = ?", projectID).Order("chapter_number asc").Limit(pageInput.PageSize).Offset((pageInput.Page - 1) * pageInput.PageSize).Find(&rows).Error; err != nil {
		return model.ListResult[model.Chapter]{}, mapDBError(err, "chapter not found")
	}
	items := make([]model.Chapter, 0, len(rows))
	for _, row := range rows {
		items = append(items, toChapter(row))
	}
	return model.ListResult[model.Chapter]{Items: items, Total: int(total)}, nil
}

func (r *chapterRepository) GetByID(ctx context.Context, id string) (model.Chapter, error) {
	var row persistencemodels.Chapter
	if err := r.dbFor(ctx).First(&row, "id = ?", id).Error; err != nil {
		return model.Chapter{}, mapDBError(err, "chapter not found")
	}
	return toChapter(row), nil
}

func (r *chapterRepository) Create(ctx context.Context, chapter model.Chapter) (model.Chapter, error) {
	if chapter.ID == "" {
		chapter.ID = r.nextID("chapter")
	}
	if chapter.Status == "" {
		chapter.Status = "committed"
	}
	row := persistencemodels.Chapter{
		ID:            chapter.ID,
		ProjectID:     chapter.ProjectID,
		ChapterNumber: chapter.ChapterNumber,
		Title:         chapter.Title,
		Summary:       chapter.Summary,
		Content:       chapter.Content,
		AuthorNote:    chapter.AuthorNote,
		Status:        chapter.Status,
		WordCount:     chapter.WordCount,
		CommittedAt:   chapter.CommittedAt,
	}
	if err := r.dbFor(ctx).Create(&row).Error; err != nil {
		mapped := mapDBError(err, "chapter not found")
		if isConflict(mapped) {
			return model.Chapter{}, pkgerr.Conflict(pkgerr.CodeConflict, "chapter number conflict")
		}
		return model.Chapter{}, mapped
	}
	return toChapter(row), nil
}

type memoryRepository struct {
	*container
}

func (r *memoryRepository) ListByCharacterID(ctx context.Context, characterID string, limit int) ([]model.Memory, error) {
	if limit <= 0 {
		limit = 20
	}
	var rows []persistencemodels.CharacterMemory
	if err := r.dbFor(ctx).Where("character_id = ?", characterID).Order("created_at desc").Limit(limit).Find(&rows).Error; err != nil {
		return nil, mapDBError(err, "memory not found")
	}
	items := make([]model.Memory, 0, len(rows))
	for _, row := range rows {
		items = append(items, toMemory(row))
	}
	return items, nil
}

func (r *memoryRepository) Create(ctx context.Context, characterID string, input model.CreateMemoryInput) (model.Memory, error) {
	memory := model.Memory{
		ID:          r.nextID("memory"),
		CharacterID: characterID,
		Content:     input.Content,
		Importance:  input.Importance,
		Note:        input.Note,
		Status:      "active",
		CreatedAt:   r.now(),
	}
	if err := r.CreateBatch(ctx, []model.Memory{memory}); err != nil {
		return model.Memory{}, err
	}
	return memory, nil
}

func (r *memoryRepository) CreateBatch(ctx context.Context, memories []model.Memory) error {
	if len(memories) == 0 {
		return nil
	}
	rows := make([]persistencemodels.CharacterMemory, 0, len(memories))
	for _, memory := range memories {
		if memory.ID == "" {
			memory.ID = r.nextID("memory")
		}
		if memory.Status == "" {
			memory.Status = "active"
		}
		rows = append(rows, persistencemodels.CharacterMemory{
			ID:              memory.ID,
			CharacterID:     memory.CharacterID,
			Content:         memory.Content,
			SourceChapterID: memory.SourceChapterID,
			Importance:      memory.Importance,
			Note:            memory.Note,
			Status:          memory.Status,
			CreatedAt:       memory.CreatedAt,
		})
	}
	return mapDBError(r.dbFor(ctx).Create(&rows).Error, "memory not found")
}
