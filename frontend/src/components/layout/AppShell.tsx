import { BookOpen } from 'lucide-react';
import { Link, Outlet } from 'react-router-dom';

export function AppShell() {
  return (
    <div className="app-shell">
      <header className="topbar">
        <Link className="brand" to="/">
          <BookOpen size={20} />
          <span>NovelOS</span>
        </Link>
      </header>
      <Outlet />
    </div>
  );
}
