# Story 编排重构计划书 (v2)

> **本文件 v2 取代 v1。** 实现以本文件为唯一准绳。
>
> **v1 → v2 关键变更（在实现的 AI 必读）：**
> 1. 模拟调用**不再产正文**（删除 `content` / `draft_delta` 输出）。
> 2. 模拟调用**不再产记忆**（`memory_patch` 从模拟段移出）。
> 3. **新增「复盘/记忆」段**：模拟结束后，单独一次 LLM 调用产出「摘要 + 状态变化统计 + 各角色记忆」。
> 4. **记忆提交时机改为「编排结束、摘要完成时」**（run 完成那一刻），**不再是裁章时**；提交逻辑从 `StoryChapterCutter` 移到 `StorySessionAdvancer`。
> 5. **正文(stage③)与裁章彻底解耦、本期推迟**；只描述设计，不实现。
>
> 范围：本次只动 `StoryRunGenerator` 这条线（`internal/infrastructure/ai/eino/story_*.go`）+ `StorySessionAdvancer` 的收尾 + `StoryChapterCutter` 的记忆剥离 + config。
> 不动用户侧统一对话 Agent `DialogueRunGenerator`。
>
> ⚠️ 根 `.gitignore` 忽略 `*.md`，本文件默认不被 git 跟踪；要入库用 `git add -f docs/story-orchestration-refactor.md`。

---

## 0. 决策日志（DECIDED —— 不要重新讨论）

| # | 决策 |
|---|---|
| D1 | 模拟用**单次全知流式调用**，输出 **NDJSON**，后端逐行转发为**现有 SSE 事件**，前端幂等增量渲染。 |
| D2 | 视角受限靠**构造上下文（共享可观测层 + 每角色私有视图表）+ 提示词契约**，不靠逐角色单独请求。 |
| D3 | **模拟段不产正文、不产记忆**。只产 events / interactions / turns(台词) / plot_variable。 |
| D4 | **新增复盘/记忆段**（独立 LLM 调用）：产**摘要 + 状态变化(记忆/关系/世界 delta) + 各角色记忆**；记忆本身**视角受限**（角色只记亲历/感知到的）。 |
| D5 | **记忆在 run 完成（②摘要做完）时提交**到外部记忆库，**不是裁章**。提交从 `StoryChapterCutter` 移到 `StorySessionAdvancer.Generate`，并给记忆打 `source_run_id`/`branch_id`/`source_event_id` + `trigger=run_completion`。 |
| D6 | **正文(stage③)与裁章解耦、本期不实现**。两候选方案（精选自动成文 / 作者对话驱动）只做设计；本期只保证账本素材充足，两条路都能后续接。 |
| D7 | `port.StoryRunGenerator.Generate` 接口签名**不变** → service/run executor/SSE 端点零改动。**完整 turns（含独白/旁白）必须落进事件账本**。 |
| D8 | 同地点 / 交涉归属等不变量仍由 **Go 确定性派生**（`buildStoryLocationGroups`/`validateStoryTurnInteraction`/`upsertStoryInteractionTurn`），不让模型直接给。 |
| D9 | 泄漏兜底 **P0 只做第一档（结构约束 + 提示词契约）**，不预造自检字段/关键词校验；观察到硬剧透泄漏再补「定向重问」。 |

**本期范围（P0）= 阶段① + 阶段②。**
**推迟（后续）= 阶段③ 正文 + 裁章语义重构。**

---

## 1. 目标架构：三段式流水线

```
① 模拟 (单次流式 LLM)        →  events + interactions + turns(台词) + plot_variable        [不产正文/不产记忆]
② 复盘/记忆 (单次 LLM)       →  scene summary + 状态变化 delta + 各角色记忆(视角受限)        [不产正文]
   └─(run 完成时)→ 提交各角色记忆到外部记忆库 + 写 scene_resolved 事件
─────────────────────────  以上是「演绎」，run 到此 completed  ─────────────────────────
③ 正文 (推迟, 解耦, 裁章/作者触发) → 章节 prose                                            [本期不实现]
```

