import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Check, Pencil, Save, Send, Trash2, X } from 'lucide-react';
import { type FormEvent, useMemo, useState, type ReactNode } from 'react';
import { useParams } from 'react-router-dom';

import {
  advanceStorySession,
  createStorySession,
  deleteStorySession,
  listStorySessions,
  updateStorySession,
} from '../../api/storySessions';
import { commitStoryRun, getStoryRun, getStoryRunResult, listStoryRunEventHistory } from '../../api/storyRuns';
import { EmptyState } from '../../components/feedback/EmptyState';
import { ErrorState } from '../../components/feedback/ErrorState';
import { LoadingState } from '../../components/feedback/LoadingState';
import { MarkdownRenderer } from '../../components/MarkdownRenderer';
import type {
  StorySession,
  StoryCharacterMemoryUpdate,
  StoryMemoryPatch,
  StoryPlotVariable,
  StoryRelationshipUpdate,
  StoryReviewReport,
  StoryRunResult,
  StoryWorldStateUpdate,
} from '../../types/api';
import { formatRelativeTime } from '../../utils/format';
import { submitTextareaOnEnter } from '../../utils/keyboard';
import { useStoryRunEvents } from './useStoryRunEvents';

const copy = {
  sessionTitle: '写作会话',
  createSessionPlaceholder: '为这一段剧情起一个工作标题',
  createSessionButton: '创建写作会话',
  editSessionTitle: '编辑题目',
  saveSessionTitle: '保存题目',
  cancelSessionEdit: '取消编辑',
  deleteSessionTitle: '删除会话',
  deleteSessionConfirm: '删除这个写作会话？已提交章节不会删除。',
  workspaceTitle: '剧情工作台',
  workspaceDesc: '正文在中间阅读，运行状态和补丁审校放右侧，整个流程更接近真实写作节奏。',
  emptyTitle: '先创建一个写作会话',
  emptyDesc: '建立会话后，正文生成、审校和提交都会围绕这个会话展开。',
  untitledDraft: '未命名草稿',
  chapterPrefix: '第',
  chapterSuffix: '章',
  wordSuffix: '字',
  draftEmpty: '完整正文会在角色回合完成后统一生成，运行中先查看下方实时角色回合。',
  advancePlaceholder: '输入你想推进的情节方向、冲突目标、人物选择或文风要求',
  advanceButton: '推进剧情',
  reviewTitle: '运行与提交',
  authorNotePlaceholder: '记录这章提交的作者批注',
  commitButton: '提交为正式章节',
  missingIds: '请先等待完整的 draft 和 memory patch 结果。',
  committed: '这次运行已经提交，不需要重复 commit。',
  runHistory: '运行记录',
  orchestrationStarted: '主控已接收 idea 并启动',
  liveTurnsTitle: '实时角色回合',
  liveTurnsEmpty: '启动后，这里会实时显示哪个角色说了什么、做了什么。',
  turnPrefix: '回合',
  speechLabel: '说：',
  actionLabel: '做：',
  targetLabel: '指向：',
  loadingHistory: '正在加载运行记录',
  noHistory: '暂无持久化运行事件。',
  plotTitle: '情节压力线',
  pressureSource: '压力来源',
  coreChoice: '核心选择',
  optionA: '选项 A',
  optionB: '选项 B',
  costA: '代价 A',
  costB: '代价 B',
  irreversible: '不可逆影响',
  worldPressure: '世界压力',
  reviewReport: '审校报告',
  reviewPass: '通过',
  reviewPending: '待处理',
  hardViolations: '硬违规',
  continuityIssues: '连贯性问题',
  styleIssues: '风格问题',
  suggestedFixes: '修正建议',
  patchTitle: '记忆与状态补丁',
  memoryPatch: '角色记忆',
  relationshipPatch: '关系变化',
  worldPatch: '世界状态',
  noMemoryPatch: '没有角色记忆更新。',
  noRelationshipPatch: '没有关系变化。',
  noWorldPatch: '没有世界状态更新。',
  unknownCharacter: '未指定角色',
  importance: '重要度',
  relationshipUpdate: '关系更新',
  viewCount: '条视角',
  eventCount: '个事件',
  noExtraItems: '无附加事项',
  worldUpdate: '世界状态更新',
  statusIdle: '未开始',
  statusConnecting: '连接中',
  statusOpen: '已连接',
  statusError: '异常',
} as const;

