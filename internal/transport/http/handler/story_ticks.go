package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/service"
	"github.com/fishimei/NovelOS/internal/transport/http/dto"
	"github.com/fishimei/NovelOS/internal/transport/http/presenter"
)

type StoryTicksHandler struct {
	advancer *service.StoryTickAdvancer
}

func NewStoryTicksHandler(advancer *service.StoryTickAdvancer) StoryTicksHandler {
	return StoryTicksHandler{advancer: advancer}
}

func (h StoryTicksHandler) Advance(c *gin.Context) {
	var req dto.AdvanceStoryTickRequest
	if !bindJSON(c, &req) {
		return
	}
	result, err := h.advancer.Advance(c.Request.Context(), c.Param("project_id"), model.AdvanceStoryTickInput{TickHours: req.TickHours})
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Data(c, http.StatusAccepted, result)
}

func (h StoryTicksHandler) CurrentState(c *gin.Context) {
	result, err := h.advancer.CurrentState(c.Request.Context(), c.Param("project_id"))
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Data(c, http.StatusOK, result)
}

func (h StoryTicksHandler) Events(c *gin.Context) {
	result, err := h.advancer.Events(c.Request.Context(), c.Param("tick_id"))
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Data(c, http.StatusOK, result)
}

func (h StoryTicksHandler) Snapshot(c *gin.Context) {
	result, err := h.advancer.Snapshot(c.Request.Context(), c.Param("tick_id"))
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Data(c, http.StatusOK, result)
}
