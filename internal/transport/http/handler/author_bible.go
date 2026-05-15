package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/transport/http/dto"
	"github.com/fishimei/NovelOS/internal/transport/http/presenter"
)

type AuthorBibleHandler struct {
	authorBibles port.AuthorBibleRepository
}

func NewAuthorBibleHandler(authorBibles port.AuthorBibleRepository) AuthorBibleHandler {
	return AuthorBibleHandler{authorBibles: authorBibles}
}

func (h AuthorBibleHandler) Get(c *gin.Context) {
	result, err := h.authorBibles.GetByProjectID(c.Request.Context(), c.Param("project_id"))
	if err != nil {
		presenter.Error(c, err)
		return
	}

	presenter.Data(c, http.StatusOK, result)
}

func (h AuthorBibleHandler) Update(c *gin.Context) {
	var req dto.UpdateAuthorBibleRequest
	if !bindJSON(c, &req) {
		return
	}

	worldState := make([]model.WorldStateEntry, 0, len(req.InitialWorldState))
	for _, entry := range req.InitialWorldState {
		worldState = append(worldState, model.WorldStateEntry{
			Key:   entry.Key,
			Value: entry.Value,
			Note:  entry.Note,
		})
	}

	result, err := h.authorBibles.UpdateByProjectID(c.Request.Context(), c.Param("project_id"), model.UpdateAuthorBibleInput{
		Theme:               req.Theme,
		StyleGuide:          req.StyleGuide,
		WorldRules:          req.WorldRules,
		AestheticPrinciples: req.AestheticPrinciples,
		HardConstraints:     req.HardConstraints,
		SoftPreferences:     req.SoftPreferences,
		ForbiddenMoves:      req.ForbiddenMoves,
		InitialWorldState:   worldState,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}

	presenter.Data(c, http.StatusOK, result)
}
