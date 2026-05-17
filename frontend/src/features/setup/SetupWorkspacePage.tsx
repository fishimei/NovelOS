import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { AlertTriangle, Check, Clock3, FileText, Globe2, Link2, Send, Sparkles, Users } from 'lucide-react';
import { FormEvent, useMemo, useState } from 'react';
import { useParams } from 'react-router-dom';

import { getSetupRun, getSetupRunResult, listSetupRunEventHistory } from '../../api/setupRuns';
import { advanceSetupSession, applySetupRun, createSetupSession, listSetupSessions } from '../../api/setupSessions';
import { EmptyState } from '../../components/feedback/EmptyState';
import { ErrorState } from '../../components/feedback/ErrorState';
import { LoadingState } from '../../components/feedback/LoadingState';
import type { Relationship, Run, RunEvent, SetupDraft, SetupSession, WorldStateEntry } from '../../types/api';

const copy = {
  sessionTitle: '\u8bbe\u5b9a\u4f1a\u8bdd',
  sessionSubtitle: '\u4ece\u6545\u4e8b\u79cd\u5b50\u5f00\u59cb\uff0c\u7ef4\u62a4\u4e00\u6761\u53ef\u56de\u6eaf\u7684\u8bbe\u5b9a\u8ba8\u8bba\u6d41\u3002',
  sessionHistoryTitle: '\u5386\u53f2\u4f1a\u8bdd',
  noSessions: '\u8fd8\u6ca1\u6709\u8bbe\u5b9a\u4f1a\u8bdd\u3002',
  seedPlaceholder: '\u8f93\u5165\u6545\u4e8b\u79cd\u5b50\uff0c\u4f8b\u5982\u4e3b\u9898\u3001\u4e16\u754c\u89c2\u3001\u4e3b\u89d2\u51b2\u7a81\u6216\u60c5\u7eea\u57fa\u8c03',
  createSessionButton: '\u521b\u5efa\u8bbe\u5b9a\u4f1a\u8bdd',
  createSessionLoading: '\u521b\u5efa\u4e2d...',
  workspaceTitle: '\u8bbe\u5b9a\u5de5\u4f5c\u53f0',
  workspaceDesc: '\u628a\u4f1a\u8bdd\u79cd\u5b50\u6574\u7406\u6210\u7ed3\u6784\u5316\u8349\u6848\uff0c\u518d\u6309\u6a21\u5757\u5ba1\u6838\u5e76\u5e94\u7528\u5230\u9879\u76ee\u8bbe\u5b9a\u3002',
  breadcrumb: '\u5199\u4f5c / \u8bbe\u5b9a\u5de5\u4f5c\u53f0',
  connectedHint: '\u5f53\u524d\u4f1a\u8bdd\u5df2\u5c31\u7eea',
  waitingHint: '\u7b49\u5f85\u521b\u5efa\u4f1a\u8bdd',
  emptyTitle: '\u5148\u521b\u5efa\u4e00\u4e2a\u8bbe\u5b9a\u4f1a\u8bdd',
  emptyDesc: '\u4ece\u79cd\u5b50\u60f3\u6cd5\u5f00\u59cb\uff0c\u9010\u6b65\u751f\u6210\u4f5c\u8005\u5723\u7ecf\u3001\u89d2\u8272\u3001\u5173\u7cfb\u4e0e\u4e16\u754c\u72b6\u6001\u8349\u6848\u3002',
  stepContext: '\u5f53\u524d\u4f1a\u8bdd\u4e0a\u4e0b\u6587',
  stepContextDesc: '\u5148\u786e\u8ba4\u8fd9\u6b21\u8bbe\u5b9a\u751f\u6210\u8981\u56f4\u7ed5\u54ea\u4e2a\u79cd\u5b50\u548c\u54ea\u4e9b\u8865\u5145\u3002',
  stepCompose: '\u8865\u5145\u8bbe\u5b9a\u8981\u6c42',
  stepComposeDesc: '\u6307\u51fa\u4f60\u60f3\u5f3a\u5316\u7684\u4e3b\u9898\u3001\u98ce\u683c\u3001\u4eba\u7269\u51b2\u7a81\u6216\u4e16\u754c\u89c4\u5219\u3002',
  stepDraft: '\u8bbe\u5b9a\u8349\u6848',
  stepDraftDesc: '\u5148\u770b\u6a21\u5757\u9aa8\u67b6\uff0c\u751f\u6210\u540e\u518d\u9010\u4e2a\u5ba1\u9605\u5e76\u51b3\u5b9a\u662f\u5426\u5e94\u7528\u3002',
  sessionSeedTitle: '\u79cd\u5b50\u6784\u60f3',
  lastSupplementTitle: '\u6700\u8fd1\u8865\u5145',
  noSupplement: '\u8fd8\u6ca1\u6709\u989d\u5916\u8865\u5145\u3002',
  emptySeed: '\u8be5\u4f1a\u8bdd\u8fd8\u6ca1\u6709\u5199\u5165\u79cd\u5b50\u5185\u5bb9\u3002',
  advancePlaceholder:
    '\u8865\u5145\u4f60\u7684\u8bbe\u5b9a\u8981\u6c42\uff0c\u4f8b\u5982\u4e3b\u9898\u3001\u6587\u98ce\u3001\u89d2\u8272\u52a8\u673a\u3001\u5173\u7cfb\u5f20\u529b\u6216\u4e16\u754c\u89c4\u5219',
  advanceButton: '\u751f\u6210\u8349\u6848',
  advanceRunning: '\u751f\u6210\u4e2d...',
  advanceHint: '\u57fa\u4e8e\u5f53\u524d\u4f1a\u8bdd\u7ee7\u7eed\u751f\u6210\uff0c\u901a\u5e38\u4f1a\u5728 20 - 40 \u79d2\u8fd4\u56de\u7ed3\u679c\u3002',
  resultLoading: '\u6b63\u5728\u751f\u6210\u8bbe\u5b9a\u8349\u6848',
  resultEmpty: '\u8fd8\u6ca1\u6709\u53ef\u5ba1\u9605\u7684\u8349\u6848\uff0c\u4f46\u5de5\u4f5c\u53f0\u5df2\u7ecf\u4e3a\u4f60\u9884\u7559\u4e86\u6a21\u5757\u9aa8\u67b6\u3002',
  resultReady: '\u5df2\u751f\u6210',
  resultOpenQuestions: '\u5f85\u786e\u8ba4\u95ee\u9898',
  resultOpenQuestionsEmpty: '\u6ca1\u6709\u989d\u5916\u5f85\u786e\u8ba4\u95ee\u9898\u3002',
  visualDraftTitle: '\u4e3b\u63a7 Agent \u8349\u6848\u770b\u677f',
  visualDraftSubtitle: '\u8fd9\u662f agent \u5185\u90e8\u6df1\u5316\u540e\u901a\u8fc7\u5de5\u5177\u5c55\u793a\u7684\u5b8c\u6574\u8be6\u7ec6\u8349\u6848\uff0c\u786e\u8ba4\u524d\u4e0d\u4f1a\u5199\u5165\u6b63\u5f0f\u8bbe\u5b9a\u3002',
  visualStyle: '\u98ce\u683c',
  visualTone: '\u6c14\u8d28',
  visualBoldness: '\u5927\u80c6\u7a0b\u5ea6',
  visualWorldPressure: '\u4e16\u754c\u538b\u529b',
  visualCharacterCards: '\u4eba\u7269\u5361\u7247',
  visualRelationshipGraph: '\u5173\u7cfb\u7f51\u7edc',
  visualNextAgents: '\u5efa\u8bae\u4e0b\u4e00\u6b65',
  visualNoBoard: '\u8fd9\u7248\u7ed3\u679c\u6ca1\u6709\u8fd4\u56de\u53ef\u89c6\u5316\u770b\u677f\uff0c\u4f46\u4e0b\u65b9\u7ed3\u6784\u5316\u8349\u6848\u4ecd\u53ef\u5ba1\u9605\u548c\u5e94\u7528\u3002',
  regenerateDraft: '\u91cd\u8d77\u8349\u4e00\u7248',
  cancelDraft: '\u53d6\u6d88\u8fd9\u7248\u8349\u6848',
  applyTitle: '\u5e94\u7528\u8349\u6848',
  applySubtitle: '\u9009\u62e9\u8981\u5199\u5165\u9879\u76ee\u7684\u6a21\u5757\uff0c\u5e76\u4e3a\u8fd9\u6b21\u63d0\u4ea4\u7559\u4e0b\u5907\u6ce8\u3002',
  applyScopeTitle: '\u5e94\u7528\u8303\u56f4',
  applyBible: '\u4f5c\u8005\u5723\u7ecf',
  applyCharacters: '\u89d2\u8272\u8bbe\u5b9a',
  applyRelationships: '\u5173\u7cfb\u8bbe\u5b9a',
  applyWorld: '\u4e16\u754c\u72b6\u6001',
  scopePending: '\u5f85\u751f\u6210',
  noteTitle: '\u5e94\u7528\u5907\u6ce8\uff08\u53ef\u9009\uff09',
  authorNotePlaceholder: '\u8bb0\u5f55\u8fd9\u6b21\u5e94\u7528\u7684\u51b3\u7b56\u3001\u53d6\u820d\u6216\u540e\u7eed\u5f85\u8c03\u6574\u70b9',
  applyButton: '\u5e94\u7528\u5230\u9879\u76ee',
  applyDisabled: '\u8bf7\u5148\u751f\u6210\u8349\u6848',
  applyPending: '\u5e94\u7528\u4e2d...',
  eventHistoryTitle: '\u8fd0\u884c\u8bb0\u5f55',
  eventHistoryHint: '\u4f1a\u8bdd\u521b\u5efa\u3001\u8349\u6848\u751f\u6210\u4e0e\u5e94\u7528\u8fc7\u7a0b\u90fd\u4f1a\u8bb0\u5f55\u5728\u8fd9\u91cc\u3002',
  eventHistoryLoading: '\u6b63\u5728\u52a0\u8f7d\u8fd0\u884c\u8bb0\u5f55',
  eventHistoryEmpty: '\u5f00\u59cb\u7b2c\u4e00\u6b21\u8bbe\u5b9a\u751f\u6210\u540e\uff0c\u8fd9\u91cc\u4f1a\u51fa\u73b0\u65f6\u95f4\u7ebf\u8bb0\u5f55\u3002',
  noRunHistory: '\u751f\u6210\u8349\u6848\u540e\u624d\u4f1a\u51fa\u73b0\u8fd0\u884c\u8bb0\u5f55\u3002',
  expandEvents: '\u5c55\u5f00\u5168\u90e8',
  collapseEvents: '\u6536\u8d77',
  summaryTitle: '\u8349\u6848\u6458\u8981',
  bibleTitle: '\u4f5c\u8005\u5723\u7ecf',
  theme: '\u4e3b\u9898',
  styleGuide: '\u6587\u98ce',
  worldRules: '\u4e16\u754c\u89c4\u5219',
  aesthetic: '\u5ba1\u7f8e\u539f\u5219',
  hardConstraints: '\u786c\u7ea6\u675f',
  softPreferences: '\u8f6f\u504f\u597d',
  forbiddenMoves: '\u7981\u7528\u5957\u8def',
  charactersTitle: '\u89d2\u8272\u8349\u6848',
  relationshipsTitle: '\u5173\u7cfb\u8349\u6848',
  worldStateTitle: '\u4e16\u754c\u72b6\u6001',
  charactersUnit: '\u4eba',
  relationshipsUnit: '\u6761',
  worldStateUnit: '\u9879',
  questionsUnit: '\u4e2a',
  defaultRole: '\u672a\u8bbe\u5b9a\u89d2\u8272\u5b9a\u4f4d',
  defaultProfile: '\u672a\u63d0\u4f9b\u89d2\u8272\u7b80\u4ecb\u3002',
  personality: '\u6027\u683c',
  voiceStyle: '\u53e3\u543b',
  goals: '\u76ee\u6807',
  fears: '\u6050\u60e7',
  secrets: '\u79d8\u5bc6',
  constraints: '\u7ea6\u675f',
  noCharacters: '\u672a\u751f\u6210\u89d2\u8272\u8349\u6848\u3002',
  noRelationships: '\u672a\u751f\u6210\u5173\u7cfb\u8349\u6848\u3002',
  relationshipDraft: '\u5173\u7cfb\u8349\u6848',
  volatility: '\u6ce2\u52a8',
  defaultRelationshipSummary: '\u672a\u63d0\u4f9b\u5173\u7cfb\u6458\u8981\u3002',
  anchors: '\u5173\u7cfb\u951a\u70b9',
  tensionPoints: '\u5f20\u529b\u70b9',
  noWorldState: '\u672a\u751f\u6210\u4e16\u754c\u72b6\u6001\u3002',
  defaultNote: '\u672a\u63d0\u4f9b\u8bf4\u660e',
  defaultQuestionReason: '\u672a\u8bf4\u660e\u8be5\u95ee\u9898\u7684\u5f71\u54cd\u3002',
  noQuestions: '\u6ca1\u6709\u989d\u5916\u5f85\u786e\u8ba4\u7684\u95ee\u9898\u3002',
  listEmpty: '-',
  rolePrefix: '\u89d2\u8272',
  statePrefix: '\u72b6\u6001',
  questionPrefix: '\u95ee\u9898',
  roleA: '\u89d2\u8272 A',
  roleB: '\u89d2\u8272 B',
  viewSuffix: '\u7684\u89c6\u89d2',
  surfaceView: '\u8868\u9762\uff1a',
  privateView: '\u79c1\u4e0b\uff1a',
  misunderstanding: '\u8bef\u5224\uff1a',
  masking: '\u4f2a\u88c5\uff1a',
  moduleApply: '\u52a0\u5165\u5e94\u7528',
  moduleApplied: '\u5df2\u9009\u4e2d',
  modulePrompt: '\u751f\u6210\u6b64\u6a21\u5757\u63d0\u793a',
  waitingGenerate: '\u5f85\u751f\u6210',
  failureTitle: '\u4e0a\u6b21\u751f\u6210\u5931\u8d25',
  failurePendingTitle: '\u8fd9\u6761\u4f1a\u8bdd\u5f85\u91cd\u8bd5',
  failurePendingMessage: '\u4e0a\u6b21\u751f\u6210\u6ca1\u6709\u5b8c\u6210\uff0c\u4f60\u53ef\u4ee5\u76f4\u63a5\u8865\u5145\u8981\u6c42\u540e\u518d\u91cd\u65b0\u751f\u6210\uff0c\u4e0d\u9700\u8981\u65b0\u5efa\u4f1a\u8bdd\u3002',
  failureAction: '\u751f\u6210\u4e00\u6761\u91cd\u8bd5\u63d0\u793a',
  failureFallback: '\u53ef\u4ee5\u7f29\u5c0f\u8303\u56f4\u3001\u8865\u5145\u7ea6\u675f\u6216\u91cd\u65b0\u63cf\u8ff0\u79cd\u5b50\u540e\u518d\u8bd5\u4e00\u6b21\u3002',
} as const;

