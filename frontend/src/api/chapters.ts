// Chapters API 客户端。前端当前只读取章节；章节会在 story run 提交为正史后出现。
import { getData, getPage, pageParams } from './http';
import type { Chapter } from '../types/api';

export function listChapters(projectId: string, page = 1, pageSize = 20, signal?: AbortSignal) {
  return getPage<Chapter>(`/projects/${projectId}/chapters?${pageParams(page, pageSize)}`, signal);
}

export function getChapter(chapterId: string, signal?: AbortSignal) {
  return getData<Chapter>(`/chapters/${chapterId}`, signal);
}
