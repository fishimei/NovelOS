// 全局视觉外壳。所有路由共用顶部栏，并在下方渲染当前匹配页面。
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
