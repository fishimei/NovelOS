// Package bootstrap 是应用程序的组合根（Composition Root），负责：
// 1. 初始化所有基础设施组件（数据库、事件流等）
// 2. 实例化所有仓库接口的具体实现
// 3. 创建应用服务的具体实现
// 4. 组装 HTTP 处理器并配置路由
// 5. 启动 HTTP 服务器并处理优雅关闭
package bootstrap

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/application/service"
	"github.com/fishimei/NovelOS/internal/config"
	einoai "github.com/fishimei/NovelOS/internal/infrastructure/ai/eino"
	memoryinfra "github.com/fishimei/NovelOS/internal/infrastructure/memory"
	gormstore "github.com/fishimei/NovelOS/internal/infrastructure/persistence/gorm"
	"github.com/fishimei/NovelOS/internal/infrastructure/persistence/gorm/repository"
	worldgen "github.com/fishimei/NovelOS/internal/infrastructure/worldgen/ironarachne"
	transporthttp "github.com/fishimei/NovelOS/internal/transport/http"
	"github.com/fishimei/NovelOS/internal/transport/http/handler"
)

// App 是应用程序的主结构体，包含配置和 HTTP 服务器实例。
// 不持有任何业务逻辑，仅负责生命周期管理和请求路由。
type App struct {
	config      config.Config
	server      *http.Server
	runExecutor *service.RunExecutor
}

