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
	sessions  port.StorySessionRepository
	audit     port.AuditRepository
	events    port.GenerationEventStream
	advancer  *service.StorySessionAdvancer
	committer *service.StoryRunCommitter
}

func NewStorySessionsHandler(
	sessions port.StorySessionRepository,
	audit port.AuditRepository,
	events port.GenerationEventStream,
	advancer *service.StorySessionAdvancer,
	committer *service.StoryRunCommitter,
) StorySessionsHandler {
	return StorySessionsHandler{
		sessions:  sessions,
		audit:     audit,
		events:    events,
		advancer:  advancer,
		committer: committer,
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
	// 历史事件读取持久化 RunEvent；实时流仍由 Subscribe 负责，避免把 SSE 当成正史来源。
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

func (h StorySessionsHandler) CommitRun(c *gin.Context) {
	var req dto.CommitStoryRunRequest
	if !bindJSON(c, &req) {
		return
	}

	result, err := h.committer.Commit(c.Request.Context(), c.Param("run_id"), model.CommitStoryRunInput{
		DraftID:       req.DraftID,
		MemoryPatchID: req.MemoryPatchID,
		AuthorNote:    req.AuthorNote,
	})
	if err != nil {
		presenter.Error(c, err)
		return
	}

	presenter.Data(c, http.StatusOK, result)
}
