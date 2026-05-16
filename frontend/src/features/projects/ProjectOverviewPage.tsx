// 项目总览页。展示项目统计、提供轻量项目编辑，并连接到主要项目工作区。
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { BookMarked, FileText, GitBranch, PenLine, Save, Users, X } from 'lucide-react';
import { FormEvent, useState } from 'react';
import { Link, useOutletContext, useParams } from 'react-router-dom';

import { updateProject } from '../../api/projects';
import { ErrorState } from '../../components/feedback/ErrorState';
import { useRecentProjects } from '../../hooks/useRecentProjects';
import type { Project } from '../../types/api';

const actions = [
  { to: 'setup', label: '设定建模', icon: PenLine, description: '从一句想法生成结构化设定草稿。' },
  { to: 'bible', label: '作者圣经', icon: BookMarked, description: '维护主题、风格、世界规则和约束。' },
  { to: 'characters', label: '角色', icon: Users, description: '管理人物设定、目标和记忆。' },
  { to: 'relationships', label: '关系', icon: GitBranch, description: '维护角色之间的锚点与张力。' },
  { to: 'story', label: '故事推进', icon: FileText, description: '输入推进语，生成草稿并提交正史。' },
];

export function ProjectOverviewPage() {
  const { projectId } = useParams();
  const { project } = useOutletContext<{ project?: Project }>();
  const queryClient = useQueryClient();
  const { rememberProject } = useRecentProjects();
  const [isEditing, setIsEditing] = useState(false);
  const [title, setTitle] = useState('');
  const [genre, setGenre] = useState('');
  const [description, setDescription] = useState('');

  const updateProjectMutation = useMutation({
    mutationFn: () =>
      updateProject(projectId ?? '', {
        title: title.trim(),
        genre: genre.trim() || undefined,
        description: description.trim() || undefined,
      }),
    onSuccess: (updatedProject) => {
      // 项目编辑成功后，同步外壳标题和最近项目快捷入口。
      queryClient.setQueryData(['project', projectId], updatedProject);
      queryClient.invalidateQueries({ queryKey: ['project', projectId] });
      rememberProject(updatedProject);
      setIsEditing(false);
    },
  });

  const startEditing = () => {
    setTitle(project?.title ?? '');
    setGenre(project?.genre ?? '');
    setDescription(project?.description ?? '');
    setIsEditing(true);
  };

  const cancelEditing = () => {
    setIsEditing(false);
    updateProjectMutation.reset();
  };

  const saveProject = (event: FormEvent) => {
    event.preventDefault();
    updateProjectMutation.mutate();
  };

  return (
    <div className="page">
      <div className="page__header">
        <div>
          <h1>{project?.title ?? '项目总览'}</h1>
          <p>{project?.description || '项目详情加载后会显示在这里。'}</p>
        </div>
        <button className="button button--secondary" disabled={!project || isEditing} onClick={startEditing} type="button">
          <PenLine size={17} />
          编辑项目
        </button>
      </div>

      {isEditing ? (
        <form className="project-edit-form" onSubmit={saveProject}>
          {updateProjectMutation.isError ? <ErrorState message={(updateProjectMutation.error as Error).message} /> : null}
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
            <textarea value={description} onChange={(event) => setDescription(event.target.value)} rows={4} />
          </label>
          <div className="form-actions">
            <button className="button" disabled={!title.trim() || updateProjectMutation.isPending} type="submit">
              <Save size={17} />
              保存
            </button>
            <button className="button button--secondary" disabled={updateProjectMutation.isPending} onClick={cancelEditing} type="button">
              <X size={17} />
              取消
            </button>
          </div>
        </form>
      ) : null}

      <div className="stats-row">
        <div className="stat">
          <span>角色</span>
          <strong>{project?.stats?.character_count ?? '-'}</strong>
        </div>
        <div className="stat">
          <span>章节</span>
          <strong>{project?.stats?.chapter_count ?? '-'}</strong>
        </div>
        <div className="stat">
          <span>最后章节</span>
          <strong>{project?.stats?.last_committed_chapter_number ?? '-'}</strong>
        </div>
      </div>

      <div className="action-grid">
        {actions.map((action) => {
          const Icon = action.icon;
          return (
            <Link className="action-tile" key={action.to} to={`/projects/${projectId}/${action.to}`}>
              <Icon size={20} />
              <strong>{action.label}</strong>
              <span>{action.description}</span>
            </Link>
          );
        })}
      </div>
    </div>
  );
}
