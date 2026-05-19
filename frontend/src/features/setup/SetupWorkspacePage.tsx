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
import { advanceSetupSession, applySetupRun, createSetupSession, listSetupSessions } from '../../api/setupSessions';
import { EmptyState } from '../../components/feedback/EmptyState';
import { ErrorState } from '../../components/feedback/ErrorState';
import { LoadingState } from '../../components/feedback/LoadingState';
import { MarkdownRenderer } from '../../components/MarkdownRenderer';
import type {
  DialogueActionOption,
  DialogueMessage,
  DialogueSession,
  Relationship,
  Run,
  RunEvent,
  SetupDraft,
  SetupSession,
  WorldStateEntry,
} from '../../types/api';
import { submitTextareaOnEnter } from '../../utils/keyboard';

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
  stepCompose: '\u8bbe\u5b9a\u8ba8\u8bba\u533a',
  stepComposeDesc: '\u5148\u548c AI \u8f7b\u91cf\u8ba8\u8bba\u3001\u6f84\u6e05\u65b9\u5411\uff0c\u518d\u663e\u5f0f\u751f\u6210\u6216\u66f4\u65b0\u8349\u6848\u3002',
  stepDraft: '\u8bbe\u5b9a\u8349\u6848',
  stepDraftDesc: '\u5148\u770b\u6a21\u5757\u9aa8\u67b6\uff0c\u751f\u6210\u540e\u518d\u9010\u4e2a\u5ba1\u9605\u5e76\u51b3\u5b9a\u662f\u5426\u5e94\u7528\u3002',
  sessionSeedTitle: '\u79cd\u5b50\u6784\u60f3',
  lastSupplementTitle: '\u6700\u8fd1\u8865\u5145',
  noSupplement: '\u8fd8\u6ca1\u6709\u989d\u5916\u8865\u5145\u3002',
  emptySeed: '\u8be5\u4f1a\u8bdd\u8fd8\u6ca1\u6709\u5199\u5165\u79cd\u5b50\u5185\u5bb9\u3002',
  advanceButton: '\u751f\u6210 / \u66f4\u65b0\u8349\u6848',
  advanceRunning: '\u751f\u6210\u4e2d...',
  advanceTooltip: '\u5c06\u57fa\u4e8e\u5f53\u524d\u8ba8\u8bba\u4e0a\u4e0b\u6587\u751f\u6210\u8349\u6848\uff0c\u9884\u8ba1 20-40 \u79d2\u3002',
  generatePanelTitle: '\u751f\u6210\u524d\u8865\u5145\u8bf4\u660e\uff08\u53ef\u9009\uff09',
  generatePanelDesc: '\u4e0d\u9700\u8981\u91cd\u65b0\u603b\u7ed3\u8ba8\u8bba\u3002\u53ea\u586b\u8fd9\u6b21\u751f\u6210\u9700\u8981\u989d\u5916\u9075\u5faa\u7684\u4e00\u53e5\u8bdd\u3002',
  generateNotePlaceholder: '\u4f8b\u5982\uff1a\u8fd9\u6b21\u53ea\u66f4\u65b0\u4e16\u754c\u72b6\u6001\uff0c\u4e0d\u6539\u89d2\u8272\u5173\u7cfb',
  generateConfirm: '\u751f\u6210\u8349\u6848',
  generateCancel: '\u53d6\u6d88',
  discussionScopeTitle: '\u4f5c\u7528\u57df',
  discussionScopeHelp: '\u4f5c\u7528\u57df\u53ea\u7ea6\u675f AI \u5199\u5165\u8349\u6848\u7684\u8303\u56f4\uff0c\u4e0d\u9650\u5236\u8ba8\u8bba\u8bdd\u9898\u3002',
  discussionStarterIntro: 'AI \u5df2\u8bfb\u53d6\u4f60\u7684\u79cd\u5b50\u6784\u60f3\uff0c\u60f3\u5148\u548c\u4f60\u786e\u8ba4\u51e0\u4ef6\u4e8b\uff1a',
  discussionPlaceholder: '\u548c AI \u8ba8\u8bba\u8bbe\u5b9a\u65b9\u5411\uff0c\u4f8b\u5982\u201c\u90ae\u5dee\u8eab\u4efd\u80fd\u4e0d\u80fd\u5e26\u70b9\u8bc5\u5492\u611f\uff1f\u201d',
  discussionSend: '\u53d1\u9001',
  discussionSending: '\u8ba8\u8bba\u4e2d...',
  discussionReady: '\u666e\u901a\u8ba8\u8bba\u4e0d\u4f1a\u89e6\u53d1\u8349\u6848\u751f\u6210\u3002',
  discussionCreateHint: '\u9996\u6b21\u53d1\u9001\u65f6\u4f1a\u4e3a\u5f53\u524d\u8bbe\u5b9a\u4f1a\u8bdd\u521b\u5efa\u4e00\u6761\u8ba8\u8bba\u7ebf\u3002',
  discussionLoading: '\u6b63\u5728\u52a0\u8f7d\u8ba8\u8bba\u6d88\u606f',
  discussionActionTitle: '\u5f85\u786e\u8ba4\u64cd\u4f5c',
  discussionConfirmAction: '\u786e\u8ba4\u6267\u884c',
  discussionRejectAction: '\u62d2\u7edd',
  discussionConfirming: '\u6267\u884c\u4e2d...',
  discussionRejecting: '\u62d2\u7edd\u4e2d...',
  draftRequestTitle: '\u8349\u6848\u751f\u6210\u8f93\u5165',
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
  scopePending: '\u6682\u65e0\u8349\u6848',
  scopeDiscussing: '\u8ba8\u8bba\u4e2d \u00b7 \u6682\u65e0\u8349\u6848',
  scopeGenerating: '\u751f\u6210\u4e2d',
  scopeReady: '\u5f85\u5e94\u7528',
  scopeApplied: '\u5df2\u5e94\u7528',
  noteTitle: '\u5e94\u7528\u5907\u6ce8\uff08\u53ef\u9009\uff09',
  authorNotePlaceholder: '\u8bb0\u5f55\u8fd9\u6b21\u5e94\u7528\u7684\u51b3\u7b56\u3001\u53d6\u820d\u6216\u540e\u7eed\u5f85\u8c03\u6574\u70b9',
  applyButton: '\u5e94\u7528\u5230\u9879\u76ee',
  applyDisabled: '\u8bf7\u5148\u751f\u6210\u8349\u6848',
  applyDisabledHint: '\u751f\u6210\u8349\u6848\u540e\u53ef\u9010\u6a21\u5757\u5e94\u7528\u3002',
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
type DiscussionScope = 'all' | 'author_bible' | 'characters' | 'relationships' | 'world';
type ApplyPanelState = 'empty' | 'discussing' | 'generating' | 'ready' | 'applied';
type FlowStepTone = 'idle' | 'active' | 'done';

