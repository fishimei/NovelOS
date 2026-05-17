// 首页。负责创建新项目、按已知项目 ID 打开项目，并展示本地记录的最近项目。
import { useMutation } from '@tanstack/react-query';
import { ArrowRight, Plus } from 'lucide-react';
import { FormEvent, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';

import { createProject } from '../../api/projects';
import { ErrorState } from '../../components/feedback/ErrorState';
import { useRecentProjects } from '../../hooks/useRecentProjects';

export function HomePage() {
  const navigate = useNavigate();
  const { recentProjects, rememberProject } = useRecentProjects();
  const [title, setTitle] = useState('');
  const [genre, setGenre] = useState('');
  const [description, setDescription] = useState('');
  const [projectId, setProjectId] = useState('');
  const canOpenProject = isValidProjectId(projectId);

  const createProjectMutation = useMutation({
    mutationFn: createProject,
    onSuccess: (project) => {
      // 创建项目是 MVP 流程第一步；后端返回项目 id 后，立即进入项目工作台。
      rememberProject(project);
      navigate(`/projects/${project.id}`);
    },
  });

  const handleCreate = (event: FormEvent) => {
    event.preventDefault();
    createProjectMutation.mutate({
      title: title.trim(),
      genre: genre.trim() || undefined,
      description: description.trim() || undefined,
    });
  };

  const openProject = (event: FormEvent) => {
    event.preventDefault();
    const normalizedProjectId = projectId.trim();
    if (isValidProjectId(normalizedProjectId)) {
      navigate(`/projects/${normalizedProjectId}`);
    }
  };

  return (
    <main className="home">
      <section className="home__intro">
        <h1>NovelOS 创作工作台</h1>
        <p>从设定建模到故事推进，把小说项目、角色关系、章节草稿和记忆状态放在同一个工作流里。</p>
      </section>

      <section className="home__grid">
        <section className="home__primary panel">
          <div className="panel__header">
            <h2>创建项目</h2>
          </div>
          <form className="home-create-form" onSubmit={handleCreate}>
            {createProjectMutation.isError ? <ErrorState message={(createProjectMutation.error as Error).message} /> : null}
            <label className="field">
              <span>标题</span>
              <input value={title} onChange={(event) => setTitle(event.target.value)} required />
            </label>
            <label className="field">
              <span>类型</span>
              <input value={genre} onChange={(event) => setGenre(event.target.value)} placeholder="悬疑 / 都市 / 情感" />
            </label>
            <label className="field field--stack">
              <span>简介</span>
              <textarea value={description} onChange={(event) => setDescription(event.target.value)} rows={6} />
            </label>
            <div className="form-actions">
              <button className="button" disabled={!title.trim() || createProjectMutation.isPending} type="submit">
                <Plus size={17} />
                创建
              </button>
            </div>
          </form>
        </section>

        <div className="panel">
          <div className="panel__header">
            <h2>打开项目</h2>
          </div>
          <form className="inline-form" onSubmit={openProject}>
            <input value={projectId} onChange={(event) => setProjectId(event.target.value)} placeholder="project_..." />
            <button className="button button--secondary" disabled={!canOpenProject} type="submit">
              <ArrowRight size={17} />
              打开
            </button>
          </form>

          <div className="recent-list">
            <h3>最近项目</h3>
            {recentProjects.length === 0 ? <p className="muted">暂无最近项目。</p> : null}
            {recentProjects.filter((project) => isValidProjectId(project.id)).map((project) => (
              <Link className="recent-list__item" key={project.id} to={`/projects/${project.id.trim()}`}>
                <span>{project.title}</span>
                <small>{project.genre || project.id}</small>
              </Link>
            ))}
          </div>
        </div>
      </section>
    </main>
  );
}

function isValidProjectId(id: string) {
  const normalized = id.trim();
  return normalized !== '' && normalized !== 'undefined' && normalized !== 'null';
}
