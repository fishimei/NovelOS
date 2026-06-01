import { getData, postData } from './http';
import type { CutChapterRequest, CutChapterResult, Run, RunEvent, StoryRunResult } from '../types/api';

export function getStoryRun(runId: string, signal?: AbortSignal) {
  return getData<Run>(`/story-runs/${runId}`, signal);
}

export function getStoryRunResult(runId: string, signal?: AbortSignal) {
  return getData<StoryRunResult>(`/story-runs/${runId}/result`, signal);
}

export function listStoryRunEventHistory(runId: string, signal?: AbortSignal) {
  return getData<RunEvent[]>(`/story-runs/${runId}/event-history`, signal);
}

export function cutStoryChapter(runId: string, body: CutChapterRequest) {
  return postData<CutChapterResult, CutChapterRequest>(`/story-runs/${runId}/cut-chapter`, body);
}

export function storyRunEventsUrl(runId: string) {
  return `/api/v1/story-runs/${runId}/events`;
}