export function StoryWorkspacePage() {
  const { projectId = '' } = useParams();
  const queryClient = useQueryClient();
  const [sessionTitle, setSessionTitle] = useState('');
  const [activeSessionId, setActiveSessionId] = useState('');
  const [authorMessage, setAuthorMessage] = useState('');
  const [activeRunId, setActiveRunId] = useState('');
  const [authorNote, setAuthorNote] = useState('');
  const [editingSessionId, setEditingSessionId] = useState('');
  const [editingSessionTitle, setEditingSessionTitle] = useState('');

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

  const updateSessionMutation = useMutation({
    mutationFn: ({ sessionId, title }: { sessionId: string; title: string }) =>
      updateStorySession(sessionId, { title }),
    onSuccess: (session) => {
      setEditingSessionId('');
      setEditingSessionTitle('');
      queryClient.invalidateQueries({ queryKey: ['storySessions', projectId] });
      queryClient.invalidateQueries({ queryKey: ['storySession', session.id] });
    },
  });

  const deleteSessionMutation = useMutation({
    mutationFn: deleteStorySession,
    onSuccess: (_result, sessionId) => {
      if (selectedSessionId === sessionId) {
        setActiveSessionId('');
        setActiveRunId('');
        setAuthorMessage('');
        setAuthorNote('');
      }
      if (editingSessionId === sessionId) {
        setEditingSessionId('');
        setEditingSessionTitle('');
      }
      queryClient.invalidateQueries({ queryKey: ['storySessions', projectId] });
      queryClient.invalidateQueries({ queryKey: ['project', projectId] });
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
  const isRunCommitted = runQuery.data?.status === 'committed' || Boolean(runQuery.data?.committed_at);
  const committedCharacterIds = useMemo(() => {
    const updates = resultQuery.data?.memory_patch?.character_memory_updates ?? [];
    return Array.from(new Set(updates.map((update) => update.character_id).filter(Boolean)));
  }, [resultQuery.data?.memory_patch?.character_memory_updates]);

  const commitMutation = useMutation({
    mutationFn: () =>
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
      setAuthorNote('');
    },
  });

  const visibleDraft = useMemo(() => {
    return resultQuery.data?.content ?? resultQuery.data?.draft?.content ?? '';
  }, [resultQuery.data]);

  const storyProgress = [
    {
      label: '实时连接',
      state:
        eventState.connectionStatus === 'open'
          ? 'done'
          : eventState.connectionStatus === 'connecting'
            ? 'active'
            : eventState.connectionStatus === 'error'
              ? 'error'
              : 'idle',
      detail: formatConnectionStatus(eventState.connectionStatus),
    },
    {
      label: '草稿生成',
      state: visibleDraft ? 'done' : activeRunId ? 'active' : 'idle',
      detail: runQuery.data?.current_step || '等待会话推进',
    },
    {
      label: '记忆补丁',
      state: memoryPatchId ? 'done' : activeRunId ? 'active' : 'idle',
      detail: memoryPatchId ? '已可审校' : '结果生成后出现',
    },
  ] as const;

  const startSession = (event: FormEvent) => {
    event.preventDefault();
    if (createSessionMutation.isPending) {
      return;
    }
    createSessionMutation.mutate();
  };

  const sendAdvance = (event: FormEvent) => {
    event.preventDefault();
    if (!authorMessage.trim() || !selectedSessionId || advanceMutation.isPending) {
      return;
    }
    advanceMutation.mutate();
  };

  const beginEditSession = (session: StorySession) => {
    setEditingSessionId(session.id);
    setEditingSessionTitle(session.title || '');
  };

  const cancelEditSession = () => {
    setEditingSessionId('');
    setEditingSessionTitle('');
  };

  const saveSessionTitle = () => {
    const title = editingSessionTitle.trim();
    if (!editingSessionId || !title || updateSessionMutation.isPending) {
      return;
    }
    updateSessionMutation.mutate({ sessionId: editingSessionId, title });
  };

  const removeSession = (session: StorySession) => {
    if (deleteSessionMutation.isPending) {
      return;
    }
    if (window.confirm(`${copy.deleteSessionConfirm}\n\n${session.title || session.id}`)) {
      deleteSessionMutation.mutate(session.id);
    }
  };

  return (
    <div className="workspace workspace--three">
      <aside className="workspace-panel">
        <h2>{copy.sessionTitle}</h2>
        <form className="stack-list" onSubmit={startSession}>
          <input
            value={sessionTitle}
            onChange={(event) => setSessionTitle(event.target.value)}
            placeholder={copy.createSessionPlaceholder}
          />
          <button className="button" disabled={createSessionMutation.isPending} type="submit">
            {copy.createSessionButton}
          </button>
        </form>
        {sessionsQuery.isLoading ? <LoadingState /> : null}
        {updateSessionMutation.isError ? <ErrorState message={(updateSessionMutation.error as Error).message} /> : null}
        {deleteSessionMutation.isError ? <ErrorState message={(deleteSessionMutation.error as Error).message} /> : null}
        <div className="session-list">
          {sessions.map((session) => {
            const isEditing = editingSessionId === session.id;
            return (
              <article
                className={
                  selectedSessionId === session.id
                    ? 'session-item session-item--story session-item--active'
                    : 'session-item session-item--story'
                }
                key={session.id}
              >
                {isEditing ? (
                  <form
                    className="session-item__edit"
                    onSubmit={(event) => {
                      event.preventDefault();
                      saveSessionTitle();
                    }}
                  >
                    <input
                      aria-label={copy.editSessionTitle}
                      autoFocus
                      onChange={(event) => setEditingSessionTitle(event.target.value)}
                      onKeyDown={(event) => {
                        if (event.key === 'Escape') {
                          event.preventDefault();
                          cancelEditSession();
                        }
                      }}
                      value={editingSessionTitle}
                    />
                    <div className="session-item__edit-actions">
                      <button
                        aria-label={copy.saveSessionTitle}
                        className="icon-button session-item__action"
                        disabled={!editingSessionTitle.trim() || updateSessionMutation.isPending}
                        title={copy.saveSessionTitle}
                        type="submit"
                      >
                        <Save size={15} />
                      </button>
                      <button
                        aria-label={copy.cancelSessionEdit}
                        className="icon-button session-item__action"
                        disabled={updateSessionMutation.isPending}
                        onClick={cancelEditSession}
                        title={copy.cancelSessionEdit}
                        type="button"
                      >
                        <X size={15} />
                      </button>
                    </div>
                  </form>
                ) : (
                  <>
                    <button
                      aria-label={`选择写作会话 ${session.title || session.id}`}
                      className="session-item__body"
                      onClick={() => setActiveSessionId(session.id)}
                      type="button"
                    >
                      <strong>{session.title || session.id}</strong>
                      <span>{session.status || 'ready'}</span>
                    </button>
                    <div className="session-item__actions">
                      <button
                        aria-label={copy.editSessionTitle}
                        className="icon-button session-item__action"
                        onClick={() => beginEditSession(session)}
                        title={copy.editSessionTitle}
                        type="button"
                      >
                        <Pencil size={15} />
                      </button>
                      <button
                        aria-label={copy.deleteSessionTitle}
                        className="icon-button session-item__action session-item__action--danger"
                        disabled={deleteSessionMutation.isPending}
                        onClick={() => removeSession(session)}
                        title={copy.deleteSessionTitle}
                        type="button"
                      >
                        <Trash2 size={15} />
                      </button>
                    </div>
                  </>
                )}
              </article>
            );
          })}
        </div>
      </aside>

      <section className="workspace-main workspace-main--story">
        <div className="page__header">
          <div>
            <h1>{copy.workspaceTitle}</h1>
            <p>{copy.workspaceDesc}</p>
          </div>
          {runQuery.data?.status ? <span className="status-pill">{runQuery.data.status}</span> : null}
        </div>

        {createSessionMutation.isError ? <ErrorState message={(createSessionMutation.error as Error).message} /> : null}
        {advanceMutation.isError ? <ErrorState message={(advanceMutation.error as Error).message} /> : null}
        {resultQuery.isError ? <ErrorState message={(resultQuery.error as Error).message} /> : null}

        {!selectedSessionId ? (
          <div className="story-empty-state">
            <EmptyState title={copy.emptyTitle} description={copy.emptyDesc} />
            <ol className="story-empty-steps">
              <li>创建写作会话，给这次推进一个标题。</li>
              <li>输入推进语，说明你要推动的情节、冲突或人物决断。</li>
              <li>审校草稿与记忆补丁，确认后再提交为正式章节。</li>
            </ol>
          </div>
        ) : (
          <>
            <div className="draft-surface">
              {resultQuery.data?.draft ? (
                <div className="draft-meta">
                  <strong>{resultQuery.data.draft.title ?? copy.untitledDraft}</strong>
                  <span>
                    {copy.chapterPrefix} {resultQuery.data.draft.chapter_number ?? '-'} {copy.chapterSuffix}
                  </span>
                  <span>
                    {resultQuery.data.draft.word_count ?? 0} {copy.wordSuffix}
                  </span>
                </div>
              ) : null}
              {visibleDraft ? <MarkdownRenderer source={visibleDraft} variant="reading" /> : <p className="muted">{copy.draftEmpty}</p>}
            </div>

            <LiveCharacterTurnsPanel starts={eventState.orchestrationStarts} turns={eventState.characterTurns} />

            <form className="composer" onSubmit={sendAdvance}>
              <textarea
                value={authorMessage}
                onChange={(event) => setAuthorMessage(event.target.value)}
                onKeyDown={submitTextareaOnEnter}
                placeholder={copy.advancePlaceholder}
                rows={4}
              />
              <button
                className="button"
                disabled={!authorMessage.trim() || !selectedSessionId || advanceMutation.isPending}
                type="submit"
              >
                <Send size={17} />
                {copy.advanceButton}
              </button>
            </form>
          </>
        )}
      </section>

      <aside className="workspace-panel">
        <h2>{copy.reviewTitle}</h2>
        <div className="story-status-card">
          <div className="story-status-card__header">
            <strong>{runQuery.data?.status || '等待开始'}</strong>
            <small>{activeRunId ? `最近一次运行 ${formatRelativeTime(runQuery.data?.updated_at ?? runQuery.data?.created_at)}` : '尚未运行'}</small>
          </div>
          <div className="story-progress-list">
            {storyProgress.map((item) => (
              <div className="story-progress-item" key={item.label}>
                <span className={`story-progress-item__dot story-progress-item__dot--${item.state}`} aria-hidden="true" />
                <div>
                  <strong>{item.label}</strong>
                  <small>{item.detail}</small>
                </div>
              </div>
            ))}
          </div>
        </div>

        {resultQuery.data ? <StoryResultPreview result={resultQuery.data} /> : null}

        <div className="commit-box">
          <textarea
            value={authorNote}
            onChange={(event) => setAuthorNote(event.target.value)}
            placeholder={copy.authorNotePlaceholder}
            rows={4}
          />
          <button
            className="button"
            disabled={!activeRunId || !draftId || !memoryPatchId || isRunCommitted || commitMutation.isPending}
            onClick={() => commitMutation.mutate()}
            type="button"
          >
            <Check size={17} />
            {copy.commitButton}
          </button>
          {!draftId || !memoryPatchId ? <small className="muted">{copy.missingIds}</small> : null}
          {isRunCommitted ? <small className="muted">{copy.committed}</small> : null}
          {commitMutation.isError ? <ErrorState message={(commitMutation.error as Error).message} /> : null}
        </div>

        <details className="event-disclosure">
          <summary>查看运行记录</summary>
          <div className="event-section">
            {eventHistoryQuery.isLoading ? <LoadingState label={copy.loadingHistory} /> : null}
            {!eventHistoryQuery.isLoading && (eventHistoryQuery.data ?? []).length === 0 ? <p className="muted">{copy.noHistory}</p> : null}
            {(eventHistoryQuery.data ?? []).map((event) => (
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
        </details>

        <details className="event-disclosure">
          <summary>查看生成细节</summary>
          <div className="structured-result">
            <InspectorSection emptyText="暂无步骤数据。" title="生成步骤">
              {eventState.generationSteps.map((item, index) => (
                <pre key={index}>{JSON.stringify(item, null, 2)}</pre>
              ))}
            </InspectorSection>
            <InspectorSection emptyText="暂无剧情变量事件。" title="剧情变量">
              {eventState.plotVariables.map((item, index) => (
                <pre key={index}>{JSON.stringify(item, null, 2)}</pre>
              ))}
            </InspectorSection>
            <InspectorSection emptyText="暂无角色轮次事件。" title="角色轮次">
              {eventState.characterTurns.map((item, index) => (
                <pre key={index}>{JSON.stringify(item, null, 2)}</pre>
              ))}
            </InspectorSection>
            <InspectorSection emptyText="暂无审校事件。" title="审校触发">
              {eventState.reviewItems.map((item, index) => (
                <pre key={index}>{JSON.stringify(item, null, 2)}</pre>
              ))}
            </InspectorSection>
          </div>
        </details>
      </aside>
    </div>
  );
}

function LiveCharacterTurnsPanel({
  starts,
  turns,
}: {
  starts: Array<{ author_message?: string; author_intent?: string; opening_situation?: string }>;
  turns: Array<{
    turn_index?: number;
    actor_name?: string;
    actor_id?: string;
    action_type?: string;
    speech?: string;
    action_summary?: string;
    target_actor_ids?: string[];
  }>;
}) {
  const latestStart = starts.length > 0 ? starts[starts.length - 1] : undefined;

  return (
    <section className="result-section story-live-turns">
      <div className="result-section__header">
        <h2>{copy.liveTurnsTitle}</h2>
        {latestStart ? <span className="status-pill">{copy.orchestrationStarted}</span> : null}
      </div>
      {latestStart ? <p className="muted">{latestStart.author_message || latestStart.author_intent || latestStart.opening_situation}</p> : null}
      {turns.length > 0 ? (
        <div className="setup-card-list">
          {turns.map((turn, index) => (
            <article className="setup-card" key={`${turn.turn_index ?? index}-${turn.actor_id ?? turn.actor_name ?? 'turn'}`}>
              <div className="setup-card__header">
                <strong>
                  {copy.turnPrefix} {turn.turn_index ?? index + 1} · {turn.actor_name || turn.actor_id || copy.unknownCharacter}
                </strong>
                <span>{turn.action_type || '-'}</span>
              </div>
              {turn.speech ? <p className="setup-card__copy">{copy.speechLabel}{turn.speech}</p> : null}
              {turn.action_summary ? <p className="setup-card__copy">{copy.actionLabel}{turn.action_summary}</p> : null}
              {turn.target_actor_ids && turn.target_actor_ids.length > 0 ? (
                <p className="setup-card__copy setup-card__copy--muted">{copy.targetLabel}{turn.target_actor_ids.join(' / ')}</p>
              ) : null}
            </article>
          ))}
        </div>
      ) : (
        <p className="muted">{copy.liveTurnsEmpty}</p>
      )}
    </section>
  );
}

function InspectorSection({
  title,
  emptyText,
  children,
}: {
  title: string;
  emptyText: string;
  children: ReactNode;
}) {
  const hasChildren = Array.isArray(children) ? children.length > 0 : Boolean(children);

  return (
    <section className="result-section">
      <div className="result-section__header">
        <h2>{title}</h2>
      </div>
      {hasChildren ? children : <p className="muted">{emptyText}</p>}
    </section>
  );
}

function formatConnectionStatus(status: string) {
  if (status === 'idle') {
    return copy.statusIdle;
  }
  if (status === 'connecting') {
    return copy.statusConnecting;
  }
  if (status === 'open') {
    return copy.statusOpen;
  }
  if (status === 'error') {
    return copy.statusError;
  }
  return status;
}

function isActiveRunStatus(status?: string) {
  return status === 'queued' || status === 'running' || status === 'loading_state';
}

function StoryResultPreview({ result }: { result: StoryRunResult }) {
  const plot = result.plot_variable;
  const review = result.review;
  const patch = result.memory_patch;

  return (
    <div className="structured-result">
      {plot ? <PlotVariablePreview plot={plot} /> : null}
      {review ? <ReviewPreview review={review} /> : null}
      {patch ? <MemoryPatchPreview patch={patch} /> : null}
    </div>
  );
}

function PlotVariablePreview({ plot }: { plot: StoryPlotVariable }) {
  return (
    <section className="result-section">
      <div className="result-section__header">
        <h2>{copy.plotTitle}</h2>
        {plot.focal_character_id ? <span className="status-pill">{plot.focal_character_id}</span> : null}
      </div>
      <div className="key-value-grid">
        <KeyValue label={copy.pressureSource} value={plot.pressure_source} />
        <KeyValue label={copy.coreChoice} value={plot.core_choice} />
        <KeyValue label={copy.optionA} value={plot.option_a} />
        <KeyValue label={copy.costA} value={plot.cost_a} />
        <KeyValue label={copy.optionB} value={plot.option_b} />
        <KeyValue label={copy.costB} value={plot.cost_b} />
        <KeyValue label={copy.irreversible} value={plot.irreversible_effect} />
        <KeyValue label={copy.worldPressure} value={plot.world_state_pressure?.join(', ')} />
      </div>
    </section>
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
    [copy.hardViolations, review.hard_violations],
    [copy.continuityIssues, review.continuity_issues],
    [copy.styleIssues, review.style_issues],
    [copy.suggestedFixes, review.suggested_fixes],
  ] as const;

  return (
    <section className="result-section">
      <div className="result-section__header">
        <h2>{copy.reviewReport}</h2>
        <span className={review.pass ? 'status-pill' : 'status-pill status-pill--warning'}>
          {review.pass ? copy.reviewPass : copy.reviewPending}
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
              <p className="muted">-</p>
            )}
          </div>
        ))}
      </div>
    </section>
  );
}

