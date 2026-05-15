// Package config 负责应用程序配置的管理。
// 支持从 YAML 配置文件、环境变量和命令行参数加载配置。
// 环境变量使用 NOVEL_OS_ 前缀，例如：NOVEL_OS_POSTGRES_DSN 映射到配置中的 postgres.dsn。
package config

import (
	"errors"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config 是应用程序的根配置结构，聚合了所有子配置块。
// 使用 mapstructure 标签以便从 YAML/JSON/环境变量等来源反序列化。
type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Postgres PostgresConfig `mapstructure:"postgres"`
	AI       AIConfig       `mapstructure:"ai"`
	SSE      SSEConfig      `mapstructure:"sse"`
}

// AppConfig 包含应用运行时的基本配置。
type AppConfig struct {
	Name string `mapstructure:"name"`
	Env  string `mapstructure:"env"`
	Port int    `mapstructure:"port"`
}

// PostgresConfig 包含 PostgreSQL 数据库连接配置。
type PostgresConfig struct {
	DSN          string `mapstructure:"dsn"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	AutoMigrate  bool   `mapstructure:"auto_migrate"`
}

// AIConfig 包含 AI 模型提供者的配置。
type AIConfig struct {
	Provider   string           `mapstructure:"provider"`
	BaseURL    string           `mapstructure:"base_url"`
	APIKey     string           `mapstructure:"api_key"`
	Model      string           `mapstructure:"model"`
	StoryAgent StoryAgentConfig `mapstructure:"story_agent"`
}

type StoryAgentConfig struct {
	MaxTurns         int    `mapstructure:"max_turns"`
	ControllerPrompt string `mapstructure:"controller_prompt"`
	ToolPrompt       string `mapstructure:"tool_prompt"`
	ResultPrompt     string `mapstructure:"result_prompt"`
}

// SSEConfig 包含服务器发送事件（Server-Sent Events）的配置。
type SSEConfig struct {
	HeartbeatSeconds int `mapstructure:"heartbeat_seconds"`
}

const defaultStoryAgentControllerPrompt = `你是 NovelOS 的故事回合裁决 agent。你的职责不是写对白，而是围绕当前故事状态判断演绎是否应该停止，以及下一轮应该由谁发言或产生动作。每次需要推进时调用 choose_next_story_actor；需要停止时调用 decide_story_stop 并 finalize_story_plan。最多 25 个业务回合。`

const defaultStoryAgentToolPrompt = `load_story_context 用于读取当前故事上下文；choose_next_story_actor 用于记录下一行动者；decide_story_stop 用于判断是否停止；finalize_story_plan 用于提交结构化摘要。不要直接编造已存在事实，不要写入数据库，不要生成完整人物对白。`

const defaultStoryAgentResultPrompt = `将回合裁决结果整理为简洁摘要，供后端构造占位 StoryRunResult。`

// Load 从多个来源加载配置并返回 Config 结构体。
// 配置优先级（从高到低）：命令行参数 > 环境变量 > 配置文件 > 默认值。
// 具体步骤如下：
// 1. 创建空的 Viper 实例（不共享全局配置）
// 2. 设置环境变量前缀为 NOVEL_OS，使 NOVEL_OS_POSTGRES_DSN 映射到 postgres.dsn
// 3. 启用自动环境变量映射（将环境变量绑定到配置键）
// 4. 设置所有配置项的默认值（确保即使没有配置也能启动）
// 5. 如果传入了配置文件路径：
//   - 设置配置文件路径
//   - 读取配置文件（忽略"文件不存在"错误，因为可能有环境变量覆盖）
//   - 如果有其他读取错误，返回错误
//
// 6. 将配置反序列化到 Config 结构体
// 7. 返回配置和任何反序列化错误
func Load(configFile string) (Config, error) {
	v := viper.New()
	v.SetEnvPrefix("NOVEL_OS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetDefault("app.name", "NovelOS")
	v.SetDefault("app.env", "development")
	v.SetDefault("app.port", 8080)
	v.SetDefault("postgres.max_open_conns", 10)
	v.SetDefault("postgres.max_idle_conns", 5)
	v.SetDefault("postgres.auto_migrate", false)
	v.SetDefault("ai.provider", "openai_compatible")
	v.SetDefault("ai.base_url", "")
	v.SetDefault("ai.api_key", "")
	v.SetDefault("ai.model", "claude-sonnet-4-6")
	v.SetDefault("ai.story_agent.max_turns", 25)
	v.SetDefault("ai.story_agent.controller_prompt", defaultStoryAgentControllerPrompt)
	v.SetDefault("ai.story_agent.tool_prompt", defaultStoryAgentToolPrompt)
	v.SetDefault("ai.story_agent.result_prompt", defaultStoryAgentResultPrompt)
	v.SetDefault("sse.heartbeat_seconds", 15)

	if configFile != "" {
		v.SetConfigFile(configFile)
		if err := v.ReadInConfig(); err != nil && !errors.As(err, &viper.ConfigFileNotFoundError{}) && !os.IsNotExist(err) {
			return Config{}, err
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
