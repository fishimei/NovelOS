import { getData, postData, putData } from './http';
import type { CreateProjectRequest, Project, UpdateProjectRequest } from '../types/api';

export function createProject(body: CreateProjectRequest) {
  return postData<Project, CreateProjectRequest>('/projects', body);
}

export function getProject(projectId: string, signal?: AbortSignal) {
  return getData<Project>(`/projects/${projectId}`, signal);
}

export function updateProject(projectId: string, body: UpdateProjectRequest) {
  return putData<Project, UpdateProjectRequest>(`/projects/${projectId}`, body);
}
