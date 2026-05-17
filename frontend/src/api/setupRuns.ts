// SetupRun API 客户端。run 表示设定会话中的异步 AI 任务，result 接口暴露生成出的结构化草稿。
import { getData } from './http';
import type { Run, RunEvent, SetupRunResult } from '../types/api';

export function getSetupRun(runId: string, signal?: AbortSignal) {
  return getData<Run>(`/setup-runs/${runId}`, signal);
}

export function getSetupRunResult(runId: string, signal?: AbortSignal) {
  return getData<SetupRunResult>(`/setup-runs/${runId}/result`, signal);
}

export function listSetupRunEventHistory(runId: string, signal?: AbortSignal) {
  return getData<RunEvent[]>(`/setup-runs/${runId}/event-history`, signal);
}