const defaultAcceptSections = {
  authorBible: true,
  characters: true,
  relationships: true,
  worldState: true,
};

type AcceptSectionsState = typeof defaultAcceptSections;
type ModuleKey = keyof AcceptSectionsState;

const moduleDefinitions: Array<{
  key: ModuleKey;
  icon: typeof FileText;
  label: string;
  prompt: string;
}> = [
  {
    key: 'authorBible',
    icon: FileText,
    label: copy.applyBible,
    prompt: '\u8bf7\u91cd\u65b0\u751f\u6210\u4f5c\u8005\u5723\u7ecf\uff0c\u91cd\u70b9\u8865\u5f3a\u4e3b\u9898\u3001\u6587\u98ce\u4e0e\u4e16\u754c\u89c4\u5219\u3002',
  },
  {
    key: 'characters',
    icon: Users,
    label: copy.applyCharacters,
    prompt: '\u8bf7\u91cd\u65b0\u751f\u6210\u89d2\u8272\u8349\u6848\uff0c\u91cd\u70b9\u8865\u5f3a\u89d2\u8272\u52a8\u673a\u3001\u51b2\u7a81\u548c\u53e3\u543b\u533a\u5206\u3002',
  },
  {
    key: 'relationships',
    icon: Link2,
    label: copy.applyRelationships,
    prompt: '\u8bf7\u91cd\u65b0\u751f\u6210\u5173\u7cfb\u8349\u6848\uff0c\u91cd\u70b9\u8865\u5f3a\u53cc\u5411\u89c6\u89d2\u3001\u5f20\u529b\u70b9\u548c\u5173\u7cfb\u6f14\u5316\u7ebf\u7d22\u3002',
  },
  {
    key: 'worldState',
    icon: Globe2,
    label: copy.applyWorld,
    prompt: '\u8bf7\u91cd\u65b0\u751f\u6210\u4e16\u754c\u72b6\u6001\uff0c\u91cd\u70b9\u8865\u5145\u89c4\u5219\u8bbe\u5b9a\u3001\u5173\u952e\u53d8\u91cf\u548c\u7ea6\u675f\u6761\u4ef6\u3002',
  },
];

