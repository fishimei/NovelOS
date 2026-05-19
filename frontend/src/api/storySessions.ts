// StorySession API 客户端。故事会话接收作者推进语，并创建用于生成候选正文的 story run。
import { deleteData, getData, getPage, pageParams, postData, putData } from './http';
import type {
  AdvanceStorySessionRequest,
  CreateStorySessionRequest,
  Run,
  StorySession,
  UpdateStorySessionRequest,
} from '../types/api';

export function createStorySession(projectId: string, body: CreateStorySessionRequest) {
  return postData<StorySession, CreateStorySessionRequest>(`/projects/${projectId}/story-sessions`, body);
}

export function listStorySessions(projectId: string, page = 1, pageSize = 20, signal?: AbortSignal) {
  return getPage<StorySession>(`/projects/${projectId}/story-sessions?${pageParams(page, pageSize)}`, signal);
}

export function getStorySession(sessionId: string, signal?: AbortSignal) {
  return getData<StorySession>(`/story-sessions/${sessionId}`, signal);
}

export function updateStorySession(sessionId: string, body: UpdateStorySessionRequest) {
  return putData<StorySession, UpdateStorySessionRequest>(`/story-sessions/${sessionId}`, body);
}

export function deleteStorySession(sessionId: string) {
  return deleteData<{ deleted: boolean }>(`/story-sessions/${sessionId}`);
}

export function advanceStorySession(sessionId: string, body: AdvanceStorySessionRequest) {
  return postData<Run, AdvanceStorySessionRequest>(`/story-sessions/${sessionId}/advance`, body);
}
