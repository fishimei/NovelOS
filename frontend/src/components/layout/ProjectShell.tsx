// 项目级外壳。这里统一加载当前项目，渲染项目侧边栏，并通过 outlet context 把项目数据传给子页面。
import { useQuery } from '@tanstack/react-query';
import { Navigate, Outlet, useParams } from 'react-router-dom';

import { getProject } from '../../api/projects';
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
        <div className="project-sidebar__title">
          {projectQuery.isLoading ? '读取项目' : projectQuery.data?.title ?? '未命名项目'}
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
