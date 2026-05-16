// AuthorBible API 客户端：读取和保存项目级正式写作规则、风格指南、约束和初始世界状态。
import { getData, putData } from './http';
import type { AuthorBible, UpdateAuthorBibleRequest } from '../types/api';

export function getAuthorBible(projectId: string, signal?: AbortSignal) {
  return getData<AuthorBible>(`/projects/${projectId}/author-bible`, signal);
}

export function updateAuthorBible(projectId: string, body: UpdateAuthorBibleRequest) {
  return putData<AuthorBible, UpdateAuthorBibleRequest>(`/projects/${projectId}/author-bible`, body);
}
