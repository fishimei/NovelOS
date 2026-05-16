// StoryRun API 客户端。它负责读取 run 状态和结果、把接受的候选结果提交为正史，并提供 SSE 事件地址。
import { getData, postData } from './http';
import type { CommitStoryRunRequest, CommitStoryRunResult, Run, StoryRunResult } from '../types/api';

export function getStoryRun(runId: string, signal?: AbortSignal) {
  return getData<Run>(`/story-runs/${runId}`, signal);
}

export function getStoryRunResult(runId: string, signal?: AbortSignal) {
  return getData<StoryRunResult>(`/story-runs/${runId}/result`, signal);
}

export function commitStoryRun(runId: string, body: CommitStoryRunRequest) {
  return postData<CommitStoryRunResult, CommitStoryRunRequest>(`/story-runs/${runId}/commit`, body);
}

export function storyRunEventsUrl(runId: string) {
  return `/api/v1/story-runs/${runId}/events`;
}