export function SetupWorkspacePage() {
  const { projectId = '' } = useParams();
  const queryClient = useQueryClient();
  const [seedIdea, setSeedIdea] = useState('');
  const [message, setMessage] = useState('');
  const [activeSessionId, setActiveSessionId] = useState('');
  const [activeRunId, setActiveRunId] = useState('');
  const [authorNote, setAuthorNote] = useState('');
  const [acceptSections, setAcceptSections] = useState(defaultAcceptSections);
  const [noteExpanded, setNoteExpanded] = useState(false);

  const sessionsQuery = useQuery({
    queryKey: ['setupSessions', projectId, 1, 20],
    queryFn: ({ signal }) => listSetupSessions(projectId, 1, 20, signal),
    enabled: Boolean(projectId),
  });

  const sessions = sessionsQuery.data?.data ?? [];
  const selectedSessionId = activeSessionId || sessions[0]?.id || '';
  const currentSession = sessions.find((session) => session.id === selectedSessionId) ?? sessions[0] ?? null;

  const runQuery = useQuery({
    queryKey: ['setupRun', activeRunId],
    queryFn: ({ signal }) => getSetupRun(activeRunId, signal),
    enabled: Boolean(activeRunId),
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      return isActiveSetupRunStatus(status) ? 1500 : false;
    },
  });

  const resultQuery = useQuery({
    queryKey: ['setupRunResult', activeRunId],
    queryFn: ({ signal }) => getSetupRunResult(activeRunId, signal),
    enabled: Boolean(activeRunId) && hasSetupRunResult(runQuery.data?.status),
  });

  const eventHistoryQuery = useQuery({
    queryKey: ['setupRunEventHistory', activeRunId],
    queryFn: ({ signal }) => listSetupRunEventHistory(activeRunId, signal),
    enabled: Boolean(activeRunId),
  });

  const createSessionMutation = useMutation({
    mutationFn: () => createSetupSession(projectId, { seed_idea: seedIdea.trim() }),
    onSuccess: (session) => {
      setActiveSessionId(session.id);
      setActiveRunId('');
      setSeedIdea('');
      setAuthorNote('');
      queryClient.invalidateQueries({ queryKey: ['setupSessions', projectId] });
    },
  });

  const advanceMutation = useMutation({
    mutationFn: (overrideMessage?: string) => advanceSetupSession(selectedSessionId, { user_message: (overrideMessage ?? message).trim() }),
    onSuccess: (run) => {
      setActiveRunId(run.run_id ?? run.id ?? '');
      setMessage('');
    },
  });

  const applyMutation = useMutation({
    mutationFn: () =>
      applySetupRun(selectedSessionId, {
        run_id: activeRunId,
        accept_author_bible: acceptSections.authorBible,
        accept_characters: acceptSections.characters,
        accept_relationships: acceptSections.relationships,
        accept_world_state: acceptSections.worldState,
        author_note: authorNote.trim() || undefined,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['project', projectId] });
      queryClient.invalidateQueries({ queryKey: ['authorBible', projectId] });
      queryClient.invalidateQueries({ queryKey: ['characters', projectId] });
      queryClient.invalidateQueries({ queryKey: ['relationships', projectId] });
      queryClient.invalidateQueries({ queryKey: ['setupSessions', projectId] });
      queryClient.invalidateQueries({ queryKey: ['setupRunEventHistory', activeRunId] });
    },
  });

  const draft = resultQuery.data?.setup_draft;
  const moduleCounts = useMemo(() => getModuleCounts(draft), [draft]);
  const hasDraft = hasDraftContent(draft);
  const isRunActive = isActiveSetupRunStatus(runQuery.data?.status);
  const canApply = Boolean(
    activeRunId &&
      selectedSessionId &&
      hasDraft &&
      hasApplicableSection(acceptSections, moduleCounts) &&
      !applyMutation.isPending &&
      !isRunActive,
  );
  const applyButtonLabel = applyMutation.isPending
    ? copy.applyPending
    : hasDraft
      ? copy.applyButton
      : copy.applyDisabled;
  const workspaceStatus = getWorkspaceDisplayStatus(runQuery.data?.status, currentSession?.status);
  const runStatusLabel = formatRunStatus(workspaceStatus);
  const runStatusTone = getStatusTone(workspaceStatus);
  const runErrorMessage = firstText(getRunErrorMessage(runQuery.data), getRunEventErrorMessage(eventHistoryQuery.data));
  const sessionNeedsRetry = !activeRunId && currentSession?.status === 'failed';
  const showFailureCard = Boolean(runErrorMessage || sessionNeedsRetry);
  const failureTitle = runErrorMessage ? copy.failureTitle : copy.failurePendingTitle;
  const failureMessage = runErrorMessage || copy.failurePendingMessage;

  const startSession = (event: FormEvent) => {
    event.preventDefault();
    createSessionMutation.mutate();
  };

  const sendMessage = (event: FormEvent) => {
    event.preventDefault();
    advanceMutation.mutate(message);
  };

  const toggleSection = (key: ModuleKey) => {
    setAcceptSections((current) => ({ ...current, [key]: !current[key] }));
  };

  const fillModulePrompt = (moduleKey: ModuleKey) => {
    const definition = moduleDefinitions.find((item) => item.key === moduleKey);
    if (!definition) {
      return;
    }
    setMessage(definition.prompt);
  };

  const fillRetryPrompt = () => {
    setMessage(
      '\u8bf7\u91cd\u65b0\u751f\u6210\u8fd9\u6b21\u8bbe\u5b9a\u8349\u6848\uff0c\u5e76\u5728\u4e0d\u6539\u53d8\u6838\u5fc3\u79cd\u5b50\u7684\u524d\u63d0\u4e0b\u7f29\u5c0f\u8303\u56f4\uff0c\u8865\u5145\u786c\u7ea6\u675f\u4e0e\u4e16\u754c\u89c4\u5219\u3002',
    );
  };

  const regenerateDraft = () => {
    if (!selectedSessionId || advanceMutation.isPending || isRunActive) {
      return;
    }
    advanceMutation.mutate('\u8bf7\u57fa\u4e8e\u5f53\u524d\u4f1a\u8bdd\u91cd\u65b0\u8d77\u8349\u4e00\u7248\uff0c\u4e0d\u8981\u76f4\u63a5\u6cbf\u7528\u4e0a\u4e00\u7248\uff1b\u65b9\u5411\u66f4\u5927\u80c6\u3001\u66f4\u5929\u9a6c\u884c\u7a7a\uff0c\u540c\u65f6\u4fdd\u7559\u53ef\u843d\u5730\u7684\u4eba\u7269\u52a8\u673a\u548c\u5173\u7cfb\u5f20\u529b\u3002');
  };

  const cancelDraft = () => {
    setActiveRunId('');
    setAuthorNote('');
    setAcceptSections(defaultAcceptSections);
  };

  return (
    <div className="setup-page">
      <div className="setup-statusbar">
        <div className="setup-statusbar__crumb">{copy.breadcrumb}</div>
        <div className="setup-statusbar__meta">
          <span className={`status-pill status-pill--${runStatusTone}`}>{runStatusLabel}</span>
          <span>{currentSession ? copy.connectedHint : copy.waitingHint}</span>
          {currentSession ? <span>{getSessionTitle(currentSession)}</span> : null}
        </div>
      </div>

      <div className="workspace workspace--three workspace--setup-layout">
        <aside className="workspace-panel workspace-panel--setup">
          <div className="setup-panel__intro">
            <h2>{copy.sessionTitle}</h2>
            <p>{copy.sessionSubtitle}</p>
          </div>

          <form className="stack-list setup-session-form" onSubmit={startSession}>
            <textarea
              className="setup-grow-textarea"
              value={seedIdea}
              onChange={(event) => setSeedIdea(event.target.value)}
              placeholder={copy.seedPlaceholder}
              rows={4}
            />
            <button className="button setup-full-button" disabled={!seedIdea.trim() || createSessionMutation.isPending} type="submit">
              {createSessionMutation.isPending ? copy.createSessionLoading : copy.createSessionButton}
            </button>
          </form>

          <div className="setup-panel__section">
            <div className="setup-panel__section-header">
              <h3>{copy.sessionHistoryTitle}</h3>
            </div>

            {sessionsQuery.isLoading ? <LoadingState /> : null}
            {!sessionsQuery.isLoading && sessions.length === 0 ? <p className="muted">{copy.noSessions}</p> : null}

            <div className="session-list session-list--setup">
              {sessions.map((session) => (
                <button
                  className={selectedSessionId === session.id ? 'session-item session-item--active' : 'session-item'}
                  key={session.id}
                  onClick={() => {
                    setActiveSessionId(session.id);
                    setActiveRunId('');
                  }}
                  type="button"
                >
                  <div className="session-item__headline">
                    <strong>{getSessionTitle(session)}</strong>
                    <span className={`session-state session-state--${getStatusTone(session.status)}`}>
                      {formatRunStatus(session.status)}
                    </span>
                  </div>
                  <p className="session-item__summary">{getSessionSummary(session)}</p>
                  <div className="session-item__meta">
                    <span>{formatRelativeTime(session.updated_at ?? session.created_at)}</span>
                    <span>{session.last_user_message ? '\u5df2\u8865\u5145' : '\u4ec5\u542b\u79cd\u5b50'}</span>
                  </div>
                </button>
              ))}
            </div>
          </div>

        </aside>

        <section className="workspace-main workspace-main--setup">
          <div className="page__header setup-page__header">
            <div>
              <h1>{copy.workspaceTitle}</h1>
              <p>{copy.workspaceDesc}</p>
            </div>
          </div>

          {createSessionMutation.isError ? <ErrorState message={(createSessionMutation.error as Error).message} /> : null}
          {advanceMutation.isError ? <ErrorState message={(advanceMutation.error as Error).message} /> : null}
          {resultQuery.isError ? <ErrorState message={(resultQuery.error as Error).message} /> : null}

          {!selectedSessionId ? (
            <EmptyState title={copy.emptyTitle} description={copy.emptyDesc} />
          ) : (
            <>
              <section className="setup-stage">
                <div className="setup-stage__header">
                  <div className="step-badge">1</div>
                  <div>
                    <h2>{copy.stepContext}</h2>
                    <p>{copy.stepContextDesc}</p>
                  </div>
                </div>

                <div className="setup-context-card">
                  <div className="setup-context-card__topline">
                    <strong>{getSessionTitle(currentSession)}</strong>
                    <span>{formatRelativeTime(currentSession?.updated_at ?? currentSession?.created_at)}</span>
                  </div>
                  <div className="setup-context-card__block">
                    <small>{copy.sessionSeedTitle}</small>
                    <p>{firstText(currentSession?.seed_idea, copy.emptySeed)}</p>
                  </div>
                  <div className="setup-context-card__block">
                    <small>{copy.lastSupplementTitle}</small>
                    <p>{firstText(currentSession?.last_user_message, copy.noSupplement)}</p>
                  </div>
                </div>
              </section>

              <form className="setup-stage composer composer--setup" onSubmit={sendMessage}>
                <div className="setup-stage__header">
                  <div className="step-badge">2</div>
                  <div>
                    <h2>{copy.stepCompose}</h2>
                    <p>{copy.stepComposeDesc}</p>
                  </div>
                </div>

                <textarea
                  className="setup-grow-textarea setup-grow-textarea--large"
                  value={message}
                  onChange={(event) => setMessage(event.target.value)}
                  placeholder={copy.advancePlaceholder}
                  rows={6}
                />

                <div className="setup-composer__footer">
                  <div className="setup-composer__hint">
                    <Clock3 size={15} />
                    <span>{copy.advanceHint}</span>
                  </div>
                  <button
                    className="button"
                    disabled={!message.trim() || !selectedSessionId || advanceMutation.isPending || isRunActive}
                    type="submit"
                  >
                    <Send size={17} />
                    {advanceMutation.isPending || isRunActive ? copy.advanceRunning : copy.advanceButton}
                  </button>
                </div>
              </form>

              {showFailureCard ? (
                <section className="setup-alert-card setup-alert-card--inline">
                  <div className="setup-alert-card__header">
                    <AlertTriangle size={16} />
                    <strong>{failureTitle}</strong>
                  </div>
                  <p>{failureMessage}</p>
                  <p className="muted">{copy.failureFallback}</p>
                  <button className="button button--secondary" onClick={fillRetryPrompt} type="button">
                    {copy.failureAction}
                  </button>
                </section>
              ) : null}

              <section className="setup-stage">
                <div className="setup-stage__header">
                  <div className="step-badge">3</div>
                  <div>
                    <h2>{copy.stepDraft}</h2>
                    <p>{copy.stepDraftDesc}</p>
                  </div>
                  {activeRunId || sessionNeedsRetry ? (
                    <span className={`status-pill status-pill--${getStatusTone(workspaceStatus)}`}>
                      {formatRunStatus(workspaceStatus)}
                    </span>
                  ) : null}
                </div>

                <SetupDraftPreview
                  acceptSections={acceptSections}
                  draft={draft}
                  isLoading={runQuery.isLoading || isRunActive}
                  onCancelDraft={cancelDraft}
                  onFillModulePrompt={fillModulePrompt}
                  onRegenerateDraft={regenerateDraft}
                  onToggleSection={toggleSection}
                />
              </section>
            </>
          )}
        </section>

        <aside className="workspace-panel workspace-panel--setup">
          <div className="setup-panel__intro">
            <h2>{copy.applyTitle}</h2>
            <p>{copy.applySubtitle}</p>
          </div>

          <section className="setup-scope">
            <div className="setup-panel__section-header">
              <h3>{copy.applyScopeTitle}</h3>
            </div>

            <div className="setup-scope__list">
              {moduleDefinitions.map((module) => {
                const count = moduleCounts[module.key];
                const Icon = module.icon;
                return (
                  <label className={count > 0 ? 'setup-scope__item' : 'setup-scope__item setup-scope__item--disabled'} key={module.key}>
                    <input
                      checked={acceptSections[module.key]}
                      disabled={count === 0}
                      onChange={() => toggleSection(module.key)}
                      type="checkbox"
                    />
                    <span className="setup-scope__label">
                      <Icon size={15} />
                      <span>{module.label}</span>
                    </span>
                    <span className="setup-count-badge">{formatModuleCount(module.key, count)}</span>
                  </label>
                );
              })}
            </div>
          </section>

          <section className="setup-note-box">
            <div className="setup-panel__section-header">
              <h3>{copy.noteTitle}</h3>
            </div>
            <textarea
              className="setup-grow-textarea"
              onBlur={() => setNoteExpanded(authorNote.trim().length > 0)}
              onChange={(event) => setAuthorNote(event.target.value)}
              onFocus={() => setNoteExpanded(true)}
              placeholder={copy.authorNotePlaceholder}
              rows={noteExpanded || authorNote.trim() ? 4 : 1}
              value={authorNote}
            />
          </section>

          <button className="button setup-full-button" disabled={!canApply} onClick={() => applyMutation.mutate()} type="button">
            <Check size={17} />
            {applyButtonLabel}
          </button>

          {applyMutation.isError ? <ErrorState message={(applyMutation.error as Error).message} /> : null}

          <SetupRunEventHistoryPanel
            error={eventHistoryQuery.error as Error | null}
            events={eventHistoryQuery.data ?? []}
            hasRun={Boolean(activeRunId)}
            isLoading={eventHistoryQuery.isLoading}
          />
        </aside>
      </div>
    </div>
  );
}