**段数固定、不随角色/回合数膨胀。** 我们消除的是「随角色数 N 倍增的扇出」（旧 `generateTurnContent` 逐回合请求），不是「所有多段调用」。`①(1) + ②(1)` 是两段固定调用。

**职责归属表：**

| 关注点 | 由谁产出 | 落到哪 |
|---|---|---|
| 事件模拟 / 台词 / 交涉 | ① 模拟段 (eino) | `result.EventPlan` + `result.Turns` + `result.InteractionAnalysis/Transcripts` |
| 剧情变量 | ① 确认（确定性种子 + 模型微调） | `result.PlotVariable` |
| 场景摘要 + 状态变化 + 角色记忆 | ② 复盘段 (eino) | `result.SceneSummary` + `result.MemoryPatch` |
| 写 StoryEvent（action_scheduled + scene_resolved） | `StorySessionAdvancer.persistResultEvents` (service) | 事件账本 |
| **提交角色记忆到外部库** | `StorySessionAdvancer.Generate`（run 完成时） | mem0/qdrant via `CharacterMemoryService.Commit` |
| 正文 prose | ③（推迟） | Chapter.Content（后续） |

---

## 2. 运行生命周期（演绎优先，章节后置）

```
StorySessionAdvancer.Advance(sessionID, input)      story_flow.go:49     [不变]
   建/取分支 → 追加用户消息 → 建 run(queued)
RunExecutor.handle → storyAdvancer.Generate(runID)  run_executor.go:90   [不变]
StorySessionAdvancer.Generate(runID):                story_flow.go:117    [改：收尾加记忆提交]
   1. load run/session；publish loading_state(10)
   2. result = storyGenerator.Generate(ctx, {Run,Session})   ← 内部跑 ①+②
   3. result.BranchID/BaseEventID/Status 落定
   4. persistResultEvents(run, result)   ← 写 action_scheduled + scene_resolved（含 turns/summary/memory_patch）
   5. ★ commitCharacterMemories(run, result)   ← 新增：把 result.MemoryPatch 的角色记忆提交外部库
   6. SaveRunResult；update session(CurrentPlotVariableSummary=SceneSummary)；publish completed
裁章 = 独立、后续的动作（见 §6），run 完成后随时可由 ③ / 作者触发，不在演绎链路里。
```

`port.StoryRunGenerator.Generate(input) (StoryRunResult, error)` 签名不变（runtime.go:65）。①②都在 generator 内部，靠 `deps.events` 发 SSE。

---

## 3. 阶段① 模拟（单次流式 NDJSON 调用）

### 3.1 输入：场景上下文（`buildSceneContext`，新增）

从 `loadStoryContext` 的全量快照拆成两层喂模型（视角受限靠这个结构，不靠"叮嘱"）：

**共享可观测层 `shared_observable`**：所有在场角色都能感知、对谁都安全的地面真相。
- 地点提示、时间；公开世界状态（`importance>=4 || volatility>=4`，复用 `visibleWorldForCharacter` 阈值口径）；公开可见动作；随回合进入的已说出口台词。

**每角色私有视图表 `character_views[]`**：每个 related character 一行，自包含（全部复用现成函数）：

| 字段 | 来源 |
|---|---|
| identity(profile/personality/voice_style/role/goals/fears/constraints) | `compactCharacters` / `model.Character` |
| known_facts | `storyVariableKnownFacts` |
| misreadings(尤其 believed_target_attitude) | `storyVariableMisreadings` / `relationshipViewsForCharacter` |
| secrets / private_attitude | `Character.SecretsJSON` + 该角色为 source 的 `RelationshipView` |
| recent_memories | `snapshot.RecentMemories[charID]`（来自上一轮②写入，见 §5 闭环） |
| emotional_pressure / action_bias | `CharacterVariableView`(`storyVariableCharacterViews`) |
| visible_world | `visibleWorldForCharacter(worldState, character)` |

`plot_variable_seed`：保留现 `generateStoryVariable` 的确定性逻辑（story_generator.go:186），作为种子；模型在 `plot_variable` 记录里确认/微调。