function MemoryPatchPreview({ patch }: { patch: StoryMemoryPatch }) {
  const memoryUpdates = patch.character_memory_updates ?? [];
  const relationshipUpdates = patch.relationship_updates ?? [];
  const worldUpdates = patch.world_state_updates ?? [];

  return (
    <section className="result-section">
      <div className="result-section__header">
        <h2>{copy.patchTitle}</h2>
        {patch.status ? <span className="status-pill">{patch.status}</span> : null}
      </div>

      <PatchGroup title={copy.memoryPatch}>
        {memoryUpdates.length > 0 ? memoryUpdates.map((item, index) => <CharacterMemoryPatchItem item={item} key={`memory-${index}`} />) : <p className="muted">{copy.noMemoryPatch}</p>}
      </PatchGroup>

      <PatchGroup title={copy.relationshipPatch}>
        {relationshipUpdates.length > 0 ? relationshipUpdates.map((item, index) => <RelationshipPatchItem item={item} key={`relationship-${index}`} />) : <p className="muted">{copy.noRelationshipPatch}</p>}
      </PatchGroup>

      <PatchGroup title={copy.worldPatch}>
        {worldUpdates.length > 0 ? worldUpdates.map((item, index) => <WorldStatePatchItem item={item} key={`world-${index}`} />) : <p className="muted">{copy.noWorldPatch}</p>}
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

function CharacterMemoryPatchItem({ item }: { item: StoryCharacterMemoryUpdate }) {
  return (
    <div className="patch-item">
      <div className="patch-item__body">
        <strong>{item.character_id || copy.unknownCharacter}</strong>
        <p>{item.content || '-'}</p>
        <small>{[item.type, item.importance ? `${copy.importance} ${item.importance}` : undefined].filter(Boolean).join(' / ')}</small>
      </div>
    </div>
  );
}

function RelationshipPatchItem({ item }: { item: StoryRelationshipUpdate }) {
  return (
    <div className="patch-item">
      <div className="patch-item__body">
        <strong>{item.pair_id || item.pair?.id || copy.relationshipUpdate}</strong>
        <p>{item.summary || item.tension_delta || '-'}</p>
        <small>
          {[
            item.views?.length ? `${item.views.length} ${copy.viewCount}` : undefined,
            item.events?.length ? `${item.events.length} ${copy.eventCount}` : undefined,
          ]
            .filter(Boolean)
            .join(' / ') || copy.noExtraItems}
        </small>
      </div>
    </div>
  );
}

function WorldStatePatchItem({ item }: { item: StoryWorldStateUpdate }) {
  return (
    <div className="patch-item">
      <div className="patch-item__body">
        <strong>{item.key || copy.worldUpdate}</strong>
        <p>{item.note || stringifyValue(item.value)}</p>
        <small>{item.operation || 'set'}</small>
      </div>
    </div>
  );
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
