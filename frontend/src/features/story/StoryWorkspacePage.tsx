// 故事推进工作台。这是 MVP 主创作闭环：创建/选择 story session、推进 run、查看流式输出并提交正史。
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Check, Send } from 'lucide-react';
import { FormEvent, useMemo, useState, type ReactNode } from 'react';
import { useParams } from 'react-router-dom';

import { advanceStorySession, createStorySession, listStorySessions } from '../../api/storySessions';
import { commitStoryRun, getStoryRun, getStoryRunResult, listStoryRunEventHistory } from '../../api/storyRuns';
import { EmptyState } from '../../components/feedback/EmptyState';
import { ErrorState } from '../../components/feedback/ErrorState';
import { LoadingState } from '../../components/feedback/LoadingState';
import type {
  RunEvent,
  StoryCharacterMemoryUpdate,
  StoryMemoryPatch,
  StoryRelationshipUpdate,
  StoryReviewReport,
  StoryRunResult,
  StoryWorldStateUpdate,
} from '../../types/api';
import { useStoryRunEvents } from './useStoryRunEvents';

type PatchDecision = 'accept' | 'defer';

export function StoryWorkspacePage() {
  const { projectId = '' } = useParams();
  const queryClient = useQueryClient();
  const [sessionTitle, setSessionTitle] = useState('');
  const [activeSessionId, setActiveSessionId] = useState('');
  const [authorMessage, setAuthorMessage] = useState('');
  const [activeRunId, setActiveRunId] = useState('');
  const [authorNote, setAuthorNote] = useState('');
  const [patchDecisions, setPatchDecisions] = useState<Record<string, PatchDecision>>({});

  const sessionsQuery = useQuery({
    queryKey: ['storySessions', projectId, 1, 20],
    queryFn: ({ signal }) => listStorySessions(projectId, 1, 20, signal),
    enabled: Boolean(projectId),
  });

  const sessions = sessionsQuery.data?.data ?? [];
  const selectedSessionId = activeSessionId || sessions[0]?.id || '';

  const runQuery = useQuery({
    queryKey: ['storyRun', activeRunId],
    queryFn: ({ signal }) => getStoryRun(activeRunId, signal),
    enabled: Boolean(activeRunId),
    refetchInterval: (query) => {
      // run 状态用轮询；正文增量可通过 SSE 到达。
      const status = query.state.data?.status;
      return isActiveRunStatus(status) ? 1500 : false;
    },
  });

  const resultQuery = useQuery({
    queryKey: ['storyRunResult', activeRunId],
    queryFn: ({ signal }) => getStoryRunResult(activeRunId, signal),
    enabled: Boolean(activeRunId) && Boolean(runQuery.data?.status) && !isActiveRunStatus(runQuery.data?.status),
  });

  const eventHistoryQuery = useQuery({
    queryKey: ['storyRunEventHistory', activeRunId],
    queryFn: ({ signal }) => listStoryRunEventHistory(activeRunId, signal),
    enabled: Boolean(activeRunId),
  });

  const eventState = useStoryRunEvents(activeRunId);

  const createSessionMutation = useMutation({
    mutationFn: () => createStorySession(projectId, { title: sessionTitle.trim() || undefined }),
    onSuccess: (session) => {
      setActiveSessionId(session.id);
      setSessionTitle('');
      queryClient.invalidateQueries({ queryKey: ['storySessions', projectId] });
    },
  });

  const advanceMutation = useMutation({
    mutationFn: () => advanceStorySession(selectedSessionId, { author_message: authorMessage.trim() }),
    onSuccess: (run) => {
      setActiveRunId(run.run_id ?? run.id ?? '');
      setAuthorMessage('');
    },
  });

  const draftId = resultQuery.data?.draft?.id ?? resultQuery.data?.draft_id ?? '';
  const memoryPatchId = resultQuery.data?.memory_patch?.id ?? resultQuery.data?.memory_patch_id ?? '';
  const hasDeferredPatchItems = Object.values(patchDecisions).includes('defer');
  const isRunCommitted = runQuery.data?.status === 'committed' || Boolean(runQuery.data?.committed_at);
  const committedCharacterIds = useMemo(() => {
    const updates = resultQuery.data?.memory_patch?.character_memory_updates ?? [];
    return Array.from(new Set(updates.map((update) => update.character_id).filter(Boolean)));
  }, [resultQuery.data?.memory_patch?.character_memory_updates]);

  const setPatchDecision = (key: string, decision: PatchDecision) => {
    setPatchDecisions((current) => ({ ...current, [key]: decision }));
  };

  const commitMutation = useMutation({
    mutationFn: () =>
      // commit 是前端唯一把候选故事结果写入正式章节/记忆数据的动作。
      commitStoryRun(activeRunId, {
        draft_id: draftId,
        memory_patch_id: memoryPatchId,
        author_note: authorNote.trim() || undefined,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['chapters', projectId] });
      queryClient.invalidateQueries({ queryKey: ['project', projectId] });
      queryClient.invalidateQueries({ queryKey: ['storySessions', projectId] });
      queryClient.invalidateQueries({ queryKey: ['storyRun', activeRunId] });
      queryClient.invalidateQueries({ queryKey: ['storyRunResult', activeRunId] });
      queryClient.invalidateQueries({ queryKey: ['storyRunEventHistory', activeRunId] });
      queryClient.invalidateQueries({ queryKey: ['relationships', projectId] });
      queryClient.invalidateQueries({ queryKey: ['authorBible', projectId] });
      queryClient.invalidateQueries({ queryKey: ['characters', projectId] });
      committedCharacterIds.forEach((characterId) => {
        queryClient.invalidateQueries({ queryKey: ['memories', characterId] });
      });
      setPatchDecisions({});
      setAuthorNote('');
    },
  });

  const visibleDraft = useMemo(() => {
    // 有最终 result 时优先展示最终正文，否则展示 SSE 流式增量。
    return resultQuery.data?.content ?? resultQuery.data?.draft?.content ?? eventState.draftText;
  }, [eventState.draftText, resultQuery.data]);

  const startSession = (event: FormEvent) => {
    event.preventDefault();
    createSessionMutation.mutate();
  };

  const sendAdvance = (event: FormEvent) => {
    event.preventDefault();
    advanceMutation.mutate();
  };

  return (
    <div className="workspace workspace--three">
      <aside className="workspace-panel">
        <h2>故事会话</h2>
        <form className="stack-list" onSubmit={startSession}>
          <input
            value={sessionTitle}
            onChange={(event) => setSessionTitle(event.target.value)}
            placeholder="会话标题，可选"
          />
          <button className="button" disabled={createSessionMutation.isPending} type="submit">
            创建会话
          </button>
        </form>
        {sessionsQuery.isLoading ? <LoadingState /> : null}
        <div className="session-list">
          {sessions.map((session) => (
            <button
              className={selectedSessionId === session.id ? 'session-item session-item--active' : 'session-item'}
              key={session.id}
              onClick={() => setActiveSessionId(session.id)}
              type="button"
            >
              <strong>{session.title || session.id}</strong>
              <span>{session.status || 'ready'}</span>
            </button>
          ))}
        </div>
      </aside>

      <section className="workspace-main">
        <div className="page__header">
          <div>
            <h1>故事推进工作台</h1>
            <p>输入推进语，查看生成过程，审阅候选结果后再提交为正史。</p>
          </div>
          {runQuery.data?.status ? <span className="status-pill">{runQuery.data.status}</span> : null}
        </div>

        {createSessionMutation.isError ? <ErrorState message={(createSessionMutation.error as Error).message} /> : null}
        {advanceMutation.isError ? <ErrorState message={(advanceMutation.error as Error).message} /> : null}
        {resultQuery.isError ? <ErrorState message={(resultQuery.error as Error).message} /> : null}

        {!selectedSessionId ? (
          <EmptyState title="请先创建故事会话" description="创建后输入推进语，系统会生成候选章节草稿。" />
        ) : (
          <>
            <div className="draft-surface">
              {resultQuery.data?.draft ? (
                <div className="draft-meta">
                  <strong>{resultQuery.data.draft.title ?? '未命名草稿'}</strong>
                  <span>第 {resultQuery.data.draft.chapter_number ?? '-'} 章</span>
                  <span>{resultQuery.data.draft.word_count ?? 0} 字</span>
                </div>
              ) : null}
              {visibleDraft ? <pre>{visibleDraft}</pre> : <p className="muted">生成的章节草稿会显示在这里。</p>}
            </div>
            {resultQuery.data ? (
              <StoryResultPreview
                result={resultQuery.data}
                patchDecisions={patchDecisions}
                onPatchDecisionChange={setPatchDecision}
              />
            ) : null}
            <form className="composer" onSubmit={sendAdvance}>
              <textarea
                value={authorMessage}
                onChange={(event) => setAuthorMessage(event.target.value)}
                placeholder="例如：在雨夜对峙中揭露一个线索"
                rows={4}
              />
              <button
                className="button"
                disabled={!authorMessage.trim() || !selectedSessionId || advanceMutation.isPending}
                type="submit"
              >
                <Send size={17} />
                推进
              </button>
            </form>
          </>
        )}
      </section>

      <aside className="workspace-panel">
        <h2>运行事件</h2>
        <div className="status-line">实时流：{formatConnectionStatus(eventState.connectionStatus)}</div>
        <div className="event-section">
          <h3>生成步骤</h3>
          {eventState.generationSteps.map((item, index) => (
            <pre key={index}>{JSON.stringify(item, null, 2)}</pre>
          ))}
        </div>
        <div className="event-section">
          <h3>剧情变量</h3>
          {eventState.plotVariables.map((item, index) => (
            <pre key={index}>{JSON.stringify(item, null, 2)}</pre>
          ))}
        </div>
        <div className="event-section">
          <h3>角色回合</h3>
          {eventState.characterTurns.map((item, index) => (
            <pre key={index}>{JSON.stringify(item, null, 2)}</pre>
          ))}
        </div>
        <div className="event-section">
          <h3>等待审阅</h3>
          {eventState.reviewItems.map((item, index) => (
            <pre key={index}>{JSON.stringify(item, null, 2)}</pre>
          ))}
        </div>
        <RunEventHistoryPanel
          events={eventHistoryQuery.data ?? []}
          isLoading={eventHistoryQuery.isLoading}
          error={eventHistoryQuery.error as Error | null}
        />
        <div className="commit-box">
          <textarea
            value={authorNote}
            onChange={(event) => setAuthorNote(event.target.value)}
            placeholder="提交备注"
            rows={4}
          />
          <button
            className="button"
            disabled={
              !activeRunId || !draftId || !memoryPatchId || hasDeferredPatchItems || isRunCommitted || commitMutation.isPending
            }
            onClick={() => commitMutation.mutate()}
            type="button"
          >
            <Check size={17} />
            提交正史
          </button>
          {!draftId || !memoryPatchId ? (
            <small className="muted">等待生成结果中的 draft_id 和 memory_patch_id。</small>
          ) : null}
          {hasDeferredPatchItems ? <small className="muted">提交前请先接受或移除暂缓的补丁项。</small> : null}
          {isRunCommitted ? <small className="muted">这次运行已经提交过。</small> : null}
          {commitMutation.isError ? <ErrorState message={(commitMutation.error as Error).message} /> : null}
        </div>
      </aside>
    </div>
  );
}

