// 前端共享 API 类型。当前 OpenAPI 里的 StandardResponse.data 仍是泛型对象，
// 所以这里尽量按 docs/openapi (1).yaml 手写对齐各资源的实际结构。
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

export type RelationshipPair = {
  id: string;
  project_id?: string;
  left_character_id: string;
  right_character_id: string;
  summary: string;
  anchors?: string[];
  tension_points?: string[];
  shared_history?: string[];
  volatility?: number;
  status?: string;
  created_at?: string;
  updated_at?: string;
};

export type RelationshipView = {
  id: string;
  project_id?: string;
  pair_id?: string;
  source_character_id: string;
  target_character_id: string;
  public_attitude?: string;
  private_attitude?: string;
  believed_target_attitude?: string;
  masking_strategy?: string;
  status?: string;
  created_at?: string;
  updated_at?: string;
};

export type RelationshipEvent = {
  id: string;
  project_id?: string;
  pair_id?: string;
  event_type?: string;
  summary?: string;
  payload?: Record<string, unknown>;
  created_at?: string;
};

export type Relationship = {
  pair: RelationshipPair;
  views?: RelationshipView[];
  recent_events?: RelationshipEvent[];
  character_a_view?: RelationshipView;
  character_b_view?: RelationshipView;
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
  last_user_message?: string;
  status?: string;
  messages?: ConversationMessage[];
  created_at?: string;
  updated_at?: string;
};

export type ConversationMessage = {
  id: string;
  session_id?: string;
  role: string;
  content: string;
  created_at?: string;
};

export type CreateSetupSessionRequest = {
  seed_idea: string;
};

export type AdvanceSetupSessionRequest = {
  user_message: string;
};

export type RunStatus = 'queued' | 'running' | 'succeeded' | 'failed' | 'cancelled' | string;

export type Run = {
  id?: string;
  run_id?: string;
  session_id?: string;
  project_id?: string;
  status?: RunStatus;
  current_step?: string;
  progress?: number;
  committed_at?: string;
  error?: string | { message?: string; code?: string };
  created_at?: string;
  updated_at?: string;
};

export type SetupQuestion = {
  key?: string;
  question?: string;
  why_it_matters?: string;
};

export type SetupDraft = {
  author_bible?: Partial<AuthorBible>;
  characters?: Partial<Character>[];
  relationships?: Partial<Relationship>[];
  world_state?: WorldStateEntry[];
  open_questions?: SetupQuestion[];
  assistant_summary?: string;
};

export type SetupRunResult = {
  run_id?: string;
  session_id?: string;
  status?: string;
  setup_draft?: SetupDraft;
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

export type ApplySetupRunResult = {
  project_id?: string;
  run_id?: string;
  status?: string;
};

export type StorySession = {
  id: string;
  project_id?: string;
  title?: string;
  opening_situation?: string;
  author_intent?: string;
  last_author_message?: string;
  status?: string;
  current_plot_variable_summary?: string;
  messages?: ConversationMessage[];
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

export type StoryDraft = {
  id?: string;
  title?: string;
  chapter_number?: number;
  content?: string;
  summary?: string;
  word_count?: number;
};

export type StoryPlotVariable = {
  pressure_source?: string;
  focal_character_id?: string;
  core_choice?: string;
  option_a?: string;
  option_b?: string;
  cost_a?: string;
  cost_b?: string;
  irreversible_effect?: string;
  related_character_ids?: string[];
  world_state_pressure?: string[];
};

export type StoryReviewReport = {
  pass?: boolean;
  hard_violations?: string[];
  continuity_issues?: string[];
  style_issues?: string[];
  suggested_fixes?: string[];
};

export type StoryCharacterMemoryUpdate = {
  character_id?: string;
  type?: string;
  content?: string;
  importance?: number;
};

export type StoryRelationshipViewUpdate = {
  view_id?: string;
  pair_id?: string;
  source_character_id?: string;
  target_character_id?: string;
  public_attitude?: string;
  private_attitude?: string;
  believed_target_attitude?: string;
  masking_strategy?: string;
};

export type StoryRelationshipUpdate = {
  pair_id?: string;
  summary?: string;
  tension_delta?: string;
  pair?: Partial<RelationshipPair>;
  views?: StoryRelationshipViewUpdate[];
  events?: Partial<RelationshipEvent>[];
};

export type StoryWorldStateUpdate = {
  key?: string;
  operation?: 'set' | 'update' | 'delete' | string;
  value?: unknown;
  note?: string;
};

export type StoryMemoryPatch = {
  id?: string;
  status?: string;
  character_memory_updates?: StoryCharacterMemoryUpdate[];
  relationship_updates?: StoryRelationshipUpdate[];
  world_state_updates?: StoryWorldStateUpdate[];
};

export type StoryRunResult = {
  run_id?: string;
  session_id?: string;
  status?: string;
  plot_variable?: StoryPlotVariable;
  draft?: StoryDraft;
  review?: StoryReviewReport;
  memory_patch?: StoryMemoryPatch;
  // Legacy fallbacks kept while old mock/runtime payloads may still be in circulation.
  draft_id?: string;
  memory_patch_id?: string;
  content?: string;
  [key: string]: unknown;
};

export type CommitStoryRunRequest = {
  draft_id: string;
  memory_patch_id: string;
  author_note?: string;
};

export type CommitStoryRunResult = {
  chapter?: Chapter;
  patch?: StoryMemoryPatch;
  story_run?: Run;
};

export type StoryGenerationStepEvent = {
  step?: string;
  progress?: number;
  error?: string;
  stop?: boolean;
  reason?: string;
  [key: string]: unknown;
};

export type StoryDraftDeltaEvent = {
  turn_index?: number;
  actor_id?: string;
  content?: string;
  text?: string;
  delta?: string;
  [key: string]: unknown;
};

export type RunEvent = {
  id: string;
  run_kind?: string;
  run_id?: string;
  event_name: string;
  sequence?: number;
  payload?: Record<string, unknown>;
  created_at?: string;
};

export type Chapter = {
  id: string;
  project_id?: string;
  title?: string;
  chapter_number?: number;
  summary?: string;
  content?: string;
  author_note?: string;
  status?: string;
  word_count?: number;
  committed_at?: string;
  created_at?: string;
  updated_at?: string;
};

export type Memory = {
  id: string;
  character_id?: string;
  content: string;
  source_chapter_id?: string;
  importance?: number;
  note?: string;
  status?: string;
  created_at?: string;
};

export type CreateMemoryRequest = {
  content: string;
  importance?: number;
  note?: string;
};
