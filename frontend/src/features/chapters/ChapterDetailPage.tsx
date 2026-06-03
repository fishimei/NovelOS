// 章节阅读页。展示后端从 story event span 裁出的正史章节内容。
import { useQuery } from '@tanstack/react-query';
import { ArrowLeft, List, PenLine } from 'lucide-react';
import { Link, useNavigate, useParams } from 'react-router-dom';

import { getChapter } from '../../api/chapters';
import { ErrorState } from '../../components/feedback/ErrorState';
import { LoadingState } from '../../components/feedback/LoadingState';
import { MarkdownRenderer } from '../../components/MarkdownRenderer';

export function ChapterDetailPage() {
  const { chapterId = '' } = useParams();
  const navigate = useNavigate();

  const chapterQuery = useQuery({
    queryKey: ['chapter', chapterId],
    queryFn: ({ signal }) => getChapter(chapterId, signal),
    enabled: Boolean(chapterId),
  });
  const projectId = chapterQuery.data?.project_id ?? '';

  const goBack = () => {
    if (window.history.length > 1) {
      navigate(-1);
      return;
    }
    navigate(projectId ? `/projects/${projectId}/chapters` : '/');
  };

  return (
    <main className="reader-page">
      <div className="reader-toolbar">
        <button className="button button--secondary" onClick={goBack} type="button">
          <ArrowLeft size={17} />
          返回
        </button>
        {projectId ? (
          <>
            <Link className="button button--ghost" to={`/projects/${projectId}/chapters`}>
              <List size={17} />
              章节列表
            </Link>
            <Link className="button button--ghost" to={`/projects/${projectId}/story`}>
              <PenLine size={17} />
              写作工作台
            </Link>
          </>
        ) : null}
      </div>
      {chapterQuery.isLoading ? <LoadingState /> : null}
      {chapterQuery.isError ? <ErrorState message={(chapterQuery.error as Error).message} /> : null}
      {chapterQuery.data ? (
        <article className="chapter-reader">
          <header>
            <h1>{chapterQuery.data.title ?? `第 ${chapterQuery.data.chapter_number ?? '-'} 章`}</h1>
            <span>{chapterQuery.data.updated_at ?? chapterQuery.data.created_at}</span>
          </header>
          <MarkdownRenderer className="chapter-content" source={chapterQuery.data.content ?? '章节内容为空。'} variant="reading" />
        </article>
      ) : null}
    </main>
  );
}