function RunEventHistoryPanel({
  events,
  isLoading,
  error,
}: {
  events: RunEvent[];
  isLoading: boolean;
  error: Error | null;
}) {
  return (
    <div className="event-section">
      <h3>历史事件</h3>
      <small className="muted">这里显示已持久化的运行事件；上方实时流仍然只属于运行态。</small>
      {isLoading ? <LoadingState label="正在加载历史事件" /> : null}
      {error ? <ErrorState message={error.message} /> : null}
      {!isLoading && !error && events.length === 0 ? <p className="muted">暂无已持久化事件。</p> : null}
      {events.map((event) => (
        <pre key={event.id}>
          {JSON.stringify(
            {
              sequence: event.sequence,
              event_name: event.event_name,
              payload: event.payload,
              created_at: event.created_at,
            },
            null,
            2,
          )}
        </pre>
      ))}
    </div>
  );
}

function formatConnectionStatus(status: string) {
  if (status === 'idle') {
    return '空闲';
  }
  if (status === 'connecting') {
    return '连接中';
  }
  if (status === 'open') {
    return '已连接';
  }
  if (status === 'error') {
    return '异常';
  }
  return status;
}

function isActiveRunStatus(status?: string) {
  return status === 'queued' || status === 'running' || status === 'loading_state';
}

