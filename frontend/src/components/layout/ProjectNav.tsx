import { BookMarked, FileText, GitBranch, Home, PenLine, ScrollText, Users } from 'lucide-react';
import { NavLink, useParams } from 'react-router-dom';

const items = [
  { to: '', label: '\u6982\u89c8', icon: Home, end: true },
  { to: 'setup', label: '\u8bbe\u5b9a\u5de5\u4f5c\u53f0', icon: PenLine },
  { to: 'bible', label: '\u4f5c\u8005\u5723\u7ecf', icon: BookMarked },
  { to: 'characters', label: '\u89d2\u8272', icon: Users },
  { to: 'relationships', label: '\u5173\u7cfb', icon: GitBranch },
  { to: 'story', label: '\u5199\u4f5c', icon: FileText },
  { to: 'chapters', label: '\u7ae0\u8282', icon: ScrollText },
];

export function ProjectNav({ collapsed = false, onNavigate }: { collapsed?: boolean; onNavigate?: () => void }) {
  const { projectId } = useParams();

  return (
    <nav className={collapsed ? 'project-nav project-nav--collapsed' : 'project-nav'} aria-label={'\u9879\u76ee\u5bfc\u822a'}>
      {items.map((item) => {
        const Icon = item.icon;
        const to = item.to ? `/projects/${projectId}/${item.to}` : `/projects/${projectId}`;

        return (
          <NavLink aria-label={item.label} className="project-nav__item" end={item.end} key={item.label} onClick={onNavigate} title={item.label} to={to}>
            <Icon size={17} />
            <span className="project-nav__label">{item.label}</span>
          </NavLink>
        );
      })}
    </nav>
  );
}
