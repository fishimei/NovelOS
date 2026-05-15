import { getData, putData } from './http';
import type { AuthorBible, UpdateAuthorBibleRequest } from '../types/api';

export function getAuthorBible(projectId: string, signal?: AbortSignal) {
  return getData<AuthorBible>(`/projects/${projectId}/author-bible`, signal);
}

export function updateAuthorBible(projectId: string, body: UpdateAuthorBibleRequest) {
  return putData<AuthorBible, UpdateAuthorBibleRequest>(`/projects/${projectId}/author-bible`, body);
}
