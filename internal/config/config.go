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
	App         AppConfig         `mapstructure:"app"`
	Postgres    PostgresConfig    `mapstructure:"postgres"`
	AI          AIConfig          `mapstructure:"ai"`
	Memory      MemoryConfig      `mapstructure:"memory"`
	World       WorldConfig       `mapstructure:"world"`
	SSE         SSEConfig         `mapstructure:"sse"`
	RunExecutor RunExecutorConfig `mapstructure:"run_executor"`
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
	Provider      string              `mapstructure:"provider"`
	BaseURL       string              `mapstructure:"base_url"`
	APIKey        string              `mapstructure:"api_key"`
	Model         string              `mapstructure:"model"`
	SetupAgent    SetupAgentConfig    `mapstructure:"setup_agent"`
	StoryAgent    StoryAgentConfig    `mapstructure:"story_agent"`
	DialogueAgent DialogueAgentConfig `mapstructure:"dialogue_agent"`
}

type SetupAgentConfig struct {
	Prompt string `mapstructure:"prompt"`
}

type StoryAgentConfig struct {
	MaxTurns         int    `mapstructure:"max_turns"`
	ControllerPrompt string `mapstructure:"controller_prompt"`
	ToolPrompt       string `mapstructure:"tool_prompt"`
	ResultPrompt     string `mapstructure:"result_prompt"`
	NarrativePrompt  string `mapstructure:"narrative_prompt"`
	VariablePrompt   string `mapstructure:"variable_prompt"`
	SimulationPrompt string `mapstructure:"simulation_prompt"`
}

type DialogueAgentConfig struct {
	Prompt   string `mapstructure:"prompt"`
	MaxSteps int    `mapstructure:"max_steps"`
}

type MemoryConfig struct {
	Provider  string          `mapstructure:"provider"`
	Embedding EmbeddingConfig `mapstructure:"embedding"`
	Mem0      Mem0Config      `mapstructure:"mem0"`
	Qdrant    QdrantConfig    `mapstructure:"qdrant"`
}

type EmbeddingConfig struct {
	Provider string `mapstructure:"provider"`
	BaseURL  string `mapstructure:"base_url"`
	APIKey   string `mapstructure:"api_key"`
	Model    string `mapstructure:"model"`
	Dims     int    `mapstructure:"dims"`
}

type Mem0Config struct {
	BaseURL string `mapstructure:"base_url"`
	APIKey  string `mapstructure:"api_key"`
	AppID   string `mapstructure:"app_id"`
	TopK    int    `mapstructure:"top_k"`
	Rerank  bool   `mapstructure:"rerank"`
}

type QdrantConfig struct {
	URL        string `mapstructure:"url"`
	APIKey     string `mapstructure:"api_key"`
	Collection string `mapstructure:"collection"`
}

type WorldConfig struct {
	Enabled       bool   `mapstructure:"enabled"`
	Seed          string `mapstructure:"seed"`
	LocationCount int    `mapstructure:"location_count"`
	NearbyRadius  int    `mapstructure:"nearby_radius"`
	MapWidth      int    `mapstructure:"map_width"`
	MapHeight     int    `mapstructure:"map_height"`
}

// SSEConfig 包含服务器发送事件（Server-Sent Events）的配置。
type SSEConfig struct {
	HeartbeatSeconds int `mapstructure:"heartbeat_seconds"`
}

type RunExecutorConfig struct {
	Enabled             bool `mapstructure:"enabled"`
	PollIntervalSeconds int  `mapstructure:"poll_interval_seconds"`
	StaleAfterSeconds   int  `mapstructure:"stale_after_seconds"`
	BatchSize           int  `mapstructure:"batch_size"`
	RunTimeoutSeconds   int  `mapstructure:"run_timeout_seconds"`
}

const defaultSetupAgentPrompt = `你是 NovelOS 的 Setup 编剧 agent。用户只需要说想创作哪类小说或给出粗略灵感，你要主动推理类型约定、世界压力、人物功能位、关系张力和初始状态。不要把 setup 做成问卷；只有无法合理推断且会改变主方向的信息，才放进 open_questions。输出必须是 JSON 对象。`

const defaultStoryAgentControllerPrompt = `你是 NovelOS 的故事回合裁决 agent。你的职责是围绕当前故事状态判断演绎是否应该停止，以及下一轮应该由谁发言或产生动作。每次需要推进时调用 choose_next_story_actor；需要停止时调用 decide_story_stop 并 finalize_story_plan。最多 25 个业务回合。`

const defaultStoryAgentToolPrompt = `load_story_context 用于读取当前故事上下文；choose_next_story_actor 用于记录下一行动者；decide_story_stop 用于判断是否停止；finalize_story_plan 用于提交结构化摘要。不要直接编造已存在事实，不要写入数据库。`

