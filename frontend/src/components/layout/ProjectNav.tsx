// 项目工作台内的侧边栏导航。
import { BookMarked, FileText, GitBranch, Home, PenLine, ScrollText, Users } from 'lucide-react';
import { NavLink, useParams } from 'react-router-dom';

const items = [
  { to: '', label: '总览', icon: Home, end: true },
  { to: 'setup', label: '设定建模', icon: PenLine },
  { to: 'bible', label: '作者圣经', icon: BookMarked },
  { to: 'characters', label: '角色', icon: Users },
  { to: 'relationships', label: '关系', icon: GitBranch },
  { to: 'story', label: '故事推进', icon: FileText },
  { to: 'chapters', label: '章节', icon: ScrollText },
];

export function ProjectNav() {
  const { projectId } = useParams();

  return (
    <nav className="project-nav" aria-label="项目导航">
      {items.map((item) => {
        const Icon = item.icon;
        const to = item.to ? `/projects/${projectId}/${item.to}` : `/projects/${projectId}`;

        return (
          <NavLink className="project-nav__item" end={item.end} key={item.label} to={to}>
            <Icon size={17} />
            {item.label}
          </NavLink>
        );
      })}
    </nav>
  );
}
