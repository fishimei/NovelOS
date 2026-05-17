import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Plus } from 'lucide-react';
import { FormEvent, useMemo, useState } from 'react';
import { Link, useParams } from 'react-router-dom';

import { listCharacters } from '../../api/characters';
import { createRelationship, listRelationships } from '../../api/relationships';
import { EmptyState } from '../../components/feedback/EmptyState';
import { ErrorState } from '../../components/feedback/ErrorState';
import { LoadingState } from '../../components/feedback/LoadingState';
import { CharacterSelect } from '../../components/forms/CharacterSelect';
import { TextArrayField } from '../../components/forms/TextArrayField';
import type { Character, CreateRelationshipRequest } from '../../types/api';

const emptyRelationshipForm: CreateRelationshipRequest = {
  character_a_id: '',
  character_b_id: '',
  summary: '',
  anchors: [],
  tension_points: [],
  volatility: 0,
};

export function RelationshipsPage() {
  const { projectId = '' } = useParams();
  const queryClient = useQueryClient();
  const [form, setForm] = useState<CreateRelationshipRequest>(emptyRelationshipForm);

  const charactersQuery = useQuery({
    queryKey: ['characters', projectId, 1, 100],
    queryFn: ({ signal }) => listCharacters(projectId, 1, 100, signal),
    enabled: Boolean(projectId),
  });

  const relationshipsQuery = useQuery({
    queryKey: ['relationships', projectId, 1, 50],
    queryFn: ({ signal }) => listRelationships(projectId, 1, 50, signal),
    enabled: Boolean(projectId),
  });

  const createMutation = useMutation({
    mutationFn: () => createRelationship(projectId, normalizeCreateRelationshipForm(form)),
    onSuccess: () => {
      setForm(emptyRelationshipForm);
      queryClient.invalidateQueries({ queryKey: ['relationships', projectId] });
      queryClient.invalidateQueries({ queryKey: ['project', projectId] });
    },
  });

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault();
    createMutation.mutate();
  };

  const characters = charactersQuery.data?.data ?? [];
  const relationships = relationshipsQuery.data?.data ?? [];
  const characterNameMap = useMemo(() => buildCharacterNameMap(characters), [characters]);
  const canCreate =
    Boolean(form.character_a_id && form.character_b_id && form.character_a_id !== form.character_b_id && form.summary.trim()) &&
    !createMutation.isPending;

  return (
    <div className="page page--wide page--relationships">
      <div className="page__header">
        <div>
          <h1>关系网络</h1>
          <p>用角色姓名组织人物关系、锚点和张力变化，而不是直接暴露内部 ID。</p>
        </div>
      </div>

      <form className="form-grid" onSubmit={handleSubmit}>
        <CharacterSelect
          label="角色 A"
          value={form.character_a_id}
          characters={characters}
          onChange={(character_a_id) => setForm({ ...form, character_a_id })}
        />
        <CharacterSelect
          label="角色 B"
          value={form.character_b_id}
          characters={characters}
          onChange={(character_b_id) => setForm({ ...form, character_b_id })}
        />
        <label className="field">
          <span>关系摘要</span>
          <textarea value={form.summary} onChange={(event) => setForm({ ...form, summary: event.target.value })} required />
        </label>
        <label className="field">
          <span>波动值</span>
          <input
            type="number"
            value={form.volatility ?? 0}
            onChange={(event) => setForm({ ...form, volatility: Number(event.target.value) })}
          />
        </label>
        <TextArrayField label="关系锚点" values={form.anchors ?? []} onChange={(anchors) => setForm({ ...form, anchors })} />
        <TextArrayField
          label="张力点"
          values={form.tension_points ?? []}
          onChange={(tension_points) => setForm({ ...form, tension_points })}
        />
        <div className="form-actions">
          <button className="button" disabled={!canCreate} type="submit">
            <Plus size={17} />
            创建关系
          </button>
        </div>
      </form>

      {relationshipsQuery.isLoading ? <LoadingState /> : null}
      {relationshipsQuery.isError ? <ErrorState message={(relationshipsQuery.error as Error).message} /> : null}
      {createMutation.isError ? <ErrorState message={(createMutation.error as Error).message} /> : null}
      {!relationshipsQuery.isLoading && relationships.length === 0 ? (
        <EmptyState title="还没有关系记录" description="先选择两个角色，再记录他们的关系摘要、锚点和张力点。" />
      ) : null}

      <div className="list-grid">
        {relationships.map((relationship) => {
          const leftName = getCharacterName(characterNameMap, relationship.pair.left_character_id);
          const rightName = getCharacterName(characterNameMap, relationship.pair.right_character_id);
          return (
            <Link className="list-card" key={relationship.pair.id} to={`/relationships/${relationship.pair.id}`}>
              <strong>{relationship.pair.summary}</strong>
              <span>
                {leftName} {'->'} {rightName}
              </span>
              {relationship.pair.anchors?.length ? <p>锚点：{relationship.pair.anchors.join(' / ')}</p> : null}
              {relationship.pair.tension_points?.length ? <p>张力：{relationship.pair.tension_points.join(' / ')}</p> : null}
            </Link>
          );
        })}
      </div>
    </div>
  );
}

function buildCharacterNameMap(characters: Character[]) {
  return new Map(characters.map((character) => [character.id, character.name]));
}

function getCharacterName(characterMap: Map<string, string>, id: string) {
  return characterMap.get(id) ?? id;
}

function normalizeCreateRelationshipForm(form: CreateRelationshipRequest): CreateRelationshipRequest {
  return {
    character_a_id: form.character_a_id,
    character_b_id: form.character_b_id,
    summary: form.summary.trim(),
    anchors: normalizeStringList(form.anchors),
    tension_points: normalizeStringList(form.tension_points),
    volatility: form.volatility,
  };
}

function normalizeStringList(values?: string[]) {
  const normalized = values?.map((value) => value.trim()).filter(Boolean);
  return normalized && normalized.length > 0 ? normalized : undefined;
}