User payload 结构：
```json
{
  "story_run_id":"...","project_id":"...","session_id":"...",
  "session":{"title":"...","opening_situation":"...","author_intent":"...","last_author_message":"...","current_plot_variable":"..."},
  "author_bible": { /* compactAuthorBible */ },
  "plot_variable_seed": { /* PlotVariable 种子 */ },
  "shared_observable": { "location_hints":[...], "public_world_state":[...], "recent_chapters":[...] },
  "character_views": [ { "character_id":"c1","identity":{...},"known_facts":[...],"secrets":[...],"misreadings":[...],"private_attitude":[...],"recent_memories":[...],"visible_world":[...],"emotional_pressure":"...","action_bias":"..." } ],
  "constraints": { "max_turns": 25, "max_interactions": 3 }
}
```

### 3.2 输出：NDJSON 记录（一行一条 JSON，按固定顺序）

顺序：`plot_variable → event* → interaction* → turn* → stop`
**注意：v2 删除了 v1 的 `result`(含正文) 与 `draft_delta` 记录。终止记录改为 `stop`，不含正文、不含 memory。**

1) 剧情变量（1 条）— `model.PlotVariable`
```json
{"type":"plot_variable","plot_variable":{"pressure_source":"...","focal_character_id":"c1","core_choice":"...","option_a":"...","option_b":"...","cost_a":"...","cost_b":"...","irreversible_effect":"...","related_character_ids":["c1","c2"],"world_state_pressure":["k1"]}}
```

2) 事件模拟（N 条）— `model.StoryEventPlan`。**硬约束：必须产出**，`persistResultEvents` 靠它写 `action_scheduled`
```json
{"type":"event","event":{"time_index":1,"character_id":"c1","character_name":"林","location_key":"study","location_name":"书房","action_type":"action","summary":"独自销毁密信","intent":"灭证","visibility":"private","target_actor_ids":[]}}
```

3) 交涉组（0..3 条）— `model.StoryInteractionGroup`。`character_ids` 必须同地点
```json
{"type":"interaction","interaction_group":{"id":"interaction_1","location_key":"hall","location_name":"前厅","character_ids":["c1","c2"],"event_ids":["story_event_2","story_event_3"],"should_interact":true,"interaction_type":"confrontation","stakes":"林是否暴露昨晚行踪","rationale":"...","priority":1}}
```

4) 回合（M 条，**人物台词/动作的逐条载体，前端逐条渲染；也是③正文的素材**）
```json
{"type":"turn","turn":{"turn_index":1,"actor_id":"c2","actor_name":"沈","action_type":"speak","speech":"你昨晚去过书房？","action_summary":"","intent":"试探林","target_actor_ids":["c1"],"interaction_group_id":"interaction_1","location_key":"hall","phase":"negotiation"}}
```

5) 终止（1 条，最后）
```json
{"type":"stop","stop_reason":"交涉以僵局收束"}
```

### 3.3 NDJSON → SSE 映射（`consumeSceneStream`，新增）

事件名全部复用 `domain/status.go` 常量；前端无需新增事件类型。

| NDJSON 记录 | SSE 事件 | run step | 后端确定性动作 |
|---|---|---|---|
| 流开始 | `generation_step` | `generating_plot_variable`(20)→`planning_events`(30) | — |
| `plot_variable` | `plot_variable` | `generating_plot_variable`(20) | 校验 focal/related 合法（`normalizeStoryVariable`） |
| `event` | `story_event_planned` | `planning_events`(30) | 累积 EventPlan；块末用 `buildStoryLocationGroups` 派生同地点候选 |
| （事件块末，后端派生） | `same_location_candidates` | `selecting_interactions`(45) | 后端算 location_groups，**不要模型给** |
| `interaction` | `interaction_analysis` + (should_interact 时)`interaction_selected` | should_interact→`negotiating_interactions`(60)，否则 `selecting_interactions`(45) | 校验同地点（复用 selectStoryInteraction 校验）；≤3 组 |
| `turn` | `character_turn`（interaction_group_id 非空再发 `negotiation_turn`） | 首条 turn→`driving_character_turns`(status.go:22，启用) | 校验 actor/targets 属于交涉组（`validateStoryTurnInteraction`）；`upsertStoryInteractionTurn` 建 transcript；超 `max_turns` 截断 |
| `stop` | `generation_step` | — | 落定 stop_reason，结束① |

