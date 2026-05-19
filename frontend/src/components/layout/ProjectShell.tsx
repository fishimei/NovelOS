import { useQuery } from '@tanstack/react-query';
import { ChevronLeft, ChevronRight } from 'lucide-react';
import { useEffect, useState } from 'react';
import { Navigate, Outlet, useParams } from 'react-router-dom';

import { getProject } from '../../api/projects';
import { formatRelativeTime } from '../../utils/format';
import { ErrorState } from '../feedback/ErrorState';
import { LoadingState } from '../feedback/LoadingState';
import { ProjectNav } from './ProjectNav';

type SidebarMode = 'expanded' | 'collapsed';

export function ProjectShell() {
  const { projectId = '' } = useParams();
  const [sidebarMode, setSidebarMode] = useState<SidebarMode>(() => readSidebarMode());
  const [mobileNavOpen, setMobileNavOpen] = useState(false);
  const [isCompact, setIsCompact] = useState(() => isCompactViewport());
  const hasValidProjectId = isValidProjectId(projectId);
  const projectQuery = useQuery({
    queryKey: ['project', projectId],
    queryFn: ({ signal }) => getProject(projectId, signal),
    enabled: hasValidProjectId,
  });

  useEffect(() => {
    window.localStorage.setItem('novelos.projectSidebarMode', sidebarMode);
  }, [sidebarMode]);

  useEffect(() => {
    const mediaQuery = window.matchMedia('(max-width: 960px)');
    const syncViewport = () => setIsCompact(mediaQuery.matches);
    syncViewport();
    mediaQuery.addEventListener('change', syncViewport);
    return () => mediaQuery.removeEventListener('change', syncViewport);
  }, []);

  if (!hasValidProjectId) {
    return <Navigate replace to="/" />;
  }

  if (projectQuery.isLoading) {
    return (
      <main className="project-main">
        <LoadingState />
      </main>
    );
  }

  if (projectQuery.isError) {
    return (
      <main className="project-main">
        <ErrorState message={(projectQuery.error as Error).message} />
      </main>
    );
  }

  if (!projectQuery.data) {
    return (
      <main className="project-main">
        <ErrorState title="项目加载失败" message="未获取到项目数据" />
      </main>
    );
  }

  const sidebarCollapsed = sidebarMode === 'collapsed';
  const sidebarClassName = [
    'project-sidebar',
    sidebarCollapsed ? 'project-sidebar--collapsed' : '',
    mobileNavOpen ? 'project-sidebar--mobile-open' : '',
  ]
    .filter(Boolean)
    .join(' ');
  const shellClassName = [
    'project-shell',
    `project-shell--sidebar-${sidebarMode}`,
    mobileNavOpen ? 'project-shell--mobile-nav-open' : '',
  ]
    .filter(Boolean)
    .join(' ');
  const toggleLabel = getSidebarToggleLabel(sidebarMode, isCompact, mobileNavOpen);
  const togglePointsRight = isCompact ? !mobileNavOpen : sidebarCollapsed;

  const toggleSidebar = () => {
    if (isCompact) {
      if (mobileNavOpen) {
        setMobileNavOpen(false);
        return;
      }
      setMobileNavOpen(true);
      return;
    }

    setSidebarMode(nextSidebarMode(sidebarMode));
    setMobileNavOpen(false);
  };

  return (
    <div className={shellClassName}>
      <aside className={sidebarClassName} aria-label="项目侧边栏">
        <div className="project-sidebar__meta">
          <small>当前项目</small>
          <div className="project-sidebar__title">
            {projectQuery.data.title ?? '未命名项目'}
          </div>
          <p className="project-sidebar__subtitle">
            {projectQuery.data.genre || '小说工程'}
            {' · '}
            最近编辑 {formatRelativeTime(projectQuery.data.updated_at ?? projectQuery.data.created_at)}
          </p>
        </div>
        <ProjectNav collapsed={sidebarCollapsed} onNavigate={() => setMobileNavOpen(false)} />
      </aside>
      {mobileNavOpen ? (
        <button aria-label="关闭项目导航" className="project-sidebar-backdrop" onClick={() => setMobileNavOpen(false)} type="button" />
      ) : null}
      <button aria-label={toggleLabel} className="project-sidebar-edge-toggle" onClick={toggleSidebar} title={toggleLabel} type="button">
        {togglePointsRight ? <ChevronRight size={18} /> : <ChevronLeft size={18} />}
      </button>
      <main className="project-main">
        <Outlet context={{ project: projectQuery.data }} />
      </main>
    </div>
  );
}

function isValidProjectId(projectId: string) {
  const normalized = projectId.trim();
  return normalized !== '' && normalized !== 'undefined' && normalized !== 'null';
}

function readSidebarMode(): SidebarMode {
  if (typeof window === 'undefined') {
    return 'expanded';
  }
  const value = window.localStorage.getItem('novelos.projectSidebarMode');
  return value === 'collapsed' ? 'collapsed' : 'expanded';
}

function nextSidebarMode(mode: SidebarMode): SidebarMode {
  return mode === 'expanded' ? 'collapsed' : 'expanded';
}

function getSidebarToggleLabel(mode: SidebarMode, isCompact: boolean, mobileNavOpen: boolean) {
  if (isCompact) {
    return mobileNavOpen ? '关闭项目导航' : '打开项目导航';
  }
  switch (mode) {
    case 'expanded':
      return '收起项目导航';
    default:
      return '展开项目导航';
  }
}

function isCompactViewport() {
  return typeof window !== 'undefined' && window.matchMedia('(max-width: 960px)').matches;
}