// New 创建并初始化一个新的 App 实例。
// 初始化顺序：
// 1. 数据库连接和存储层
// 2. 基础设施工具（时钟、ID生成器、事务管理器）
// 3. 仓库层实现
// 4. 应用服务层（Setup 和 Story 相关的服务）
// 5. HTTP 处理器
// 6. Gin 路由引擎
func New(cfg config.Config) *App {
	store, err := gormstore.New(context.Background(), cfg.Postgres)
	if err != nil {
		log.Fatalf("bootstrap postgres: %v", err)
	}

	clock := serviceClock{}
	idGenerator := gormstore.NewIDGenerator()
	txManager := gormstore.NewTxManager(store.DB())
	repos := repository.New(store.DB(), idGenerator, clock)
	eventStream := service.NewInMemoryEventStream()
	memoryService := configuredMemoryService(cfg)

	setupStarter := service.NewSetupSessionStarter(repos.SetupSessions)
	setupGenerator, err := einoai.NewSetupRunGenerator(context.Background(), einoai.SetupRunGeneratorDeps{
		Config: cfg.AI,
		Events: eventStream,
		Clock:  clock,
		IDs:    idGenerator,
	})
	if err != nil {
		log.Fatalf("bootstrap setup generator: %v", err)
	}
	setupAdvancer := service.NewSetupSessionAdvancer(repos.SetupSessions, repos.Audit, setupGenerator, eventStream)
	setupApplier := service.NewSetupRunApplier(
		repos.SetupSessions,
		repos.AuthorBibles,
		repos.WorldState,
		repos.Characters,
		repos.Relationships,
		repos.Audit,
		txManager,
		clock,
		idGenerator,
		repos.StoryEvents,
		worldgen.NewInitializer(idGenerator),
		service.WorldInitializationSettings{
			Enabled:       cfg.World.Enabled,
			Seed:          cfg.World.Seed,
			LocationCount: cfg.World.LocationCount,
			MapWidth:      cfg.World.MapWidth,
			MapHeight:     cfg.World.MapHeight,
		},
	)
	storyGenerator, err := einoai.NewStoryRunGenerator(context.Background(), einoai.StoryRunGeneratorDeps{
		Config:        cfg.AI,
		Sessions:      repos.StorySessions,
		AuthorBibles:  repos.AuthorBibles,
		WorldState:    repos.WorldState,
		Characters:    repos.Characters,
		Relationships: repos.Relationships,
		Chapters:      repos.Chapters,
		Memories:      repos.Memories,
		MemoryService: memoryService,
		Events:        eventStream,
		Audit:         repos.Audit,
		Clock:         clock,
		IDs:           idGenerator,
	})
	if err != nil {
		log.Fatalf("bootstrap story generator: %v", err)
	}
	storyStarter := service.NewStorySessionStarter(repos.StorySessions)
	storyAdvancer := service.NewStorySessionAdvancer(
		repos.StorySessions,
		repos.StoryEvents,
		repos.Audit,
		storyGenerator,
		memoryService,
		eventStream,
		clock,
		idGenerator,
	)
	storyCutter := service.NewStoryChapterCutter(
		repos.StorySessions,
		repos.StoryEvents,
		repos.Chapters,
		repos.Audit,
		txManager,
		clock,
		idGenerator,
	)
	storyEventLog := service.NewStoryEventLogService(repos.StorySessions, repos.StoryEvents, storyAdvancer, clock)
	dialogueValidator := service.NewDialogueActionValidator(repos.SetupSessions, repos.StorySessions, repos.StoryEvents)
	dialogueExecutor := service.NewDialogueActionExecutor(
		repos.DialogueSessions,
		setupStarter,
		setupAdvancer,
		setupApplier,
		storyStarter,
		storyAdvancer,
		storyCutter,
		storyEventLog,
		repos.Audit,
		dialogueValidator,
		clock,
	)
	dialogueGenerator, err := einoai.NewDialogueRunGenerator(context.Background(), einoai.DialogueRunGeneratorDeps{
		Config:           cfg.AI,
		Projects:         repos.Projects,
		AuthorBibles:     repos.AuthorBibles,
		WorldState:       repos.WorldState,
		Characters:       repos.Characters,
		Relationships:    repos.Relationships,
		SetupSessions:    repos.SetupSessions,
		StorySessions:    repos.StorySessions,
		Chapters:         repos.Chapters,
		DialogueSessions: repos.DialogueSessions,
		ActionExecutor:   dialogueExecutor,
		OptionValidator:  dialogueValidator,
		Events:           eventStream,
		Clock:            clock,
		IDs:              idGenerator,
	})
	if err != nil {
		log.Fatalf("bootstrap dialogue generator: %v", err)
	}
	dialogueStarter := service.NewDialogueSessionStarter(repos.DialogueSessions)
	dialogueAdvancer := service.NewDialogueSessionAdvancer(repos.DialogueSessions, repos.Audit, dialogueGenerator, eventStream)
	runExecutor := service.NewRunExecutor(
		repos,
		setupAdvancer,
		storyAdvancer,
		dialogueAdvancer,
		service.RunExecutorSettings{
			Enabled:             cfg.RunExecutor.Enabled,
			PollIntervalSeconds: cfg.RunExecutor.PollIntervalSeconds,
			StaleAfterSeconds:   cfg.RunExecutor.StaleAfterSeconds,
			BatchSize:           cfg.RunExecutor.BatchSize,
			RunTimeoutSeconds:   cfg.RunExecutor.RunTimeoutSeconds,
		},
		clock,
	)

	handlers := transporthttp.Handlers{
		Projects:         handler.NewProjectsHandler(repos.Projects),
		AuthorBibles:     handler.NewAuthorBibleHandler(repos.AuthorBibles),
		Characters:       handler.NewCharactersHandler(repos.Characters),
		Relationships:    handler.NewRelationshipsHandler(repos.Relationships),
		SetupSessions:    handler.NewSetupSessionsHandler(repos.SetupSessions, repos.Audit, setupStarter, setupAdvancer, setupApplier),
		DialogueSessions: handler.NewDialogueSessionsHandler(repos.DialogueSessions, repos.Audit, eventStream, dialogueStarter, dialogueAdvancer, dialogueExecutor),
		StorySessions:    handler.NewStorySessionsHandler(repos.StorySessions, repos.Audit, eventStream, storyAdvancer, storyCutter, storyEventLog),
		World:            handler.NewWorldHandler(repos.Projects, repos.StoryEvents),
		Chapters:         handler.NewChaptersHandler(repos.Chapters),
		Memories:         handler.NewMemoriesHandler(repos.Memories),
	}
	router := transporthttp.NewRouter(handlers)

	return &App{
		config:      cfg,
		runExecutor: runExecutor,
		server: &http.Server{
			Addr:              fmt.Sprintf(":%d", cfg.App.Port),
			Handler:           router,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

func configuredMemoryService(cfg config.Config) port.CharacterMemoryService {
	mem0Enabled := memoryinfra.ProviderEnabled(cfg.Memory.Provider, "mem0")
	qdrantEnabled := memoryinfra.ProviderEnabled(cfg.Memory.Provider, "qdrant")
	if !mem0Enabled && !qdrantEnabled {
		return nil
	}
	var primary port.CharacterMemoryService
	writers := make([]port.CharacterMemoryService, 0, 2)
	if mem0Enabled {
		mem0 := memoryinfra.NewMem0Service(cfg.Memory.Mem0)
		primary = mem0
		writers = append(writers, mem0)
	}
	if qdrantEnabled {
		qdrant := memoryinfra.NewQdrantService(cfg.Memory.Qdrant, cfg.Memory.Embedding)
		primary = qdrant
		writers = append(writers, qdrant)
	}
	return memoryinfra.NewFanoutService(primary, writers...)
}

// serviceClock 是服务层的时钟实现，使用 UTC 时间。
// 将其作为 port.Clock 接口的实现注入到需要时间的组件中。
type serviceClock struct{}

func (serviceClock) Now() time.Time {
	return time.Now().UTC()
}

// Run 启动 HTTP 服务器并阻塞，直到收到上下文取消信号或发生错误。
// 支持优雅关闭：收到信号后会在 5 秒内完成正在处理的请求。
func (a *App) Run(ctx context.Context) error {
	if a.runExecutor != nil {
		a.runExecutor.Start(ctx)
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("NovelOS listening on %s", a.server.Addr)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return a.server.Shutdown(shutdownCtx)
	case err, ok := <-errCh:
		if !ok {
			return nil
		}
		return err
	}
}
