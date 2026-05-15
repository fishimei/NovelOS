package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/transport/http/dto"
	"github.com/fishimei/NovelOS/internal/transport/http/presenter"
)

type CharactersHandler struct {
	characters port.CharacterRepository
}

func NewCharactersHandler(characters port.CharacterRepository) CharactersHandler {
	return CharactersHandler{characters: characters}
}

func (h CharactersHandler) Create(c *gin.Context) {
	var req dto.CreateCharacterRequest
	if !bindJSON(c, &req) {
		return
	}

	result, err := h.characters.Create(c.Request.Context(), c.Param("project_id"), model.CreateCharacterInput{
		Name:        req.Name,
		Role:        req.Role,
		Profile:     req.Profile,
		Personality: req.Personality,
		VoiceStyle:  req.VoiceStyle,
		Goals:       req.Goals,
		Fears:       req.Fears,
		Secrets:     req.Secrets,
		Constraints: req.Constraints,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}

	presenter.Data(c, http.StatusCreated, result)
}

func (h CharactersHandler) List(c *gin.Context) {
	var query dto.PageQuery
	if !bindQuery(c, &query) {
		return
	}

	pageInput := normalizePageInput(query.Page, query.PageSize)
	result, err := h.characters.ListByProjectID(c.Request.Context(), c.Param("project_id"), pageInput)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	presenter.PaginatedData(c, http.StatusOK, result.Items, pageInput.Page, pageInput.PageSize, result.Total)
}

func (h CharactersHandler) Get(c *gin.Context) {
	result, err := h.characters.GetByID(c.Request.Context(), c.Param("character_id"))
	if err != nil {
		presenter.Error(c, err)
		return
	}

	presenter.Data(c, http.StatusOK, result)
}

func (h CharactersHandler) Update(c *gin.Context) {
	var req dto.UpdateCharacterRequest
	if !bindJSON(c, &req) {
		return
	}

	result, err := h.characters.Update(c.Request.Context(), c.Param("character_id"), model.UpdateCharacterInput{
		Name:        req.Name,
		Role:        req.Role,
		Profile:     req.Profile,
		Personality: req.Personality,
		VoiceStyle:  req.VoiceStyle,
		Goals:       req.Goals,
		Fears:       req.Fears,
		Secrets:     req.Secrets,
		Constraints: req.Constraints,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}

	presenter.Data(c, http.StatusOK, result)
}
