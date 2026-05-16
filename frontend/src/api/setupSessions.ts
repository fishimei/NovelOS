// SetupSession API 客户端。设定会话把种子想法和作者补充信息推进成可审校、可应用的候选设定草稿。
import { getData, getPage, pageParams, postData } from './http';
import type {
  AdvanceSetupSessionRequest,
  ApplySetupRunResult,
  ApplySetupRunRequest,
  CreateSetupSessionRequest,
  Run,
  SetupSession,
} from '../types/api';

export function createSetupSession(projectId: string, body: CreateSetupSessionRequest) {
  return postData<SetupSession, CreateSetupSessionRequest>(`/projects/${projectId}/setup-sessions`, body);
}

export function listSetupSessions(projectId: string, page = 1, pageSize = 20, signal?: AbortSignal) {
  return getPage<SetupSession>(`/projects/${projectId}/setup-sessions?${pageParams(page, pageSize)}`, signal);
}

export function getSetupSession(sessionId: string, signal?: AbortSignal) {
  return getData<SetupSession>(`/setup-sessions/${sessionId}`, signal);
}

export function advanceSetupSession(sessionId: string, body: AdvanceSetupSessionRequest) {
  return postData<Run, AdvanceSetupSessionRequest>(`/setup-sessions/${sessionId}/advance`, body);
}

export function applySetupRun(sessionId: string, body: ApplySetupRunRequest) {
  return postData<ApplySetupRunResult, ApplySetupRunRequest>(`/setup-sessions/${sessionId}/apply`, body);
}