### 3.4 ① 的持久化（D7 硬约束：turns 落账本）

①结束后，`result` 至少带齐：`EventPlan[]`、`Turns[]`（**全部回合，含独白/旁白，不只交涉**）、`InteractionAnalysis`、`InteractionTranscripts[]`、`PlotVariable`。
其中 `Turns[]` 是新加的持久化字段（见 §7 模型变更），`persistResultEvents` 会把它写进 `scene_resolved` payload，供③后续渲染。

---

## 4. 阶段② 复盘/记忆（单次 LLM 调用）

模拟结束、拿到完整场景后，单独一次调用做"复盘"。**本质是把现有 `generateNarrativeMetadata`(story_generator.go:522) 抽成独立段**，去掉 prose、加每角色视角。

### 4.1 输入：`buildReflectionContext`（新增，由 Go 从①输出 + 快照拼）
```json
{
  "scene": { "plot_variable":{...}, "events":[...StoryEventPlan...], "turns":[...全部回合...], "interaction_transcripts":[...] },
  "characters": [{"id":"c1","name":"林","role":"..."}],
  "perception_index": [ {"character_id":"c1","witnessed_turn_indexes":[1,3],"witnessed_event_ids":["story_event_1"]} ],
  "prior_memories": { "c1":["..已有记忆.."], "c2":[...] },
  "relationships": [...], "world_state": [...]
}
```
`perception_index` 由 Go 确定性算：某角色"看得见"某 turn/event ⇔ 他是 actor / 在 target_actor_ids / 与该 turn 同 location_key（且该 event 非他人 private）。这把"谁感知到什么"变成可校验数据，供②做视角受限记忆。

### 4.2 输出（单 JSON；若要流式见 4.4）
```json
{
  "summary": "本场景一句到一段的整体摘要",
  "character_takeaways": [{"character_id":"c2","summary":"沈开始怀疑林"}],
  "memory_patch": {
    "character_memory_updates": [{"character_id":"c2","type":"belief","content":"林昨晚可能去过书房","importance":4}],
    "relationship_updates": [{"pair_id":"p1","summary":"...","tension_delta":"+1","events":[{"event_type":"negotiation","...":"..."}]}],
    "world_state_updates": [{"key":"...","operation":"update","value":"...","note":"..."}]
  }
}
```
直接对应现成 `model.MemoryPatch`(types.go:802) 三件套：`CharacterMemoryUpdate` / `RelationshipUpdate` / `WorldStateUpdate`。

### 4.3 ② 的视角受限契约（提示词）
- 每条 `character_memory_update` 只能基于该角色 `perception_index` 内的 turn/event。
- 不在场的角色不得记下密谈内容，至多记下其可观测到的外部事实（"看见两人进了书房"）。
- 允许写入"误读"（基于 misreadings 的错误信念），不得用全局真相纠正。
- 对 `prior_memories` 去重：已知的不重复写；只写"这一场景新产生/被改变"的。
- importance(1-5)、type(belief/event/emotion/relationship…) 由模型按戏剧权重赋值。

### 4.4 ② 是否流式（可选）
②较短，可不流式（一次 `model.Generate` 出单 JSON）。若要让前端看到"复盘在进行"：可新增可选 SSE 事件 `scene_summary` / `memory_update`（见 §7 新增常量），run step 用现成 `generating_memory_patch`(90)。**P0 可先不流式。**

### 4.5 ② 的产物去向
- `result.SceneSummary = summary`；`result.MemoryPatch = memory_patch`（run-local 拷贝，用于写事件 + 审计）。
- 角色记忆的**真正提交**在 service 层 run 完成时做（§5）。
- `summary` 写进 `scene_resolved` payload，裁章/③后续用它做 `chapter.Summary`（现 `createChapterFromEvents:190` 从 draft.Summary 收，改从 summary 收）。