const defaultStoryAgentResultPrompt = `将回合裁决结果整理为简洁摘要，供后端生成剧情变量、章节草稿和状态补丁。`

const defaultStoryAgentNarrativePrompt = `你是 NovelOS 的受限视角多角色演绎 agent。你会收到回合计划和故事上下文。生成正文时，每个角色只能依据自己的 profile、personality、voice_style、goals、fears、secrets、constraints、recent memories，以及该角色自己的 relationship view 行动；不要让角色知道全局真相、他人秘密或他人 private_attitude，除非上下文明确显示该角色已知道。输出必须是 JSON 对象。`

const defaultStoryAgentVariablePrompt = `你是 NovelOS 的剧情变量 agent。你的职责是在角色演绎之前，基于作者意图、当前故事状态、世界压力、角色目标和关系张力，生成一个会推动本章状态变化的核心变量，并为每个相关角色生成受限视角下可感知的变量切片。全局剧情变量可以知道完整结构；角色切片只能包含该角色合理知道、误读、感受到的压力和行动倾向。输出必须是 JSON 对象。`

const defaultStoryAgentSimulationPrompt = `你是 NovelOS 的世界模拟行动裁决 agent。你会收到角色当前位置、坐标、当前地点势力影响、附近地点与距离信息。你只决定该角色在这些已提供信息下接下来要做什么、持续多久、为什么。不要写章节正文，不要制造多人相遇，不要让角色感知未提供的信息。输出必须是 JSON 对象，包含 action_type、description、duration_hours、rationale。`

const defaultDialogueAgentPrompt = `你是 NovelOS 的统一对话 Agent。每轮必须先调用 load_dialogue_context。你的职责是和用户交流、澄清目标、读取当前项目状态，并通过 propose_* 工具提出可确认选项。用户未明确确认前，不能调用 execute_confirmed_action；不要声称已经修改项目状态。涉及 setup/story 状态变更时，只创建待确认 option；若缺少 run/session/draft ID，先 inspect 或 list，再不足则提出澄清问题。结束本轮必须调用 finalize_dialogue_response。`

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
	v.SetDefault("ai.setup_agent.prompt", defaultSetupAgentPrompt)
	v.SetDefault("ai.story_agent.max_turns", 25)
	v.SetDefault("ai.story_agent.controller_prompt", defaultStoryAgentControllerPrompt)
	v.SetDefault("ai.story_agent.tool_prompt", defaultStoryAgentToolPrompt)
	v.SetDefault("ai.story_agent.result_prompt", defaultStoryAgentResultPrompt)
	v.SetDefault("ai.story_agent.narrative_prompt", defaultStoryAgentNarrativePrompt)
	v.SetDefault("ai.story_agent.variable_prompt", defaultStoryAgentVariablePrompt)
	v.SetDefault("ai.story_agent.simulation_prompt", defaultStoryAgentSimulationPrompt)
	v.SetDefault("ai.dialogue_agent.prompt", defaultDialogueAgentPrompt)
	v.SetDefault("ai.dialogue_agent.max_steps", 16)
	v.SetDefault("memory.provider", "local")
	v.SetDefault("memory.embedding.provider", "openai_compatible")
	v.SetDefault("memory.embedding.base_url", "")
	v.SetDefault("memory.embedding.api_key", "")
	v.SetDefault("memory.embedding.model", "")
	v.SetDefault("memory.embedding.dims", 1536)
	v.SetDefault("memory.mem0.base_url", "https://api.mem0.ai")
	v.SetDefault("memory.mem0.api_key", "")
	v.SetDefault("memory.mem0.app_id", "novelos")
	v.SetDefault("memory.mem0.top_k", 12)
	v.SetDefault("memory.mem0.rerank", false)
	v.SetDefault("memory.qdrant.url", "")
	v.SetDefault("memory.qdrant.api_key", "")
	v.SetDefault("memory.qdrant.collection", "novelos_character_memories")
	v.SetDefault("world.enabled", true)
	v.SetDefault("world.seed", "")
	v.SetDefault("world.location_count", 15)
	v.SetDefault("world.nearby_radius", 25)
	v.SetDefault("world.map_width", 1024)
	v.SetDefault("world.map_height", 1024)
	v.SetDefault("sse.heartbeat_seconds", 15)
	v.SetDefault("run_executor.enabled", true)
	v.SetDefault("run_executor.poll_interval_seconds", 2)
	v.SetDefault("run_executor.stale_after_seconds", 600)
	v.SetDefault("run_executor.batch_size", 10)
	v.SetDefault("run_executor.run_timeout_seconds", 600)

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