function hasApplicableSection(sections: AcceptSectionsState, counts: Record<ModuleKey, number>) {
  return (
    (sections.authorBible && counts.authorBible > 0) ||
    (sections.characters && counts.characters > 0) ||
    (sections.relationships && counts.relationships > 0) ||
    (sections.worldState && counts.worldState > 0)
  );
}

function hasDraftContent(draft?: SetupDraft) {
  if (!draft) {
    return false;
  }

  return Boolean(
    draft.assistant_summary ||
      countAuthorBibleEntries(draft) > 0 ||
      (draft.characters?.length ?? 0) > 0 ||
      (draft.relationships?.length ?? 0) > 0 ||
      (draft.world_state?.length ?? 0) > 0 ||
      (draft.open_questions?.length ?? 0) > 0,
  );
}

function getModuleCounts(draft?: SetupDraft): Record<ModuleKey, number> {
  return {
    authorBible: countAuthorBibleEntries(draft),
    characters: draft?.characters?.length ?? 0,
    relationships: draft?.relationships?.length ?? 0,
    worldState: draft?.world_state?.length ?? 0,
  };
}

function countAuthorBibleEntries(draft?: SetupDraft) {
  const bible = draft?.author_bible;
  if (!bible) {
    return 0;
  }

  const arrays = [
    bible.world_rules,
    bible.aesthetic_principles,
    bible.hard_constraints,
    bible.soft_preferences,
    bible.forbidden_moves,
  ];

  let count = 0;
  if (bible.theme?.trim()) {
    count += 1;
  }
  if (bible.style_guide?.trim()) {
    count += 1;
  }
  arrays.forEach((items) => {
    count += items?.filter((item) => item?.trim()).length ?? 0;
  });
  return count;
}

