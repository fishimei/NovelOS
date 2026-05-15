import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Plus } from 'lucide-react';
import { FormEvent, useState } from 'react';
import { Link, useParams } from 'react-router-dom';

import { createCharacter, listCharacters } from '../../api/characters';
import { EmptyState } from '../../components/feedback/EmptyState';
import { ErrorState } from '../../components/feedback/ErrorState';
import { LoadingState } from '../../components/feedback/LoadingState';

export function CharactersPage() {
  const { projectId = '' } = useParams();
  const queryClient = useQueryClient();
  const [name, setName] = useState('');
  const [role, setRole] = useState('');

  const charactersQuery = useQuery({
    queryKey: ['characters', projectId, 1, 50],
    queryFn: ({ signal }) => listCharacters(projectId, 1, 50, signal),
    enabled: Boolean(projectId),
  });

  const createMutation = useMutation({
    mutationFn: () => createCharacter(projectId, { name: name.trim(), role: role.trim() }),
    onSuccess: () => {
      setName('');
      setRole('');
      queryClient.invalidateQueries({ queryKey: ['characters', projectId] });
      queryClient.invalidateQueries({ queryKey: ['project', projectId] });
    },
  });

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault();
    createMutation.mutate();
  };

  const characters = charactersQuery.data?.data ?? [];

  return (
    <div className="page">
      <div className="page__header">
        <div>
          <h1>角色</h1>
          <p>管理人物身份、声音、目标、恐惧和约束。</p>
        </div>
      </div>

      <form className="toolbar-form" onSubmit={handleSubmit}>
        <input value={name} onChange={(event) => setName(event.target.value)} placeholder="角色名" required />
        <input value={role} onChange={(event) => setRole(event.target.value)} placeholder="角色定位" required />
        <button className="button" disabled={!name.trim() || !role.trim() || createMutation.isPending} type="submit">
          <Plus size={17} />
          创建角色
        </button>
      </form>

      {charactersQuery.isLoading ? <LoadingState /> : null}
      {charactersQuery.isError ? <ErrorState message={(charactersQuery.error as Error).message} /> : null}
      {createMutation.isError ? <ErrorState message={(createMutation.error as Error).message} /> : null}
      {!charactersQuery.isLoading && characters.length === 0 ? (
        <EmptyState title="还没有角色" description="可以先手动创建，也可以之后通过设定建模生成角色草稿。" />
      ) : null}

      <div className="list-grid">
        {characters.map((character) => (
          <Link className="list-card" key={character.id} to={`/characters/${character.id}`}>
            <strong>{character.name}</strong>
            <span>{character.role}</span>
            {character.profile ? <p>{character.profile}</p> : null}
          </Link>
        ))}
      </div>
    </div>
  );
}
