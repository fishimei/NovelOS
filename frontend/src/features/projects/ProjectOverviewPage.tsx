import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowRight, BookMarked, FileText, GitBranch, PenLine, Save, ScrollText, Users, X } from 'lucide-react';
import { FormEvent, useState } from 'react';
import { Link, useOutletContext, useParams } from 'react-router-dom';

import { listChapters } from '../../api/chapters';
import { updateProject } from '../../api/projects';
import { listStorySessions } from '../../api/storySessions';
import { ErrorState } from '../../components/feedback/ErrorState';
import { useRecentProjects } from '../../hooks/useRecentProjects';
import type { Project } from '../../types/api';
import { formatDateLabel, formatRelativeTime } from '../../utils/format';

const actions = [
  { to: 'setup', label: '设定工作台', icon: PenLine, description: '从一句灵感推进设定草稿，把可用内容写回项目。' },
  { to: 'bible', label: '作者圣经', icon: BookMarked, description: '维护主题、文风、禁区与世界规则，统一创作边界。' },
  { to: 'characters', label: '角色档案', icon: Users, description: '集中编辑角色画像、目标、恐惧、秘密与约束。' },
  { to: 'relationships', label: '关系网络', icon: GitBranch, description: '整理人物之间的摘要、锚点和张力变化。' },
  { to: 'story', label: '继续写作', icon: FileText, description: '推进会话、生成正文草稿、审校并提交为正式章节。' },
  { to: 'chapters', label: '章节卷轴', icon: ScrollText, description: '按时间线查看已提交章节与最新创作进度。' },
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

  const chaptersQuery = useQuery({
    queryKey: ['chapters', projectId, 1, 6],
    queryFn: ({ signal }) => listChapters(projectId ?? '', 1, 6, signal),
    enabled: Boolean(projectId),
  });

  const storySessionsQuery = useQuery({
    queryKey: ['storySessions', projectId, 1, 6],
    queryFn: ({ signal }) => listStorySessions(projectId ?? '', 1, 6, signal),
    enabled: Boolean(projectId),
  });

  const updateProjectMutation = useMutation({
    mutationFn: () =>
      updateProject(projectId ?? '', {
        title: title.trim(),
        genre: genre.trim() || undefined,
        description: description.trim() || undefined,
      }),
    onSuccess: (updatedProject) => {
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

  const chapters = chaptersQuery.data?.data ?? [];
  const latestChapter = chapters[0];
  const storySessions = storySessionsQuery.data?.data ?? [];
  const latestSession = storySessions[0];

  const overviewStats = [
    {
      label: '角色',
      value: project?.stats?.character_count ?? 0,
      meta: project?.stats?.character_count ? '已建立人物档案' : '等待补充核心人物',
    },
    {
      label: '章节',
      value: project?.stats?.chapter_count ?? 0,
      meta: latestChapter ? `最近一章 ${formatRelativeTime(latestChapter.updated_at ?? latestChapter.created_at)}` : '尚未提交正文',
    },
    {
      label: '最新进度',
      value: project?.stats?.last_committed_chapter_number ?? '-',
      meta: latestChapter?.title ?? '从设定或写作工作台开始',
    },
  ];

  return (
    <div className="page page--wide page--overview">
      <div className="page__header">
        <div>
          <h1>{project?.title ?? '项目概览'}</h1>
          <p>{project?.description || '先写下这本书的方向、气质和核心冲突，概览页会围绕它组织你的工作流。'}</p>
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

      <section className="overview-ribbon">
        <div className="overview-ribbon__stats">
          {overviewStats.map((item) => (
            <div className="overview-ribbon__stat" key={item.label}>
              <span>{item.label}</span>
              <strong>{item.value}</strong>
              <small>{item.meta}</small>
            </div>
          ))}
        </div>
        <div className="overview-ribbon__status">
          <span>项目状态</span>
          <strong>{project?.genre || '长篇创作中'}</strong>
          <small>最后更新 {formatDateLabel(project?.updated_at ?? project?.created_at)}</small>
        </div>
      </section>

      <section className="continue-card">
        <div className="continue-card__copy">
          <small>今日继续写作</small>
          <h2>{latestChapter?.title ?? latestSession?.title ?? '先开启第一段创作流程'}</h2>
          <p>
            {latestChapter?.summary ||
              latestChapter?.content?.slice(0, 120) ||
              latestSession?.current_plot_variable_summary ||
              '从最新会话继续推进正文，或先补齐世界设定与角色资料。'}
          </p>
          <div className="continue-card__meta">
            <span>最近章节：{latestChapter ? `第 ${latestChapter.chapter_number ?? '-'} 章` : '暂无正式章节'}</span>
            <span>最近活动：{formatRelativeTime(latestChapter?.updated_at ?? latestSession?.updated_at ?? project?.updated_at)}</span>
          </div>
        </div>
        <div className="continue-card__actions">
          <Link className="button" to={`/projects/${projectId}/story`}>
            <ArrowRight size={17} />
            继续写作
          </Link>
          <Link className="button button--secondary" to={`/projects/${projectId}/chapters`}>
            查看章节卷轴
          </Link>
        </div>
      </section>

      <div className="action-grid action-grid--overview">
        {actions.map((action) => {
          const Icon = action.icon;
          const activity =
            action.to === 'story'
              ? latestSession?.updated_at
              : action.to === 'chapters'
                ? latestChapter?.updated_at
                : project?.updated_at;

          return (
            <Link className="action-tile action-tile--rich" key={action.to} to={`/projects/${projectId}/${action.to}`}>
              <div className="action-tile__head">
                <Icon size={20} />
                <small>{activity ? `最近活动 ${formatRelativeTime(activity)}` : '等待开始'}</small>
              </div>
              <strong>{action.label}</strong>
              <span>{action.description}</span>
            </Link>
          );
        })}
      </div>

      <div className="overview-secondary">
        <section className="panel overview-panel">
          <div className="panel__header">
            <h2>最近章节</h2>
          </div>
          {chapters.length === 0 ? (
            <p className="muted">还没有正式章节。完成一次写作提交后，这里会出现你的章节卷轴。</p>
          ) : (
            <div className="recent-story-list">
              {chapters.slice(0, 3).map((chapter) => (
                <Link className="recent-story-list__item" key={chapter.id} to={`/chapters/${chapter.id}`}>
                  <strong>{chapter.title ?? `第 ${chapter.chapter_number ?? '-'} 章`}</strong>
                  <span>{formatDateLabel(chapter.updated_at ?? chapter.created_at)}</span>
                </Link>
              ))}
            </div>
          )}
        </section>

        <section className="panel overview-panel">
          <div className="panel__header">
            <h2>写作会话</h2>
          </div>
          {storySessions.length ? (
            <div className="recent-story-list">
              {storySessions.slice(0, 3).map((session) => (
                <Link className="recent-story-list__item" key={session.id} to={`/projects/${projectId}/story`}>
                  <strong>{session.title || '未命名会话'}</strong>
                  <span>{session.current_plot_variable_summary || `最近更新 ${formatRelativeTime(session.updated_at)}`}</span>
                </Link>
              ))}
            </div>
          ) : (
            <p className="muted">还没有写作会话。可以先用设定工作台建立世界基础，再进入写作页面。</p>
          )}
        </section>
      </div>
    </div>
  );
}
