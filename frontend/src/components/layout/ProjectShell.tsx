import { useQuery } from '@tanstack/react-query';
import { Navigate, Outlet, useParams } from 'react-router-dom';

import { getProject } from '../../api/projects';
import { formatRelativeTime } from '../../utils/format';
import { ErrorState } from '../feedback/ErrorState';
import { LoadingState } from '../feedback/LoadingState';
import { ProjectNav } from './ProjectNav';

export function ProjectShell() {
  const { projectId = '' } = useParams();
  const hasValidProjectId = isValidProjectId(projectId);
  const projectQuery = useQuery({
    queryKey: ['project', projectId],
    queryFn: ({ signal }) => getProject(projectId, signal),
    enabled: hasValidProjectId,
  });

  if (!hasValidProjectId) {
    return <Navigate replace to="/" />;
  }

  return (
    <div className="project-shell">
      <aside className="project-sidebar">
        <div className="project-sidebar__meta">
          <small>当前项目</small>
          <div className="project-sidebar__title">
            {projectQuery.isLoading ? '加载项目中' : projectQuery.data?.title ?? '未命名项目'}
          </div>
          <p className="project-sidebar__subtitle">
            {projectQuery.data?.genre || '小说工程'}
            {' · '}
            最近编辑 {formatRelativeTime(projectQuery.data?.updated_at ?? projectQuery.data?.created_at)}
          </p>
        </div>
        <ProjectNav />
      </aside>
      <main className="project-main">
        {projectQuery.isLoading ? <LoadingState /> : null}
        {projectQuery.isError ? <ErrorState message={(projectQuery.error as Error).message} /> : null}
        <Outlet context={{ project: projectQuery.data }} />
      </main>
    </div>
  );
}

function isValidProjectId(projectId: string) {
  const normalized = projectId.trim();
  return normalized !== '' && normalized !== 'undefined' && normalized !== 'null';
}
