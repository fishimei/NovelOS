package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/transport/http/dto"
	"github.com/fishimei/NovelOS/internal/transport/http/presenter"
)

type MemoriesHandler struct {
	memories port.MemoryRepository
}

func NewMemoriesHandler(memories port.MemoryRepository) MemoriesHandler {
	return MemoriesHandler{memories: memories}
}

func (h MemoriesHandler) List(c *gin.Context) {
	var query dto.LimitQuery
	if !bindQuery(c, &query) {
		return
	}

	result, err := h.memories.ListByCharacterID(c.Request.Context(), c.Param("character_id"), normalizeLimit(query.Limit))
	if err != nil {
		presenter.Error(c, err)
		return
	}

	presenter.Data(c, http.StatusOK, result)
}

func (h MemoriesHandler) Create(c *gin.Context) {
	var req dto.CreateMemoryRequest
	if !bindJSON(c, &req) {
		return
	}

	result, err := h.memories.Create(c.Request.Context(), c.Param("character_id"), model.CreateMemoryInput{
		Content:    req.Content,
		Importance: req.Importance,
		Note:       req.Note,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}

	presenter.Data(c, http.StatusCreated, result)
}