const discussionScopes: Array<{ key: DiscussionScope; label: string }> = [
  { key: 'all', label: '\u5168\u90e8' },
  { key: 'author_bible', label: '\u4f5c\u8005\u5723\u7ecf' },
  { key: 'characters', label: '\u89d2\u8272' },
  { key: 'relationships', label: '\u5173\u7cfb' },
  { key: 'world', label: '\u4e16\u754c' },
];

const starterPrompts = [
  '\u4e3b\u89d2\u6700\u60f3\u9003\u907f\u7684\u8fc7\u5f80\u6216\u5fc3\u7ed3\u5177\u4f53\u662f\u4ec0\u4e48\uff1f',
  '\u8fd9\u4e2a\u4e16\u754c\u91cc\u6700\u91cd\u8981\u7684\u89c4\u5219\u6216\u4ee3\u4ef7\u662f\u4ec0\u4e48\uff1f',
  '\u89d2\u8272\u5173\u7cfb\u60f3\u5148\u4ece\u4fe1\u4efb\u3001\u4e8f\u6b20\u8fd8\u662f\u51b2\u7a81\u5f00\u59cb\uff1f',
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

  const sessionsQuery = useQuery({
    queryKey: ['setupSessions', projectId, 1, 20],
    queryFn: ({ signal }) => listSetupSessions(projectId, 1, 20, signal),
    enabled: Boolean(projectId),
  });

  const sessions = sessionsQuery.data?.data ?? [];
  const selectedSessionId = activeSessionId || sessions[0]?.id || '';
  const currentSession = sessions.find((session) => session.id === selectedSessionId) ?? sessions[0] ?? null;

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
    mutationFn: (optionId: string) => rejectDialogueActionOption(optionId, { reason: '\u4f5c\u8005\u5728\u8bbe\u5b9a\u8ba8\u8bba\u533a\u62d2\u7edd\u4e86\u8fd9\u4e2a\u64cd\u4f5c\u3002' }),
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

  const openGeneratePanel = () => {
    if (!selectedSessionId || advanceMutation.isPending || isRunActive) {
      return;
    }
    setGeneratePanelOpen(true);
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
      '\u8bf7\u91cd\u65b0\u751f\u6210\u8fd9\u6b21\u8bbe\u5b9a\u8349\u6848\uff0c\u5e76\u5728\u4e0d\u6539\u53d8\u6838\u5fc3\u79cd\u5b50\u7684\u524d\u63d0\u4e0b\u7f29\u5c0f\u8303\u56f4\uff0c\u8865\u5145\u786c\u7ea6\u675f\u4e0e\u4e16\u754c\u89c4\u5219\u3002',
    );
    setGeneratePanelOpen(true);
  };

  const regenerateDraft = () => {
    if (!selectedSessionId || advanceMutation.isPending || isRunActive) {
      return;
    }
    startDraftGeneration('\u8bf7\u57fa\u4e8e\u5f53\u524d\u4f1a\u8bdd\u91cd\u65b0\u8d77\u8349\u4e00\u7248\uff0c\u4e0d\u8981\u76f4\u63a5\u6cbf\u7528\u4e0a\u4e00\u7248\uff1b\u65b9\u5411\u66f4\u5927\u80c6\u3001\u66f4\u5929\u9a6c\u884c\u7a7a\uff0c\u540c\u65f6\u4fdd\u7559\u53ef\u843d\u5730\u7684\u4eba\u7269\u52a8\u673a\u548c\u5173\u7cfb\u5f20\u529b\u3002');
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
                return (
                  <button
                    className={selectedSessionId === session.id ? 'session-item session-item--active' : 'session-item'}
                    key={session.id}
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
}: {
  activeRunId: string;
  message: string;
  scope: DiscussionScope;
  session: SetupSession | null;
}) {
  const setupRunId = firstText(activeRunId, session?.latest_run_id);
  return [
    '__setup_discussion_context__',
    `scope=${scope}`,
    `setup_session_id=${session?.id ?? ''}`,
    `setup_run_id=${setupRunId}`,
    `seed_idea=${session?.seed_idea ?? ''}`,
    `last_setup_supplement=${session?.last_user_message ?? ''}`,
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
}: {
  activeRunId: string;
  discussionMessages: DialogueMessage[];
  optionalInstruction: string;
  scope: DiscussionScope;
  session: SetupSession | null;
}) {
  const setupRunId = firstText(activeRunId, session?.latest_run_id);
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

  return [
    '__setup_draft_generation_context__',
    `scope=${scope}`,
    `setup_session_id=${session?.id ?? ''}`,
    `setup_run_id=${setupRunId}`,
    `seed_idea=${session?.seed_idea ?? ''}`,
    `last_setup_supplement=${session?.last_user_message ?? ''}`,
    'discussion_context:',
    discussionContext || '\u6682\u65e0\u8ba8\u8bba\u6d88\u606f\uff0c\u8bf7\u57fa\u4e8e\u79cd\u5b50\u6784\u60f3\u8d77\u8349\u3002',
    optionalInstruction.trim() ? `optional_instruction=${optionalInstruction.trim()}` : '',
    'instruction=\u8bf7\u57fa\u4e8e\u79cd\u5b50\u6784\u60f3\u548c\u5f53\u524d\u8bbe\u5b9a\u8ba8\u8bba\u4e0a\u4e0b\u6587\u751f\u6210\u7ed3\u6784\u5316\u8bbe\u5b9a\u8349\u6848\uff1b\u4e0d\u8981\u8981\u6c42\u4f5c\u8005\u91cd\u590d\u8f93\u5165\u5df2\u7ecf\u8ba8\u8bba\u8fc7\u7684\u4fe1\u606f\u3002',
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
      return '\u5f85\u786e\u8ba4';
    case 'confirmed':
      return '\u5df2\u786e\u8ba4';
    case 'executing':
      return '\u6267\u884c\u4e2d';
    case 'executed':
      return '\u5df2\u6267\u884c';
    case 'rejected':
      return '\u5df2\u62d2\u7edd';
    case 'failed':
      return '\u5931\u8d25';
    default:
      return '\u672a\u77e5';
  }
}