function SetupDraftPreview({
  draft,
  isLoading,
  acceptSections,
  onToggleSection,
  onFillModulePrompt,
  onRegenerateDraft,
  onCancelDraft,
}: {
  draft?: SetupDraft;
  isLoading: boolean;
  acceptSections: AcceptSectionsState;
  onToggleSection: (key: ModuleKey) => void;
  onFillModulePrompt: (key: ModuleKey) => void;
  onRegenerateDraft: () => void;
  onCancelDraft: () => void;
}) {
  const characterNames = useMemo(() => {
    const names = new Map<string, string>();
    (draft?.characters ?? []).forEach((character, index) => {
      const id = character.id?.trim();
      if (id) {
        names.set(id, firstText(character.name, `${copy.rolePrefix} ${index + 1}`));
      }
    });
    return names;
  }, [draft?.characters]);

  if (!draft) {
    return (
      <div className="result-preview result-preview--setup">
        {isLoading ? <LoadingState label={copy.resultLoading} /> : null}
        <p className="muted">{copy.resultEmpty}</p>
        <div className="setup-skeleton-grid">
          {moduleDefinitions.map((module) => {
            const Icon = module.icon;
            return (
              <div className="setup-skeleton-card" key={module.key}>
                <div className="setup-skeleton-card__header">
                  <span className="setup-skeleton-card__title">
                    <Icon size={15} />
                    {module.label}
                  </span>
                  <span className="setup-count-badge">{copy.scopePending}</span>
                </div>
                <div className="setup-skeleton-line" />
                <div className="setup-skeleton-line setup-skeleton-line--short" />
              </div>
            );
          })}
        </div>
      </div>
    );
  }

  return (
    <div className="structured-result structured-result--setup">
      <VisualDraftBoard draft={draft} onCancelDraft={onCancelDraft} onRegenerateDraft={onRegenerateDraft} />

      {draft.assistant_summary ? (
        <section className="result-section">
          <div className="result-section__header">
            <h2>{copy.summaryTitle}</h2>
            <span className="status-pill status-pill--success">{copy.resultReady}</span>
          </div>
          <p className="result-copy">{draft.assistant_summary}</p>
        </section>
      ) : null}

      <SetupModuleCard
        countLabel={formatModuleCount('authorBible', countAuthorBibleEntries(draft))}
        icon={FileText}
        isSelected={acceptSections.authorBible}
        onPrompt={() => onFillModulePrompt('authorBible')}
        onToggle={() => onToggleSection('authorBible')}
        title={copy.bibleTitle}
      >
        <div className="key-value-grid">
          <KeyValue label={copy.theme} value={draft.author_bible?.theme} />
          <KeyValue label={copy.styleGuide} value={draft.author_bible?.style_guide} />
        </div>
        <div className="setup-grid">
          <SetupList title={copy.worldRules} items={draft.author_bible?.world_rules} />
          <SetupList title={copy.aesthetic} items={draft.author_bible?.aesthetic_principles} />
          <SetupList title={copy.hardConstraints} items={draft.author_bible?.hard_constraints} />
          <SetupList title={copy.softPreferences} items={draft.author_bible?.soft_preferences} />
          <SetupList title={copy.forbiddenMoves} items={draft.author_bible?.forbidden_moves} />
        </div>
      </SetupModuleCard>

      <SetupModuleCard
        countLabel={formatModuleCount('characters', draft.characters?.length ?? 0)}
        icon={Users}
        isSelected={acceptSections.characters}
        onPrompt={() => onFillModulePrompt('characters')}
        onToggle={() => onToggleSection('characters')}
        title={copy.charactersTitle}
      >
        {draft.characters && draft.characters.length > 0 ? (
          <div className="setup-card-list">
            {draft.characters.map((character, index) => (
              <article className="setup-card" key={character.id ?? `character-${index}`}>
                <div className="setup-card__header">
                  <strong>{firstText(character.name, `${copy.rolePrefix} ${index + 1}`)}</strong>
                  <span>{firstText(character.role, copy.defaultRole)}</span>
                </div>
                <p className="setup-card__copy">{firstText(character.profile, copy.defaultProfile)}</p>
                <div className="key-value-grid">
                  <KeyValue label={copy.personality} value={character.personality} />
                  <KeyValue label={copy.voiceStyle} value={character.voice_style} />
                </div>
                <div className="setup-grid">
                  <SetupList title={copy.goals} items={character.goals} />
                  <SetupList title={copy.fears} items={character.fears} />
                  <SetupList title={copy.secrets} items={character.secrets} />
                  <SetupList title={copy.constraints} items={character.constraints} />
                </div>
              </article>
            ))}
          </div>
        ) : (
          <p className="muted">{copy.noCharacters}</p>
        )}
      </SetupModuleCard>

      <SetupModuleCard
        countLabel={formatModuleCount('relationships', draft.relationships?.length ?? 0)}
        icon={Link2}
        isSelected={acceptSections.relationships}
        onPrompt={() => onFillModulePrompt('relationships')}
        onToggle={() => onToggleSection('relationships')}
        title={copy.relationshipsTitle}
      >
        {draft.relationships && draft.relationships.length > 0 ? (
          <div className="setup-card-list">
            {draft.relationships.map((relationship, index) => (
              <RelationshipDraftCard
                characterNames={characterNames}
                key={relationship.pair?.id ?? `relationship-${index}`}
                relationship={relationship}
              />
            ))}
          </div>
        ) : (
          <p className="muted">{copy.noRelationships}</p>
        )}
      </SetupModuleCard>

      <SetupModuleCard
        countLabel={formatModuleCount('worldState', draft.world_state?.length ?? 0)}
        icon={Globe2}
        isSelected={acceptSections.worldState}
        onPrompt={() => onFillModulePrompt('worldState')}
        onToggle={() => onToggleSection('worldState')}
        title={copy.worldStateTitle}
      >
        {draft.world_state && draft.world_state.length > 0 ? (
          <div className="setup-card-list">
            {draft.world_state.map((entry, index) => (
              <article className="setup-card" key={`${entry.key ?? 'world'}-${index}`}>
                <div className="setup-card__header">
                  <strong>{firstText(entry.key, `${copy.statePrefix} ${index + 1}`)}</strong>
                  <span>{firstText(entry.note, copy.defaultNote)}</span>
                </div>
                <pre className="inline-json">{stringifyValue(entry.value)}</pre>
              </article>
            ))}
          </div>
        ) : (
          <p className="muted">{copy.noWorldState}</p>
        )}
      </SetupModuleCard>

      <section className="result-section result-section--questions">
        <div className="result-section__header">
          <h2>{copy.resultOpenQuestions}</h2>
          <span className="status-pill status-pill--warning">
            {draft.open_questions?.length ?? 0} {copy.questionsUnit}
          </span>
        </div>
        {draft.open_questions && draft.open_questions.length > 0 ? (
          <div className="setup-card-list">
            {draft.open_questions.map((question, index) => (
              <article className="setup-card" key={question.key ?? `question-${index}`}>
                <div className="setup-card__header">
                  <strong>{firstText(question.question, `${copy.questionPrefix} ${index + 1}`)}</strong>
                </div>
                <p className="setup-card__copy">{firstText(question.why_it_matters, copy.defaultQuestionReason)}</p>
              </article>
            ))}
          </div>
        ) : (
          <p className="muted">{copy.resultOpenQuestionsEmpty}</p>
        )}
      </section>
    </div>
  );
}

