import { getData, getPage, pageParams, postData, putData } from './http';
import type {
  CreateRelationshipRequest,
  Relationship,
  UpdateRelationshipRequest,
} from '../types/api';

export function listRelationships(projectId: string, page = 1, pageSize = 20, signal?: AbortSignal) {
  return getPage<Relationship>(`/projects/${projectId}/relationships?${pageParams(page, pageSize)}`, signal);
}

export function createRelationship(projectId: string, body: CreateRelationshipRequest) {
  return postData<Relationship, CreateRelationshipRequest>(`/projects/${projectId}/relationships`, body);
}

export function getRelationship(relationshipId: string, signal?: AbortSignal) {
  return getData<Relationship>(`/relationships/${relationshipId}`, signal);
}

export function updateRelationship(relationshipId: string, body: UpdateRelationshipRequest) {
  return putData<Relationship, UpdateRelationshipRequest>(`/relationships/${relationshipId}`, body);
}