---

## 5. 记忆提交与分支作用域（D5）

**时机：run 完成（②摘要做完）即提交，不是裁章。** 把提交逻辑从 cutter 移到 advancer。

`StorySessionAdvancer.Generate` 在 `persistResultEvents` 之后、`SaveRunResult` 之前，新增 `commitCharacterMemories`：
- 由 `result.MemoryPatch.CharacterMemoryUpdates` 构造 `[]model.Memory`，每条打标：
  - `SourceRunID = run.RunID`，`BranchID = run.BranchID`，`SourceEventID = sceneEvent.ID`（scene_resolved 事件 id）
  - `Note = MemoryScopeExternalCommitted + ":" + MemoryCommitTriggerRunCompletion`（新增 trigger 常量）
- 调 `CharacterMemoryService.Commit`（需把 `memoryService` 注入 `StorySessionAdvancer`，见 §7）。
- 失败处理参照现有：记审计事件 `external_memory_flush_failed`，不阻断 run 完成。

**`scene_resolved` 事件仍带 `StateDelta.MemoryPatch`**（审计 / 重放 / 分支用），与外部提交是两条并行：账本是真相，外部库是可检索投影。

**分支作用域注记（重要但不阻塞 P0）：** 因为账本可 fork/rollback，run 完成即入外部库意味着外部库会含"某分支上发生过"的记忆。靠上面打的 `branch_id`/`source_event_id` 标签，`recallCharacterMemories`(story_tools.go:150) 后续应能按"当前分支祖先链"过滤，避免跨废弃分支串味。P0 先打标签、recall 过滤留作 follow-up（在 `CharacterMemoryService.Recall` 入参里加 branch 维度）。

**cutter 不再提交记忆**（§7 删除 `flushExternalMemories` 及其调用），避免双写。

---

## 6. 阶段③ 正文（推迟、解耦 —— 仅设计，本期不实现）

裁章/正文是"演绎之后"的事。run 完成 = 演绎完成 + 记忆已更新；写不写章、怎么写，互不影响。

两条候选路线（D6，后续二选一或并存）：

**③-A 精选 + 小说写法自动成文**
1. 选材 agent：扫分支事件链，按戏剧权重选出"出彩的事情"（高 stakes 的 scene_resolved / 关键 turn）。
2. 成文 agent：把选中 span 的 `turns`(speech/action/intent) + `interaction_transcripts` + `author_bible.style_guide` + `recent_chapters`(声音/连贯)，按小说自有写法（POV 纪律、scene/sequel、show-don't-tell）铺成 prose。
适合"自动出精华稿"。

**③-B 作者对话驱动成文**
作者跟 agent 对话表达意图（"把书房对峙写成林的视角、短句、紧张"），agent 拉取相关已模拟素材（events/turns/transcripts/summary），调用"写作工具"按作者指令渲染 prose。落在现有统一对话 Agent 那条线上扩展。适合"作者在环、可控"。

**两条路都吃同一份账本素材**，所以本期唯一义务：**把 §3.4 的 turns + §4.5 的 summary 充分落进 `scene_resolved`**，使后续任一路线都能取到料。

**裁章语义随③重构（本期不做，只标）：**
- 现 `createChapterFromEvents`(story_chapter_cut.go:173) 把 scene_resolved 的 `draft.Content` 拼成章节，且 :213 没 content 就报错。
- 演绎段不再产 prose 后：裁章应退化为"圈定章节 span + 章节壳"，prose 由③填。
- **P0 最小护栏**：把 :213 的硬失败改为"允许空 content（产出 summary-only 章节壳）"，避免无 prose 时崩溃；完整重构与③一起做。

---

## 7. 文件级改动清单

