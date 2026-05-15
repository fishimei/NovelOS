import { getData, getPage, pageParams, postData, putData } from './http';
import type {
  Character,
  CreateCharacterRequest,
  PaginatedResponse,
  UpdateCharacterRequest,
} from '../types/api';

export function listCharacters(projectId: string, page = 1, pageSize = 20, signal?: AbortSignal) {
  return getPage<Character>(`/projects/${projectId}/characters?${pageParams(page, pageSize)}`, signal);
}

export function createCharacter(projectId: string, body: CreateCharacterRequest) {
  return postData<Character, CreateCharacterRequest>(`/projects/${projectId}/characters`, body);
}

export function getCharacter(characterId: string, signal?: AbortSignal) {
  return getData<Character>(`/characters/${characterId}`, signal);
}

export function updateCharacter(characterId: string, body: UpdateCharacterRequest) {
  return putData<Character, UpdateCharacterRequest>(`/characters/${characterId}`, body);
}

export type CharacterPage = PaginatedResponse<Character>;
