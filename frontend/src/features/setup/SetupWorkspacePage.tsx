import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  AlertTriangle,
  Bot,
  Check,
  CheckCircle2,
  FileText,
  Globe2,
  Link2,
  MessageCircle,
  Send,
  Sparkles,
  Target,
  Trash2,
  Users,
  XCircle,
} from 'lucide-react';
import { type FormEvent, useEffect, useMemo, useState } from 'react';
import { useParams } from 'react-router-dom';

import {
  advanceDialogueSession,
  confirmDialogueActionOption,
  createDialogueSession,
  getDialogueRun,
  getDialogueRunResult,
  getDialogueSession,
  listDialogueSessions,
  rejectDialogueActionOption,
} from '../../api/dialogueSessions';
import { getSetupRun, getSetupRunResult, listSetupRunEventHistory } from '../../api/setupRuns';
import {
  advanceSetupSession,
  applySetupRun,
  createSetupSession,
  deleteSetupSession,
  listSetupSessions,
  updateSetupSession,
} from '../../api/setupSessions';
import { EmptyState } from '../../components/feedback/EmptyState';
import { ErrorState } from '../../components/feedback/ErrorState';
import { LoadingState } from '../../components/feedback/LoadingState';
import { MarkdownRenderer } from '../../components/MarkdownRenderer';
import type {
  DialogueActionOption,
  DialogueMessage,
  DialogueSession,
  PaginatedResponse,
  Relationship,
  Run,
  RunEvent,
  SetupDraft,
  SetupSession,
  WorldStateEntry,
} from '../../types/api';
import { submitTextareaOnEnter } from '../../utils/keyboard';

const copy = {
  sessionTitle: '设定会话',
  sessionSubtitle: '从故事种子开始，维护一条可回溯的设定讨论流。',
  sessionHistoryTitle: '历史会话',
  deleteSessionTitle: '删除会话',
  deleteSessionConfirm: '删除这个设定会话？已应用到项目的正式设定不会删除。',
  deleteSessionPending: '删除中...',
  noSessions: '还没有设定会话。',
  seedPlaceholder: '输入故事种子，例如主题、世界观、主角冲突或情绪基调',
  createSessionButton: '创建设定会话',
  createSessionLoading: '创建中...',
  workspaceTitle: '设定工作台',
  workspaceDesc: '把会话种子整理成结构化草案，再按模块审核并应用到项目设定。',
  breadcrumb: '写作 / 设定工作台',
  connectedHint: '当前会话已就绪',
  waitingHint: '等待创建会话',
  emptyTitle: '先创建一个设定会话',
  emptyDesc: '从种子想法开始，逐步生成作者圣经、角色、关系与世界状态草案。',
  stepContext: '当前会话上下文',
  stepContextDesc: '先确认这次设定生成要围绕哪个种子和哪些补充。',
  stepCompose: '设定讨论区',
  stepComposeDesc: '先和 AI 轻量讨论、澄清方向，再显式生成或更新草案。',
  stepDraft: '设定草案',
  stepDraftDesc: '先看模块骨架，生成后再逐个审阅并决定是否应用。',
  sessionSeedTitle: '种子构想',
  lastSupplementTitle: '最近补充',
  supplementEditButton: '编辑补充',
  supplementManualButton: '手动补充',
  supplementEditorTitle: '整理补充内容',
  supplementEditorPlaceholder: '保留要采纳的主题、设定要求或修改意见，支持 Markdown。',
  supplementSave: '保存到补充区',
  supplementSaving: '保存中...',
  supplementCancel: '取消',
  supplementAdopt: '采纳为补充',
  supplementAdoptWhole: '采纳全文',
  noSupplement: '还没有额外补充。',
  emptySeed: '该会话还没有写入种子内容。',
  advanceButton: '生成 / 更新草案',
  advanceRunning: '生成中...',
  advanceTooltip: '将基于当前讨论上下文生成草案，预计 20-40 秒。',
  generatePanelTitle: '生成前补充说明（可选）',
  generatePanelDesc: '不需要重新总结讨论。只填这次生成需要额外遵循的一句话。',
  generateNotePlaceholder: '例如：这次只更新世界状态，不改角色关系',
  generateConfirm: '生成草案',
  generateCancel: '取消',
  discussionScopeTitle: '作用域',
  discussionScopeHelp: '作用域只约束 AI 写入草案的范围，不限制讨论话题。',
  discussionStarterIntro: 'AI 已读取你的种子构想，想先和你确认几件事：',
  discussionPlaceholder: '和 AI 讨论设定方向，例如“邮差身份能不能带点诅咒感？”',
  discussionSend: '发送',
  discussionSending: '讨论中...',
  discussionReady: '普通讨论不会触发草案生成。',
  discussionCreateHint: '首次发送时会为当前设定会话创建一条讨论线。',
  discussionLoading: '正在加载讨论消息',
  discussionActionTitle: '待确认操作',
  discussionConfirmAction: '确认执行',
  discussionRejectAction: '拒绝',
  discussionConfirming: '执行中...',
  discussionRejecting: '拒绝中...',
  draftRequestTitle: '草案生成输入',
  resultLoading: '正在生成设定草案',
  resultEmpty: '还没有可审阅的草案，但工作台已经为你预留了模块骨架。',
  resultReady: '已生成',
  resultOpenQuestions: '待确认问题',
  resultOpenQuestionsEmpty: '没有额外待确认问题。',
  visualDraftTitle: '主控 Agent 草案看板',
  visualDraftSubtitle: '这是 agent 内部深化后通过工具展示的完整详细草案，确认前不会写入正式设定。',
  visualStyle: '风格',
  visualTone: '气质',
  visualBoldness: '大胆程度',
  visualWorldPressure: '世界压力',
  visualCharacterCards: '人物卡片',
  visualRelationshipGraph: '关系网络',
  visualNextAgents: '建议下一步',
  visualNoBoard: '这版结果没有返回可视化看板，但下方结构化草案仍可审阅和应用。',
  regenerateDraft: '重起草一版',
  cancelDraft: '取消这版草案',
  applyTitle: '应用草案',
  applySubtitle: '选择要写入项目的模块，并为这次提交留下备注。',
  applyScopeTitle: '应用范围',
  applyBible: '作者圣经',
  applyCharacters: '角色设定',
  applyRelationships: '关系设定',
  applyWorld: '世界状态',
  scopePending: '暂无草案',
  scopeDiscussing: '讨论中 · 暂无草案',
  scopeGenerating: '生成中',
  scopeReady: '待应用',
  scopeApplied: '已应用',
  noteTitle: '应用备注（可选）',
  authorNotePlaceholder: '记录这次应用的决策、取舍或后续待调整点',
  applyButton: '应用到项目',
  applyDisabled: '请先生成草案',
  applyDisabledHint: '生成草案后可逐模块应用。',
  applyPending: '应用中...',
  eventHistoryTitle: '运行记录',
  eventHistoryHint: '会话创建、草案生成与应用过程都会记录在这里。',
  eventHistoryLoading: '正在加载运行记录',
  eventHistoryEmpty: '开始第一次设定生成后，这里会出现时间线记录。',
  noRunHistory: '生成草案后才会出现运行记录。',
  expandEvents: '展开全部',
  collapseEvents: '收起',
  summaryTitle: '草案摘要',
  bibleTitle: '作者圣经',
  theme: '主题',
  styleGuide: '文风',
  worldRules: '世界规则',
  aesthetic: '审美原则',
  hardConstraints: '硬约束',
  softPreferences: '软偏好',
  forbiddenMoves: '禁用套路',
  charactersTitle: '角色草案',
  relationshipsTitle: '关系草案',
  worldStateTitle: '世界状态',
  charactersUnit: '人',
  relationshipsUnit: '条',
  worldStateUnit: '项',
  questionsUnit: '个',
  defaultRole: '未设定角色定位',
  defaultProfile: '未提供角色简介。',
  personality: '性格',
  voiceStyle: '口吻',
  goals: '目标',
  fears: '恐惧',
  secrets: '秘密',
  constraints: '约束',
  noCharacters: '未生成角色草案。',
  noRelationships: '未生成关系草案。',
  relationshipDraft: '关系草案',
  volatility: '波动',
  defaultRelationshipSummary: '未提供关系摘要。',
  anchors: '关系锚点',
  tensionPoints: '张力点',
  noWorldState: '未生成世界状态。',
  defaultNote: '未提供说明',
  defaultQuestionReason: '未说明该问题的影响。',
  noQuestions: '没有额外待确认的问题。',
  listEmpty: '-',
  rolePrefix: '角色',
  statePrefix: '状态',
  questionPrefix: '问题',
  roleA: '角色 A',
  roleB: '角色 B',
  viewSuffix: '的视角',
  surfaceView: '表面：',
  privateView: '私下：',
  misunderstanding: '误判：',
  masking: '伪装：',
  moduleApply: '加入应用',
  moduleApplied: '已选中',
  modulePrompt: '生成此模块提示',
  waitingGenerate: '待生成',
  failureTitle: '上次生成失败',
  failurePendingTitle: '这条会话待重试',
  failurePendingMessage: '上次生成没有完成，你可以直接补充要求后再重新生成，不需要新建会话。',
  failureAction: '生成一条重试提示',
  failureFallback: '可以缩小范围、补充约束或重新描述种子后再试一次。',
} as const;

