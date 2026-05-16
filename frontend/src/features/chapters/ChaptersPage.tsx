// 章节列表页。只展示 story run commit 后进入正史的章节。
import { useQuery } from '@tanstack/react-query';
import { Link, useParams } from 'react-router-dom';

import { listChapters } from '../../api/chapters';
import { EmptyState } from '../../components/feedback/EmptyState';
import { ErrorState } from '../../components/feedback/ErrorState';
import { LoadingState } from '../../components/feedback/LoadingState';

export function ChaptersPage() {
  const { projectId = '' } = useParams();

  const chaptersQuery = useQuery({
    queryKey: ['chapters', projectId, 1, 50],
    queryFn: ({ signal }) => listChapters(projectId, 1, 50, signal),
    enabled: Boolean(projectId),
  });

  const chapters = chaptersQuery.data?.data ?? [];

  return (
    <div className="page">
      <div className="page__header">
        <div>
          <h1>章节</h1>
          <p>查看已经提交为正史的章节。</p>
        </div>
      </div>

      {chaptersQuery.isLoading ? <LoadingState /> : null}
      {chaptersQuery.isError ? <ErrorState message={(chaptersQuery.error as Error).message} /> : null}
      {!chaptersQuery.isLoading && chapters.length === 0 ? (
        <EmptyState title="还没有章节" description="在故事推进工作台提交草稿后，章节会出现在这里。" />
      ) : null}

      <div className="list-grid">
        {chapters.map((chapter) => (
          <Link className="list-card" key={chapter.id} to={`/chapters/${chapter.id}`}>
            <strong>{chapter.title ?? `第 ${chapter.chapter_number ?? '-'} 章`}</strong>
            <span>{chapter.updated_at ?? chapter.created_at ?? chapter.id}</span>
            {chapter.content ? <p>{chapter.content.slice(0, 120)}</p> : null}
          </Link>
        ))}
      </div>
    </div>
  );
}
