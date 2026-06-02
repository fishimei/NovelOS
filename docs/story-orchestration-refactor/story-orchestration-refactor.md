# Story 多人物交涉编排重构 —— 方案 A 设计文档

> 目标：把一次 story run 里"事件模拟 + 同地点分析 + 多人物交涉 + 逐回合渲染 + 元数据 + 章节拼装"从
> **「ReAct 多步循环 + N 次逐回合请求 + 2 次后处理请求」** 收敛为 **一次结构化流式调用**；
> 视角受限不靠"逐角色单独请求做物理隔离"，改为**靠构造上下文（共享可观测层 + 每角色私有视图表）+ 提示词契约**来约束；
> 流式不丢失——单次调用输出 **NDJSON**，后端边解析边复用**现有 SSE 事件**，前端基本不改。
>
> 本文档只描述改造方案，不含代码实现。
>
> ⚠️ 仓库根 `.gitignore` 忽略 `*.md`（仅 `CLAUDE.md` 例外），本文件默认不会被 git 跟踪。若要纳入版本库，用 `git add -f docs/story-orchestration-refactor.md`，或在 `.gitignore` 增加 `!docs/*.md` 例外。

---

## 0. 适用范围（先对齐）

本次改造**只动 `StoryRunGenerator`** 这条线（`internal/infrastructure/ai/eino/story_*.go`）——也就是
故事模拟里"多人物交涉 / 受限视角演绎"。

**不动**用户侧的统一对话 Agent `DialogueRunGenerator`（`dialogue_*.go`，用户↔编排助手提 option）。
两者在当前分支都被改过，注意别混。

---

## 1. 现状：一次 story run 到底发生了什么

入口链路（**保持不变**）：

```
StorySessionAdvancer.Advance        story_flow.go:49    建分支/追加用户消息/建 run(queued)
   → RunExecutor.handle              run_executor.go:90  轮询 claim，调 storyAdvancer.Generate
   → StorySessionAdvancer.Generate   story_flow.go:117   load run/session → generator.Generate → persistResultEvents → SaveRunResult
        → StoryRunGenerator.Generate story_generator.go:94   ← 本次唯一要重写的函数体
```

`StoryRunGenerator.Generate` 现在内部的 LLM 调用（**这是要改的部分**）：

| 步骤 | 位置 | 是否 LLM | 说明 |
|---|---|---|---|
| `generateStoryVariable` | story_generator.go:186 | **否（纯 Go）** | 拼 PlotVariable + 每角色 `CharacterVariableView`；`variable_prompt` 是配了但**没接**的死配置 |
| ReAct 编排 `agent.Generate` | story_generator.go:126 | 是，**多步**（`max_react_steps:80` / `max_turns:25`） | `record_story_event`/`select_story_interaction`/`choose_next_story_actor`(产出 speech)/`decide_story_stop`/`finalize_story_plan`，逐回合推 SSE。产出 plan（turns 带 speech、event_plan、interaction_analysis、transcripts） |
| `generateTurnContent` | story_generator.go:489 | 是，**每回合一次**（N 次） | 只喂 `perspectiveForTurn(actor)`，重渲染 `turn.Content`——**台词被生成第二遍**；这就是"因为人物多次请求"的根源 |
| `generateNarrativeMetadata` | story_generator.go:522 | 是，1 次 | title/summary/plot_variable/memory_patch/review |
| `assembleChapterDraft` | story_generator.go:557 | 是，1 次 | 把 turns[].content 拼成章节正文 |

合计 ≈ **ReAct(几十步) + N + 1 + 1** 次往返，台词生成两遍。

### 现状的两点关键事实（决定了改造方式）

1. **决策层（ReAct 编排器）本来就是全知的**：`load_story_context`（story_tools.go:89）把所有角色 / 世界 / 关系 /
   秘密全喂进去。所以真正的"谁说什么"是全知模型拍的板。
2. **唯一做了物理视角隔离的是渲染层**：`generateTurnContent` 只喂 `perspectiveForTurn`（story_generator.go:674），
   靠"只给这个角色看得见的切片"在物理上防泄漏。**这正是逐回合扇出存在的理由**。

