import { getData, postData } from './http';
import type { CommitStoryRunRequest, Run, StoryRunResult } from '../types/api';

export function getStoryRun(runId: string, signal?: AbortSignal) {
  return getData<Run>(`/story-runs/${runId}`, signal);
}

export function getStoryRunResult(runId: string, signal?: AbortSignal) {
  return getData<StoryRunResult>(`/story-runs/${runId}/result`, signal);
}

export function commitStoryRun(runId: string, body: CommitStoryRunRequest) {
  return postData<unknown, CommitStoryRunRequest>(`/story-runs/${runId}/commit`, body);
}

export function storyRunEventsUrl(runId: string) {
  return `/api/v1/story-runs/${runId}/events`;
}