### `internal/infrastructure/ai/eino/story_generator.go`
- **重写 `Generate`**(:94)：`loadStoryContext → seedPlotVariable → simulateScene(①) → reflectScene(②) → assembleStoryRunResult`。保留 `publishStoryOrchestrationStarted` 和进度推送骨架。
- **新增**：`buildSceneContext`、`simulateScene`（model.Stream + `consumeSceneStream` NDJSON 解析）、`buildReflectionContext`、`reflectScene`（model.Generate 出复盘 JSON）、`assembleStoryRunResult`。
- **删除**：`generateNarrative`(:471)、`generateTurnContent`(:489)、`generateNarrativeMetadata`(:522)（其职责并入 `reflectScene`）、`assembleChapterDraft`(:557)、`perspectiveForTurn`(:674，视角逻辑并入 `buildSceneContext`)、`turnNarrativePrompt`/`statePatchPrompt`/`chapterAssemblerPrompt`/`variableSystemPrompt` 及相关 fallback/merge。
- **保留并复用**：`generateStoryVariable`→`seedPlotVariable`(:186)、`visibleWorldForCharacter`(:778)、`relationshipViewsForCharacter`(:795)、`storyVariableCharacterViews`(:268)、`storyVariableKnownFacts`/`Misreadings`、`normalizeStoryVariable`(:815)、`compact*`。`buildResult`(:693) 改造为"无正文版"（Draft.Content 空、Draft.Title/Summary 取自②）。`nextChapterNumber` 移交③/cut（本期可留着不调）。

### `internal/infrastructure/ai/eino/story_tools.go`
- **删除 ReAct 工具层**：`newStoryTools`(:49) 及 5 个 InferTool 包装；`storyRunState` 中仅服务工具的可变状态/方法。
- **保留并复用为"流消费期的确定性派生/校验"**：`buildStoryLocationGroups`、`locationGroupByKey`、`validateStoryTurnInteraction`、`interactionLocation`、`upsertStoryInteractionTurn`、`resolveStoryActor`、`normalizeStoryActionType`、`validStoryTargetActorIDs`、`copyStory*`、`publishStoryEvent`、`updateStoryRunStep`、`storyTurnDisplayPayload`。从"工具回调"改为"`consumeSceneStream` 逐记录调用"。新增 `buildPerceptionIndex`（§4.1）。

### `internal/infrastructure/ai/eino/story_types.go`
- **新增**：① NDJSON 记录类型（`sceneRecord{Type,...}` 及各 payload）、`SceneContext`、② 的 `ReflectionContext` / `ReflectionResult`。
- **保留**：`StoryTurnPlan`/`StoryPlanResult`/`CharacterPerspective`/`CharacterVariableView` 作累积/上下文类型。纯逐回合渲染相关类型可删。

### `internal/application/model/types.go`
- **`StoryRunResult` 新增**：`Turns []StoryTurn`（持久化全部回合，D7）、`SceneSummary string`。
- **新增 `model.StoryTurn`**：持久化字段 = turn_index, actor_id/name, action_type, speech, action_summary, intent, target_actor_ids, interaction_group_id, location_key/name, phase（即 eino `StoryTurnPlan` 去掉 Content/Rationale）。
- **`model.Memory` 确认/新增标签字段**：`SourceRunID`、`BranchID`、`SourceEventID`（若无则加）供 §5 打标与后续分支过滤。

### `internal/application/service/story_flow.go`
- **`StorySessionAdvancer` 注入 `memoryService port.CharacterMemoryService`**（构造函数 + bootstrap 接线）。
- **`Generate`**(:117)：`persistResultEvents` 后新增 `commitCharacterMemories(run, result, sceneEventID)`（§5）；`session.CurrentPlotVariableSummary` 改取 `result.SceneSummary`。
- **`persistResultEvents`**(:163)：`scene_resolved` payload 增加 `"turns": result.Turns` 与 `"summary": result.SceneSummary`；`"draft"` 改为空壳或移除（无正文）。`StateDelta.MemoryPatch` 仍写。需把 sceneEvent.ID 回传给 `commitCharacterMemories`。

### `internal/application/service/story_chapter_cut.go`
- **删除记忆提交**：`flushExternalMemories`(:273)、`appendExternalMemoryCommittedEvent`、`createChapterFromEvents` 中构造 `memories` 的段（:198-211）；`StoryChapterCutter` 去掉 `memoryService` 依赖。
- **`createChapterFromEvents`**：`summary` 改从 scene_resolved payload `summary` 收；:213 硬失败改为"允许空 content"（§6 护栏）。完整重构随③。