function StoryResultPreview({
  result,
  patchDecisions,
  onPatchDecisionChange,
}: {
  result: StoryRunResult;
  patchDecisions: Record<string, PatchDecision>;
  onPatchDecisionChange: (key: string, decision: PatchDecision) => void;
}) {
  const plot = result.plot_variable;
  const review = result.review;
  const patch = result.memory_patch;

  return (
    <div className="structured-result">
      {plot ? (
        <section className="result-section">
          <div className="result-section__header">
            <h2>剧情变量</h2>
            {plot.focal_character_id ? <span className="status-pill">{plot.focal_character_id}</span> : null}
          </div>
          <div className="key-value-grid">
            <KeyValue label="压力来源" value={plot.pressure_source} />
            <KeyValue label="核心选择" value={plot.core_choice} />
            <KeyValue label="选项 A" value={plot.option_a} />
            <KeyValue label="代价 A" value={plot.cost_a} />
            <KeyValue label="选项 B" value={plot.option_b} />
            <KeyValue label="代价 B" value={plot.cost_b} />
            <KeyValue label="不可逆影响" value={plot.irreversible_effect} />
            <KeyValue label="世界压力" value={plot.world_state_pressure?.join(', ')} />
          </div>
        </section>
      ) : null}

      {review ? <ReviewPreview review={review} /> : null}

      {patch ? (
        <MemoryPatchPreview
          patch={patch}
          decisions={patchDecisions}
          onDecisionChange={onPatchDecisionChange}
        />
      ) : null}
    </div>
  );
}

