// AuthorBible API 客户端：读取和保存项目级正式写作规则、风格指南、约束和初始世界状态。
import { getData, putData } from './http';
import type { AuthorBible, UpdateAuthorBibleRequest, WorldStateEntry } from '../types/api';

type WorldStateEntryPayload = WorldStateEntry & {
  Key?: string;
  Value?: unknown;
  Note?: string;
};

type AuthorBiblePayload = AuthorBible & {
  ID?: string;
  ProjectID?: string;
  Theme?: string;
  StyleGuide?: string;
  WorldRules?: string[];
  AestheticPrinciples?: string[];
  HardConstraints?: string[];
  SoftPreferences?: string[];
  ForbiddenMoves?: string[];
  InitialWorldState?: WorldStateEntryPayload[];
};

export async function getAuthorBible(projectId: string, signal?: AbortSignal) {
  return normalizeAuthorBible(await getData<AuthorBiblePayload>(`/projects/${projectId}/author-bible`, signal));
}

export async function updateAuthorBible(projectId: string, body: UpdateAuthorBibleRequest) {
  return normalizeAuthorBible(
    await putData<AuthorBiblePayload, UpdateAuthorBibleRequest>(`/projects/${projectId}/author-bible`, body),
  );
}

function normalizeAuthorBible(bible: AuthorBiblePayload): AuthorBible {
  const worldState = bible.initial_world_state ?? bible.InitialWorldState ?? [];

  return {
    ...bible,
    id: bible.id ?? bible.ID,
    project_id: bible.project_id ?? bible.ProjectID,
    theme: bible.theme ?? bible.Theme,
    style_guide: bible.style_guide ?? bible.StyleGuide,
    world_rules: bible.world_rules ?? bible.WorldRules ?? [],
    aesthetic_principles: bible.aesthetic_principles ?? bible.AestheticPrinciples ?? [],
    hard_constraints: bible.hard_constraints ?? bible.HardConstraints ?? [],
    soft_preferences: bible.soft_preferences ?? bible.SoftPreferences ?? [],
    forbidden_moves: bible.forbidden_moves ?? bible.ForbiddenMoves ?? [],
    initial_world_state: worldState.map(normalizeWorldStateEntry),
  };
}

function normalizeWorldStateEntry(entry: WorldStateEntryPayload): WorldStateEntry {
  return {
    key: entry.key ?? entry.Key ?? '',
    value: entry.value ?? entry.Value,
    note: entry.note ?? entry.Note,
  };
}