> 推论：当前花 N 次请求换来的视角隔离，只作用在"渲染口吻"层，而决策早被全知模型定死了——
> 两头不讨好。方案 A 接受"决策也由单次（全知）调用完成"，把视角约束**前移到上下文构造 + 提示词契约**，
> 并提供可选的自检/校验钩子来兜底（见 §6）。

---

## 2. 目标架构（方案 A）

```
StoryRunGenerator.Generate(ctx, input):
  1. loadStoryContext            ← 保留：拉全量快照（角色/世界/关系/记忆）
  2. seedPlotVariable            ← 保留 generateStoryVariable 的确定性逻辑，作为「变量种子」喂进上下文
  3. buildSceneContext           ← 新增：把快照拆成「共享可观测层」+「每角色私有视图表」
  4. model.Stream(scenePrompt, sceneContext)   ← 唯一一次 LLM 调用，输出 NDJSON
  5. consumeSceneStream          ← 新增：逐行解析 NDJSON
        每条记录 →  (a) publishStoryEvent 复用现有 SSE 事件
                    (b) 累积进内存 plan/draft/patch
                    (c) 复用现有确定性校验/派生（同地点分组、交涉归属、transcript upsert）
  6. assembleStoryRunResult      ← 流结束后用累积结果拼 StoryRunResult（含 EventPlan，硬约束）
```

**LLM 往返：从 (几十+N+2) 降到 1。** 台词只生成一次（在 `turn` 记录里）。

**接口与持久化边界不变**（重要——把改动半径锁住）：
- `port.StoryRunGenerator.Generate` 签名不变（runtime.go:65）→ service / run executor / SSE 端点**零改动**。
- 流式发布仍在 eino 层用 `deps.events`（generator 持有 `GenerationEventStream`），所以
  `StorySessionAdvancer.Generate` 与 `persistResultEvents` 几乎不动（只需确认 EventPlan 仍被填充）。

---

## 3. 「规范」：单次调用的流式输出契约（NDJSON）

这是你说的"要规范一些东西"的核心。模型**不输出一个大 JSON**，而是输出**逐行 JSON（NDJSON / JSON Lines）**，
每行一条记录，带 `type` 判别字段，按固定顺序产出。后端每读满一行就解析一条、立即映射成 SSE 推给前端。

> 为什么是 NDJSON 而不是"流式解析一个大 JSON"：大对象在流中是半截的（`{"turns":[{...},{...`），
> 没有可靠的行边界无法稳定增量解析。NDJSON 用 `\n` 给出干净的记录边界，是可流式结构化输出的标准做法。

### 3.1 记录类型与产出顺序

模型被要求按此**规范顺序**输出（提示词里硬性规定）：

```
plot_variable → event*（全部事件）→ interaction*（交涉组）→ turn*（逐回合台词/动作）→ [draft_delta*] → result
```

每条记录的 schema（字段沿用现有 model 类型，便于直接反序列化）：

1) 剧情变量（1 条）— 对应 `model.PlotVariable`
```json
{"type":"plot_variable","plot_variable":{"pressure_source":"...","focal_character_id":"c1","core_choice":"...","option_a":"...","option_b":"...","cost_a":"...","cost_b":"...","irreversible_effect":"...","related_character_ids":["c1","c2"],"world_state_pressure":["k1"]}}
```

2) 事件模拟（N 条，每条一个事件）— 对应 `model.StoryEventPlan`。**硬约束：必须产出，`persistResultEvents` 靠它写 `action_scheduled` 事件**
```json
{"type":"event","event":{"time_index":1,"character_id":"c1","character_name":"林","location_key":"study","location_name":"书房","action_type":"action","summary":"独自销毁密信","intent":"灭证","visibility":"private","target_actor_ids":[]}}
```

