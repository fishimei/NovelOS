// 角色列表和创建页。覆盖项目级角色集合接口，并在 POST 前整理表单值。
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Plus } from 'lucide-react';
import { FormEvent, useState } from 'react';
import { Link, useParams } from 'react-router-dom';

import { createCharacter, listCharacters } from '../../api/characters';
import { EmptyState } from '../../components/feedback/EmptyState';
import { ErrorState } from '../../components/feedback/ErrorState';
import { LoadingState } from '../../components/feedback/LoadingState';
import { TextArrayField } from '../../components/forms/TextArrayField';
import type { CreateCharacterRequest } from '../../types/api';

const emptyCharacterForm: CreateCharacterRequest = {
  name: '',
  role: '',
  profile: '',
  personality: '',
  voice_style: '',
  goals: [],
  fears: [],
  secrets: [],
  constraints: [],
};

export function CharactersPage() {
  const { projectId = '' } = useParams();
  const queryClient = useQueryClient();
  const [form, setForm] = useState<CreateCharacterRequest>(emptyCharacterForm);

  const charactersQuery = useQuery({
    queryKey: ['characters', projectId, 1, 50],
    queryFn: ({ signal }) => listCharacters(projectId, 1, 50, signal),
    enabled: Boolean(projectId),
  });

  const createMutation = useMutation({
    mutationFn: () => createCharacter(projectId, normalizeCharacterForm(form)),
    onSuccess: () => {
      // 创建角色后刷新角色列表和项目外壳统计。
      setForm(emptyCharacterForm);
      queryClient.invalidateQueries({ queryKey: ['characters', projectId] });
      queryClient.invalidateQueries({ queryKey: ['project', projectId] });
    },
  });

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault();
    createMutation.mutate();
  };

  const characters = charactersQuery.data?.data ?? [];
  const canCreate = Boolean(form.name.trim() && form.role.trim()) && !createMutation.isPending;

  return (
    <div className="page">
      <div className="page__header">
        <div>
          <h1>角色</h1>
          <p>管理人物身份、声音、目标、恐惧和约束。</p>
        </div>
      </div>

      <form className="form-grid" onSubmit={handleSubmit}>
        <label className="field">
          <span>姓名</span>
          <input value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} required />
        </label>
        <label className="field">
          <span>定位</span>
          <input value={form.role} onChange={(event) => setForm({ ...form, role: event.target.value })} required />
        </label>
        <label className="field">
          <span>简介</span>
          <textarea value={form.profile ?? ''} onChange={(event) => setForm({ ...form, profile: event.target.value })} />
        </label>
        <label className="field">
          <span>性格</span>
          <textarea
            value={form.personality ?? ''}
            onChange={(event) => setForm({ ...form, personality: event.target.value })}
          />
        </label>
        <label className="field">
          <span>说话风格</span>
          <textarea
            value={form.voice_style ?? ''}
            onChange={(event) => setForm({ ...form, voice_style: event.target.value })}
          />
        </label>
        <TextArrayField label="目标" values={form.goals ?? []} onChange={(goals) => setForm({ ...form, goals })} />
        <TextArrayField label="恐惧" values={form.fears ?? []} onChange={(fears) => setForm({ ...form, fears })} />
        <TextArrayField label="秘密" values={form.secrets ?? []} onChange={(secrets) => setForm({ ...form, secrets })} />
        <TextArrayField
          label="约束"
          values={form.constraints ?? []}
          onChange={(constraints) => setForm({ ...form, constraints })}
        />
        <div className="form-actions">
          <button className="button" disabled={!canCreate} type="submit">
          <Plus size={17} />
          创建角色
          </button>
        </div>
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

function normalizeCharacterForm(form: CreateCharacterRequest): CreateCharacterRequest {
  // 只有作者实际填写的可选字段才发送给后端。
  return {
    name: form.name.trim(),
    role: form.role.trim(),
    profile: normalizeOptionalText(form.profile),
    personality: normalizeOptionalText(form.personality),
    voice_style: normalizeOptionalText(form.voice_style),
    goals: normalizeStringList(form.goals),
    fears: normalizeStringList(form.fears),
    secrets: normalizeStringList(form.secrets),
    constraints: normalizeStringList(form.constraints),
  };
}

function normalizeOptionalText(value?: string) {
  const trimmed = value?.trim();
  return trimmed || undefined;
}

function normalizeStringList(values?: string[]) {
  const normalized = values?.map((value) => value.trim()).filter(Boolean);
  return normalized && normalized.length > 0 ? normalized : undefined;
}
