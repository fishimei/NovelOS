package handler

import (
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/application/service"
	"github.com/fishimei/NovelOS/internal/transport/http/dto"
	"github.com/fishimei/NovelOS/internal/transport/http/presenter"
)

type StorySessionsHandler struct {
	sessions port.StorySessionRepository
	audit    port.AuditRepository
	events   port.GenerationEventStream
	advancer *service.StorySessionAdvancer
	cutter   *service.StoryChapterCutter
	log      *service.StoryEventLogService
}

func NewStorySessionsHandler(
	sessions port.StorySessionRepository,
	audit port.AuditRepository,
	events port.GenerationEventStream,
	advancer *service.StorySessionAdvancer,
	cutter *service.StoryChapterCutter,
	logService *service.StoryEventLogService,
) StorySessionsHandler {
	return StorySessionsHandler{
		sessions: sessions,
		audit:    audit,
		events:   events,
		advancer: advancer,
		cutter:   cutter,
		log:      logService,
	}
}

func (h StorySessionsHandler) Create(c *gin.Context) {
	var req dto.CreateStorySessionRequest
	if !bindJSON(c, &req) {
		return
	}
	result, err := h.sessions.CreateSession(c.Request.Context(), c.Param("project_id"), model.CreateStorySessionInput{
		Title:            req.Title,
		OpeningSituation: req.OpeningSituation,
		AuthorIntent:     req.AuthorIntent,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Data(c, http.StatusCreated, result)
}

func (h StorySessionsHandler) List(c *gin.Context) {
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

func (h StorySessionsHandler) Get(c *gin.Context) {
	result, err := h.sessions.GetSessionByID(c.Request.Context(), c.Param("session_id"))
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Data(c, http.StatusOK, result)
}

func (h StorySessionsHandler) Update(c *gin.Context) {
	var req dto.UpdateStorySessionRequest
	if !bindJSON(c, &req) {
		return
	}
	session, err := h.sessions.GetSessionByID(c.Request.Context(), c.Param("session_id"))
	if err != nil {
		presenter.Error(c, err)
		return
	}
	session.Title = strings.TrimSpace(req.Title)
	result, err := h.sessions.UpdateSession(c.Request.Context(), session)
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Data(c, http.StatusOK, result)
}

func (h StorySessionsHandler) Delete(c *gin.Context) {
	if err := h.sessions.DeleteSession(c.Request.Context(), c.Param("session_id")); err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Data(c, http.StatusOK, gin.H{"deleted": true})
}

func (h StorySessionsHandler) Advance(c *gin.Context) {
	var req dto.AdvanceStorySessionRequest
	if !bindJSON(c, &req) {
		return
	}
	result, err := h.advancer.Advance(c.Request.Context(), c.Param("session_id"), model.AdvanceStorySessionInput{
		AuthorMessage: req.AuthorMessage,
		BranchID:      req.BranchID,
		BaseEventID:   req.BaseEventID,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Data(c, http.StatusAccepted, result)
}

func (h StorySessionsHandler) ListEvents(c *gin.Context) {
	result, err := h.log.ListSessionEvents(c.Request.Context(), c.Param("session_id"))
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Data(c, http.StatusOK, result)
}

func (h StorySessionsHandler) GetEvent(c *gin.Context) {
	result, err := h.log.GetEvent(c.Request.Context(), c.Param("event_id"))
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Data(c, http.StatusOK, result)
}

func (h StorySessionsHandler) GetEventState(c *gin.Context) {
	result, err := h.log.GetEventState(c.Request.Context(), c.Param("event_id"))
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Data(c, http.StatusOK, result)
}

func (h StorySessionsHandler) ForkEvent(c *gin.Context) {
	var req dto.ForkStoryEventRequest
	if !bindJSON(c, &req) {
		return
	}
	result, err := h.log.ForkEvent(c.Request.Context(), c.Param("event_id"), model.ForkStoryEventInput{
		Name:          req.Name,
		AuthorMessage: req.AuthorMessage,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Data(c, http.StatusCreated, result)
}

func (h StorySessionsHandler) AdvanceBranch(c *gin.Context) {
	var req dto.AdvanceStorySessionRequest
	if !bindJSON(c, &req) {
		return
	}
	result, err := h.log.AdvanceBranch(c.Request.Context(), c.Param("branch_id"), model.AdvanceStorySessionInput{
		AuthorMessage: req.AuthorMessage,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Data(c, http.StatusAccepted, result)
}

func (h StorySessionsHandler) GetRun(c *gin.Context) {
	result, err := h.sessions.GetRunByID(c.Request.Context(), c.Param("run_id"))
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Data(c, http.StatusOK, result)
}

func (h StorySessionsHandler) GetRunResult(c *gin.Context) {
	result, err := h.sessions.GetRunResultByID(c.Request.Context(), c.Param("run_id"))
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Data(c, http.StatusOK, result)
}

func (h StorySessionsHandler) GetRunEventHistory(c *gin.Context) {
	result, err := h.audit.ListRunEvents(c.Request.Context(), "story", c.Param("run_id"))
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Data(c, http.StatusOK, result)
}

func (h StorySessionsHandler) Subscribe(c *gin.Context) {
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

func (h StorySessionsHandler) CutChapter(c *gin.Context) {
	var req dto.CutChapterRequest
	if !bindJSON(c, &req) {
		return
	}
	result, err := h.cutter.CutChapter(c.Request.Context(), c.Param("run_id"), model.CutChapterInput{
		BranchID:    req.BranchID,
		FromEventID: req.FromEventID,
		ToEventID:   req.ToEventID,
		Title:       req.Title,
		AuthorNote:  req.AuthorNote,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}
	presenter.Data(c, http.StatusOK, result)
}
