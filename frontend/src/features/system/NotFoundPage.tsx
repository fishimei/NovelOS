// 没有任何路由匹配时展示的兜底页面。
import { Link } from 'react-router-dom';

export function NotFoundPage() {
  return (
    <main className="center-page">
      <h1>页面不存在</h1>
      <Link className="button" to="/">
        返回首页
      </Link>
    </main>
  );
}