3) 交涉组（0..3 条）— 对应 `model.StoryInteractionGroup`。`character_ids` 必须是**同地点**已出现的角色
```json
{"type":"interaction","interaction_group":{"id":"interaction_1","location_key":"hall","location_name":"前厅","character_ids":["c1","c2"],"event_ids":["story_event_2","story_event_3"],"should_interact":true,"interaction_type":"confrontation","stakes":"林是否暴露昨晚行踪","rationale":"...","priority":1}}
```

4) 回合（M 条，**人物对话/动作的逐条载体，前端逐条渲染**）— 对应 `StoryTurnPlan`
```json
{"type":"turn","turn":{"turn_index":1,"actor_id":"c2","actor_name":"沈","action_type":"speak","speech":"你昨晚去过书房？","action_summary":"","intent":"试探林的反应","target_actor_ids":["c1"],"interaction_group_id":"interaction_1","location_key":"hall","phase":"negotiation","content":""}}
```
> `content` 可空：方案 A 让 `speech`/`action_summary` 直接作为可读内容，章节正文由 `result.content` 或 `draft_delta` 给出，**不再逐回合重渲染**。

5) 章节正文增量（可选，0..K 条，Phase 2 再做）— 用于让正文也流式
```json
{"type":"draft_delta","text":"沈推开门，烛火在穿堂风里抖了一下。"}
```

6) 终态（1 条，必须最后）
```json
{"type":"result","title":"...","summary":"...","content":"（若未用 draft_delta，则此处给完整章节正文）","stop_reason":"交涉以僵局收束","review":{"pass":true,"hard_violations":[],"continuity_issues":[],"style_issues":[],"suggested_fixes":[]},"memory_patch":{"character_memory_updates":[{"character_id":"c2","type":"belief","content":"开始怀疑林","importance":4}],"relationship_updates":[{"pair_id":"p1","summary":"...","tension_delta":"+1","events":[{"event_type":"negotiation","...":"..."}]}],"world_state_updates":[{"key":"...","operation":"update","value":"...","note":"..."}]}}
```

### 3.2 NDJSON 记录 → SSE 事件 映射表（后端 `consumeSceneStream` 做）

事件名全部**复用** `internal/domain/status.go` 已有常量，前端无需新增事件类型：

| NDJSON 记录 | 复用的 SSE 事件 | 进度/状态（run step） | 后端附带的确定性动作 |
|---|---|---|---|
| 流开始 | `generation_step` | `generating_plot_variable`(20) → `planning_events`(30) | — |
| `plot_variable` | `plot_variable` (EventPlotVariable) | `generating_plot_variable`(20) | 校验 focal/related 是否合法角色，非法则回退（复用 `normalizeStoryVariable`） |
| `event` | `story_event_planned` (EventStoryEventPlanned) | `planning_events`(30) | 累积 event_plan；事件块产出完后用 `buildStoryLocationGroups` 派生同地点候选 |
| （事件块结束，后端派生） | `same_location_candidates` (EventSameLocationCandidates) | `selecting_interactions`(45) | 后端算 `location_groups`，**不要模型给** |
| `interaction` | `interaction_analysis` + （should_interact 时）`interaction_selected` | should_interact → `negotiating_interactions`(60)，否则 `selecting_interactions`(45) | 校验 character_ids 同地点（复用 selectStoryInteraction 的校验逻辑）；最多 3 组 |
| `turn` | `character_turn`（且 interaction_group_id 非空时再发 `negotiation_turn`） | 首条 turn → `driving_character_turns`（status.go:22，现未用，启用它） | 校验 actor/targets 属于交涉组（复用 `validateStoryTurnInteraction`）；`upsertStoryInteractionTurn` 增量构建 transcript；超过 `max_turns` 截断 |
| `draft_delta`（可选） | `draft_delta` (EventDraftDelta，status.go:42 现未被 story 发出，正好用) | `writing_narrative`(80) | 拼接正文缓冲 |
| `result` | `generation_step` | `generating_memory_patch`(90) | 落定 title/summary/content/review/memory_patch |
| 流结束 | （由 service 发）`generation_step` `completed` | `completed` | generator 返回，service 走 persist |

