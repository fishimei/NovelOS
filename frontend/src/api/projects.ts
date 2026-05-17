// Projects API 客户端：创建、读取和更新项目。首页和项目工作台外壳都会依赖这些数据。
import { getData, postData, putData } from './http';
import type { CreateProjectRequest, Project, UpdateProjectRequest } from '../types/api';

type ProjectPayload = Project & {
  ID?: string;
  Title?: string;
  Genre?: string;
  Description?: string;
  character_count?: number;
  chapter_count?: number;
  last_committed_chapter_number?: number;
  CharacterCount?: number;
  ChapterCount?: number;
  LastCommittedChapterNumber?: number;
};

export async function createProject(body: CreateProjectRequest) {
  return normalizeProject(await postData<ProjectPayload, CreateProjectRequest>('/projects', body));
}

export async function getProject(projectId: string, signal?: AbortSignal) {
  return normalizeProject(await getData<ProjectPayload>(`/projects/${projectId}`, signal));
}

export async function updateProject(projectId: string, body: UpdateProjectRequest) {
  return normalizeProject(await putData<ProjectPayload, UpdateProjectRequest>(`/projects/${projectId}`, body));
}

function normalizeProject(project: ProjectPayload): Project {
  const lastCommittedChapterNumber =
    project.stats?.last_committed_chapter_number ?? project.last_committed_chapter_number ?? project.LastCommittedChapterNumber;

  return {
    ...project,
    id: project.id ?? project.ID ?? '',
    title: project.title ?? project.Title ?? '',
    genre: project.genre ?? project.Genre,
    description: project.description ?? project.Description,
    stats: {
      character_count: project.stats?.character_count ?? project.character_count ?? project.CharacterCount,
      chapter_count: project.stats?.chapter_count ?? project.chapter_count ?? project.ChapterCount,
      last_committed_chapter_number: lastCommittedChapterNumber,
    },
  };
}