### `internal/domain/status.go`
- 新增 `MemoryCommitTriggerRunCompletion`。
- 可选新增 `EventSceneSummary`/`EventMemoryUpdate`（②流式用，P0 可不加）。
- 启用现有 `RunStatusDrivingCharacterTurns`(:22)。`EventDraftDelta` 保留给③。

### `internal/application/port/*.go`
- `StoryRunGenerator.Generate` 签名**不变**。`CharacterMemoryService` 接口不变（继续用 `Commit`）。
- 可选：`CharacterMemoryRecallInput` 后续加 `BranchID`（§5 follow-up，P0 不强制）。

### `internal/bootstrap/app.go`
- `StorySessionAdvancer` 接线加 `memoryService`；`StoryChapterCutter` 去掉 `memoryService`。

### `deployments/config/config.example.yaml`（+ `internal/config/config.go` 同步）
- `story_agent` 下：新增 `scene_prompt`(①) 与 `reflect_prompt`(②)；新增 `max_scene_tokens`、`max_reflect_tokens`。
- 删除/合并：`controller_prompt`/`tool_prompt`/`narrative_prompt`/`variable_prompt`/`simulation_prompt`/`result_prompt`（result_prompt 可留作①非流式回退）；`max_react_steps`/`max_turn_tokens`/`max_assembler_tokens`/`max_chapter_tokens` 删除。`max_turns` 保留（提示词软上限 + 消费端硬截断）。

---

## 8. 提示词骨架（落 config）

### ① `scene_prompt`（System）
```
你是 NovelOS 的「单次场景模拟器」。你只模拟「发生了什么」：事件、同地点交涉判定、多人物逐回合台词/动作。
你不写章节正文，不总结记忆——那是后续阶段的事。

【视角受限契约 —— 最高优先级】
- 你会收到「共享可观测层」和「每角色私有视图表」。
- turns 里每个角色的台词/动作只能由(共享可观测层 + 该角色自己那一行)推出。
- 禁止让角色使用他人 secrets / private_attitude / 未公开 known_facts，除非已被说出口/被看见而进入可观测层。
- 角色 misreadings 必须照错信念行动，不得用全局真相纠正。旁白可全知但不得把不可观测知识塞进角色。

【输出 —— 严格 NDJSON，一行一条 JSON，不要数组，不要 markdown 围栏】
顺序：1 行 plot_variable → N 行 event → 0~3 行 interaction（同地点）→ M 行 turn → 1 行 stop。
最多 {max_turns} 个 turn。无可交互组就用 narration/action 推进，不硬造对话。
每行必须合法 JSON，字段缺省给空串/空数组，不要省略 type。不要输出 content、不要输出 memory。
```

### ② `reflect_prompt`（System）
```
你是 NovelOS 的「场景复盘 / 记忆 agent」。给你一个已模拟完成的场景（plot_variable、events、turns、transcripts）
和每个角色的 perception_index（他在本场景看得见哪些 turn/event）、prior_memories。

输出单个 JSON：{summary, character_takeaways[], memory_patch{character_memory_updates[],relationship_updates[],world_state_updates[]}}。

【记忆视角受限】
- 每条 character_memory_update 只能基于该角色 perception_index 内的 turn/event；不在场者至多记可观测外部事实。
- 允许写入基于 misreadings 的错误信念，不得用全局真相纠正。
- 对 prior_memories 去重，只写本场景新产生/被改变的；importance 1-5 按戏剧权重。
- relationship_updates 把交涉造成的关系变化写进 events(event_type=negotiation/interaction_outcome)。
不要写正文。只输出 JSON。
```

---

## 9. 健壮性与边界

