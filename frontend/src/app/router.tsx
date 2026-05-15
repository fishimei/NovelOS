import { createBrowserRouter } from 'react-router-dom';

import { AppShell } from '../components/layout/AppShell';
import { ProjectShell } from '../components/layout/ProjectShell';
import { AuthorBiblePage } from '../features/authorBible/AuthorBiblePage';
import { ChapterDetailPage } from '../features/chapters/ChapterDetailPage';
import { ChaptersPage } from '../features/chapters/ChaptersPage';
import { CharacterDetailPage } from '../features/characters/CharacterDetailPage';
import { CharactersPage } from '../features/characters/CharactersPage';
import { HomePage } from '../features/projects/HomePage';
import { ProjectOverviewPage } from '../features/projects/ProjectOverviewPage';
import { RelationshipsPage } from '../features/relationships/RelationshipsPage';
import { SetupWorkspacePage } from '../features/setup/SetupWorkspacePage';
import { StoryWorkspacePage } from '../features/story/StoryWorkspacePage';
import { NotFoundPage } from '../features/system/NotFoundPage';

export const router = createBrowserRouter([
  {
    element: <AppShell />,
    children: [
      { index: true, element: <HomePage /> },
      {
        path: 'projects/:projectId',
        element: <ProjectShell />,
        children: [
          { index: true, element: <ProjectOverviewPage /> },
          { path: 'setup', element: <SetupWorkspacePage /> },
          { path: 'bible', element: <AuthorBiblePage /> },
          { path: 'characters', element: <CharactersPage /> },
          { path: 'relationships', element: <RelationshipsPage /> },
          { path: 'story', element: <StoryWorkspacePage /> },
          { path: 'chapters', element: <ChaptersPage /> },
        ],
      },
      { path: 'characters/:characterId', element: <CharacterDetailPage /> },
      { path: 'chapters/:chapterId', element: <ChapterDetailPage /> },
      { path: '*', element: <NotFoundPage /> },
    ],
  },
]);
