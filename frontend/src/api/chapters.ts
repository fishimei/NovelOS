import { getData, getPage, pageParams } from './http';
import type { Chapter } from '../types/api';

export function listChapters(projectId: string, page = 1, pageSize = 20, signal?: AbortSignal) {
  return getPage<Chapter>(`/projects/${projectId}/chapters?${pageParams(page, pageSize)}`, signal);
}

export function getChapter(chapterId: string, signal?: AbortSignal) {
  return getData<Chapter>(`/chapters/${chapterId}`, signal);
}
