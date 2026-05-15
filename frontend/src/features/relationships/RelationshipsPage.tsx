import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Plus } from 'lucide-react';
import { FormEvent, useState } from 'react';
import { useParams } from 'react-router-dom';

import { listCharacters } from '../../api/characters';
import { createRelationship, listRelationships } from '../../api/relationships';
import { EmptyState } from '../../components/feedback/EmptyState';
import { ErrorState } from '../../components/feedback/ErrorState';
import { LoadingState } from '../../components/feedback/LoadingState';
import { CharacterSelect } from '../../components/forms/CharacterSelect';

export function RelationshipsPage() {
  const { projectId = '' } = useParams();
  const queryClient = useQueryClient();
  const [characterAId, setCharacterAId] = useState('');
  const [characterBId, setCharacterBId] = useState('');
  const [summary, setSummary] = useState('');

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
    mutationFn: () =>
      createRelationship(projectId, {
        character_a_id: characterAId,
        character_b_id: characterBId,
        summary: summary.trim(),
      }),
    onSuccess: () => {
      setCharacterAId('');
      setCharacterBId('');
      setSummary('');
      queryClient.invalidateQueries({ queryKey: ['relationships', projectId] });
    },
  });

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault();
    createMutation.mutate();
  };

  const characters = charactersQuery.data?.data ?? [];
  const relationships = relationshipsQuery.data?.data ?? [];

  return (
    <div className="page">
      <div className="page__header">
        <div>
          <h1>关系</h1>
          <p>管理角色关系、锚点、张力点和波动值。</p>
        </div>
      </div>

      <form className="relationship-form" onSubmit={handleSubmit}>
        <CharacterSelect label="角色 A" value={characterAId} characters={characters} onChange={setCharacterAId} />
        <CharacterSelect label="角色 B" value={characterBId} characters={characters} onChange={setCharacterBId} />
        <label className="field">
          <span>关系摘要</span>
          <input value={summary} onChange={(event) => setSummary(event.target.value)} required />
        </label>
        <button
          className="button"
          disabled={!characterAId || !characterBId || characterAId === characterBId || !summary.trim()}
          type="submit"
        >
          <Plus size={17} />
          创建关系
        </button>
      </form>

      {relationshipsQuery.isLoading ? <LoadingState /> : null}
      {relationshipsQuery.isError ? <ErrorState message={(relationshipsQuery.error as Error).message} /> : null}
      {createMutation.isError ? <ErrorState message={(createMutation.error as Error).message} /> : null}
      {!relationshipsQuery.isLoading && relationships.length === 0 ? (
        <EmptyState title="还没有关系" description="至少需要两个角色，才能建立角色关系。" />
      ) : null}

      <div className="list-grid">
        {relationships.map((relationship) => (
          <article className="list-card" key={relationship.id}>
            <strong>{relationship.summary}</strong>
            <span>
              {relationship.character_a_id} {'->'} {relationship.character_b_id}
            </span>
            {relationship.tension_points?.length ? <p>张力点：{relationship.tension_points.join('、')}</p> : null}
          </article>
        ))}
      </div>
    </div>
  );
}