function KeyValue({ label, value }: { label: string; value?: string }) {
  return (
    <div className="kv-item">
      <span>{label}</span>
      <strong>{value || '-'}</strong>
    </div>
  );
}

function ReviewPreview({ review }: { review: StoryReviewReport }) {
  const groups = [
    ['硬性违规', review.hard_violations],
    ['连续性问题', review.continuity_issues],
    ['风格问题', review.style_issues],
    ['建议修复', review.suggested_fixes],
  ] as const;

  return (
    <section className="result-section">
      <div className="result-section__header">
        <h2>审校报告</h2>
        <span className={review.pass ? 'status-pill' : 'status-pill status-pill--warning'}>
          {review.pass ? '通过' : '需审阅'}
        </span>
      </div>
      <div className="review-grid">
        {groups.map(([label, values]) => (
          <div className="review-list" key={label}>
            <h3>{label}</h3>
            {values && values.length > 0 ? (
              <ul>
                {values.map((value, index) => (
                  <li key={`${label}-${index}`}>{value}</li>
                ))}
              </ul>
            ) : (
              <p className="muted">无</p>
            )}
          </div>
        ))}
      </div>
    </section>
  );
}

function MemoryPatchPreview({
  patch,
  decisions,
  onDecisionChange,
}: {
  patch: StoryMemoryPatch;
  decisions: Record<string, PatchDecision>;
  onDecisionChange: (key: string, decision: PatchDecision) => void;
}) {
  const memoryUpdates = patch.character_memory_updates ?? [];
  const relationshipUpdates = patch.relationship_updates ?? [];
  const worldUpdates = patch.world_state_updates ?? [];

  return (
    <section className="result-section">
      <div className="result-section__header">
        <h2>记忆补丁</h2>
        {patch.status ? <span className="status-pill">{patch.status}</span> : null}
      </div>

      <PatchGroup title="角色记忆">
        {memoryUpdates.length > 0 ? (
          memoryUpdates.map((item, index) => (
            <CharacterMemoryPatchItem
              item={item}
              itemKey={patchItemKey('memory', index, item.character_id)}
              key={patchItemKey('memory', index, item.character_id)}
              decision={decisions[patchItemKey('memory', index, item.character_id)] ?? 'accept'}
              onDecisionChange={onDecisionChange}
            />
          ))
        ) : (
          <p className="muted">暂无角色记忆更新。</p>
        )}
      </PatchGroup>

      <PatchGroup title="关系更新">
        {relationshipUpdates.length > 0 ? (
          relationshipUpdates.map((item, index) => (
            <RelationshipPatchItem
              item={item}
              itemKey={patchItemKey('relationship', index, item.pair_id ?? item.summary)}
              key={patchItemKey('relationship', index, item.pair_id ?? item.summary)}
              decision={decisions[patchItemKey('relationship', index, item.pair_id ?? item.summary)] ?? 'accept'}
              onDecisionChange={onDecisionChange}
            />
          ))
        ) : (
          <p className="muted">暂无关系更新。</p>
        )}
      </PatchGroup>

      <PatchGroup title="世界状态更新">
        {worldUpdates.length > 0 ? (
          worldUpdates.map((item, index) => (
            <WorldStatePatchItem
              item={item}
              itemKey={patchItemKey('world', index, item.key)}
              key={patchItemKey('world', index, item.key)}
              decision={decisions[patchItemKey('world', index, item.key)] ?? 'accept'}
              onDecisionChange={onDecisionChange}
            />
          ))
        ) : (
          <p className="muted">暂无世界状态更新。</p>
        )}
      </PatchGroup>
    </section>
  );
}