> 注意：`same_location_candidates`、`interaction_analysis`、`interaction_transcripts` 这些**派生物仍由 Go 确定性计算**
> （复用现有 `buildStoryLocationGroups` / `upsertStoryInteractionTurn`），不让模型直接给——保持当前的不变量（交涉必须同地点、回合必须属于已选组）。

### 3.3 前端「规范」（流式渲染侧）

前端继续监听现有 SSE 事件，关键是**幂等增量渲染**：
- `character_turn` / `negotiation_turn`：按 `turn_index` upsert（同 index 覆盖），保证重复/乱序也稳定。
- `story_event_planned`：按 event `id` upsert 到"事件时间线"。
- `interaction_selected`：按 `interaction_group.id` upsert 到"交涉组"视图。
- `draft_delta`：追加到正文区（可选，做了就有正文流式）。
- `generation_step`：驱动进度条/阶段标签。

> 这样"单次调用 + NDJSON 后端转发"对前端表现为：和现在几乎一样的事件流，只是来源从 ReAct 工具回调变成单次流解析。

---

## 4. 视角受限：靠"构造上下文"而非"逐角色请求"

单次调用是全知的，所以"提示词下功夫"的**强形式**是把约束写进上下文结构，让越界变得显眼可查，而不是只加一句"请扮演时别偷看"。

`buildSceneContext`（新增）把 `loadStoryContext` 的快照拆成两层喂给模型：

### 4.1 共享可观测层（Shared Observable Layer）
所有在场角色都能感知、对谁都"安全"的地面真相：
- 当前地点 / 时间 / 公开世界状态（`importance>=4 || volatility>=4` 视为公开，复用 `visibleWorldForCharacter` 的阈值口径）
- 公开可见的动作、已**说出口**的台词（随回合推进自然进入可观测层）

### 4.2 每角色私有视图表（Per-Character Private View Table）
对每个 related character 一行，**自包含**（直接复用现有派生函数）：

| 字段 | 来源（现有函数/数据） |
|---|---|
| identity（profile/personality/voice_style/role/goals/fears/constraints） | `compactCharacters` / `model.Character` |
| known_facts | `storyVariableKnownFacts` |
| misreadings（尤其 believed_target_attitude） | `storyVariableMisreadings` / `relationshipViewsForCharacter` |
| secrets / private_attitude | `model.Character.SecretsJSON` + 该角色为 source 的 `RelationshipView` |
| recent_memories | `snapshot.RecentMemories[characterID]` |
| emotional_pressure / action_bias | `CharacterVariableView`（即 `storyVariableCharacterViews`） |
| visible_world | `visibleWorldForCharacter(worldState, character)` |

> 这些视图**代码里已经算好**，现在只在被丢弃的逐回合渲染里用一次（`perspectiveForTurn`，story_generator.go:674）。
> 改造后它们成为单次调用的**编排脚手架**。

### 4.3 提示词里的视角契约（硬规则）
- 每个角色在 `turns[]` 里的选择 / 台词 / 动作，**只能**由（共享可观测层）+（该角色自己那一行）推出。
- 不得使用他人 secrets / private_attitude / known_facts，**除非该信息已进入共享可观测层**（被说出口或被看见）。
- 角色 misreadings 要让其**照错的信念行动**，不得用全局真相偷偷纠正。
- narration（旁白）可以比角色知道得多，但**不得把不可观测的知识塞进某个角色的嘴里或心里**。

---

## 5. 单次调用的提示词骨架（可直接落到 config）

把现有 `controller_prompt / tool_prompt / result_prompt / narrative_prompt / variable_prompt / simulation_prompt`
**合并为一个 `scene_prompt`**（保留 1~2 个旧字段作非流式回退用，见 §7）。

