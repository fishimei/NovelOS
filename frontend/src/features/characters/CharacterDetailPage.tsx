import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Plus, Save } from 'lucide-react';
import { FormEvent, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';

import { getCharacter, updateCharacter } from '../../api/characters';
import { createCharacterMemory, listCharacterMemories } from '../../api/memories';
import { ErrorState } from '../../components/feedback/ErrorState';
import { LoadingState } from '../../components/feedback/LoadingState';
import { TextArrayField } from '../../components/forms/TextArrayField';
import type { UpdateCharacterRequest } from '../../types/api';

export function CharacterDetailPage() {
  const { characterId = '' } = useParams();
  const queryClient = useQueryClient();
  const [form, setForm] = useState<UpdateCharacterRequest>({});
  const [memoryContent, setMemoryContent] = useState('');

  const characterQuery = useQuery({
    queryKey: ['character', characterId],
    queryFn: ({ signal }) => getCharacter(characterId, signal),
    enabled: Boolean(characterId),
  });

  const memoriesQuery = useQuery({
    queryKey: ['characterMemories', characterId, 20],
    queryFn: ({ signal }) => listCharacterMemories(characterId, 20, signal),
    enabled: Boolean(characterId),
  });

  useEffect(() => {
    if (characterQuery.data) {
      setForm({
        name: characterQuery.data.name,
        role: characterQuery.data.role,
        profile: characterQuery.data.profile ?? '',
        personality: characterQuery.data.personality ?? '',
        voice_style: characterQuery.data.voice_style ?? '',
        goals: characterQuery.data.goals ?? [],
        fears: characterQuery.data.fears ?? [],
        secrets: characterQuery.data.secrets ?? [],
        constraints: characterQuery.data.constraints ?? [],
      });
    }
  }, [characterQuery.data]);

  const saveMutation = useMutation({
    mutationFn: () => updateCharacter(characterId, form),
    onSuccess: (character) => {
      queryClient.invalidateQueries({ queryKey: ['character', characterId] });
      if (character.project_id) {
        queryClient.invalidateQueries({ queryKey: ['characters', character.project_id] });
      }
    },
  });

  const createMemoryMutation = useMutation({
    mutationFn: () => createCharacterMemory(characterId, { content: memoryContent.trim() }),
    onSuccess: () => {
      setMemoryContent('');
      queryClient.invalidateQueries({ queryKey: ['characterMemories', characterId] });
    },
  });

  const addMemory = (event: FormEvent) => {
    event.preventDefault();
    createMemoryMutation.mutate();
  };

  return (
    <main className="detail-layout">
      <section className="page">
        <div className="page__header">
          <div>
            <h1>{characterQuery.data?.name ?? '角色详情'}</h1>
            <p>{characterQuery.data?.role ?? '编辑角色设定和人物记忆。'}</p>
          </div>
          <button className="button" disabled={saveMutation.isPending} onClick={() => saveMutation.mutate()} type="button">
            <Save size={17} />
            保存
          </button>
        </div>

        {characterQuery.isLoading ? <LoadingState /> : null}
        {characterQuery.isError ? <ErrorState message={(characterQuery.error as Error).message} /> : null}
        {saveMutation.isError ? <ErrorState message={(saveMutation.error as Error).message} /> : null}

        <div className="form-grid">
          <label className="field">
            <span>姓名</span>
            <input value={form.name ?? ''} onChange={(event) => setForm({ ...form, name: event.target.value })} />
          </label>
          <label className="field">
            <span>定位</span>
            <input value={form.role ?? ''} onChange={(event) => setForm({ ...form, role: event.target.value })} />
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
        </div>
      </section>

      <aside className="context-panel">
        <h2>角色记忆</h2>
        <form className="stack-list" onSubmit={addMemory}>
          <textarea
            value={memoryContent}
            onChange={(event) => setMemoryContent(event.target.value)}
            placeholder="追加一条人物记忆"
            rows={4}
          />
          <button className="button button--secondary" disabled={!memoryContent.trim()} type="submit">
            <Plus size={17} />
            添加记忆
          </button>
        </form>
        {memoriesQuery.isLoading ? <LoadingState /> : null}
        <div className="memory-list">
          {(memoriesQuery.data ?? []).map((memory) => (
            <div className="memory-item" key={memory.id}>
              <p>{memory.content}</p>
              {memory.note ? <small>{memory.note}</small> : null}
            </div>
          ))}
        </div>
      </aside>
    </main>
  );
}
