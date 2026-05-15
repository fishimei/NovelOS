import { getData, postData } from './http';
import type { CreateMemoryRequest, Memory } from '../types/api';

export function listCharacterMemories(characterId: string, limit = 20, signal?: AbortSignal) {
  return getData<Memory[]>(`/characters/${characterId}/memories?limit=${limit}`, signal);
}

export function createCharacterMemory(characterId: string, body: CreateMemoryRequest) {
  return postData<Memory, CreateMemoryRequest>(`/characters/${characterId}/memories`, body);
}