System（`scene_prompt`）草案：
```
你是 NovelOS 的「单次场景编排器」。你在一次输出里完成：剧情变量确认 → 事件模拟 → 同地点交涉判定 →
多人物逐回合交涉 → 章节正文与状态补丁。你不是逐回合被反复调用的渲染器。

【视角受限契约 —— 最高优先级】
- 你会收到「共享可观测层」和「每角色私有视图表」。
- turns 里每个角色的台词/动作只能由(共享可观测层 + 该角色自己那一行)推出。
- 禁止让角色使用他人 secrets / private_attitude / 未公开 known_facts，除非该信息已被说出口或被看见而进入可观测层。
- 角色 misreadings 必须照错信念行动，不得用全局真相纠正。
- 旁白可全知，但不得把不可观测知识写进某角色的内心或台词。

【输出格式 —— 严格 NDJSON，一行一条 JSON，不要数组，不要 markdown 代码围栏】
按此顺序输出：
  1 行 {"type":"plot_variable",...}
  N 行 {"type":"event",...}        # 事件模拟：每个相关角色本时间片在哪个地点做什么
  0~3 行 {"type":"interaction",...} # 仅从同地点角色中选会真正交涉的组；character_ids 必须同地点
  M 行 {"type":"turn",...}          # 交涉/动作逐回合；interaction_group_id/actor/targets 必须属于上面声明的组
  1 行 {"type":"result",...}        # 标题/摘要/正文/review/memory_patch；最后输出
- 最多 {max_turns} 个 turn。没有可交互组就用 narration/action 回合推进，不要硬造对话。
- memory_patch 只记录本场景真实发生、角色会记住/世界会改变的内容，不得为润色新增事实。
- 每行必须是合法 JSON；字段缺省给空字符串或空数组，不要省略 type。
```

User（`buildSceneContext` 序列化）payload 结构：
```json
{
  "story_run_id":"...","project_id":"...","session_id":"...",
  "session":{"title":"...","opening_situation":"...","author_intent":"...","last_author_message":"...","current_plot_variable":"..."},
  "author_bible": { /* compactAuthorBible */ },
  "plot_variable_seed": { /* generateStoryVariable 的确定性种子，模型在 plot_variable 记录里确认/微调 */ },
  "shared_observable": { "location_hints":[...], "public_world_state":[...], "recent_chapters":[...] },
  "character_views": [ { "character_id":"c1","identity":{...},"known_facts":[...],"secrets":[...],"misreadings":[...],"private_attitude":[...],"recent_memories":[...],"visible_world":[...],"emotional_pressure":"...","action_bias":"..." } ],
  "constraints": { "max_turns": 25, "max_interactions": 3 }
}
```

---

## 6. 视角泄漏的兜底（方案 A 内可加，按需）

你已接受"决策也由单次全知调用完成"的取舍。要在 A 内尽量加固：

1. **结构约束**（§4，已含）——最强的一档"提示词功夫"。
2. **自检字段（轻量）**：让 `result.review` 多带 `perspective_violations: []`，要求模型自审；
   或每个 `turn` 可选带 `grounded_on`（一句话说明这步基于哪条 known_fact/observable）。
3. **确定性校验（可选）**：消费 `turn` 时做关键词检查——若某角色台词命中"他人 secrets 关键词"且该秘密尚未进入可观测层，
   则标记 `review.continuity_issues`（不阻断，仅告警）。
4. **保留接缝**：未来若某个秘密是硬不变量，可对"涉密角色的那几条 turn"发**一次**定向重问（按交涉组而非按回合），
   作为例外路径，不必回到逐回合扇出。

> 取舍照实说：单次全知调用 + 提示词无法 100% 杜绝知识诅咒泄漏。若后续发现"秘密藏不住"成为硬伤，
> 升级为 §1 推论里提到的"两层 + 按交涉组"（方案 C），但那是 A 验证后再说的事。

---

## 7. 文件级改动清单（删 / 留 / 改 / 增）

### `internal/infrastructure/ai/eino/story_generator.go`
- **改写 `Generate`**（:94）：保留 `publishStoryOrchestrationStarted` / `loadStoryContext` / 进度推送骨架；
  中段（变量后处理→ReAct→逐回合→metadata→assembler）替换为 `buildSceneContext → model.Stream → consumeSceneStream → assembleStoryRunResult`。