const defaultAcceptSections = {
  authorBible: true,
  characters: true,
  relationships: true,
  worldState: true,
};

type AcceptSectionsState = typeof defaultAcceptSections;
type ModuleKey = keyof AcceptSectionsState;
type DiscussionScope = 'all' | 'author_bible' | 'characters' | 'relationships' | 'world';
type ApplyPanelState = 'empty' | 'discussing' | 'generating' | 'ready' | 'applied';
type FlowStepTone = 'idle' | 'active' | 'done';

const discussionScopes: Array<{ key: DiscussionScope; label: string }> = [
  { key: 'all', label: '全部' },
  { key: 'author_bible', label: '作者圣经' },
  { key: 'characters', label: '角色' },
  { key: 'relationships', label: '关系' },
  { key: 'world', label: '世界' },
];

const starterPrompts = [
  '主角最想逃避的过往或心结具体是什么？',
  '这个世界里最重要的规则或代价是什么？',
  '角色关系想先从信任、亏欠还是冲突开始？',
];

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
    prompt: '请重新生成作者圣经，重点补强主题、文风与世界规则。',
  },
  {
    key: 'characters',
    icon: Users,
    label: copy.applyCharacters,
    prompt: '请重新生成角色草案，重点补强角色动机、冲突和口吻区分。',
  },
  {
    key: 'relationships',
    icon: Link2,
    label: copy.applyRelationships,
    prompt: '请重新生成关系草案，重点补强双向视角、张力点和关系演化线索。',
  },
  {
    key: 'worldState',
    icon: Globe2,
    label: copy.applyWorld,
    prompt: '请重新生成世界状态，重点补充规则设定、关键变量和约束条件。',
  },
];