function VisualDraftBoard({
  draft,
  onRegenerateDraft,
  onCancelDraft,
}: {
  draft: SetupDraft;
  onRegenerateDraft: () => void;
  onCancelDraft: () => void;
}) {
  const visual = draft.visual_draft;
  if (!visual) {
    return (
      <section className="result-section setup-visual-board">
        <div className="result-section__header">
          <h2>{copy.visualDraftTitle}</h2>
          <span className="status-pill status-pill--neutral">{copy.scopePending}</span>
        </div>
        <p className="muted">{copy.visualNoBoard}</p>
        <DraftActionBar onCancelDraft={onCancelDraft} onRegenerateDraft={onRegenerateDraft} />
      </section>
    );
  }

  return (
    <section className="result-section setup-visual-board">
      <div className="result-section__header">
        <h2>
          <Sparkles size={17} />
          {copy.visualDraftTitle}
        </h2>
        <span className="status-pill status-pill--success">{copy.resultReady}</span>
      </div>
      <p className="muted">{copy.visualDraftSubtitle}</p>
      {visual.logline ? <p className="setup-visual-board__logline">{visual.logline}</p> : null}
      <div className="setup-visual-board__meta">
        <KeyValue label={copy.visualStyle} value={visual.style_tags?.join(' / ')} />
        <KeyValue label={copy.visualTone} value={visual.tone} />
        <KeyValue label={copy.visualBoldness} value={visual.boldness_level ? `${visual.boldness_level}/10` : undefined} />
      </div>

      {visual.world_pressure_cards && visual.world_pressure_cards.length > 0 ? (
        <VisualSection title={copy.visualWorldPressure} count={visual.world_pressure_cards.length}>
          <div className="setup-card-list">
            {visual.world_pressure_cards.map((card, index) => (
              <article className="setup-card" key={`${card.title ?? 'world'}-${index}`}>
                <div className="setup-card__header">
                  <strong>{firstText(card.title, `${copy.statePrefix} ${index + 1}`)}</strong>
                  <span>{card.related_world_state_keys?.join(' / ')}</span>
                </div>
                <p className="setup-card__copy">{firstText(card.detail, card.stakes, copy.defaultNote)}</p>
                {card.stakes ? <p className="setup-card__copy setup-card__copy--muted">{card.stakes}</p> : null}
              </article>
            ))}
          </div>
        </VisualSection>
      ) : null}

      {visual.character_cards && visual.character_cards.length > 0 ? (
        <VisualSection title={copy.visualCharacterCards} count={visual.character_cards.length}>
          <div className="setup-card-list">
            {visual.character_cards.map((card, index) => (
              <article className="setup-card" key={card.character_key ?? `${card.name ?? 'character'}-${index}`}>
                <div className="setup-card__header">
                  <strong>{firstText(card.name, `${copy.rolePrefix} ${index + 1}`)}</strong>
                  <span>{firstText(card.role, card.character_key, copy.defaultRole)}</span>
                </div>
                <p className="setup-card__copy">{firstText(card.hook, copy.defaultProfile)}</p>
                <div className="key-value-grid">
                  <KeyValue label={copy.goals} value={card.goal} />
                  <KeyValue label={copy.fears} value={card.fear} />
                  <KeyValue label={copy.secrets} value={card.secret} />
                </div>
              </article>
            ))}
          </div>
        </VisualSection>
      ) : null}

      {visual.relationship_edges && visual.relationship_edges.length > 0 ? (
        <VisualSection title={copy.visualRelationshipGraph} count={visual.relationship_edges.length}>
          <div className="setup-card-list">
            {visual.relationship_edges.map((edge, index) => (
              <article className="setup-card" key={`${edge.from_character_key ?? 'from'}-${edge.to_character_key ?? 'to'}-${index}`}>
                <div className="setup-card__header">
                  <strong>
                    {firstText(edge.from_character_key, copy.roleA)} / {firstText(edge.to_character_key, copy.roleB)}
                  </strong>
                  <span>{firstText(edge.tension, copy.relationshipDraft)}</span>
                </div>
                <p className="setup-card__copy">{firstText(edge.summary, copy.defaultRelationshipSummary)}</p>
                {edge.misreading ? <p className="setup-card__copy setup-card__copy--muted">{copy.misunderstanding}{edge.misreading}</p> : null}
              </article>
            ))}
          </div>
        </VisualSection>
      ) : null}

      {visual.open_questions && visual.open_questions.length > 0 ? (
        <VisualSection title={copy.resultOpenQuestions} count={visual.open_questions.length}>
          <div className="setup-card-list">
            {visual.open_questions.map((question, index) => (
              <article className="setup-card" key={question.key ?? `visual-question-${index}`}>
                <div className="setup-card__header">
                  <strong>{firstText(question.question, `${copy.questionPrefix} ${index + 1}`)}</strong>
                </div>
                <p className="setup-card__copy">{firstText(question.why_it_matters, copy.defaultQuestionReason)}</p>
              </article>
            ))}
          </div>
        </VisualSection>
      ) : null}

      {visual.next_agent_suggestions && visual.next_agent_suggestions.length > 0 ? (
        <VisualSection title={copy.visualNextAgents} count={visual.next_agent_suggestions.length}>
          <div className="setup-card-list">
            {visual.next_agent_suggestions.map((suggestion, index) => (
              <article className="setup-card" key={suggestion.key ?? `${suggestion.label ?? 'agent'}-${index}`}>
                <div className="setup-card__header">
                  <strong>{firstText(suggestion.label, suggestion.key, `${copy.questionPrefix} ${index + 1}`)}</strong>
                </div>
                <p className="setup-card__copy">{firstText(suggestion.reason, copy.defaultQuestionReason)}</p>
              </article>
            ))}
          </div>
        </VisualSection>
      ) : null}

      {visual.agent_summary ? <p className="result-copy">{visual.agent_summary}</p> : null}
      <DraftActionBar onCancelDraft={onCancelDraft} onRegenerateDraft={onRegenerateDraft} />
    </section>
  );
}

