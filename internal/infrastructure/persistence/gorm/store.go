// Package gormstore 提供基于 GORM 的 PostgreSQL 持久化实现。
// 负责数据库连接管理、连接池配置和自动迁移。
package gormstore

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/fishimei/NovelOS/internal/config"
	persistencemodels "github.com/fishimei/NovelOS/internal/infrastructure/persistence/gorm/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Store 是 GORM 存储层的封装，提供数据库访问和连接管理。
type Store struct {
	db  *gorm.DB // GORM 数据库实例
	sql *sql.DB  // 标准库 SQL 数据库实例（用于底层操作）
}

// New 创建并初始化新的数据库存储实例。
// 执行以下操作：
// 1. 打开 PostgreSQL 连接
// 2. 配置连接池参数
// 3. 执行连接测试
// 4. 如果启用自动迁移，运行数据库迁移
func New(ctx context.Context, cfg config.PostgresConfig) (*Store, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db: %w", err)
	}

	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	store := &Store{db: db, sql: sqlDB}
	if cfg.AutoMigrate {
		if err := store.AutoMigrate(); err != nil {
			return nil, err
		}
	}

	return store, nil
}

// DB 返回 GORM 数据库实例。
func (s *Store) DB() *gorm.DB {
	return s.db
}

// Close 关闭数据库连接。
func (s *Store) Close() error {
	if s.sql == nil {
		return nil
	}
	return s.sql.Close()
}

// AutoMigrate 执行数据库自动迁移。
// 按依赖顺序迁移表：基础表 → 关系表 → 会话表 → 故事表 → 审计表。
func (s *Store) AutoMigrate() error {
	groups := [][]any{
		{
			&persistencemodels.Project{},
			&persistencemodels.AuthorBible{},
			&persistencemodels.WorldStateEntry{},
			&persistencemodels.Character{},
			&persistencemodels.CharacterMemory{},
			&persistencemodels.Chapter{},
		},
		{
			&persistencemodels.RelationshipPair{},
			&persistencemodels.RelationshipView{},
			&persistencemodels.RelationshipEvent{},
		},
		{
			&persistencemodels.SetupSession{},
			&persistencemodels.SetupMessage{},
			&persistencemodels.SetupRun{},
			&persistencemodels.SetupRunResult{},
		},
		{
			&persistencemodels.StorySession{},
			&persistencemodels.StoryMessage{},
			&persistencemodels.StoryRun{},
			&persistencemodels.StoryRunResult{},
			&persistencemodels.StoryAutoRunState{},
			&persistencemodels.StoryBranch{},
			&persistencemodels.StoryEvent{},
			&persistencemodels.StorySnapshot{},
			&persistencemodels.ChapterEventSpan{},
			&persistencemodels.WorldMap{},
			&persistencemodels.MapArea{},
			&persistencemodels.MapTile{},
			&persistencemodels.LocationState{},
			&persistencemodels.FactionInfluence{},
		},
		{
			&persistencemodels.DialogueSession{},
			&persistencemodels.DialogueMessage{},
			&persistencemodels.DialogueRun{},
			&persistencemodels.DialogueRunResult{},
			&persistencemodels.DialogueActionOption{},
		},
		{
			&persistencemodels.RunEventCounter{},
			&persistencemodels.RunEvent{},
			&persistencemodels.StateRevision{},
		},
	}

	for _, group := range groups {
		if err := s.db.AutoMigrate(group...); err != nil {
			return fmt.Errorf("auto migrate: %w", err)
		}
	}

	return nil
}
