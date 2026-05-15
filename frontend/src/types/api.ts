export type ApiMeta = {
  request_id?: string;
};

export type PaginationMeta = {
  page: number;
  page_size: number;
  total: number;
};

export type StandardResponse<T> = {
  data: T;
  meta?: ApiMeta;
};

export type PaginatedResponse<T> = {
  data: T[];
  meta?: ApiMeta & {
    pagination?: PaginationMeta;
  };
};

export type ApiErrorPayload = {
  error?: {
    code?: string;
    message?: string;
    details?: unknown;
  };
  meta?: ApiMeta;
};

export type Project = {
  id: string;
  title: string;
  genre?: string;
  description?: string;
  stats?: {
    character_count?: number;
    chapter_count?: number;
    last_committed_chapter_number?: number;
  };
  created_at?: string;
  updated_at?: string;
};

export type CreateProjectRequest = {
  title: string;
  genre?: string;
  description?: string;
};

export type UpdateProjectRequest = Partial<CreateProjectRequest>;

export type WorldStateEntry = {
  key: string;
  value?: unknown;
  note?: string;
};

export type AuthorBible = {
  id?: string;
  project_id?: string;
  theme?: string;
  style_guide?: string;
  world_rules?: string[];
  aesthetic_principles?: string[];
  hard_constraints?: string[];
  soft_preferences?: string[];
  forbidden_moves?: string[];
  initial_world_state?: WorldStateEntry[];
};

export type UpdateAuthorBibleRequest = Omit<AuthorBible, 'id' | 'project_id'>;

export type Character = {
  id: string;
  project_id?: string;
  name: string;
  role: string;
  profile?: string;
  personality?: string;
  voice_style?: string;
  goals?: string[];
  fears?: string[];
  secrets?: string[];
  constraints?: string[];
  created_at?: string;
  updated_at?: string;
};

export type CreateCharacterRequest = {
  name: string;
  role: string;
  profile?: string;
  personality?: string;
  voice_style?: string;
  goals?: string[];
  fears?: string[];
  secrets?: string[];
  constraints?: string[];
};

export type UpdateCharacterRequest = Partial<CreateCharacterRequest>;

export type Relationship = {
  id: string;
  project_id?: string;
  character_a_id: string;
  character_b_id: string;
  summary: string;
  anchors?: string[];
  tension_points?: string[];
  volatility?: number;
  created_at?: string;
  updated_at?: string;
};

export type CreateRelationshipRequest = {
  character_a_id: string;
  character_b_id: string;
  summary: string;
  anchors?: string[];
  tension_points?: string[];
  volatility?: number;
};

export type UpdateRelationshipRequest = Omit<Partial<CreateRelationshipRequest>, 'character_a_id' | 'character_b_id'>;

export type SetupSession = {
  id: string;
  project_id?: string;
  seed_idea?: string;
  status?: string;
  created_at?: string;
  updated_at?: string;
};

export type CreateSetupSessionRequest = {
  seed_idea: string;
};

export type AdvanceSetupSessionRequest = {
  user_message: string;
};

export type RunStatus = 'queued' | 'running' | 'succeeded' | 'failed' | 'cancelled' | string;

export type Run = {
  id: string;
  status?: RunStatus;
  error?: string | { message?: string; code?: string };
  created_at?: string;
  updated_at?: string;
};

export type SetupRunResult = {
  author_bible?: Partial<AuthorBible>;
  characters?: Partial<Character>[];
  relationships?: Partial<Relationship>[];
  world_state?: WorldStateEntry[];
  questions?: string[];
  [key: string]: unknown;
};

export type ApplySetupRunRequest = {
  run_id: string;
  accept_author_bible?: boolean;
  accept_characters?: boolean;
  accept_relationships?: boolean;
  accept_world_state?: boolean;
  author_note?: string;
};

export type StorySession = {
  id: string;
  project_id?: string;
  title?: string;
  opening_situation?: string;
  author_intent?: string;
  status?: string;
  created_at?: string;
  updated_at?: string;
};

export type CreateStorySessionRequest = {
  title?: string;
  opening_situation?: string;
  author_intent?: string;
};

export type AdvanceStorySessionRequest = {
  author_message: string;
};

export type StoryRunResult = {
  draft_id?: string;
  memory_patch_id?: string;
  content?: string;
  draft?: {
    id?: string;
    content?: string;
    [key: string]: unknown;
  };
  memory_patch?: {
    id?: string;
    [key: string]: unknown;
  };
  review?: unknown;
  [key: string]: unknown;
};

export type CommitStoryRunRequest = {
  draft_id: string;
  memory_patch_id: string;
  author_note?: string;
};

export type Chapter = {
  id: string;
  project_id?: string;
  title?: string;
  chapter_number?: number;
  content?: string;
  created_at?: string;
  updated_at?: string;
};

export type Memory = {
  id: string;
  character_id?: string;
  content: string;
  importance?: number;
  note?: string;
  created_at?: string;
};

export type CreateMemoryRequest = {
  content: string;
  importance?: number;
  note?: string;
};