function VisualSection({ title, count, children }: { title: string; count: number; children: React.ReactNode }) {
  return (
    <div className="setup-visual-section">
      <div className="setup-visual-section__header">
        <h3>{title}</h3>
        <span className="setup-count-badge">{count}</span>
      </div>
      {children}
    </div>
  );
}

function DraftActionBar({ onRegenerateDraft, onCancelDraft }: { onRegenerateDraft: () => void; onCancelDraft: () => void }) {
  return (
    <div className="setup-visual-board__actions">
      <button className="button button--secondary" onClick={onRegenerateDraft} type="button">
        {copy.regenerateDraft}
      </button>
      <button className="button button--ghost" onClick={onCancelDraft} type="button">
        {copy.cancelDraft}
      </button>
    </div>
  );
}

function SetupModuleCard({
  title,
  icon: Icon,
  countLabel,
  isSelected,
  onToggle,
  onPrompt,
  children,
}: {
  title: string;
  icon: typeof FileText;
  countLabel: string;
  isSelected: boolean;
  onToggle: () => void;
  onPrompt: () => void;
  children: React.ReactNode;
}) {
  return (
    <details className="result-section setup-module" open>
      <summary className="setup-module__summary">
        <span className="setup-module__title">
          <Icon size={16} />
          {title}
        </span>
        <span className="status-pill status-pill--success">{countLabel}</span>
      </summary>
      <div className="setup-module__body">
        <div className="setup-module__actions">
          <button className={isSelected ? 'button button--secondary' : 'button button--ghost'} onClick={onToggle} type="button">
            {isSelected ? copy.moduleApplied : copy.moduleApply}
          </button>
          <button className="button button--ghost" onClick={onPrompt} type="button">
            {copy.modulePrompt}
          </button>
        </div>
        {children}
      </div>
    </details>
  );
}

function RelationshipDraftCard({
  relationship,
  characterNames,
}: {
  relationship: Partial<Relationship>;
  characterNames: Map<string, string>;
}) {
  const pair = relationship.pair;
  const leftName = firstText(
    pair?.left_character_id ? characterNames.get(pair.left_character_id) : undefined,
    pair?.left_character_id,
    copy.roleA,
  );
  const rightName = firstText(
    pair?.right_character_id ? characterNames.get(pair.right_character_id) : undefined,
    pair?.right_character_id,
    copy.roleB,
  );

  return (
    <article className="setup-card">
      <div className="setup-card__header">
        <strong>
          {leftName} / {rightName}
        </strong>
        <span>{pair?.volatility != null ? `${copy.volatility} ${pair.volatility}` : copy.relationshipDraft}</span>
      </div>
      <p className="setup-card__copy">{firstText(pair?.summary, copy.defaultRelationshipSummary)}</p>
      <div className="setup-grid">
        <SetupList title={`${leftName}${copy.viewSuffix}`} items={relationshipViewLines(relationship, pair?.left_character_id)} />
        <SetupList title={`${rightName}${copy.viewSuffix}`} items={relationshipViewLines(relationship, pair?.right_character_id)} />
        <SetupList title={copy.anchors} items={pair?.anchors} />
        <SetupList title={copy.tensionPoints} items={pair?.tension_points} />
      </div>
    </article>
  );
}

function SetupList({ title, items }: { title: string; items?: string[] }) {
  return (
    <div className="review-list">
      <h3>{title}</h3>
      {items && items.length > 0 ? (
        <ul>
          {items.map((item, index) => (
            <li key={`${title}-${index}`}>{item}</li>
          ))}
        </ul>
      ) : (
        <p className="muted">{copy.listEmpty}</p>
      )}
    </div>
  );
}

function KeyValue({ label, value }: { label: string; value?: string }) {
  return (
    <div className="kv-item">
      <span>{label}</span>
      <strong>{firstText(value, '-')}</strong>
    </div>
  );
}

function relationshipViewLines(relationship: Partial<Relationship>, sourceCharacterID?: string) {
  const view = relationship.views?.find((item) => item.source_character_id === sourceCharacterID);
  if (!view) {
    return [];
  }

  return [
    view.public_attitude ? `${copy.surfaceView}${view.public_attitude}` : undefined,
    view.private_attitude ? `${copy.privateView}${view.private_attitude}` : undefined,
    view.believed_target_attitude ? `${copy.misunderstanding}${view.believed_target_attitude}` : undefined,
    view.masking_strategy ? `${copy.masking}${view.masking_strategy}` : undefined,
  ].filter(Boolean) as string[];
}

function firstText(...values: Array<string | undefined>) {
  for (const value of values) {
    if (value && value.trim()) {
      return value;
    }
  }
  return '';
}

function stringifyValue(value: WorldStateEntry['value']) {
  if (typeof value === 'string') {
    return value;
  }
  if (value == null) {
    return '-';
  }
  return JSON.stringify(value, null, 2);
}

function SetupRunEventHistoryPanel({
  events,
  isLoading,
  error,
  hasRun,
}: {
  events: RunEvent[];
  isLoading: boolean;
  error: Error | null;
  hasRun: boolean;
}) {
  const [expanded, setExpanded] = useState(false);
  const visibleEvents = expanded ? events : events.slice(0, 6);

  return (
    <div className="event-section event-section--timeline">
      <div className="setup-panel__section-header">
        <h3>{copy.eventHistoryTitle}</h3>
      </div>
      <p className="muted">{copy.eventHistoryHint}</p>

      {!hasRun ? <p className="muted">{copy.noRunHistory}</p> : null}
      {hasRun && isLoading ? <LoadingState label={copy.eventHistoryLoading} /> : null}
      {hasRun && error ? <ErrorState message={error.message} /> : null}
      {hasRun && !isLoading && !error && events.length === 0 ? <p className="muted">{copy.eventHistoryEmpty}</p> : null}

      {hasRun && !isLoading && !error && events.length > 0 ? (
        <>
          <div className="event-timeline">
            {visibleEvents.map((event) => (
              <div className="event-timeline__item" key={event.id}>
                <span className={`event-timeline__dot event-timeline__dot--${getEventTone(event)}`} />
                <div className="event-timeline__body">
                  <div className="event-timeline__row">
                    <strong>{getEventTitle(event)}</strong>
                    <span>{formatDateTime(event.created_at)}</span>
                  </div>
                  <p>{getEventDescription(event)}</p>
                </div>
              </div>
            ))}
          </div>
          {events.length > 6 ? (
            <button className="button button--ghost setup-full-button" onClick={() => setExpanded((value) => !value)} type="button">
              {expanded ? copy.collapseEvents : `${copy.expandEvents} ${events.length} \u6761`}
            </button>
          ) : null}
        </>
      ) : null}
    </div>
  );
}

