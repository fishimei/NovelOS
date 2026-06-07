package config

import (
	"errors"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	App         AppConfig         `mapstructure:"app"`
	Postgres    PostgresConfig    `mapstructure:"postgres"`
	AI          AIConfig          `mapstructure:"ai"`
	Memory      MemoryConfig      `mapstructure:"memory"`
	World       WorldConfig       `mapstructure:"world"`
	SSE         SSEConfig         `mapstructure:"sse"`
	RunExecutor RunExecutorConfig `mapstructure:"run_executor"`
}

type AppConfig struct {
	Name string `mapstructure:"name"`
	Env  string `mapstructure:"env"`
	Port int    `mapstructure:"port"`
}

type PostgresConfig struct {
	DSN          string `mapstructure:"dsn"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	AutoMigrate  bool   `mapstructure:"auto_migrate"`
}

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
	MaxSceneTokens   int    `mapstructure:"max_scene_tokens"`
	MaxReflectTokens int    `mapstructure:"max_reflect_tokens"`
	ScenePrompt      string `mapstructure:"scene_prompt"`
	ReflectPrompt    string `mapstructure:"reflect_prompt"`
	ResultPrompt     string `mapstructure:"result_prompt"`
	SimulationPrompt string `mapstructure:"simulation_prompt"`
}

type DialogueAgentConfig struct {
	Prompt    string `mapstructure:"prompt"`
	MaxSteps  int    `mapstructure:"max_steps"`
	AutoPilot bool   `mapstructure:"auto_pilot"`
}

const (
	DefaultStoryAgentMaxTurns         = 25
	DefaultStoryAgentMaxSceneTokens   = 7000
	DefaultStoryAgentMaxReflectTokens = 3000
	DefaultDialogueAgentMaxSteps      = 16
)

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

const defaultStoryAgentScenePrompt = `You are NovelOS' scene simulator.

You receive shared public context, private per-character views, authoritative planned_actions, constraints, and output_contract.
Core rules:
- Simulate only consequences after planned_actions.
- planned_actions are authoritative. Do not change listed characters' goals.
- Do not add participants who are not in planned_actions or characters.
- Do not write chapter prose. Do not output content, draft_delta, or memory_patch.
- Keep each character grounded in shared context plus their own private view.
- Characters may act on misreadings. Do not correct them with omniscient truth.
- Secrets/private facts become available to others only if spoken or directly observed.
- Use only action_locations details supplied in shared context; do not invent uninitialized interiors, routes, resources, or access rules.

Output strict NDJSON: one JSON object per line, no array, no markdown fence.
Allowed record types only: plot_variable, event, interaction, turn, stop.

Conditional order:
If planned_actions is non-empty: plot_variable, 0-3 interaction records, turn records, stop. Do not emit event records.
If planned_actions is empty: plot_variable, event records, 0-3 interaction records, turn records, stop.
Every selected interaction must use characters at the same location. Maximum turns: constraints.max_turns.`

const defaultStoryAgentReflectPrompt = `You are NovelOS' scene reflection and memory agent. Given a completed simulated scene plus perception_index and prior_memories, output exactly one JSON object:
{"summary":"...","character_takeaways":[{"character_id":"...","summary":"..."}],"memory_patch":{"character_memory_updates":[],"relationship_updates":[],"world_state_updates":[]}}

Memory perspective contract:
- Write only new or changed memories; Deduplicate prior_memories and do not restate them.
- Each character_memory_update must be based only on turn/event ids visible to that character in perception_index.
- A character not present may record only observable external facts, not private dialogue.
- Misreadings are allowed as beliefs when grounded in that character's view; do not correct with global truth.
- Relationship updates must be based on actual interaction or observable consequence.
- World state updates must be concrete state changes, not mood or prose.
Do not write prose. Output JSON only.`

const defaultStoryAgentResultPrompt = `You are NovelOS' non-streaming scene simulation fallback. Output one JSON object with plot_variable, event_plan or events, interaction_groups, turns, and stop_reason. Do not output title, content, draft_delta, or memory_patch.`

const defaultStoryAgentSimulationPrompt = `你是 NovelOS 的单角色行动裁决 agent。

任务：只为当前 character 决定下一步行动。你不是导演，不安排戏剧场面，不制造多人相遇；只能使用输入 JSON 中提供给该角色可见的信息。如果信息不足，选择保守、可解释、符合角色目标/恐惧/约束的行动。

决策原则：
1. 行动必须符合 character 的 personality、goals、fears、constraints、recent_memory_summary。
2. private_facts / relationships 只代表该角色的主观认知，可以导致误判。
3. 不要让角色利用未提供的信息，也不要替其他角色做决定。
4. 你会看到 reachable_location_refs，它们只是可达地点引用，不等于地点详情。
5. 需要移动、接触、搜索、潜入、冲突或争夺资源时，先调用 inspect_location 了解目标地点；最多 inspect 两个地点。
6. target_location_key 必须来自 current_location.id 或 reachable_location_refs[].id；如果原地行动，可填当前 location id。
7. participant_ids 只填本行动主动接触、寻找、跟踪、攻击、保护或谈判的角色 ID；不要为了热闹添加旁观者。
8. affected_resource_keys 只填本行动会占用、改变或争夺的资源 key，可为空数组。
9. duration_hours 必须是正整数，通常 1-6，除非行动明显需要更久。
10. 不输出 StartAt、ArriveAt、EffectAt、EndsAt 等绝对时间字段。

只返回 JSON 对象：{"action_type":"observe|action|speak|silence","description":"...","duration_hours":1,"target_location_key":"...","participant_ids":[],"affected_resource_keys":[],"rationale":"..."}`

const defaultDialogueAgentPrompt = `你是 NovelOS 的统一对话 Agent。每轮必须先调用 load_dialogue_context。你的职责是和用户交流、澄清目标、读取当前项目状态，并通过 propose_* 工具提出可确认选项。用户未明确确认前，不能调用 execute_confirmed_action；不要声称已经修改项目状态。涉及 setup/story 状态变更时，只创建待确认 option；若缺少 run/session/draft ID，先 inspect 或 list，再不足则提出澄清问题。结束本轮必须调用 finalize_dialogue_response。`

func DefaultStoryAgentScenePrompt() string {
	return defaultStoryAgentScenePrompt
}

func DefaultStoryAgentReflectPrompt() string {
	return defaultStoryAgentReflectPrompt
}

func DefaultStoryAgentResultPrompt() string {
	return defaultStoryAgentResultPrompt
}

func DefaultStoryAgentSimulationPrompt() string {
	return defaultStoryAgentSimulationPrompt
}

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
	v.SetDefault("ai.story_agent.max_turns", DefaultStoryAgentMaxTurns)
	v.SetDefault("ai.story_agent.max_scene_tokens", DefaultStoryAgentMaxSceneTokens)
	v.SetDefault("ai.story_agent.max_reflect_tokens", DefaultStoryAgentMaxReflectTokens)
	v.SetDefault("ai.story_agent.scene_prompt", defaultStoryAgentScenePrompt)
	v.SetDefault("ai.story_agent.reflect_prompt", defaultStoryAgentReflectPrompt)
	v.SetDefault("ai.story_agent.result_prompt", defaultStoryAgentResultPrompt)
	v.SetDefault("ai.story_agent.simulation_prompt", defaultStoryAgentSimulationPrompt)
	v.SetDefault("ai.dialogue_agent.prompt", defaultDialogueAgentPrompt)
	v.SetDefault("ai.dialogue_agent.max_steps", DefaultDialogueAgentMaxSteps)
	v.SetDefault("ai.dialogue_agent.auto_pilot", false)
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
