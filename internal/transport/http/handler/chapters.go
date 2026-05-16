package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/transport/http/dto"
	"github.com/fishimei/NovelOS/internal/transport/http/presenter"
)

type ChaptersHandler struct {
	chapters port.ChapterRepository
}

func NewChaptersHandler(chapters port.ChapterRepository) ChaptersHandler {
	return ChaptersHandler{chapters: chapters}
}

func (h ChaptersHandler) List(c *gin.Context) {
	var query dto.PageQuery
	if !bindQuery(c, &query) {
		return
	}

	pageInput := normalizePageInput(query.Page, query.PageSize)
	result, err := h.chapters.ListByProjectID(c.Request.Context(), c.Param("project_id"), pageInput)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	presenter.PaginatedData(c, http.StatusOK, result.Items, pageInput.Page, pageInput.PageSize, result.Total)
}

func (h ChaptersHandler) Get(c *gin.Context) {
	result, err := h.chapters.GetByID(c.Request.Context(), c.Param("chapter_id"))
	if err != nil {
		presenter.Error(c, err)
		return
	}

	presenter.Data(c, http.StatusOK, result)
}