function getSessionTitle(session: SetupSession | null) {
  const seed = session?.seed_idea?.trim();
  if (!seed) {
    return '\u672a\u547d\u540d\u8bbe\u5b9a\u4f1a\u8bdd';
  }

  const firstLine = seed.split(/\r?\n/)[0]?.trim() ?? seed;
  return truncateText(firstLine, 18);
}

function getSessionSummary(session: SetupSession) {
  const summary = session.last_user_message?.trim() || session.seed_idea?.trim();
  return truncateText(summary || '\u6682\u65e0\u6458\u8981', 58);
}

function getWorkspaceDisplayStatus(runStatus?: string, sessionStatus?: string) {
  if (runStatus) {
    return runStatus;
  }
  return getSessionDisplayStatus(sessionStatus);
}

function getSessionDisplayStatus(status?: string) {
  if (status === 'failed') {
    return 'retryable';
  }
  return status;
}

function truncateText(value: string, maxLength: number) {
  if (value.length <= maxLength) {
    return value;
  }
  return `${value.slice(0, maxLength).trimEnd()}...`;
}

function formatRunStatus(status?: string) {
  switch (status) {
    case 'queued':
      return '\u6392\u961f\u4e2d';
    case 'running':
    case 'loading_state':
      return '\u751f\u6210\u4e2d';
    case 'review_required':
      return '\u5f85\u5ba1\u9605';
    case 'succeeded':
      return '\u5df2\u751f\u6210';
    case 'retryable':
      return '\u5f85\u91cd\u8bd5';
    case 'failed':
      return '\u5931\u8d25';
    case 'applied':
      return '\u5df2\u5e94\u7528';
    case 'cancelled':
      return '\u5df2\u53d6\u6d88';
    default:
      return '\u5c31\u7eea';
  }
}

function getStatusTone(status?: string) {
  switch (status) {
    case 'review_required':
    case 'succeeded':
    case 'applied':
      return 'success';
    case 'retryable':
      return 'warning';
    case 'failed':
      return 'danger';
    case 'queued':
    case 'running':
    case 'loading_state':
      return 'warning';
    default:
      return 'neutral';
  }
}

function isActiveSetupRunStatus(status?: string) {
  return status === 'queued' || status === 'running' || status === 'loading_state';
}

function hasSetupRunResult(status?: string) {
  return status === 'review_required' || status === 'succeeded' || status === 'applied';
}

function formatModuleCount(key: ModuleKey, count: number) {
  if (count === 0) {
    return copy.scopePending;
  }

  switch (key) {
    case 'characters':
      return `${count} ${copy.charactersUnit}`;
    case 'relationships':
      return `${count} ${copy.relationshipsUnit}`;
    case 'worldState':
      return `${count} ${copy.worldStateUnit}`;
    default:
      return `${count} \u9879`;
  }
}

function getRunErrorMessage(run?: Run) {
  const value = run?.error;
  if (!value) {
    return '';
  }
  if (typeof value === 'string') {
    return value;
  }
  return firstText(value.message, value.code, copy.failureFallback);
}

function getRunEventErrorMessage(events?: RunEvent[]) {
  if (!events?.length) {
    return '';
  }

  for (let index = events.length - 1; index >= 0; index -= 1) {
    const payload = events[index].payload ?? {};
    const message = pickText(payload.error, payload.message, payload.detail, payload.reason);
    if (message) {
      return message;
    }
  }

  return '';
}

function getEventTone(event: RunEvent) {
  const name = event.event_name.toLowerCase();
  const payloadText = eventPayloadText(event).toLowerCase();
  if (name.includes('fail') || name.includes('error') || payloadText.includes('failed') || payloadText.includes('error')) {
    return 'danger';
  }
  if (name.includes('queue') || name.includes('run') || name.includes('start') || name.includes('progress')) {
    return 'warning';
  }
  return 'success';
}

function getEventTitle(event: RunEvent) {
  const name = event.event_name.toLowerCase();
  const payloadText = eventPayloadText(event).toLowerCase();
  if (name.includes('session') && name.includes('create')) {
    return '\u521b\u5efa\u8bbe\u5b9a\u4f1a\u8bdd';
  }
  if (name.includes('apply') && (name.includes('success') || name.includes('done') || name.includes('finish'))) {
    return '\u5e94\u7528\u8349\u6848';
  }
  if (name.includes('apply')) {
    return '\u51c6\u5907\u5e94\u7528\u8349\u6848';
  }
  if (name.includes('fail') || name.includes('error') || payloadText.includes('failed') || payloadText.includes('error')) {
    return '\u8349\u6848\u751f\u6210\u5931\u8d25';
  }
  if (name.includes('success') || name.includes('finish') || name.includes('complete')) {
    return '\u8349\u6848\u751f\u6210\u5b8c\u6210';
  }
  if (name.includes('queue')) {
    return '\u8349\u6848\u8fdb\u5165\u961f\u5217';
  }
  if (name.includes('start') || name.includes('run')) {
    return '\u5f00\u59cb\u751f\u6210\u8349\u6848';
  }
  return normalizeEventName(event.event_name);
}

function getEventDescription(event: RunEvent) {
  const payload = event.payload ?? {};
  const message = pickText(payload.error, payload.message, payload.detail, payload.reason, payload.step, payload.status);
  if (message) {
    return message;
  }
  if (event.sequence != null) {
    return `#${event.sequence} ${normalizeEventName(event.event_name)}`;
  }
  return normalizeEventName(event.event_name);
}

function eventPayloadText(event: RunEvent) {
  const payload = event.payload ?? {};
  return pickText(payload.error, payload.message, payload.detail, payload.reason, payload.step, payload.status);
}

function normalizeEventName(value: string) {
  return value
    .split(/[._-]/)
    .filter(Boolean)
    .join(' ')
    .trim();
}

function pickText(...values: unknown[]) {
  for (const value of values) {
    if (typeof value === 'string' && value.trim()) {
      return value;
    }
  }
  return '';
}

function formatRelativeTime(value?: string) {
  if (!value) {
    return '\u521a\u521a\u66f4\u65b0';
  }

  const timestamp = new Date(value).getTime();
  if (Number.isNaN(timestamp)) {
    return value;
  }

  const diff = Date.now() - timestamp;
  const minute = 60 * 1000;
  const hour = 60 * minute;
  const day = 24 * hour;

  if (diff < minute) {
    return '\u521a\u521a\u66f4\u65b0';
  }

  if (diff < hour) {
    return `${Math.floor(diff / minute)} \u5206\u949f\u524d`;
  }

  if (diff < day) {
    return `${Math.floor(diff / hour)} \u5c0f\u65f6\u524d`;
  }

  if (diff < day * 7) {
    return `${Math.floor(diff / day)} \u5929\u524d`;
  }

  return new Intl.DateTimeFormat('zh-CN', {
    month: 'short',
    day: 'numeric',
  }).format(new Date(timestamp));
}

function formatDateTime(value?: string) {
  if (!value) {
    return '\u6682\u65e0\u8bb0\u5f55';
  }

  const timestamp = new Date(value);
  if (Number.isNaN(timestamp.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat('zh-CN', {
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(timestamp);
}
