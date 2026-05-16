// 章节阅读页。展示后端在 story run commit 后返回的正史章节内容。
import { useQuery } from '@tanstack/react-query';
import { useParams } from 'react-router-dom';

import { getChapter } from '../../api/chapters';
import { ErrorState } from '../../components/feedback/ErrorState';
import { LoadingState } from '../../components/feedback/LoadingState';

export function ChapterDetailPage() {
  const { chapterId = '' } = useParams();

  const chapterQuery = useQuery({
    queryKey: ['chapter', chapterId],
    queryFn: ({ signal }) => getChapter(chapterId, signal),
    enabled: Boolean(chapterId),
  });

  return (
    <main className="reader-page">
      {chapterQuery.isLoading ? <LoadingState /> : null}
      {chapterQuery.isError ? <ErrorState message={(chapterQuery.error as Error).message} /> : null}
      {chapterQuery.data ? (
        <article className="chapter-reader">
          <header>
            <h1>{chapterQuery.data.title ?? `第 ${chapterQuery.data.chapter_number ?? '-'} 章`}</h1>
            <span>{chapterQuery.data.updated_at ?? chapterQuery.data.created_at}</span>
          </header>
          <div className="chapter-content">{chapterQuery.data.content ?? '章节内容为空。'}</div>
        </article>
      ) : null}
    </main>
  );
}