- **删除**：`generateNarrative`(:471)、`generateTurnContent`(:489)、`generateNarrativeMetadata`(:522)、
  `assembleChapterDraft`(:557)、`perspectiveForTurn`(:674)、`turnNarrativePrompt`(:656)、`statePatchPrompt`(:662)、
  `chapterAssemblerPrompt`(:668)、`variableSystemPrompt`(:650) 及其 fallback（`fallbackNarrativeMetadata`/`fallbackAssembledDraft`/`fallbackTurnContent`/`mergeNarrativeResult`/`mergeNarrativeReview`）。
- **保留并复用**：`generateStoryVariable`(:186) 改名/降级为 `seedPlotVariable`（喂上下文用，不再是最终值）；
  `visibleWorldForCharacter`(:778)、`relationshipViewsForCharacter`(:795)、`storyVariableCharacterViews`(:268)、
  `storyVariableKnownFacts`/`Misreadings`、`normalizeStoryVariable`(:815)、`compact*`、`nextChapterNumber`(:750)、`buildResult`(:693)（适配为从累积结果拼装）。
- **新增**：`buildSceneContext`、`consumeSceneStream`、`assembleStoryRunResult`、NDJSON 行解析器（容错见 §8）。

### `internal/infrastructure/ai/eino/story_tools.go`
- **删除 ReAct 工具层**：`newStoryTools`(:49) 及 `record_story_event`/`select_story_interaction`/
  `choose_next_story_actor`/`decide_story_stop`/`finalize_story_plan` 的 `InferTool` 包装；
  `storyRunState` 里只服务于工具的可变状态/方法。
- **保留并复用为"流消费期的确定性派生/校验"**：`buildStoryLocationGroups`(:497)、`locationGroupByKey`、
  `validateStoryTurnInteraction`(:550)、`interactionLocation`、`upsertStoryInteractionTurn`(:580)、
  `resolveStoryActor`(:395)、`normalizeStoryActionType`(:370)、`validStoryTargetActorIDs`(:421)、
  `copyStory*`、`publishStoryEvent`(:638)、`updateStoryRunStep`(:631)、`storyTurnDisplayPayload`(:445)。
  这些从"工具回调里调用"改为"在 `consumeSceneStream` 里逐记录调用"。

### `internal/infrastructure/ai/eino/story_types.go`
- **新增**：NDJSON 记录类型（`sceneRecord` 带 `Type` + 各 payload）、`SceneContext`（§5 的 user payload）。
- **保留**：`StoryTurnPlan`/`StoryPlanResult`/`StoryEventRecordResult`/`CharacterPerspective`/`CharacterVariableView`
  作为累积目标与上下文类型；`StoryNarrative*` 中仍被 `result` 记录复用的部分保留，纯逐回合渲染相关的可删。

### `internal/application/service/story_flow.go`
- **基本不动**。确认 `assembleStoryRunResult` 仍填充 `result.EventPlan`（`persistResultEvents`:177 依赖它写 `action_scheduled`）
  与 `InteractionAnalysis`/`InteractionTranscripts`（scene_resolved payload:220 依赖）。
- 若模型偶发只给 turn 不给 event：在 generator 内**确定性兜底**——从 turns 反推最小 event_plan（actor+location+action_type+summary），保证 persist 有料。

### `internal/application/port/*.go`
- **不动**。`StoryRunGenerator.Generate` 签名不变；流式发布走 generator 已持有的 `GenerationEventStream`。

### `deployments/config/config.example.yaml`（`internal/config/config.go` 同步加字段）
- `story_agent` 下：新增 `scene_prompt`（§5）；新增 `max_scene_tokens`（替代 `max_chapter_tokens`/`max_turn_tokens`/`max_assembler_tokens`）。
- `max_turns` 保留（提示词软上限 + 消费端硬截断）；`max_react_steps` 变为无用，删除。
- `controller_prompt`/`tool_prompt`/`result_prompt`/`narrative_prompt`/`variable_prompt`/`simulation_prompt`：
  合入 `scene_prompt`；如需非流式回退（§8）保留 `result_prompt` 一个即可，其余删除。

