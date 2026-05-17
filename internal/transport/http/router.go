// Package transporthttp 提供 HTTP 传输层的实现。
// 负责 HTTP 请求处理、路由配置和响应格式化。
package transporthttp

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/fishimei/NovelOS/internal/transport/http/handler"
	"github.com/fishimei/NovelOS/internal/transport/http/middleware"
	"github.com/fishimei/NovelOS/internal/transport/http/presenter"
)

// Handlers 聚合所有 HTTP 处理器。
type Handlers struct {
	Projects      handler.ProjectsHandler
	AuthorBibles  handler.AuthorBibleHandler
	Characters    handler.CharactersHandler
	Relationships handler.RelationshipsHandler
	SetupSessions handler.SetupSessionsHandler
	StorySessions handler.StorySessionsHandler
	Chapters      handler.ChaptersHandler
	Memories      handler.MemoriesHandler
}

// NewRouter 创建并配置 Gin 路由引擎。
// 注册的路由包括：
// - GET /healthz - 健康检查
// - /api/v1/* - API v1 路由组
func NewRouter(handlers Handlers) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.RequestID())
	router.Use(errorMiddleware())
	router.GET("/healthz", func(c *gin.Context) {
		presenter.Data(c, http.StatusOK, gin.H{"status": "ok"})
	})

	api := router.Group("/api/v1")
	{
		api.POST("/projects", handlers.Projects.Create)
		api.GET("/projects/:project_id", handlers.Projects.Get)
		api.PUT("/projects/:project_id", handlers.Projects.Update)

		api.GET("/projects/:project_id/author-bible", handlers.AuthorBibles.Get)
		api.PUT("/projects/:project_id/author-bible", handlers.AuthorBibles.Update)

		api.POST("/projects/:project_id/characters", handlers.Characters.Create)
		api.GET("/projects/:project_id/characters", handlers.Characters.List)
		api.GET("/characters/:character_id", handlers.Characters.Get)
		api.PUT("/characters/:character_id", handlers.Characters.Update)

		api.POST("/projects/:project_id/relationships", handlers.Relationships.Create)
		api.GET("/projects/:project_id/relationships", handlers.Relationships.List)
		api.GET("/relationships/:relationship_id", handlers.Relationships.Get)
		api.PUT("/relationships/:relationship_id", handlers.Relationships.Update)

		api.POST("/projects/:project_id/setup-sessions", handlers.SetupSessions.Create)
		api.GET("/projects/:project_id/setup-sessions", handlers.SetupSessions.List)
		api.GET("/setup-sessions/:session_id", handlers.SetupSessions.Get)
		api.POST("/setup-sessions/:session_id/advance", handlers.SetupSessions.Advance)
		api.GET("/setup-runs/:run_id", handlers.SetupSessions.GetRun)
		api.GET("/setup-runs/:run_id/result", handlers.SetupSessions.GetRunResult)
		api.GET("/setup-runs/:run_id/event-history", handlers.SetupSessions.GetRunEventHistory)
		api.POST("/setup-sessions/:session_id/apply", handlers.SetupSessions.ApplyRun)

		api.POST("/projects/:project_id/story-sessions", handlers.StorySessions.Create)
		api.GET("/projects/:project_id/story-sessions", handlers.StorySessions.List)
		api.GET("/story-sessions/:session_id", handlers.StorySessions.Get)
		api.POST("/story-sessions/:session_id/advance", handlers.StorySessions.Advance)
		api.GET("/story-runs/:run_id", handlers.StorySessions.GetRun)
		api.GET("/story-runs/:run_id/result", handlers.StorySessions.GetRunResult)
		api.GET("/story-runs/:run_id/event-history", handlers.StorySessions.GetRunEventHistory)
		api.GET("/story-runs/:run_id/events", handlers.StorySessions.Subscribe)
		api.POST("/story-runs/:run_id/commit", handlers.StorySessions.CommitRun)

		api.GET("/projects/:project_id/chapters", handlers.Chapters.List)
		api.GET("/chapters/:chapter_id", handlers.Chapters.Get)

		api.GET("/characters/:character_id/memories", handlers.Memories.List)
		api.POST("/characters/:character_id/memories", handlers.Memories.Create)
	}

	return router
}

// errorMiddleware 是错误处理中间件。
// 收集 Gin 上下文中的错误并转换为统一的错误响应格式。
func errorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		for _, err := range c.Errors {
			presenter.Error(c, err.Err)
			return
		}
	}
}
