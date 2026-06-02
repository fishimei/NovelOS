package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/pkgerr"
	"github.com/fishimei/NovelOS/internal/transport/http/presenter"
)

type WorldHandler struct {
	projects port.ProjectRepository
	events   port.StoryEventStore
}

func NewWorldHandler(projects port.ProjectRepository, events port.StoryEventStore) WorldHandler {
	return WorldHandler{projects: projects, events: events}
}

func (h WorldHandler) GetMap(c *gin.Context) {
	projectID := c.Param("project_id")
	if err := h.ensureProject(c.Request.Context(), projectID); err != nil {
		presenter.Error(c, err)
		return
	}
	result, err := h.events.GetWorldMapByProjectID(c.Request.Context(), projectID)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Data(c, http.StatusOK, result)
}

func (h WorldHandler) ListMapTiles(c *gin.Context) {
	projectID := c.Param("project_id")
	if err := h.ensureProject(c.Request.Context(), projectID); err != nil {
		presenter.Error(c, err)
		return
	}
	result, err := h.events.ListMapTilesByProjectID(c.Request.Context(), projectID)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Data(c, http.StatusOK, result)
}

func (h WorldHandler) ListLocations(c *gin.Context) {
	projectID := c.Param("project_id")
	if err := h.ensureProject(c.Request.Context(), projectID); err != nil {
		presenter.Error(c, err)
		return
	}
	result, err := h.events.ListLocationsByProjectID(c.Request.Context(), projectID)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Data(c, http.StatusOK, result)
}

func (h WorldHandler) ListFactionInfluences(c *gin.Context) {
	projectID := c.Param("project_id")
	if err := h.ensureProject(c.Request.Context(), projectID); err != nil {
		presenter.Error(c, err)
		return
	}
	result, err := h.events.ListFactionInfluencesByProjectID(c.Request.Context(), projectID)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Data(c, http.StatusOK, result)
}

func (h WorldHandler) ListInFlightActions(c *gin.Context) {
	if h.events == nil {
		presenter.Error(c, pkgerr.Internal("story event store is required", nil))
		return
	}
	at, err := time.Parse(time.RFC3339, c.Query("at"))
	if err != nil {
		presenter.Error(c, pkgerr.Validation("at must be an RFC3339 story time"))
		return
	}
	branch, err := h.events.GetBranch(c.Request.Context(), c.Param("branch_id"))
	if err != nil {
		presenter.Error(c, err)
		return
	}
	result, err := h.events.InFlightActionsAt(c.Request.Context(), branch.ID, at)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Data(c, http.StatusOK, result)
}

func (h WorldHandler) ensureProject(ctx context.Context, projectID string) error {
	if h.projects == nil {
		return pkgerr.Internal("project repository is required", nil)
	}
	if h.events == nil {
		return pkgerr.Internal("story event store is required", nil)
	}
	_, err := h.projects.GetDetail(ctx, projectID)
	return err
}