---

## 8. 健壮性与边界（必须在实现时处理）

1. **NDJSON 可靠性**：低温度 + 严格格式说明 + 1 个 few-shot；解析器容错：跳过空行、剥离 ```json 围栏、
   单行解析失败则与下一行拼接重试，整体失败则**回退到一次非流式结构化调用**（`model.Generate` 出一个大 JSON，
   用旧 `result_prompt` 兜底）后再走同样的 assemble。
2. **EventPlan 硬约束**：见 §7 兜底——无 event 记录时从 turns 反推。
3. **max_turns 执行点**：从 ReAct 工具内移到 `consumeSceneStream`（累计到上限后丢弃后续 turn 并标记 stop_reason）。
4. **同地点/交涉归属不变量**：消费 `interaction`/`turn` 时复用 `selectStoryInteraction` 的同地点校验与
   `validateStoryTurnInteraction`；违例则降级（丢弃该 turn 或转 narration）+ 记 `review.continuity_issues`。
5. **持久化时序不变**：流仅用于实时 UX；权威 `StoryRunResult` 在流结束后 assemble，再由 service 持久化。
   中途失败 = run 失败（与现状一致）。事务性不变。
6. **取消/超时**：`Generate` 的 5min `context.WithTimeout`（story_generator.go:95）保留；流读取必须响应 `ctx.Done()`。
7. **token 预算**：单次调用输出含整场（turns + 正文），`max_scene_tokens` 要足够大（≥ 旧 chapter 4000 + 回合量）；
   输入侧用现有 `compact*` 压缩每角色视图，注意上下文窗口。

---

## 9. 测试影响

- `internal/infrastructure/ai/eino/story_tools_test.go`、`story_generator_test.go`：针对 ReAct 工具 / 逐回合渲染的用例需重写为
  针对 **NDJSON 解析器 + consumeSceneStream + assemble** 的用例。
- 新增建议用例：
  - 解析器：合法 NDJSON / 带围栏 / 半行拼接 / 整体失败回退。
  - 不变量：交涉非同地点被拒；turn 不属于已选组被降级；max_turns 截断。
  - EventPlan 兜底：只给 turn 不给 event 时能反推。
  - 视角校验（若启用 §6.3）：他人秘密关键词触发 continuity_issue。
  - assemble：累积记录 → StoryRunResult（EventPlan/InteractionAnalysis/Transcripts/Draft/MemoryPatch 齐全）。
- service 层 `story_flow` 现有用例应**保持通过**（接口/持久化未变）——这是改动半径正确的回归信号。

---

## 10. 分阶段落地建议

- **P0 契约**：定稿 §3 NDJSON 记录 schema + SSE 映射 + 前端幂等渲染约定（前后端共同 code 的地基）。
- **P1 上下文**：`buildSceneContext`（复用现有视图函数）。
- **P2 提示词**：`scene_prompt` + config 字段。
- **P3 流消费**：`model.Stream` → 行解析器 → SSE 发布 + 累积（含容错/回退）。
- **P4 装配**：`assembleStoryRunResult`（复用 transcript 派生/normalize/buildResult）。
- **P5 清理**：删 §7 列出的死代码。
- **P6 测试**：§9。
- **P7 前端**：增量渲染对齐（多数事件已存在，重点是 draft_delta 与幂等 upsert）。

---

## 11. 一句话总结

接口和持久化边界不动，把 `StoryRunGenerator.Generate` 的中段从"全知 ReAct 多步编排 + 逐角色渲染扇出 + 两次后处理"
换成"一次全知流式调用输出 NDJSON"；视角受限靠**共享可观测层 + 每角色私有视图表 + 提示词契约**约束（代码里现成的视图函数直接复用）；
流式靠**后端把 NDJSON 转发成现有 SSE 事件 + 前端幂等增量渲染**保住。LLM 往返从几十次降到一次，台词只生成一次。
```
