package handler

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/application/service"
	"github.com/fishimei/NovelOS/internal/transport/http/dto"
	"github.com/fishimei/NovelOS/internal/transport/http/presenter"
)

type DialogueSessionsHandler struct {
	sessions port.DialogueSessionRepository
	audit    port.AuditRepository
	events   port.GenerationEventStream
	starter  *service.DialogueSessionStarter
	advancer *service.DialogueSessionAdvancer
	executor *service.DialogueActionExecutor
}

func NewDialogueSessionsHandler(
	sessions port.DialogueSessionRepository,
	audit port.AuditRepository,
	events port.GenerationEventStream,
	starter *service.DialogueSessionStarter,
	advancer *service.DialogueSessionAdvancer,
	executor *service.DialogueActionExecutor,
) DialogueSessionsHandler {
	return DialogueSessionsHandler{sessions: sessions, audit: audit, events: events, starter: starter, advancer: advancer, executor: executor}
}

func (h DialogueSessionsHandler) Create(c *gin.Context) {
	var req dto.CreateDialogueSessionRequest
	if !bindJSON(c, &req) {
		return
	}
	result, err := h.starter.Start(c.Request.Context(), c.Param("project_id"), model.CreateDialogueSessionInput{Title: req.Title})
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Data(c, http.StatusCreated, result)
}

func (h DialogueSessionsHandler) List(c *gin.Context) {
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

func (h DialogueSessionsHandler) Get(c *gin.Context) {
	result, err := h.sessions.GetSessionByID(c.Request.Context(), c.Param("session_id"))
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Data(c, http.StatusOK, result)
}

func (h DialogueSessionsHandler) Advance(c *gin.Context) {
	var req dto.AdvanceDialogueSessionRequest
	if !bindJSON(c, &req) {
		return
	}
	result, err := h.advancer.Advance(c.Request.Context(), c.Param("session_id"), model.AdvanceDialogueSessionInput{UserMessage: req.UserMessage})
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Data(c, http.StatusAccepted, result)
}

func (h DialogueSessionsHandler) GetRun(c *gin.Context) {
	result, err := h.sessions.GetRunByID(c.Request.Context(), c.Param("run_id"))
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Data(c, http.StatusOK, result)
}

func (h DialogueSessionsHandler) GetRunResult(c *gin.Context) {
	result, err := h.sessions.GetRunResultByID(c.Request.Context(), c.Param("run_id"))
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Data(c, http.StatusOK, result)
}

func (h DialogueSessionsHandler) GetRunEventHistory(c *gin.Context) {
	result, err := h.audit.ListRunEvents(c.Request.Context(), "dialogue", c.Param("run_id"))
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Data(c, http.StatusOK, result)
}

func (h DialogueSessionsHandler) Subscribe(c *gin.Context) {
	stream, cancel, err := h.events.Subscribe(c.Request.Context(), c.Param("run_id"))
	if err != nil {
		presenter.Error(c, err)
		return
	}
	defer cancel()
	c.Stream(func(w io.Writer) bool {
		select {
		case <-c.Request.Context().Done():
			return false
		case event, ok := <-stream:
			if !ok {
				return false
			}
			c.SSEvent(event.Name, event.Data)
			return true
		}
	})
}

func (h DialogueSessionsHandler) ConfirmActionOption(c *gin.Context) {
	var req dto.ConfirmDialogueActionOptionRequest
	if !bindJSON(c, &req) {
		return
	}
	result, err := h.executor.ExecuteConfirmed(c.Request.Context(), c.Param("option_id"), model.ExecuteDialogueActionInput{Confirm: req.Confirm, AuthorNote: req.AuthorNote})
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Data(c, http.StatusOK, result)
}

func (h DialogueSessionsHandler) RejectActionOption(c *gin.Context) {
	var req dto.RejectDialogueActionOptionRequest
	if !bindJSON(c, &req) {
		return
	}
	result, err := h.executor.Reject(c.Request.Context(), c.Param("option_id"), model.RejectDialogueActionInput{Reason: req.Reason})
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Data(c, http.StatusOK, result)
}
