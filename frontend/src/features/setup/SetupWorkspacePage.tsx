// 设定建模工作台。负责创建 setup session、推进异步 run、预览结构化结果，并把接受的草稿应用为正式设定。
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Check, Send } from 'lucide-react';
import { FormEvent, useState } from 'react';
import { useParams } from 'react-router-dom';

import { getSetupRun, getSetupRunResult } from '../../api/setupRuns';
import { advanceSetupSession, applySetupRun, createSetupSession, listSetupSessions } from '../../api/setupSessions';
import { EmptyState } from '../../components/feedback/EmptyState';
import { ErrorState } from '../../components/feedback/ErrorState';
import { LoadingState } from '../../components/feedback/LoadingState';

const defaultAcceptSections = {
  authorBible: true,
  characters: true,
  relationships: true,
  worldState: true,
};

export function SetupWorkspacePage() {
  const { projectId = '' } = useParams();
  const queryClient = useQueryClient();
  const [seedIdea, setSeedIdea] = useState('');
  const [message, setMessage] = useState('');
  const [activeSessionId, setActiveSessionId] = useState('');
  const [activeRunId, setActiveRunId] = useState('');
  const [authorNote, setAuthorNote] = useState('');
  const [acceptSections, setAcceptSections] = useState(defaultAcceptSections);

  const sessionsQuery = useQuery({
    queryKey: ['setupSessions', projectId, 1, 20],
    queryFn: ({ signal }) => listSetupSessions(projectId, 1, 20, signal),
    enabled: Boolean(projectId),
  });

  const sessions = sessionsQuery.data?.data ?? [];
  const selectedSessionId = activeSessionId || sessions[0]?.id || '';

  const runQuery = useQuery({
    queryKey: ['setupRun', activeRunId],
    queryFn: ({ signal }) => getSetupRun(activeRunId, signal),
    enabled: Boolean(activeRunId),
    refetchInterval: (query) => {
      // 只在后端报告 run 仍处于活跃状态时轮询。
      const status = query.state.data?.status;
      return status === 'queued' || status === 'running' ? 1500 : false;
    },
  });

  const resultQuery = useQuery({
    queryKey: ['setupRunResult', activeRunId],
    queryFn: ({ signal }) => getSetupRunResult(activeRunId, signal),
    enabled: Boolean(activeRunId) && runQuery.data?.status !== 'queued' && runQuery.data?.status !== 'running',
  });

  const createSessionMutation = useMutation({
    mutationFn: () => createSetupSession(projectId, { seed_idea: seedIdea.trim() }),
    onSuccess: (session) => {
      setActiveSessionId(session.id);
      setSeedIdea('');
      queryClient.invalidateQueries({ queryKey: ['setupSessions', projectId] });
    },
  });

  const advanceMutation = useMutation({
    mutationFn: () => advanceSetupSession(selectedSessionId, { user_message: message.trim() }),
    onSuccess: (run) => {
      setActiveRunId(run.run_id ?? run.id ?? '');
      setMessage('');
    },
  });

  const applyMutation = useMutation({
    mutationFn: () =>
      // setup result 仍作为候选数据；只有勾选的区块会在 apply 时写入正式设定。
      applySetupRun(selectedSessionId, {
        run_id: activeRunId,
        accept_author_bible: acceptSections.authorBible,
        accept_characters: acceptSections.characters,
        accept_relationships: acceptSections.relationships,
        accept_world_state: acceptSections.worldState,
        author_note: authorNote.trim() || undefined,
      }),
    onSuccess: () => {
      // 应用 setup 结果可能更新所有正式设定资源。
      queryClient.invalidateQueries({ queryKey: ['project', projectId] });
      queryClient.invalidateQueries({ queryKey: ['authorBible', projectId] });
      queryClient.invalidateQueries({ queryKey: ['characters', projectId] });
      queryClient.invalidateQueries({ queryKey: ['relationships', projectId] });
      queryClient.invalidateQueries({ queryKey: ['setupSessions', projectId] });
    },
  });

  const startSession = (event: FormEvent) => {
    event.preventDefault();
    createSessionMutation.mutate();
  };

  const sendMessage = (event: FormEvent) => {
    event.preventDefault();
    advanceMutation.mutate();
  };

  return (
    <div className="workspace workspace--three">
      <aside className="workspace-panel">
        <h2>Setup Sessions</h2>
        <form className="stack-list" onSubmit={startSession}>
          <textarea
            value={seedIdea}
            onChange={(event) => setSeedIdea(event.target.value)}
            placeholder="Seed idea"
            rows={5}
          />
          <button className="button" disabled={!seedIdea.trim() || createSessionMutation.isPending} type="submit">
            Create Session
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
              <strong>{session.seed_idea || session.id}</strong>
              <span>{session.status || 'ready'}</span>
            </button>
          ))}
        </div>
      </aside>

      <section className="workspace-main">
        <div className="page__header">
          <div>
            <h1>Setup Workspace</h1>
            <p>Collect and review structured canon drafts before applying them to the project.</p>
          </div>
        </div>

        {createSessionMutation.isError ? <ErrorState message={(createSessionMutation.error as Error).message} /> : null}
        {advanceMutation.isError ? <ErrorState message={(advanceMutation.error as Error).message} /> : null}
        {resultQuery.isError ? <ErrorState message={(resultQuery.error as Error).message} /> : null}

        {!selectedSessionId ? (
          <EmptyState title="Create a setup session first" description="Start from one seed idea." />
        ) : (
          <>
            <form className="composer" onSubmit={sendMessage}>
              <textarea
                value={message}
                onChange={(event) => setMessage(event.target.value)}
                placeholder="Add details or answer assistant questions"
                rows={4}
              />
              <button
                className="button"
                disabled={!message.trim() || !selectedSessionId || advanceMutation.isPending}
                type="submit"
              >
                <Send size={17} />
                Send
              </button>
            </form>

            <div className="result-preview">
              <div className="result-preview__header">
                <h2>Structured Draft</h2>
                {runQuery.data?.status ? <span className="status-pill">{runQuery.data.status}</span> : null}
              </div>
              {runQuery.isLoading ? <LoadingState label="Loading run status" /> : null}
              {resultQuery.data ? (
                <pre>{JSON.stringify(resultQuery.data.setup_draft ?? resultQuery.data, null, 2)}</pre>
              ) : (
                <p className="muted">Setup run results will appear here.</p>
              )}
            </div>
          </>
        )}
      </section>

      <aside className="workspace-panel">
        <h2>Apply Canon</h2>
        <div className="stack-list">
          <label className="inline-row">
            <input
              checked={acceptSections.authorBible}
              onChange={(event) => setAcceptSections({ ...acceptSections, authorBible: event.target.checked })}
              type="checkbox"
            />
            作者圣经
          </label>
          <label className="inline-row">
            <input
              checked={acceptSections.characters}
              onChange={(event) => setAcceptSections({ ...acceptSections, characters: event.target.checked })}
              type="checkbox"
            />
            角色
          </label>
          <label className="inline-row">
            <input
              checked={acceptSections.relationships}
              onChange={(event) => setAcceptSections({ ...acceptSections, relationships: event.target.checked })}
              type="checkbox"
            />
            关系
          </label>
          <label className="inline-row">
            <input
              checked={acceptSections.worldState}
              onChange={(event) => setAcceptSections({ ...acceptSections, worldState: event.target.checked })}
              type="checkbox"
            />
            世界状态
          </label>
        </div>
        <textarea
          value={authorNote}
          onChange={(event) => setAuthorNote(event.target.value)}
          placeholder="Author note"
          rows={5}
        />
        <button
          className="button"
          disabled={!activeRunId || !selectedSessionId || !hasAcceptedSection(acceptSections) || applyMutation.isPending}
          onClick={() => applyMutation.mutate()}
          type="button"
        >
          <Check size={17} />
          Apply Selected
        </button>
        {applyMutation.isError ? <ErrorState message={(applyMutation.error as Error).message} /> : null}
      </aside>
    </div>
  );
}

function hasAcceptedSection(sections: typeof defaultAcceptSections) {
  return sections.authorBible || sections.characters || sections.relationships || sections.worldState;
}