function defaultDialogueActionLabel(actionType?: string) {
  switch (actionType) {
    case 'setup.start_and_advance':
      return '\u751f\u6210\u9879\u76ee\u8bbe\u5b9a\u8349\u6848';
    case 'setup.advance':
      return '\u66f4\u65b0\u8bbe\u5b9a\u8349\u6848';
    case 'setup.apply':
      return '\u5e94\u7528\u8bbe\u5b9a\u8349\u6848';
    case 'story.create_and_advance':
      return '\u5f00\u59cb\u5267\u60c5\u7f16\u6392';
    case 'story.advance':
      return '\u7ee7\u7eed\u5267\u60c5\u7f16\u6392';
    case 'story.commit':
      return '\u63d0\u4ea4\u7ae0\u8282\u8349\u7a3f';
    default:
      return '\u6267\u884c\u4e0b\u4e00\u6b65';
  }
}

function defaultDialogueActionDescription(actionType?: string) {
  switch (actionType) {
    case 'setup.start_and_advance':
    case 'setup.advance':
      return '\u786e\u8ba4\u540e\u4f1a\u89e6\u53d1 setup run\uff0c\u751f\u6210\u4e00\u7248\u65b0\u7684\u7ed3\u6784\u5316\u8349\u6848\u3002';
    case 'setup.apply':
      return '\u786e\u8ba4\u540e\u4f1a\u628a\u5df2\u5ba1\u6838\u7684\u8349\u6848\u5199\u5165\u9879\u76ee\u72b6\u6001\u3002';
    default:
      return '\u786e\u8ba4\u540e\u6267\u884c\u8fd9\u4e2a\u5f85\u786e\u8ba4\u64cd\u4f5c\u3002';
  }
}

