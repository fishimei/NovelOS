package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/application/service"
	"github.com/fishimei/NovelOS/internal/transport/http/dto"
	"github.com/fishimei/NovelOS/internal/transport/http/presenter"
)

type SetupSessionsHandler struct {
	sessions port.SetupSessionRepository
	audit    port.AuditRepository
	starter  *service.SetupSessionStarter
	advancer *service.SetupSessionAdvancer
	applier  *service.SetupRunApplier
}

func NewSetupSessionsHandler(
	sessions port.SetupSessionRepository,
	audit port.AuditRepository,
	starter *service.SetupSessionStarter,
	advancer *service.SetupSessionAdvancer,
	applier *service.SetupRunApplier,
) SetupSessionsHandler {
	return SetupSessionsHandler{
		sessions: sessions,
		audit:    audit,
		starter:  starter,
		advancer: advancer,
		applier:  applier,
	}
}

func (h SetupSessionsHandler) Create(c *gin.Context) {
	var req dto.CreateSetupSessionRequest
	if !bindJSON(c, &req) {
		return
	}

	result, err := h.starter.Start(c.Request.Context(), c.Param("project_id"), model.CreateSetupSessionInput{
		SeedIdea: req.SeedIdea,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}

	presenter.Data(c, http.StatusCreated, result)
}

func (h SetupSessionsHandler) List(c *gin.Context) {
	var query dto.PageQuery
	if !bindQuery(c, &query) {
		return
	}

	pageInput := normalizePageInput(query.Page, query.PageSize)
	result, err := h.sessions.ListSessionsByProjectID(c.Request.Context(), c.Param("project_id"), pageInput)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	presenter.PaginatedData(c, http.StatusOK, result.Items, pageInput.Page, pageInput.PageSize, result.Total)
}

func (h SetupSessionsHandler) Get(c *gin.Context) {
	result, err := h.sessions.GetSessionByID(c.Request.Context(), c.Param("session_id"))
	if err != nil {
		presenter.Error(c, err)
		return
	}

	presenter.Data(c, http.StatusOK, result)
}

func (h SetupSessionsHandler) Update(c *gin.Context) {
	var req dto.UpdateSetupSessionRequest
	if !bindJSON(c, &req) {
		return
	}

	session, err := h.sessions.GetSessionByID(c.Request.Context(), c.Param("session_id"))
	if err != nil {
		presenter.Error(c, err)
		return
	}

	session.LastUserMessage = strings.TrimSpace(req.LastUserMessage)
	result, err := h.sessions.UpdateSession(c.Request.Context(), session)
	if err != nil {
		presenter.Error(c, err)
		return
	}

	presenter.Data(c, http.StatusOK, result)
}

func (h SetupSessionsHandler) Delete(c *gin.Context) {
	if err := h.sessions.DeleteSession(c.Request.Context(), c.Param("session_id")); err != nil {
		presenter.Error(c, err)
		return
	}

	presenter.Data(c, http.StatusOK, gin.H{"deleted": true})
}

func (h SetupSessionsHandler) Advance(c *gin.Context) {
	var req dto.AdvanceSetupSessionRequest
	if !bindJSON(c, &req) {
		return
	}

	result, err := h.advancer.Advance(c.Request.Context(), c.Param("session_id"), model.AdvanceSetupSessionInput{
		UserMessage: req.UserMessage,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}

	presenter.Data(c, http.StatusAccepted, result)
}

func (h SetupSessionsHandler) GetRun(c *gin.Context) {
	result, err := h.sessions.GetRunByID(c.Request.Context(), c.Param("run_id"))
	if err != nil {
		presenter.Error(c, err)
		return
	}

	presenter.Data(c, http.StatusOK, result)
}

func (h SetupSessionsHandler) GetRunResult(c *gin.Context) {
	result, err := h.sessions.GetRunResultByID(c.Request.Context(), c.Param("run_id"))
	if err != nil {
		presenter.Error(c, err)
		return
	}

	presenter.Data(c, http.StatusOK, result)
}

func (h SetupSessionsHandler) GetRunEventHistory(c *gin.Context) {
	// 历史事件是运行态审计数据，setup apply 才会把候选设定写入正式状态。
	result, err := h.audit.ListRunEvents(c.Request.Context(), "setup", c.Param("run_id"))
	if err != nil {
		presenter.Error(c, err)
		return
	}

	presenter.Data(c, http.StatusOK, result)
}

func (h SetupSessionsHandler) ApplyRun(c *gin.Context) {
	var req dto.ApplySetupRunRequest
	if !bindJSON(c, &req) {
		return
	}

	result, err := h.applier.Apply(c.Request.Context(), c.Param("session_id"), model.ApplySetupRunInput{
		RunID:               req.RunID,
		AcceptAuthorBible:   req.AcceptAuthorBible,
		AcceptCharacters:    req.AcceptCharacters,
		AcceptRelationships: req.AcceptRelationships,
		AcceptWorldState:    req.AcceptWorldState,
		AuthorNote:          req.AuthorNote,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}

	presenter.Data(c, http.StatusOK, result)
}
