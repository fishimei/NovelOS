package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/transport/http/dto"
	"github.com/fishimei/NovelOS/internal/transport/http/presenter"
)

type ProjectsHandler struct {
	projects port.ProjectRepository
}

func NewProjectsHandler(projects port.ProjectRepository) ProjectsHandler {
	return ProjectsHandler{projects: projects}
}

func (h ProjectsHandler) Create(c *gin.Context) {
	var req dto.CreateProjectRequest
	if !bindJSON(c, &req) {
		return
	}

	result, err := h.projects.Create(c.Request.Context(), model.CreateProjectInput{
		Title:       req.Title,
		Genre:       req.Genre,
		Description: req.Description,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}

	presenter.Data(c, http.StatusCreated, result)
}

func (h ProjectsHandler) Get(c *gin.Context) {
	result, err := h.projects.GetDetail(c.Request.Context(), c.Param("project_id"))
	if err != nil {
		presenter.Error(c, err)
		return
	}

	presenter.Data(c, http.StatusOK, result)
}

func (h ProjectsHandler) Update(c *gin.Context) {
	var req dto.UpdateProjectRequest
	if !bindJSON(c, &req) {
		return
	}

	result, err := h.projects.Update(c.Request.Context(), c.Param("project_id"), model.UpdateProjectInput{
		Title:       req.Title,
		Genre:       req.Genre,
		Description: req.Description,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}

	presenter.Data(c, http.StatusOK, result)
}