function dialogueRoleLabel(role: string) {
  switch (role) {
    case 'assistant':
      return 'AI';
    case 'tool':
      return '\u7cfb\u7edf';
    case 'system':
      return '\u4e0a\u4e0b\u6587';
    default:
      return '\u6211';
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
      return '\u6b63\u5728\u751f\u6210\u8349\u6848\uff0c\u5b8c\u6210\u540e\u53ef\u9010\u6a21\u5757\u5e94\u7528\u3002';
    case 'ready':
      return '\u9009\u62e9\u8981\u5199\u5165\u9879\u76ee\u7684\u6a21\u5757\u3002';
    case 'applied':
      return '\u8fd9\u7248\u8349\u6848\u5df2\u5e94\u7528\uff0c\u53ef\u7ee7\u7eed\u8ba8\u8bba\u540e\u66f4\u65b0\u8349\u6848\u3002';
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
  const draftLabel = `${draftCount} \u8349\u6848${applied && draftCount > 0 ? '\u5df2\u5e94\u7528' : ''}`;
  return `${formatRelativeTime(updatedAt)} \u00b7 ${discussionCount} \u8ba8\u8bba \u00b7 ${draftLabel}`;
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
        { label: '\u8ba8\u8bba\u5b8c\u6210', tone: 'done' },
        { label: '\u8349\u6848\u5df2\u751f\u6210', tone: 'done' },
        { label: '\u5df2\u5e94\u7528', tone: 'done' },
      ];
    case 'generating':
      return [
        { label: '\u8ba8\u8bba\u5b8c\u6210', tone: 'done' },
        { label: '\u8349\u6848\u751f\u6210\u4e2d', tone: 'active' },
        { label: '\u5f85\u5e94\u7528', tone: 'idle' },
      ];
    case 'ready':
      return [
        { label: '\u8ba8\u8bba\u5b8c\u6210', tone: 'done' },
        { label: '\u8349\u6848\u5df2\u751f\u6210', tone: 'done' },
        { label: '\u5f85\u5e94\u7528', tone: 'active' },
      ];
    default:
      return [
        { label: '\u8ba8\u8bba\u4e2d', tone: 'active' },
        { label: '\u8349\u6848\u5f85\u751f\u6210', tone: 'idle' },
        { label: '\u5f85\u5e94\u7528', tone: 'idle' },
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
          ? visibleMessages.map((message) => (
              <article className={`setup-discussion-message setup-discussion-message--${dialogueRoleClass(message.role)}`} key={message.id}>
                <div className="setup-discussion-message__meta">
                  {message.role === 'assistant' ? <Bot size={14} /> : null}
                  <span>{dialogueRoleLabel(message.role)}</span>
                  <small>{formatDateTime(message.created_at)}</small>
                </div>
                <MarkdownRenderer source={displayDiscussionContent(message.content)} variant="compact" />
              </article>
            ))
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
    case 'committed':
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
