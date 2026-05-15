import { getData } from './http';
import type { Run, SetupRunResult } from '../types/api';

export function getSetupRun(runId: string, signal?: AbortSignal) {
  return getData<Run>(`/setup-runs/${runId}`, signal);
}

export function getSetupRunResult(runId: string, signal?: AbortSignal) {
  return getData<SetupRunResult>(`/setup-runs/${runId}/result`, signal);
}
