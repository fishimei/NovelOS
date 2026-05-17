// 纯本地最近项目记录。当前后端没有项目列表接口，所以首页用 localStorage 记住最近打开的项目。
import { useCallback, useEffect, useState } from 'react';

import type { Project } from '../types/api';

type RecentProject = Pick<Project, 'id' | 'title' | 'genre'> & {
  opened_at: string;
};

const STORAGE_KEY = 'novelos.recentProjects';

function readRecentProjects(): RecentProject[] {
  const raw = localStorage.getItem(STORAGE_KEY);

  if (!raw) {
    return [];
  }

  try {
    const parsed = JSON.parse(raw) as Partial<RecentProject>[];
    const seen = new Set<string>();

    return parsed.filter((project): project is RecentProject => {
      if (!isValidProjectId(project.id) || seen.has(project.id)) {
        return false;
      }
      seen.add(project.id);
      return true;
    });
  } catch {
    return [];
  }
}

function writeRecentProjects(projects: RecentProject[]) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(projects.slice(0, 8)));
}

export function useRecentProjects() {
  const [recentProjects, setRecentProjects] = useState<RecentProject[]>([]);

  useEffect(() => {
    const projects = readRecentProjects();
    writeRecentProjects(projects);
    setRecentProjects(projects);
  }, []);

  const rememberProject = useCallback((project: Pick<Project, 'id' | 'title' | 'genre'>) => {
    if (!isValidProjectId(project.id)) {
      return;
    }

    setRecentProjects((current) => {
      const next = [
        {
          id: project.id,
          title: project.title,
          genre: project.genre,
          opened_at: new Date().toISOString(),
        },
        ...current.filter((item) => item.id !== project.id),
      ].slice(0, 8);

      writeRecentProjects(next);
      return next;
    });
  }, []);

  return { recentProjects, rememberProject };
}

function isValidProjectId(id: unknown): id is string {
  if (typeof id !== 'string') {
    return false;
  }
  const normalized = id.trim();
  return normalized !== '' && normalized !== 'undefined' && normalized !== 'null';
}
