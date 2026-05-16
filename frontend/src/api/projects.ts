// Projects API 客户端：创建、读取和更新项目。首页和项目工作台外壳都会依赖这些数据。
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
