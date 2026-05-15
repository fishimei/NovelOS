import { BookMarked, FileText, GitBranch, PenLine, Users } from 'lucide-react';
import { Link, useOutletContext, useParams } from 'react-router-dom';

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

  return (
    <div className="page">
      <div className="page__header">
        <div>
          <h1>{project?.title ?? '项目总览'}</h1>
          <p>{project?.description || '项目详情加载后会显示在这里。'}</p>
        </div>
      </div>

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