export function SetupWorkspacePage() {
  const { projectId = '' } = useParams();
  const queryClient = useQueryClient();
  const [seedIdea, setSeedIdea] = useState('');
  const [discussionMessage, setDiscussionMessage] = useState('');
  const [activeDiscussionScope, setActiveDiscussionScope] = useState<DiscussionScope>('all');
  const [activeDialogueSessionId, setActiveDialogueSessionId] = useState('');
  const [activeDialogueRunId, setActiveDialogueRunId] = useState('');
  const [activeSessionId, setActiveSessionId] = useState('');
  const [activeRunId, setActiveRunId] = useState('');
  const [generationNote, setGenerationNote] = useState('');
  const [generatePanelOpen, setGeneratePanelOpen] = useState(false);
  const [authorNote, setAuthorNote] = useState('');
  const [acceptSections, setAcceptSections] = useState(defaultAcceptSections);
  const [noteExpanded, setNoteExpanded] = useState(false);
  const [supplementDraft, setSupplementDraft] = useState('');
  const [supplementEditorOpen, setSupplementEditorOpen] = useState(false);
  const [localSupplement, setLocalSupplement] = useState('');

  const sessionsQuery = useQuery({
    queryKey: ['setupSessions', projectId, 1, 20],
    queryFn: ({ signal }) => listSetupSessions(projectId, 1, 20, signal),
    enabled: Boolean(projectId),
  });

  const sessions = sessionsQuery.data?.data ?? [];
  const selectedSessionId = activeSessionId || sessions[0]?.id || '';
  const currentSession = sessions.find((session) => session.id === selectedSessionId) ?? sessions[0] ?? null;
  const currentSupplement = localSupplement || currentSession?.last_user_message || '';

  const dialogueSessionsQuery = useQuery({
    queryKey: ['dialogueSessions', projectId, 1, 50],
    queryFn: ({ signal }) => listDialogueSessions(projectId, 1, 50, signal),
    enabled: Boolean(projectId),
  });

  const dialogueSessions = dialogueSessionsQuery.data?.data ?? [];
  const setupDiscussionSession = useMemo(
    () => findSetupDiscussionSession(dialogueSessions, selectedSessionId),
    [dialogueSessions, selectedSessionId],
  );
  const selectedDialogueSessionId = activeDialogueSessionId || setupDiscussionSession?.id || '';
  const selectedDialogueRunId = activeDialogueRunId || setupDiscussionSession?.latest_run_id || '';

  const dialogueSessionQuery = useQuery({
    queryKey: ['dialogueSession', selectedDialogueSessionId],
    queryFn: ({ signal }) => getDialogueSession(selectedDialogueSessionId, signal),
    enabled: Boolean(selectedDialogueSessionId),
  });

  const dialogueRunQuery = useQuery({
    queryKey: ['dialogueRun', selectedDialogueRunId],
    queryFn: ({ signal }) => getDialogueRun(selectedDialogueRunId, signal),
    enabled: Boolean(selectedDialogueRunId),
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      return isActiveDialogueRunStatus(status) ? 1500 : false;
    },
  });

  const dialogueResultQuery = useQuery({
    queryKey: ['dialogueRunResult', selectedDialogueRunId],
    queryFn: ({ signal }) => getDialogueRunResult(selectedDialogueRunId, signal),
    enabled: Boolean(selectedDialogueRunId) && hasDialogueRunResult(dialogueRunQuery.data?.status),
  });

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

  useEffect(() => {
    setActiveDialogueSessionId('');
    setActiveDialogueRunId('');
    setActiveDiscussionScope('all');
    setDiscussionMessage('');
    setGenerationNote('');
    setGeneratePanelOpen(false);
    setSupplementDraft('');
    setSupplementEditorOpen(false);
    setLocalSupplement('');
  }, [selectedSessionId]);

  useEffect(() => {
    if (!selectedDialogueRunId || !hasDialogueRunResult(dialogueRunQuery.data?.status)) {
      return;
    }
    queryClient.invalidateQueries({ queryKey: ['dialogueSession', selectedDialogueSessionId] });
    queryClient.invalidateQueries({ queryKey: ['dialogueSessions', projectId] });
  }, [dialogueRunQuery.data?.status, projectId, queryClient, selectedDialogueRunId, selectedDialogueSessionId]);

  const createSessionMutation = useMutation({
    mutationFn: () => createSetupSession(projectId, { seed_idea: seedIdea.trim() }),
    onSuccess: (session) => {
      setActiveSessionId(session.id);
      setActiveRunId(session.latest_run_id ?? '');
      setSeedIdea('');
      setAuthorNote('');
      setGenerationNote('');
      setGeneratePanelOpen(false);
      queryClient.invalidateQueries({ queryKey: ['setupSessions', projectId] });
    },
  });

  const deleteSessionMutation = useMutation({
    mutationFn: (session: SetupSession) => deleteSetupSession(session.id),
    onSuccess: (_result, deletedSession) => {
      const deletedSessionId = deletedSession.id;
      queryClient.setQueriesData<PaginatedResponse<SetupSession>>({ queryKey: ['setupSessions', projectId] }, (page) =>
        removeSetupSessionFromPage(page, deletedSessionId),
      );
      queryClient.removeQueries({ queryKey: ['setupSession', deletedSessionId] });
      if (deletedSession.latest_run_id) {
        queryClient.removeQueries({ queryKey: ['setupRun', deletedSession.latest_run_id] });
        queryClient.removeQueries({ queryKey: ['setupRunResult', deletedSession.latest_run_id] });
        queryClient.removeQueries({ queryKey: ['setupRunEventHistory', deletedSession.latest_run_id] });
      }
      if (selectedSessionId === deletedSessionId) {
        const nextSession = sessions.find((session) => session.id !== deletedSessionId);
        setActiveSessionId(nextSession?.id ?? '');
        setActiveRunId(nextSession?.latest_run_id ?? '');
        setActiveDialogueSessionId('');
        setActiveDialogueRunId('');
        setDiscussionMessage('');
        setGenerationNote('');
        setGeneratePanelOpen(false);
        setSupplementDraft('');
        setSupplementEditorOpen(false);
        setLocalSupplement('');
        setAuthorNote('');
        setAcceptSections(defaultAcceptSections);
      }
      queryClient.invalidateQueries({ queryKey: ['setupSessions', projectId] });
      queryClient.invalidateQueries({ queryKey: ['dialogueSessions', projectId] });
    },
  });

  const advanceMutation = useMutation({
    mutationFn: (userMessage: string) => advanceSetupSession(selectedSessionId, { user_message: userMessage.trim() }),
    onSuccess: (run) => {
      setActiveRunId(run.run_id ?? run.id ?? '');
      setGenerationNote('');
      setGeneratePanelOpen(false);
      setAcceptSections(defaultAcceptSections);
      queryClient.invalidateQueries({ queryKey: ['setupSessions', projectId] });
    },
  });

  const supplementMutation = useMutation({
    mutationFn: (lastUserMessage: string) =>
      updateSetupSession(selectedSessionId, {
        last_user_message: lastUserMessage.trim(),
      }),
    onSuccess: (session) => {
      setLocalSupplement(session.last_user_message ?? '');
      setSupplementDraft('');
      setSupplementEditorOpen(false);
      queryClient.invalidateQueries({ queryKey: ['setupSessions', projectId] });
    },
  });

  const discussionMutation = useMutation({
    mutationFn: async () => {
      const rawMessage = discussionMessage.trim();
      let dialogueSessionId = selectedDialogueSessionId;
      if (!dialogueSessionId) {
        const session = await createDialogueSession(projectId, { title: getSetupDiscussionTitle(selectedSessionId) });
        dialogueSessionId = session.id;
      }
      const run = await advanceDialogueSession(dialogueSessionId, {
        user_message: buildSetupDiscussionMessage({
          activeRunId,
          message: rawMessage,
          scope: activeDiscussionScope,
          session: currentSession,
          supplement: currentSupplement,
        }),
      });
      return { dialogueSessionId, run };
    },
    onSuccess: ({ dialogueSessionId, run }) => {
      setActiveDialogueSessionId(dialogueSessionId);
      setActiveDialogueRunId(run.run_id ?? run.id ?? '');
      setDiscussionMessage('');
      queryClient.invalidateQueries({ queryKey: ['dialogueSessions', projectId] });
      queryClient.invalidateQueries({ queryKey: ['dialogueSession', dialogueSessionId] });
    },
  });

  const confirmActionMutation = useMutation({
    mutationFn: (optionId: string) => confirmDialogueActionOption(optionId, { confirm: true }),
    onSuccess: (option) => {
      const setupRunID = stringFromRecord(option.result, 'setup_run_id');
      const setupSessionID = stringFromRecord(option.result, 'setup_session_id');
      if (setupSessionID) {
        setActiveSessionId(setupSessionID);
      }
      if (setupRunID) {
        setActiveRunId(setupRunID);
      }
      queryClient.invalidateQueries({ queryKey: ['dialogueSession', selectedDialogueSessionId] });
      queryClient.invalidateQueries({ queryKey: ['dialogueRunResult', selectedDialogueRunId] });
      queryClient.invalidateQueries({ queryKey: ['dialogueSessions', projectId] });
      queryClient.invalidateQueries({ queryKey: ['setupSessions', projectId] });
      queryClient.invalidateQueries({ queryKey: ['project', projectId] });
      queryClient.invalidateQueries({ queryKey: ['authorBible', projectId] });
      queryClient.invalidateQueries({ queryKey: ['characters', projectId] });
      queryClient.invalidateQueries({ queryKey: ['relationships', projectId] });
    },
  });

  const rejectActionMutation = useMutation({
    mutationFn: (optionId: string) => rejectDialogueActionOption(optionId, { reason: '作者在设定讨论区拒绝了这个操作。' }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['dialogueSession', selectedDialogueSessionId] });
      queryClient.invalidateQueries({ queryKey: ['dialogueRunResult', selectedDialogueRunId] });
      queryClient.invalidateQueries({ queryKey: ['dialogueSessions', projectId] });
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
  const applyButtonLabel = applyMutation.isPending ? copy.applyPending : copy.applyButton;
  const workspaceStatus = getWorkspaceDisplayStatus(runQuery.data?.status, currentSession?.status);
  const runStatusLabel = formatRunStatus(workspaceStatus);
  const runStatusTone = getStatusTone(workspaceStatus);
  const runErrorMessage = firstText(getRunErrorMessage(runQuery.data), currentSession?.latest_run_error, getRunEventErrorMessage(eventHistoryQuery.data));
  const sessionNeedsRetry = !activeRunId && currentSession?.status === 'failed';
  const showFailureCard = Boolean(runErrorMessage || sessionNeedsRetry);
  const failureTitle = runErrorMessage ? copy.failureTitle : copy.failurePendingTitle;
  const failureMessage = runErrorMessage || copy.failurePendingMessage;
  const dialogueMessages = dialogueSessionQuery.data?.messages ?? [];
  const latestActionOptions = dialogueResultQuery.data?.action_options ?? [];
  const isDialogueRunActive = isActiveDialogueRunStatus(dialogueRunQuery.data?.status);
  const visibleDiscussionMessages = useMemo(
    () => dialogueMessages.filter((message) => displayDiscussionContent(message.content)),
    [dialogueMessages],
  );
  const hasDiscussionActivity = visibleDiscussionMessages.length > 0 || isDialogueRunActive || discussionMutation.isPending;
  const applyPanelState = getApplyPanelState({
    hasDiscussionActivity,
    hasDraft,
    isApplied: isSetupApplied(workspaceStatus),
    isGenerating: isRunActive,
  });
  const discussionError = firstError(
    dialogueSessionsQuery.error,
    dialogueSessionQuery.error,
    dialogueRunQuery.error,
    dialogueResultQuery.error,
    discussionMutation.error,
    confirmActionMutation.error,
    rejectActionMutation.error,
  );

  const startSession = (event: FormEvent) => {
    event.preventDefault();
    if (!seedIdea.trim() || createSessionMutation.isPending) {
      return;
    }
    createSessionMutation.mutate();
  };

  const removeSession = (session: SetupSession) => {
    if (deleteSessionMutation.isPending) {
      return;
    }
    if (window.confirm(`${copy.deleteSessionConfirm}\n\n${getSessionTitle(session)}`)) {
      deleteSessionMutation.mutate(session);
    }
  };

  const openGeneratePanel = () => {
    if (!selectedSessionId || advanceMutation.isPending || isRunActive) {
      return;
    }
    setGeneratePanelOpen(true);
  };

  const openSupplementEditor = () => {
    setSupplementDraft(formatSetupSupplementForDisplay(currentSupplement));
    setSupplementEditorOpen(true);
  };

  const adoptSupplement = (value: string) => {
    const content = value.trim();
    if (!content) {
      return;
    }
    setSupplementDraft(content);
    setSupplementEditorOpen(true);
  };

  const cancelSupplementEditor = () => {
    setSupplementDraft('');
    setSupplementEditorOpen(false);
  };

  const saveSupplement = (event: FormEvent) => {
    event.preventDefault();
    const content = supplementDraft.trim();
    if (!selectedSessionId || !content || supplementMutation.isPending) {
      return;
    }
    supplementMutation.mutate(content);
  };

  const startDraftGeneration = (optionalInstruction = generationNote) => {
    if (!selectedSessionId || advanceMutation.isPending || isRunActive) {
      return;
    }

    advanceMutation.mutate(
      buildSetupDraftGenerationMessage({
        activeRunId,
        discussionMessages: dialogueMessages,
        optionalInstruction,
        scope: activeDiscussionScope,
        session: currentSession,
        supplement: currentSupplement,
      }),
    );
  };

  const confirmDraftGeneration = (event: FormEvent) => {
    event.preventDefault();
    startDraftGeneration(generationNote);
  };

  const cancelDraftGeneration = () => {
    setGeneratePanelOpen(false);
    setGenerationNote('');
  };

  const sendDiscussionMessage = (event: FormEvent) => {
    event.preventDefault();
    if (!discussionMessage.trim() || !selectedSessionId || discussionMutation.isPending || isDialogueRunActive) {
      return;
    }
    discussionMutation.mutate();
  };

  const toggleSection = (key: ModuleKey) => {
    setAcceptSections((current) => ({ ...current, [key]: !current[key] }));
  };

  const fillModulePrompt = (moduleKey: ModuleKey) => {
    const definition = moduleDefinitions.find((item) => item.key === moduleKey);
    if (!definition) {
      return;
    }
    setGenerationNote(definition.prompt);
    setGeneratePanelOpen(true);
  };

  const fillRetryPrompt = () => {
    setGenerationNote(
      '请重新生成这次设定草案，并在不改变核心种子的前提下缩小范围，补充硬约束与世界规则。',
    );
    setGeneratePanelOpen(true);
  };

  const regenerateDraft = () => {
    if (!selectedSessionId || advanceMutation.isPending || isRunActive) {
      return;
    }
    startDraftGeneration('请基于当前会话重新起草一版，不要直接沿用上一版；方向更大胆、更天马行空，同时保留可落地的人物动机和关系张力。');
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
              onKeyDown={submitTextareaOnEnter}
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
              {sessions.map((session) => {
                const discussionCount = getSessionDiscussionCount(session, dialogueSessions, selectedSessionId, visibleDiscussionMessages);
                const isDeletingSession = deleteSessionMutation.isPending && deleteSessionMutation.variables?.id === session.id;
                return (
                  <article
                    className={
                      selectedSessionId === session.id
                        ? 'session-item session-item--setup session-item--active'
                        : 'session-item session-item--setup'
                    }
                    key={session.id}
                  >
                    <button
                      aria-label={`选择设定会话 ${getSessionTitle(session)}`}
                      className="session-item__body session-item__body--setup"
                      onClick={() => {
                        setActiveSessionId(session.id);
                        setActiveRunId(session.latest_run_id ?? '');
                      }}
                      type="button"
                    >
                      <div className="session-item__headline">
                        <strong>{getSessionTitle(session)}</strong>
                      </div>
                      <p className="session-item__summary">{getSessionSummary(session)}</p>
                      <div className="session-item__meta">
                        <span>
                          {formatSetupSessionMeta({
                            applied: isSetupApplied(session.status),
                            discussionCount,
                            draftCount: session.latest_run_id ? 1 : 0,
                            updatedAt: session.updated_at ?? session.created_at,
                          })}
                        </span>
                      </div>
                    </button>
                    <button
                      aria-label={`${copy.deleteSessionTitle} ${getSessionTitle(session)}`}
                      aria-busy={isDeletingSession}
                      className={`icon-button session-item__action session-item__action--danger${
                        isDeletingSession ? ' session-item__action--pending' : ''
                      }`}
                      disabled={deleteSessionMutation.isPending}
                      onClick={(event) => {
                        event.stopPropagation();
                        removeSession(session);
                      }}
                      title={isDeletingSession ? copy.deleteSessionPending : copy.deleteSessionTitle}
                      type="button"
                    >
                      {isDeletingSession ? <span className="session-item__action-spinner" /> : <Trash2 size={15} />}
                    </button>
                  </article>
                );
              })}
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
          {deleteSessionMutation.isError ? <ErrorState message={(deleteSessionMutation.error as Error).message} /> : null}
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
                    <div className="setup-context-card__block-heading">
                      <small>{copy.lastSupplementTitle}</small>
                      <button className="button button--ghost setup-context-card__edit" onClick={openSupplementEditor} type="button">
                        {currentSupplement ? copy.supplementEditButton : copy.supplementManualButton}
                      </button>
                    </div>
                    <SetupContextSupplement
                      lastDiscussion={getLastDiscussionText(visibleDiscussionMessages)}
                      supplement={currentSupplement}
                    />
                    {supplementEditorOpen ? (
                      <SetupSupplementEditor
                        isSaving={supplementMutation.isPending}
                        onCancel={cancelSupplementEditor}
                        onSubmit={saveSupplement}
                        onValueChange={setSupplementDraft}
                        value={supplementDraft}
                      />
                    ) : null}
                    {supplementMutation.isError ? <ErrorState message={(supplementMutation.error as Error).message} /> : null}
                  </div>
                </div>
              </section>

              <section className="setup-stage composer composer--setup">
                <div className="setup-stage__header">
                  <div className="step-badge">2</div>
                  <div>
                    <h2>{copy.stepCompose}</h2>
                    <p>{copy.stepComposeDesc}</p>
                  </div>
                </div>
                <SetupFlowProgress state={applyPanelState} />

                <SetupDiscussionPanel
                  actionOptions={latestActionOptions}
                  canGenerate={Boolean(selectedSessionId)}
                  confirmingOptionId={confirmActionMutation.isPending ? confirmActionMutation.variables : undefined}
                  error={discussionError}
                  isLoading={dialogueSessionQuery.isLoading}
                  isGenerating={advanceMutation.isPending || isRunActive}
                  isRunning={discussionMutation.isPending || isDialogueRunActive}
                  messages={dialogueMessages}
                  onConfirmAction={(optionId) => confirmActionMutation.mutate(optionId)}
                  onAdoptSupplement={adoptSupplement}
                  onGenerateDraft={openGeneratePanel}
                  onRejectAction={(optionId) => rejectActionMutation.mutate(optionId)}
                  onScopeChange={setActiveDiscussionScope}
                  onStarterSelect={setDiscussionMessage}
                  onSubmit={sendDiscussionMessage}
                  onValueChange={setDiscussionMessage}
                  rejectingOptionId={rejectActionMutation.isPending ? rejectActionMutation.variables : undefined}
                  scope={activeDiscussionScope}
                  scopes={discussionScopes}
                  value={discussionMessage}
                />
                {generatePanelOpen ? (
                  <SetupGenerateConfirmPanel
                    isRunning={advanceMutation.isPending || isRunActive}
                    onCancel={cancelDraftGeneration}
                    onSubmit={confirmDraftGeneration}
                    onValueChange={setGenerationNote}
                    value={generationNote}
                  />
                ) : null}
              </section>

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
                    <span className="setup-count-badge">{formatApplyModuleStatus(module.key, count, applyPanelState)}</span>
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

          <p className="setup-apply-hint">{getApplyPanelHint(applyPanelState)}</p>
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

function getSetupDiscussionTitle(setupSessionId: string) {
  return `setup-discussion:${setupSessionId}`;
}

function findSetupDiscussionSession(sessions: DialogueSession[], setupSessionId: string) {
  if (!setupSessionId) {
    return undefined;
  }
  return sessions.find((session) => session.title === getSetupDiscussionTitle(setupSessionId));
}

function buildSetupDiscussionMessage({
  activeRunId,
  message,
  scope,
  session,
  supplement,
}: {
  activeRunId: string;
  message: string;
  scope: DiscussionScope;
  session: SetupSession | null;
  supplement?: string;
}) {
  const setupRunId = firstText(activeRunId, session?.latest_run_id);
  return [
    '__setup_discussion_context__',
    `scope=${scope}`,
    `setup_session_id=${session?.id ?? ''}`,
    `setup_run_id=${setupRunId}`,
    `seed_idea=${session?.seed_idea ?? ''}`,
    `last_setup_supplement=${formatSetupSupplementForDisplay(supplement || session?.last_user_message)}`,
    'instruction=这是设定工作台的轻量讨论消息。除非用户明确要求生成、更新或应用草案，否则只追问、总结或给建议，不要提出执行操作。',
    '__user_message__',
    message,
  ].join('\n');
}

function buildSetupDraftGenerationMessage({
  activeRunId,
  discussionMessages,
  optionalInstruction,
  scope,
  session,
  supplement,
}: {
  activeRunId: string;
  discussionMessages: DialogueMessage[];
  optionalInstruction: string;
  scope: DiscussionScope;
  session: SetupSession | null;
  supplement?: string;
}) {
  const discussionContext = discussionMessages
    .map((message) => {
      const content = displayDiscussionContent(message.content);
      if (!content) {
        return '';
      }
      return `[${dialogueRoleLabel(message.role)}] ${content}`;
    })
    .filter(Boolean)
    .slice(-20)
    .join('\n');
  const optional = optionalInstruction.trim();
  const lastSupplement = formatSetupSupplementForDisplay(supplement || session?.last_user_message);

  return [
    '请基于当前设定讨论生成一版结构化草案。',
    '',
    `生成范围：${formatDiscussionScope(scope)}`,
    '',
    '种子构想：',
    firstText(session?.seed_idea, '暂无种子内容。'),
    lastSupplement ? `\n最近补充：\n${lastSupplement}` : '',
    '',
    '讨论上下文：',
    discussionContext || '暂无讨论消息，请基于种子构想起草。',
    optional ? `\n本次生成额外要求：${optional}` : '',
    '',
    '请不要要求作者重复输入已经讨论过的信息。',
  ]
    .filter(Boolean)
    .join('\n');
}

function displayDiscussionContent(content: string) {
  const marker = '__user_message__';
  const markerIndex = content.indexOf(marker);
  if (markerIndex >= 0) {
    return content.slice(markerIndex + marker.length).trim();
  }
  return content.trim();
}

function getSupplementAdoptionOptions(content: string) {
  const lines = content.split(/\r?\n/);
  const headings = lines
    .map((line, index) => {
      const match = line.match(/^\s{0,3}#{2,4}\s+(.+?)\s*#*\s*$/);
      if (!match) {
        return undefined;
      }
      return {
        index,
        label: cleanSupplementOptionLabel(match[1]),
      };
    })
    .filter(Boolean) as Array<{ index: number; label: string }>;

  return headings
    .filter((heading) => heading.label)
    .slice(0, 4)
    .map((heading, order, selectedHeadings) => {
      const nextHeading = selectedHeadings[order + 1] ?? headings.find((item) => item.index > heading.index);
      const endIndex = nextHeading?.index ?? lines.length;
      return {
        label: heading.label,
        content: lines.slice(heading.index, endIndex).join('\n').trim(),
      };
    })
    .filter((option) => option.content);
}

function cleanSupplementOptionLabel(value: string) {
  const label = value
    .replace(/\[([^\]]+)]\([^)]+\)/g, '$1')
    .replace(/[*_`~]/g, '')
    .replace(/[：:]\s*$/, '')
    .trim();
  return label;
}

function SetupContextSupplement({ supplement, lastDiscussion }: { supplement?: string; lastDiscussion?: string }) {
  const summary = parseSetupSupplement(supplement);
  const rows = [
    summary.action ? ['最近动作', summary.action] : undefined,
    summary.scope ? ['生成范围', summary.scope] : undefined,
    summary.instruction ? ['额外要求', summary.instruction] : undefined,
    summary.content ? ['最近补充', summary.content] : undefined,
    lastDiscussion ? ['最后讨论', lastDiscussion] : undefined,
  ].filter(Boolean) as Array<[string, string]>;

  if (rows.length === 0) {
    return <p>{copy.noSupplement}</p>;
  }

  return (
    <dl className="setup-context-summary">
      {rows.map(([label, value]) => (
        <div className="setup-context-summary__row" key={label}>
          <dt>{label}</dt>
          <dd>
            <MarkdownRenderer source={value} variant="compact" />
          </dd>
        </div>
      ))}
    </dl>
  );
}

function parseSetupSupplement(value?: string) {
  const text = value?.trim();
  if (!text) {
    return {};
  }

  if (text.includes('__setup_draft_generation_context__')) {
    return {
      action: '生成设定草案',
      instruction: extractInternalOptionalInstruction(text),
      scope: formatDiscussionScope(text.match(/^scope=([^\s\n]+)/m)?.[1]),
    };
  }

  if (text.startsWith('请基于当前设定讨论生成一版结构化草案')) {
    return {
      action: '生成设定草案',
      instruction: text.match(/^本次生成额外要求：(.+)$/m)?.[1]?.trim(),
      scope: text.match(/^生成范围：(.+)$/m)?.[1]?.trim() || formatDiscussionScope('all'),
    };
  }

  if (text.includes('__setup_discussion_context__')) {
    return {
      action: '补充设定讨论',
      content: displayDiscussionContent(text),
    };
  }

  return {
    action: '补充设定要求',
    content: text,
  };
}

function extractInternalOptionalInstruction(text: string) {
  return text.match(/^optional_instruction=(.*?)(?:\s+instruction=|$)/m)?.[1]?.trim();
}

function getLastDiscussionText(messages: DialogueMessage[]) {
  const last = [...messages].reverse().find((message) => displayDiscussionContent(message.content));
  return last ? displayDiscussionContent(last.content) : '';
}

function formatSetupSupplementForDisplay(value?: string) {
  const text = value?.trim();
  if (!text) {
    return '';
  }

  if (text.includes('__setup_discussion_context__')) {
    return displayDiscussionContent(text);
  }

  if (text.includes('__setup_draft_generation_context__')) {
    return summarizeInternalDraftGeneration(text);
  }

  if (text.startsWith('请基于当前设定讨论生成一版结构化草案')) {
    return summarizeReadableDraftGeneration(text);
  }

  return text;
}

function summarizeInternalDraftGeneration(text: string) {
  const scope = text.match(/^scope=([^\s\n]+)/m)?.[1];
  const optionalInstruction = text.match(/^optional_instruction=([^\n]+)/m)?.[1]?.trim();
  return formatDraftGenerationSummary(formatDiscussionScope(scope), optionalInstruction);
}

function summarizeReadableDraftGeneration(text: string) {
  const scope = text.match(/^生成范围：(.+)$/m)?.[1]?.trim();
  const optionalInstruction = text.match(/^本次生成额外要求：(.+)$/m)?.[1]?.trim();
  return formatDraftGenerationSummary(scope || formatDiscussionScope('all'), optionalInstruction);
}

function formatDraftGenerationSummary(scopeLabel: string, optionalInstruction?: string) {
  return [
    '已请求基于当前讨论生成设定草案。',
    `生成范围：${scopeLabel}。`,
    optionalInstruction ? `额外要求：${optionalInstruction}。` : '',
  ]
    .filter(Boolean)
    .join(' ');
}

function formatDiscussionScope(scope?: string) {
  switch (scope) {
    case 'author_bible':
      return '作者圣经';
    case 'characters':
      return '角色设定';
    case 'relationships':
      return '关系设定';
    case 'world':
      return '世界状态';
    case 'all':
    default:
      return '全部设定';
  }
}

function firstError(...values: unknown[]) {
  for (const value of values) {
    if (value instanceof Error) {
      return value;
    }
  }
  return null;
}

function stringFromRecord(record: Record<string, unknown> | undefined, key: string) {
  const value = record?.[key];
  return typeof value === 'string' ? value : '';
}

function isActiveDialogueRunStatus(status?: string) {
  return status === 'queued' || status === 'loading_state' || status === 'planning_actions' || status === 'executing_action' || status === 'running';
}

function hasDialogueRunResult(status?: string) {
  return status === 'completed' || status === 'review_required';
}

function isPendingDialogueAction(option: DialogueActionOption) {
  return option.status === 'pending' || option.status === 'confirmed';
}

function getDialogueActionTone(status?: string) {
  switch (status) {
    case 'executed':
      return 'success';
    case 'failed':
      return 'danger';
    case 'rejected':
      return 'neutral';
    case 'executing':
    case 'confirmed':
    case 'pending':
      return 'warning';
    default:
      return 'neutral';
  }
}

function formatDialogueActionStatus(status?: string) {
  switch (status) {
    case 'pending':
      return '待确认';
    case 'confirmed':
      return '已确认';
    case 'executing':
      return '执行中';
    case 'executed':
      return '已执行';
    case 'rejected':
      return '已拒绝';
    case 'failed':
      return '失败';
    default:
      return '未知';
  }
}

function defaultDialogueActionLabel(actionType?: string) {
  switch (actionType) {
    case 'setup_start_and_advance':
      return '生成项目设定草案';
    case 'setup_advance':
      return '更新设定草案';
    case 'setup_apply':
      return '应用设定草案';
    case 'story_create_and_advance':
      return '开始剧情编排';
    case 'story_advance':
      return '继续剧情编排';
    case 'story_cut_chapter':
      return '提交章节草稿';
    default:
      return '执行下一步';
  }
}

function defaultDialogueActionDescription(actionType?: string) {
  switch (actionType) {
    case 'setup_start_and_advance':
    case 'setup_advance':
      return '确认后会触发 setup run，生成一版新的结构化草案。';
    case 'setup_apply':
      return '确认后会把已审核的草案写入项目状态。';
    default:
      return '确认后执行这个待确认操作。';
  }
}

function dialogueRoleLabel(role: string) {
  switch (role) {
    case 'assistant':
      return 'AI';
    case 'tool':
      return '系统';
    case 'system':
      return '上下文';
    default:
      return '我';
  }
}

function dialogueRoleClass(role: string) {
  if (role === 'assistant' || role === 'tool' || role === 'system') {
    return role;
  }
  return 'user';
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

function getApplyPanelState({
  hasDiscussionActivity,
  hasDraft,
  isApplied,
  isGenerating,
}: {
  hasDiscussionActivity: boolean;
  hasDraft: boolean;
  isApplied: boolean;
  isGenerating: boolean;
}): ApplyPanelState {
  if (isApplied) {
    return 'applied';
  }
  if (isGenerating) {
    return 'generating';
  }
  if (hasDraft) {
    return 'ready';
  }
  if (hasDiscussionActivity) {
    return 'discussing';
  }
  return 'empty';
}

function isSetupApplied(status?: string) {
  return status === 'applied' || status === 'committed';
}

function formatApplyModuleStatus(key: ModuleKey, count: number, state: ApplyPanelState) {
  if (count > 0) {
    return formatModuleCount(key, count);
  }

  switch (state) {
    case 'applied':
      return copy.scopeApplied;
    case 'generating':
      return copy.scopeGenerating;
    case 'discussing':
      return copy.scopeDiscussing;
    case 'ready':
      return copy.scopeReady;
    default:
      return copy.scopePending;
  }
}

function getApplyPanelHint(state: ApplyPanelState) {
  switch (state) {
    case 'generating':
      return '正在生成草案，完成后可逐模块应用。';
    case 'ready':
      return '选择要写入项目的模块。';
    case 'applied':
      return '这版草案已应用，可继续讨论后更新草案。';
    default:
      return copy.applyDisabledHint;
  }
}

function getSessionDiscussionCount(
  session: SetupSession,
  dialogueSessions: DialogueSession[],
  selectedSessionId: string,
  selectedMessages: DialogueMessage[],
) {
  if (session.id === selectedSessionId) {
    return selectedMessages.length;
  }

  const dialogueSession = findSetupDiscussionSession(dialogueSessions, session.id);
  const messages = dialogueSession?.messages?.filter((message) => displayDiscussionContent(message.content));
  if (messages) {
    return messages.length;
  }
  return dialogueSession?.last_user_message ? 1 : 0;
}

function formatSetupSessionMeta({
  applied,
  discussionCount,
  draftCount,
  updatedAt,
}: {
  applied: boolean;
  discussionCount: number;
  draftCount: number;
  updatedAt?: string;
}) {
  const draftLabel = `${draftCount} 草案${applied && draftCount > 0 ? '已应用' : ''}`;
  return `${formatRelativeTime(updatedAt)} · ${discussionCount} 讨论 · ${draftLabel}`;
}

function removeSetupSessionFromPage(page: PaginatedResponse<SetupSession> | undefined, deletedSessionId: string) {
  if (!page) {
    return page;
  }

  const nextData = page.data.filter((session) => session.id !== deletedSessionId);
  if (nextData.length === page.data.length) {
    return page;
  }

  const pagination = page.meta?.pagination;
  return {
    ...page,
    data: nextData,
    meta: pagination
      ? {
          ...page.meta,
          pagination: {
            ...pagination,
            total: Math.max(0, pagination.total - 1),
          },
        }
      : page.meta,
  };
}

function SetupFlowProgress({ state }: { state: ApplyPanelState }) {
  const steps = getFlowSteps(state);
  return (
    <div className="setup-flow-progress" aria-label="设定草案流程">
      {steps.map((step) => (
        <span className={`setup-flow-progress__item setup-flow-progress__item--${step.tone}`} key={step.label}>
          <span className="setup-flow-progress__dot" />
          <span>{step.label}</span>
        </span>
      ))}
    </div>
  );
}

function getFlowSteps(state: ApplyPanelState): Array<{ label: string; tone: FlowStepTone }> {
  switch (state) {
    case 'applied':
      return [
        { label: '讨论完成', tone: 'done' },
        { label: '草案已生成', tone: 'done' },
        { label: '已应用', tone: 'done' },
      ];
    case 'generating':
      return [
        { label: '讨论完成', tone: 'done' },
        { label: '草案生成中', tone: 'active' },
        { label: '待应用', tone: 'idle' },
      ];
    case 'ready':
      return [
        { label: '讨论完成', tone: 'done' },
        { label: '草案已生成', tone: 'done' },
        { label: '待应用', tone: 'active' },
      ];
    default:
      return [
        { label: '讨论中', tone: 'active' },
        { label: '草案待生成', tone: 'idle' },
        { label: '待应用', tone: 'idle' },
      ];
  }
}

function SetupDiscussionPanel({
  actionOptions,
  canGenerate,
  confirmingOptionId,
  error,
  isGenerating,
  isLoading,
  isRunning,
  messages,
  onAdoptSupplement,
  onConfirmAction,
  onGenerateDraft,
  onRejectAction,
  onScopeChange,
  onStarterSelect,
  onSubmit,
  onValueChange,
  rejectingOptionId,
  scope,
  scopes,
  value,
}: {
  actionOptions: DialogueActionOption[];
  canGenerate: boolean;
  confirmingOptionId?: string;
  error?: Error | null;
  isGenerating: boolean;
  isLoading: boolean;
  isRunning: boolean;
  messages: DialogueMessage[];
  onAdoptSupplement: (value: string) => void;
  onConfirmAction: (optionId: string) => void;
  onGenerateDraft: () => void;
  onRejectAction: (optionId: string) => void;
  onScopeChange: (scope: DiscussionScope) => void;
  onStarterSelect: (value: string) => void;
  onSubmit: (event: FormEvent) => void;
  onValueChange: (value: string) => void;
  rejectingOptionId?: string;
  scope: DiscussionScope;
  scopes: Array<{ key: DiscussionScope; label: string }>;
  value: string;
}) {
  const visibleMessages = messages.filter((message) => displayDiscussionContent(message.content));

  return (
    <section className="setup-discussion">
      <div className="setup-discussion__messages">
        {isLoading ? <LoadingState label={copy.discussionLoading} /> : null}
        {!isLoading && visibleMessages.length === 0 ? (
          <div className="setup-discussion__empty">
            <Bot size={18} />
            <div>
              <p>{copy.discussionStarterIntro}</p>
              <ul className="setup-discussion__starter-list">
                {starterPrompts.map((prompt) => (
                  <li key={prompt}>{prompt}</li>
                ))}
              </ul>
              <div className="setup-discussion__starter-actions">
                {starterPrompts.map((prompt, index) => (
                  <button className="setup-discussion__starter-chip" key={prompt} onClick={() => onStarterSelect(prompt)} type="button">
                    回答第 {index + 1} 个
                  </button>
                ))}
                <button className="setup-discussion__starter-chip" onClick={() => onStarterSelect('')} type="button">
                  我想先说点别的
                </button>
              </div>
            </div>
          </div>
        ) : null}
        {!isLoading
          ? visibleMessages.map((message) => {
              const content = displayDiscussionContent(message.content);
              const canAdopt = message.role === 'assistant' && Boolean(content.trim());
              const adoptionOptions = canAdopt ? getSupplementAdoptionOptions(content) : [];
              return (
                <article className={`setup-discussion-message setup-discussion-message--${dialogueRoleClass(message.role)}`} key={message.id}>
                  <div className="setup-discussion-message__meta">
                    {message.role === 'assistant' ? <Bot size={14} /> : null}
                    <span>{dialogueRoleLabel(message.role)}</span>
                    <small>{formatDateTime(message.created_at)}</small>
                  </div>
                  <MarkdownRenderer source={content} variant="compact" />
                  {canAdopt ? (
                    <div className="setup-discussion-message__actions">
                      {adoptionOptions.map((option, optionIndex) => (
                        <button
                          className="setup-discussion-message__adopt"
                          key={`${option.label}-${optionIndex}`}
                          onClick={() => onAdoptSupplement(option.content)}
                          title={option.label}
                          type="button"
                        >
                          <Check size={14} />
                          <span>采纳：{option.label}</span>
                        </button>
                      ))}
                      <button className="setup-discussion-message__adopt" onClick={() => onAdoptSupplement(content)} type="button">
                        <Check size={14} />
                        {adoptionOptions.length > 0 ? copy.supplementAdoptWhole : copy.supplementAdopt}
                      </button>
                    </div>
                  ) : null}
                </article>
              );
            })
          : null}
      </div>

      {actionOptions.length > 0 ? (
        <div className="setup-discussion-actions">
          <div className="setup-panel__section-header">
            <h3>{copy.discussionActionTitle}</h3>
          </div>
          {actionOptions.map((option) => {
            const pending = isPendingDialogueAction(option);
            const confirming = confirmingOptionId === option.id;
            const rejecting = rejectingOptionId === option.id;
            return (
              <article className="setup-discussion-action" key={option.id}>
                <div className="setup-discussion-action__header">
                  <strong>{firstText(option.label, defaultDialogueActionLabel(option.action_type))}</strong>
                  <span className={`status-pill status-pill--${getDialogueActionTone(option.status)}`}>
                    {formatDialogueActionStatus(option.status)}
                  </span>
                </div>
                <p>{firstText(option.description, option.rationale, defaultDialogueActionDescription(option.action_type))}</p>
                {option.error ? <p className="setup-discussion-action__error">{option.error}</p> : null}
                {pending ? (
                  <div className="setup-discussion-action__buttons">
                    <button className="button button--secondary" disabled={confirming || rejecting} onClick={() => onConfirmAction(option.id)} type="button">
                      <CheckCircle2 size={16} />
                      {confirming ? copy.discussionConfirming : copy.discussionConfirmAction}
                    </button>
                    <button className="button button--ghost" disabled={confirming || rejecting} onClick={() => onRejectAction(option.id)} type="button">
                      <XCircle size={16} />
                      {rejecting ? copy.discussionRejecting : copy.discussionRejectAction}
                    </button>
                  </div>
                ) : null}
              </article>
            );
          })}
        </div>
      ) : null}

      {error ? <ErrorState message={error.message} /> : null}

      <form className="setup-discussion__form" onSubmit={onSubmit}>
        <div className="setup-discussion__input-tools">
          <span aria-label={copy.discussionScopeTitle} className="setup-discussion__scope-icon" title={copy.discussionScopeHelp}>
            <Target size={15} />
          </span>
          <div className="setup-discussion__scopes">
            {scopes.map((item) => (
              <button
                className={scope === item.key ? 'setup-discussion__scope setup-discussion__scope--active' : 'setup-discussion__scope'}
                key={item.key}
                onClick={() => onScopeChange(item.key)}
                type="button"
              >
                {item.label}
              </button>
            ))}
          </div>
        </div>
        <textarea
          className="setup-grow-textarea setup-discussion__input"
          onChange={(event) => onValueChange(event.target.value)}
          onKeyDown={submitTextareaOnEnter}
          placeholder={copy.discussionPlaceholder}
          rows={3}
          value={value}
        />
        <div className="setup-composer__footer">
          <div className="setup-composer__hint">
            <MessageCircle size={15} />
            <span>{copy.discussionReady}</span>
          </div>
          <div className="setup-composer__actions">
            <button className="button" disabled={!value.trim() || isRunning} type="submit">
              <Send size={17} />
              {isRunning ? copy.discussionSending : copy.discussionSend}
            </button>
            <button
              className="button button--secondary"
              disabled={!canGenerate || isGenerating}
              onClick={onGenerateDraft}
              title={copy.advanceTooltip}
              type="button"
            >
              <Sparkles size={17} />
              {isGenerating ? copy.advanceRunning : copy.advanceButton}
            </button>
          </div>
        </div>
      </form>
    </section>
  );
}

function SetupSupplementEditor({
  isSaving,
  onCancel,
  onSubmit,
  onValueChange,
  value,
}: {
  isSaving: boolean;
  onCancel: () => void;
  onSubmit: (event: FormEvent) => void;
  onValueChange: (value: string) => void;
  value: string;
}) {
  return (
    <form className="setup-supplement-editor" onSubmit={onSubmit}>
      <div className="setup-panel__section-header">
        <h3>{copy.supplementEditorTitle}</h3>
      </div>
      <textarea
        className="setup-grow-textarea"
        onChange={(event) => onValueChange(event.target.value)}
        placeholder={copy.supplementEditorPlaceholder}
        rows={5}
        value={value}
      />
      <div className="setup-supplement-editor__actions">
        <button className="button button--ghost" disabled={isSaving} onClick={onCancel} type="button">
          {copy.supplementCancel}
        </button>
        <button className="button button--secondary" disabled={!value.trim() || isSaving} type="submit">
          <Check size={16} />
          {isSaving ? copy.supplementSaving : copy.supplementSave}
        </button>
      </div>
    </form>
  );
}

function SetupGenerateConfirmPanel({
  isRunning,
  onCancel,
  onSubmit,
  onValueChange,
  value,
}: {
  isRunning: boolean;
  onCancel: () => void;
  onSubmit: (event: FormEvent) => void;
  onValueChange: (value: string) => void;
  value: string;
}) {
  return (
    <form className="setup-generate-confirm" onSubmit={onSubmit}>
      <div className="setup-panel__section-header">
        <h3>{copy.generatePanelTitle}</h3>
      </div>
      <p>{copy.generatePanelDesc}</p>
      <textarea
        className="setup-grow-textarea"
        onChange={(event) => onValueChange(event.target.value)}
        placeholder={copy.generateNotePlaceholder}
        rows={3}
        value={value}
      />
      <div className="setup-generate-confirm__actions">
        <button className="button button--ghost" disabled={isRunning} onClick={onCancel} type="button">
          {copy.generateCancel}
        </button>
        <button className="button button--secondary" disabled={isRunning} type="submit">
          <Sparkles size={16} />
          {isRunning ? copy.advanceRunning : copy.generateConfirm}
        </button>
      </div>
    </form>
  );
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
    return buildCharacterNameMap(draft);
  }, [draft]);

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
          <MarkdownRenderer className="result-copy" source={draft.assistant_summary} variant="compact" />
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
  const characterNames = useMemo(() => buildCharacterNameMap(draft), [draft]);
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
                  <span>{firstText(card.role, formatCharacterName(card.character_key, characterNames, ''), copy.defaultRole)}</span>
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
                    {formatCharacterName(edge.from_character_key, characterNames, copy.roleA)} / {formatCharacterName(edge.to_character_key, characterNames, copy.roleB)}
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

      {visual.agent_summary ? <MarkdownRenderer className="result-copy" source={visual.agent_summary} variant="compact" /> : null}
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
  const leftName = formatCharacterName(pair?.left_character_id, characterNames, copy.roleA);
  const rightName = formatCharacterName(pair?.right_character_id, characterNames, copy.roleB);

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

function buildCharacterNameMap(draft?: SetupDraft) {
  const names = new Map<string, string>();

  (draft?.characters ?? []).forEach((character, index) => {
    const displayName = firstText(character.name, `${copy.rolePrefix} ${index + 1}`);
    addCharacterNameAlias(names, character.id, displayName);
    addCharacterNameAlias(names, character.name, displayName);
  });

  (draft?.visual_draft?.character_cards ?? []).forEach((card, index) => {
    const displayName = firstText(card.name, `${copy.rolePrefix} ${index + 1}`);
    addCharacterNameAlias(names, card.character_key, displayName);
    addCharacterNameAlias(names, card.name, displayName);
  });

  return names;
}

function addCharacterNameAlias(names: Map<string, string>, alias: string | undefined, displayName: string) {
  const key = alias?.trim();
  if (!key || !displayName.trim()) {
    return;
  }

  names.set(key, displayName);
  names.set(normalizeCharacterIdentifier(key), displayName);
}

function formatCharacterName(identifier: string | undefined, characterNames: Map<string, string>, fallback: string) {
  const key = identifier?.trim();
  if (!key) {
    return fallback;
  }

  return characterNames.get(key) ?? characterNames.get(normalizeCharacterIdentifier(key)) ?? readableCharacterIdentifier(key);
}

function normalizeCharacterIdentifier(value: string) {
  return value
    .trim()
    .toLowerCase()
    .replace(/[\s-]+/g, '_');
}

function readableCharacterIdentifier(value: string) {
  const normalized = normalizeCharacterIdentifier(value);
  const knownNames: Record<string, string> = {
    a_deng: '阿灯',
    lin_wanqing: '林婉清',
    lin_zhou: '林舟',
    qiao_popo: '乔婆婆',
    su_mian: '苏眠',
  };

  if (knownNames[normalized]) {
    return knownNames[normalized];
  }

  if (/[一-鿿]/.test(value)) {
    return value;
  }

  return value
    .replace(/[_-]+/g, ' ')
    .replace(/\b\w/g, (letter) => letter.toUpperCase());
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
              {expanded ? copy.collapseEvents : `${copy.expandEvents} ${events.length} 条`}
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
    return '未命名设定会话';
  }

  const firstLine = seed.split(/\r?\n/)[0]?.trim() ?? seed;
  return truncateText(firstLine, 18);
}

function getSessionSummary(session: SetupSession) {
  const summary = formatSetupSupplementForDisplay(session.last_user_message) || session.seed_idea?.trim();
  return truncateText(summary || '暂无摘要', 58);
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
      return '排队中';
    case 'running':
    case 'loading_state':
      return '生成中';
    case 'review_required':
      return '待审阅';
    case 'succeeded':
      return '已生成';
    case 'retryable':
      return '待重试';
    case 'failed':
      return '失败';
    case 'applied':
      return '已应用';
    case 'committed':
      return '已应用';
    case 'cancelled':
      return '已取消';
    default:
      return '就绪';
  }
}

function getStatusTone(status?: string) {
  switch (status) {
    case 'review_required':
    case 'succeeded':
    case 'applied':
    case 'committed':
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
      return `${count} 项`;
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
    return '创建设定会话';
  }
  if (name.includes('apply') && (name.includes('success') || name.includes('done') || name.includes('finish'))) {
    return '应用草案';
  }
  if (name.includes('apply')) {
    return '准备应用草案';
  }
  if (name.includes('fail') || name.includes('error') || payloadText.includes('failed') || payloadText.includes('error')) {
    return '草案生成失败';
  }
  if (name.includes('success') || name.includes('finish') || name.includes('complete')) {
    return '草案生成完成';
  }
  if (name.includes('queue')) {
    return '草案进入队列';
  }
  if (name.includes('start') || name.includes('run')) {
    return '开始生成草案';
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
    return '刚刚更新';
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
    return '刚刚更新';
  }

  if (diff < hour) {
    return `${Math.floor(diff / minute)} 分钟前`;
  }

  if (diff < day) {
    return `${Math.floor(diff / hour)} 小时前`;
  }

  if (diff < day * 7) {
    return `${Math.floor(diff / day)} 天前`;
  }

  return new Intl.DateTimeFormat('zh-CN', {
    month: 'short',
    day: 'numeric',
  }).format(new Date(timestamp));
}

function formatDateTime(value?: string) {
  if (!value) {
    return '暂无记录';
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
