// DialogueSession API 客户端。统一对话会话承载轻量多轮讨论，并通过 action option 触发正式任务。
import { getData, getPage, pageParams, postData } from './http';
import type {
  AdvanceDialogueSessionRequest,
  ConfirmDialogueActionOptionRequest,
  CreateDialogueSessionRequest,
  DialogueActionOption,
  DialogueRunResult,
  DialogueSession,
  RejectDialogueActionOptionRequest,
  Run,
} from '../types/api';

export function createDialogueSession(projectId: string, body: CreateDialogueSessionRequest) {
  return postData<DialogueSession, CreateDialogueSessionRequest>(`/projects/${projectId}/dialogue-sessions`, body);
}

export function listDialogueSessions(projectId: string, page = 1, pageSize = 20, signal?: AbortSignal) {
  return getPage<DialogueSession>(`/projects/${projectId}/dialogue-sessions?${pageParams(page, pageSize)}`, signal);
}

export function getDialogueSession(sessionId: string, signal?: AbortSignal) {
  return getData<DialogueSession>(`/dialogue-sessions/${sessionId}`, signal);
}

export function advanceDialogueSession(sessionId: string, body: AdvanceDialogueSessionRequest) {
  return postData<Run, AdvanceDialogueSessionRequest>(`/dialogue-sessions/${sessionId}/advance`, body);
}

export function getDialogueRun(runId: string, signal?: AbortSignal) {
  return getData<Run>(`/dialogue-runs/${runId}`, signal);
}

export function getDialogueRunResult(runId: string, signal?: AbortSignal) {
  return getData<DialogueRunResult>(`/dialogue-runs/${runId}/result`, signal);
}

export function confirmDialogueActionOption(optionId: string, body: ConfirmDialogueActionOptionRequest) {
  return postData<DialogueActionOption, ConfirmDialogueActionOptionRequest>(`/dialogue-action-options/${optionId}/confirm`, body);
}

export function rejectDialogueActionOption(optionId: string, body: RejectDialogueActionOptionRequest) {
  return postData<DialogueActionOption, RejectDialogueActionOptionRequest>(`/dialogue-action-options/${optionId}/reject`, body);
}
