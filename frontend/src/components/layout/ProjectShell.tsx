import { useQuery } from '@tanstack/react-query';
import { Outlet, useParams } from 'react-router-dom';

import { getProject } from '../../api/projects';
import { ErrorState } from '../feedback/ErrorState';
import { LoadingState } from '../feedback/LoadingState';
import { ProjectNav } from './ProjectNav';

export function ProjectShell() {
  const { projectId = '' } = useParams();
  const projectQuery = useQuery({
    queryKey: ['project', projectId],
    queryFn: ({ signal }) => getProject(projectId, signal),
    enabled: Boolean(projectId),
  });

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
