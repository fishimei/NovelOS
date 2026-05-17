import { useQuery } from '@tanstack/react-query';
import { PenLine } from 'lucide-react';
import { Link, useParams } from 'react-router-dom';

import { listChapters } from '../../api/chapters';
import { EmptyState } from '../../components/feedback/EmptyState';
import { ErrorState } from '../../components/feedback/ErrorState';
import { LoadingState } from '../../components/feedback/LoadingState';
import { formatDateLabel } from '../../utils/format';

export function ChaptersPage() {
  const { projectId = '' } = useParams();

  const chaptersQuery = useQuery({
    queryKey: ['chapters', projectId, 1, 50],
    queryFn: ({ signal }) => listChapters(projectId, 1, 50, signal),
    enabled: Boolean(projectId),
  });

  const chapters = chaptersQuery.data?.data ?? [];
  const totalWords = chapters.reduce((sum, chapter) => sum + (chapter.word_count ?? 0), 0);
  const averageWords = chapters.length > 0 ? Math.round(totalWords / chapters.length) : 0;
  const latestUpdate = chapters[0]?.updated_at ?? chapters[0]?.created_at;

  return (
    <div className="page">
      <div className="page__header">
        <div>
          <h1>章节</h1>
          <p>正式提交后的章节会按时间线堆叠在这里，方便你回看整体推进节奏。</p>
        </div>
      </div>

      <div className="stats-row stats-row--compact">
        <div className="stat">
          <span>总章节</span>
          <strong>{chapters.length}</strong>
        </div>
        <div className="stat">
          <span>总字数</span>
          <strong>{totalWords}</strong>
        </div>
        <div className="stat">
          <span>平均字数</span>
          <strong>{averageWords}</strong>
        </div>
        <div className="stat">
          <span>最近更新</span>
          <strong>{latestUpdate ? formatDateLabel(latestUpdate) : '暂无'}</strong>
        </div>
      </div>

      {chaptersQuery.isLoading ? <LoadingState /> : null}
      {chaptersQuery.isError ? <ErrorState message={(chaptersQuery.error as Error).message} /> : null}
      {!chaptersQuery.isLoading && chapters.length === 0 ? (
        <div className="chapter-empty">
          <div className="chapter-empty__icon">
            <PenLine size={28} />
          </div>
          <EmptyState
            title="还没有章节"
            description="提交第一章之后，这里会按时间线展开你的章节卷轴，并保留每次正式写作结果。"
          />
          <Link className="button" to={`/projects/${projectId}/story`}>
            前往写作工作台
          </Link>
        </div>
      ) : null}

      <div className="chapter-timeline">
        {chapters.map((chapter) => (
          <Link className="chapter-timeline__item" key={chapter.id} to={`/chapters/${chapter.id}`}>
            <div className="chapter-timeline__marker" aria-hidden="true" />
            <article className="list-card chapter-card">
              <div className="chapter-card__eyebrow">
                <span>{chapter.status || '已提交'}</span>
                <small>{formatDateLabel(chapter.updated_at ?? chapter.created_at)}</small>
              </div>
              <strong>{chapter.title ?? `第 ${chapter.chapter_number ?? '-'} 章`}</strong>
              <p>{chapter.summary || chapter.content?.slice(0, 140) || '这章还没有摘要。'}</p>
              <div className="chapter-card__meta">
                <span>章节号 {chapter.chapter_number ?? '-'}</span>
                <span>{chapter.word_count ?? 0} 字</span>
              </div>
            </article>
          </Link>
        ))}
      </div>
    </div>
  );
}