function PatchGroup({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="patch-group">
      <h3>{title}</h3>
      <div className="patch-list">{children}</div>
    </div>
  );
}

function CharacterMemoryPatchItem({
  item,
  itemKey,
  decision,
  onDecisionChange,
}: {
  item: StoryCharacterMemoryUpdate;
  itemKey: string;
  decision: PatchDecision;
  onDecisionChange: (key: string, decision: PatchDecision) => void;
}) {
  return (
    <PatchItemShell itemKey={itemKey} decision={decision} onDecisionChange={onDecisionChange}>
      <strong>{item.character_id || '未知角色'}</strong>
      <p>{item.content || '-'}</p>
      <small>{[item.type, item.importance ? `重要度 ${item.importance}` : undefined].filter(Boolean).join(' · ')}</small>
    </PatchItemShell>
  );
}

function RelationshipPatchItem({
  item,
  itemKey,
  decision,
  onDecisionChange,
}: {
  item: StoryRelationshipUpdate;
  itemKey: string;
  decision: PatchDecision;
  onDecisionChange: (key: string, decision: PatchDecision) => void;
}) {
  return (
    <PatchItemShell itemKey={itemKey} decision={decision} onDecisionChange={onDecisionChange}>
      <strong>{item.pair_id || item.pair?.id || '新的关系更新'}</strong>
      <p>{item.summary || item.tension_delta || '-'}</p>
      <small>
        {[
          item.views?.length ? `${item.views.length} 个视角` : undefined,
          item.events?.length ? `${item.events.length} 个事件` : undefined,
        ]
          .filter(Boolean)
          .join(' · ')}
      </small>
    </PatchItemShell>
  );
}

function WorldStatePatchItem({
  item,
  itemKey,
  decision,
  onDecisionChange,
}: {
  item: StoryWorldStateUpdate;
  itemKey: string;
  decision: PatchDecision;
  onDecisionChange: (key: string, decision: PatchDecision) => void;
}) {
  return (
    <PatchItemShell itemKey={itemKey} decision={decision} onDecisionChange={onDecisionChange}>
      <strong>{item.key || '世界状态'}</strong>
      <p>{item.note || stringifyValue(item.value)}</p>
      <small>{item.operation || '更新'}</small>
    </PatchItemShell>
  );
}

function PatchItemShell({
  children,
  itemKey,
  decision,
  onDecisionChange,
}: {
  children: ReactNode;
  itemKey: string;
  decision: PatchDecision;
  onDecisionChange: (key: string, decision: PatchDecision) => void;
}) {
  return (
    <div className={decision === 'defer' ? 'patch-item patch-item--deferred' : 'patch-item'}>
      <div className="patch-item__body">{children}</div>
      <div className="segmented-control" role="group">
        <button
          className={decision === 'accept' ? 'segmented-control__item segmented-control__item--active' : 'segmented-control__item'}
          onClick={() => onDecisionChange(itemKey, 'accept')}
          type="button"
        >
          接受
        </button>
        <button
          className={decision === 'defer' ? 'segmented-control__item segmented-control__item--active' : 'segmented-control__item'}
          onClick={() => onDecisionChange(itemKey, 'defer')}
          type="button"
        >
          暂缓
        </button>
      </div>
    </div>
  );
}

function patchItemKey(kind: string, index: number, id?: string) {
  return `${kind}:${id || 'new'}:${index}`;
}

function stringifyValue(value: unknown) {
  if (typeof value === 'string') {
    return value;
  }
  if (value == null) {
    return '-';
  }
  return JSON.stringify(value);
}
