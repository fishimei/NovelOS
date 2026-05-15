package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/transport/http/dto"
	"github.com/fishimei/NovelOS/internal/transport/http/presenter"
)

type RelationshipsHandler struct {
	relationships port.RelationshipRepository
}

func NewRelationshipsHandler(relationships port.RelationshipRepository) RelationshipsHandler {
	return RelationshipsHandler{relationships: relationships}
}

func (h RelationshipsHandler) Create(c *gin.Context) {
	var req dto.CreateRelationshipRequest
	if !bindJSON(c, &req) {
		return
	}

	result, err := h.relationships.Create(c.Request.Context(), c.Param("project_id"), model.CreateRelationshipInput{
		CharacterAID:  req.CharacterAID,
		CharacterBID:  req.CharacterBID,
		Summary:       req.Summary,
		Anchors:       req.Anchors,
		TensionPoints: req.TensionPoints,
		Volatility:    req.Volatility,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}

	presenter.Data(c, http.StatusCreated, result)
}

func (h RelationshipsHandler) List(c *gin.Context) {
	var query dto.PageQuery
	if !bindQuery(c, &query) {
		return
	}

	pageInput := normalizePageInput(query.Page, query.PageSize)
	result, err := h.relationships.ListByProjectID(c.Request.Context(), c.Param("project_id"), pageInput)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	presenter.PaginatedData(c, http.StatusOK, result.Items, pageInput.Page, pageInput.PageSize, result.Total)
}

func (h RelationshipsHandler) Get(c *gin.Context) {
	result, err := h.relationships.GetByID(c.Request.Context(), c.Param("relationship_id"))
	if err != nil {
		presenter.Error(c, err)
		return
	}

	presenter.Data(c, http.StatusOK, result)
}

func (h RelationshipsHandler) Update(c *gin.Context) {
	var req dto.UpdateRelationshipRequest
	if !bindJSON(c, &req) {
		return
	}

	result, err := h.relationships.Update(c.Request.Context(), c.Param("relationship_id"), model.UpdateRelationshipInput{
		Summary:       req.Summary,
		Anchors:       req.Anchors,
		TensionPoints: req.TensionPoints,
		Volatility:    req.Volatility,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}

	presenter.Data(c, http.StatusOK, result)
}
