import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Check, Send } from 'lucide-react';
import { FormEvent, useMemo, useState } from 'react';
import { useParams } from 'react-router-dom';

import { advanceStorySession, createStorySession, listStorySessions } from '../../api/storySessions';
import { commitStoryRun, getStoryRun, getStoryRunResult } from '../../api/storyRuns';
import { EmptyState } from '../../components/feedback/EmptyState';
import { ErrorState } from '../../components/feedback/ErrorState';
import { LoadingState } from '../../components/feedback/LoadingState';
import { useStoryRunEvents } from './useStoryRunEvents';

export function StoryWorkspacePage() {
  const { projectId = '' } = useParams();
  const queryClient = useQueryClient();
  const [sessionTitle, setSessionTitle] = useState('');
  const [activeSessionId, setActiveSessionId] = useState('');
  const [authorMessage, setAuthorMessage] = useState('');
  const [activeRunId, setActiveRunId] = useState('');
  const [authorNote, setAuthorNote] = useState('');

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
      return status === 'queued' || status === 'running' ? 1500 : false;
    },
  });

  const resultQuery = useQuery({
    queryKey: ['storyRunResult', activeRunId],
    queryFn: ({ signal }) => getStoryRunResult(activeRunId, signal),
    enabled: Boolean(activeRunId) && runQuery.data?.status !== 'queued' && runQuery.data?.status !== 'running',
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
      setActiveRunId(run.id);
      setAuthorMessage('');
    },
  });

  const draftId = resultQuery.data?.draft_id ?? resultQuery.data?.draft?.id ?? '';
  const memoryPatchId = resultQuery.data?.memory_patch_id ?? resultQuery.data?.memory_patch?.id ?? '';

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
    },
  });

  const visibleDraft = useMemo(() => {
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
        <h2>Story Sessions</h2>
        <form className="stack-list" onSubmit={startSession}>
          <input
            value={sessionTitle}
            onChange={(event) => setSessionTitle(event.target.value)}
            placeholder="Session title, optional"
          />
          <button className="button" disabled={createSessionMutation.isPending} type="submit">
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
              <strong>{session.title || session.id}</strong>
              <span>{session.status || 'ready'}</span>
            </button>
          ))}
        </div>
      </aside>

      <section className="workspace-main">
        <div className="page__header">
          <div>
            <h1>Story Workspace</h1>
            <p>Advance the story, inspect streamed events, then commit accepted output as canon.</p>
          </div>
          {runQuery.data?.status ? <span className="status-pill">{runQuery.data.status}</span> : null}
        </div>

        {createSessionMutation.isError ? <ErrorState message={(createSessionMutation.error as Error).message} /> : null}
        {advanceMutation.isError ? <ErrorState message={(advanceMutation.error as Error).message} /> : null}
        {resultQuery.isError ? <ErrorState message={(resultQuery.error as Error).message} /> : null}

        {!selectedSessionId ? (
          <EmptyState title="Create a story session first" description="Then enter an author message to generate a draft." />
        ) : (
          <>
            <div className="draft-surface">
              {visibleDraft ? <pre>{visibleDraft}</pre> : <p className="muted">Generated draft text will appear here.</p>}
            </div>
            <form className="composer" onSubmit={sendAdvance}>
              <textarea
                value={authorMessage}
                onChange={(event) => setAuthorMessage(event.target.value)}
                placeholder="For example: reveal a clue during a rainy confrontation"
                rows={4}
              />
              <button
                className="button"
                disabled={!authorMessage.trim() || !selectedSessionId || advanceMutation.isPending}
                type="submit"
              >
                <Send size={17} />
                Advance
              </button>
            </form>
          </>
        )}
      </section>

      <aside className="workspace-panel">
        <h2>Run Events</h2>
        <div className="status-line">SSE: {eventState.connectionStatus}</div>
        <div className="event-section">
          <h3>Plot Variables</h3>
          {eventState.plotVariables.map((item, index) => (
            <pre key={index}>{JSON.stringify(item, null, 2)}</pre>
          ))}
        </div>
        <div className="event-section">
          <h3>Character Turns</h3>
          {eventState.characterTurns.map((item, index) => (
            <pre key={index}>{JSON.stringify(item, null, 2)}</pre>
          ))}
        </div>
        <div className="event-section">
          <h3>Review Required</h3>
          {eventState.reviewItems.map((item, index) => (
            <pre key={index}>{JSON.stringify(item, null, 2)}</pre>
          ))}
        </div>
        <div className="commit-box">
          <textarea
            value={authorNote}
            onChange={(event) => setAuthorNote(event.target.value)}
            placeholder="Commit note"
            rows={4}
          />
          <button
            className="button"
            disabled={!activeRunId || !draftId || !memoryPatchId || commitMutation.isPending}
            onClick={() => commitMutation.mutate()}
            type="button"
          >
            <Check size={17} />
            Commit Canon
          </button>
          {!draftId || !memoryPatchId ? (
            <small className="muted">Waiting for result draft_id and memory_patch_id.</small>
          ) : null}
          {commitMutation.isError ? <ErrorState message={(commitMutation.error as Error).message} /> : null}
        </div>
      </aside>
    </div>
  );
}