1. **NDJSON 可靠性**：低温度 + 严格格式 + 1 个 few-shot；解析器容错：跳过空行、剥 ```json 围栏、单行失败则与下一行拼接重试；整体失败回退到一次非流式结构化调用（用保留的 `result_prompt`），再走同样 assemble。
2. **EventPlan 硬约束**（D7）：若只给 turn 不给 event，从 turns 反推最小 event_plan（actor+location+action_type+summary），保证 persist 有料。
3. **max_turns 执行点**：移到 `consumeSceneStream`（累计到上限丢弃后续 turn + 标 stop_reason）。
4. **不变量**：消费 `interaction`/`turn` 复用同地点校验 + `validateStoryTurnInteraction`；违例降级（丢弃/转 narration）+ 记 `continuity_issues`。
5. **②失败**：②失败不应丢掉①的演绎结果——记 `external_memory_flush_failed`/`reflection_failed` 审计事件，用空 memory_patch + 占位 summary 完成 run（演绎仍 completed），记忆缺失留待补跑。
6. **持久化时序**：流仅用于实时 UX；权威 `StoryRunResult` 在①②都结束后 assemble 再持久化；中途失败=run 失败（同现状）。
7. **取消/超时**：`Generate` 的 5min ctx 保留；流读取响应 `ctx.Done()`。
8. **token**：`max_scene_tokens` 要够容纳整场 turns；输入侧用 `compact*` 压缩每角色视图。

## 10. 视角泄漏兜底（D9，P0 只做第一档）
- 第一档（**P0 做**）：§3.1/§4.1 的结构化视图 + §8 提示词契约。对 flavor 级受限视角，强模型 + 该结构足够。
- 第二档（**不做**）：result 带 `perspective_violations` 自检字段 / 每 turn 带 `grounded_on`。
- 第三档（**不做**）：确定性关键词校验。
- 仅当测试观察到"硬剧透秘密泄漏"才补：对涉密角色的那几条 turn 做**一次定向重问**（按交涉组，不回到逐回合扇出）。

## 11. 分阶段落地
- **P0-契约**：定 §3.2 NDJSON + §3.3 SSE 映射 + §4 复盘 I/O + 前端幂等渲染约定。
- **P1**：`buildSceneContext` / `buildPerceptionIndex`（复用现成视图函数）。
- **P2**：`scene_prompt` / `reflect_prompt` + config 字段。
- **P3**：`simulateScene`（Stream→行解析→SSE+累积，含容错/回退）。
- **P4**：`reflectScene` + `assembleStoryRunResult`。
- **P5**：`StorySessionAdvancer` 接 memoryService + `commitCharacterMemories`；`persistResultEvents` 加 turns/summary；cutter 删记忆 + :213 护栏。
- **P6**：删 §7 死代码 + config 收口。
- **P7**：测试（§12）。
- **后续（非 P0）**：阶段③ 正文 + 裁章语义重构 + recall 的 branch 过滤。

## 12. 测试点
- 解析器：合法 NDJSON / 带围栏 / 半行拼接 / 整体失败回退。
- 不变量：交涉非同地点被拒；turn 不属于已选组降级；max_turns 截断；只给 turn 不给 event 能反推。
- ② 视角受限：不在场角色不会记下密谈内容（用 perception_index 构造用例）；prior_memories 去重。
- 记忆提交：run 完成即 Commit（mock CharacterMemoryService）；打标 source_run/branch/event；cutter 不再二次提交。
- assemble：累积记录 → StoryRunResult（EventPlan/Turns/InteractionAnalysis/Transcripts/PlotVariable/SceneSummary/MemoryPatch 齐全，Draft.Content 空）。
- service 回归：`story_flow` 现有用例按"无正文 + run 末提交记忆"调整后通过；`story_chapter_cut` 用例去掉记忆断言、加空 content 护栏断言。

---

## 13. 一句话总结
演绎优先、章节后置：把 `StoryRunGenerator.Generate` 拆成**①单次流式模拟（出台词、不出正文/记忆）**+**②单次复盘（出摘要+状态变化+各角色视角受限记忆）**；记忆在 run 完成时即提交外部库（不是裁章）；正文是解耦的后续阶段（精选自动成文 / 作者对话驱动，二选一，本期不实现，只保证账本素材充足）。接口与不变量不变，turns 必落账本。
