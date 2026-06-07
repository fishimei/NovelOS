package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/pkgerr"
	"github.com/fishimei/NovelOS/internal/transport/http/dto"
	"github.com/fishimei/NovelOS/internal/transport/http/presenter"
)

type WorldHandler struct {
	projects  port.ProjectRepository
	events    port.StoryEventStore
	locations port.LocationInspectionService
}

func NewWorldHandler(projects port.ProjectRepository, events port.StoryEventStore, locations port.LocationInspectionService) WorldHandler {
	return WorldHandler{projects: projects, events: events, locations: locations}
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

func (h WorldHandler) ListMapAreas(c *gin.Context) {
	projectID := c.Param("project_id")
	if err := h.ensureProject(c.Request.Context(), projectID); err != nil {
		presenter.Error(c, err)
		return
	}
	result, err := h.events.ListMapAreasByProjectID(c.Request.Context(), projectID)
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
	parentID, hasParent := c.GetQuery("parent_id")
	areaID := c.Query("area_id")
	scale := c.Query("scale")
	detailState := c.Query("detail_state")
	var result []model.LocationState
	var err error
	if hasParent {
		result, err = h.events.ListLocationsByParentID(c.Request.Context(), projectID, parentID)
	} else {
		result, err = h.events.ListLocationsByProjectID(c.Request.Context(), projectID)
	}
	if err != nil {
		presenter.Error(c, err)
		return
	}
	result = filterLocations(result, areaID, scale, detailState)
	presenter.Data(c, http.StatusOK, result)
}

func (h WorldHandler) GetLocation(c *gin.Context) {
	projectID := c.Param("project_id")
	if err := h.ensureProject(c.Request.Context(), projectID); err != nil {
		presenter.Error(c, err)
		return
	}
	result, err := h.events.GetLocation(c.Request.Context(), projectID, c.Param("location_id"))
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Data(c, http.StatusOK, result)
}

func (h WorldHandler) InspectLocation(c *gin.Context) {
	projectID := c.Param("project_id")
	if err := h.ensureProject(c.Request.Context(), projectID); err != nil {
		presenter.Error(c, err)
		return
	}
	if h.locations == nil {
		presenter.Error(c, pkgerr.Internal("location inspection service is required", nil))
		return
	}
	var req dto.InspectLocationRequest
	if !bindJSON(c, &req) {
		return
	}
	result, err := h.locations.InspectLocation(c.Request.Context(), model.LocationInspectionInput{
		ProjectID:         projectID,
		CharacterID:       req.CharacterID,
		CurrentLocationID: req.CurrentLocationID,
		LocationID:        c.Param("location_id"),
		Reason:            req.Reason,
	})
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

func filterLocations(locations []model.LocationState, areaID string, scale string, detailState string) []model.LocationState {
	if areaID == "" && scale == "" && detailState == "" {
		return locations
	}
	filtered := make([]model.LocationState, 0, len(locations))
	for _, location := range locations {
		if areaID != "" && location.AreaID != areaID {
			continue
		}
		if scale != "" && location.Scale != scale {
			continue
		}
		if detailState != "" && location.DetailState != detailState {
			continue
		}
		filtered = append(filtered, location)
	}
	return filtered
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
