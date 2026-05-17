import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Plus, Save, Sparkles } from 'lucide-react';
import { FormEvent, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';

import { createCharacter, getCharacter, listCharacters, updateCharacter } from '../../api/characters';
import { createCharacterMemory, listCharacterMemories } from '../../api/memories';
import { EmptyState } from '../../components/feedback/EmptyState';
import { ErrorState } from '../../components/feedback/ErrorState';
import { LoadingState } from '../../components/feedback/LoadingState';
import { TagInputField } from '../../components/forms/TagInputField';
import type { CreateCharacterRequest, CreateMemoryRequest, UpdateCharacterRequest } from '../../types/api';
import { formatRelativeTime } from '../../utils/format';

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
  const [selectedCharacterId, setSelectedCharacterId] = useState('');
  const [isCreatingCharacter, setIsCreatingCharacter] = useState(false);
  const [memoryForm, setMemoryForm] = useState<CreateMemoryRequest>({ content: '', importance: 1, note: '' });

  const charactersQuery = useQuery({
    queryKey: ['characters', projectId, 1, 50],
    queryFn: ({ signal }) => listCharacters(projectId, 1, 50, signal),
    enabled: Boolean(projectId),
  });

  const characters = charactersQuery.data?.data ?? [];

  const characterDetailQuery = useQuery({
    queryKey: ['character', selectedCharacterId],
    queryFn: ({ signal }) => getCharacter(selectedCharacterId, signal),
    enabled: Boolean(selectedCharacterId),
  });

  const memoriesQuery = useQuery({
    queryKey: ['memories', selectedCharacterId, 20],
    queryFn: ({ signal }) => listCharacterMemories(selectedCharacterId, 20, signal),
    enabled: Boolean(selectedCharacterId),
  });

  useEffect(() => {
    if (!selectedCharacterId && characters.length > 0 && !isCreatingCharacter) {
      setSelectedCharacterId(characters[0].id);
    }
  }, [characters, isCreatingCharacter, selectedCharacterId]);

  useEffect(() => {
    if (characterDetailQuery.data) {
      setForm({
        name: characterDetailQuery.data.name,
        role: characterDetailQuery.data.role,
        profile: characterDetailQuery.data.profile ?? '',
        personality: characterDetailQuery.data.personality ?? '',
        voice_style: characterDetailQuery.data.voice_style ?? '',
        goals: characterDetailQuery.data.goals ?? [],
        fears: characterDetailQuery.data.fears ?? [],
        secrets: characterDetailQuery.data.secrets ?? [],
        constraints: characterDetailQuery.data.constraints ?? [],
      });
    }
  }, [characterDetailQuery.data]);

  const createMutation = useMutation({
    mutationFn: () => createCharacter(projectId, normalizeCharacterForm(form)),
    onSuccess: (character) => {
      setIsCreatingCharacter(false);
      setSelectedCharacterId(character.id);
      queryClient.invalidateQueries({ queryKey: ['characters', projectId] });
      queryClient.invalidateQueries({ queryKey: ['project', projectId] });
    },
  });

  const updateMutation = useMutation({
    mutationFn: () => updateCharacter(selectedCharacterId, normalizeCharacterUpdateForm(form)),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['character', selectedCharacterId] });
      queryClient.invalidateQueries({ queryKey: ['characters', projectId] });
      queryClient.invalidateQueries({ queryKey: ['project', projectId] });
    },
  });

  const createMemoryMutation = useMutation({
    mutationFn: () => createCharacterMemory(selectedCharacterId, normalizeMemoryForm(memoryForm)),
    onSuccess: () => {
      setMemoryForm({ content: '', importance: 1, note: '' });
      queryClient.invalidateQueries({ queryKey: ['memories', selectedCharacterId] });
    },
  });

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault();

    if (selectedCharacterId) {
      updateMutation.mutate();
      return;
    }

    createMutation.mutate();
  };

  const startNewCharacter = () => {
    setIsCreatingCharacter(true);
    setSelectedCharacterId('');
    setForm(emptyCharacterForm);
    setMemoryForm({ content: '', importance: 1, note: '' });
  };

  const canSave = Boolean(form.name.trim() && form.role.trim()) && !createMutation.isPending && !updateMutation.isPending;
  const selectedCharacter = characters.find((character) => character.id === selectedCharacterId);

  return (
    <div className="page page--wide page--characters">
      <div className="page__header">
        <div>
          <h1>角色</h1>
          <p>左侧维护角色清单，右侧集中编辑身份、画像和创作约束，让人物资料真正可用。</p>
        </div>
      </div>

      {charactersQuery.isLoading ? <LoadingState /> : null}
      {charactersQuery.isError ? <ErrorState message={(charactersQuery.error as Error).message} /> : null}
      {createMutation.isError ? <ErrorState message={(createMutation.error as Error).message} /> : null}
      {updateMutation.isError ? <ErrorState message={(updateMutation.error as Error).message} /> : null}

      {!charactersQuery.isLoading && characters.length === 0 && !selectedCharacterId ? (
        <EmptyState title="还没有角色" description="先建立一个主角或关键配角，后续剧情生成会稳定很多。" />
      ) : null}

      <div className="character-workbench">
        <aside className="panel character-directory">
          <div className="panel__header">
            <h2>角色清单</h2>
            <button className="button button--ghost" onClick={startNewCharacter} type="button">
              <Plus size={16} />
              新建角色
            </button>
          </div>
          <div className="character-directory__list">
            {characters.map((character) => (
              <button
                className={selectedCharacterId === character.id ? 'character-teaser character-teaser--active' : 'character-teaser'}
                key={character.id}
                onClick={() => {
                  setIsCreatingCharacter(false);
                  setSelectedCharacterId(character.id);
                }}
                type="button"
              >
                <strong>{character.name}</strong>
                <span>{character.role}</span>
                <small>{formatRelativeTime(character.updated_at ?? character.created_at)}</small>
              </button>
            ))}
          </div>
        </aside>

        <section className="panel character-editor">
          <div className="panel__header">
            <h2>{selectedCharacterId ? '角色编辑' : '创建角色'}</h2>
            <div className="character-editor__meta">
              <span>{selectedCharacter?.role || '新角色草稿'}</span>
            </div>
          </div>

          <form onSubmit={handleSubmit}>
            <div className="character-hero-grid">
              <section className="panel panel--nested">
                <div className="panel__header">
                  <h2>身份基础</h2>
                </div>
                <label className="field">
                  <span>姓名</span>
                  <input value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} required />
                </label>
                <label className="field">
                  <span>定位</span>
                  <input value={form.role} onChange={(event) => setForm({ ...form, role: event.target.value })} required />
                </label>
                <label className="field field--stack">
                  <span>性格</span>
                  <textarea
                    rows={4}
                    value={form.personality ?? ''}
                    onChange={(event) => setForm({ ...form, personality: event.target.value })}
                  />
                </label>
                <label className="field field--stack">
                  <span>说话风格</span>
                  <textarea
                    rows={4}
                    value={form.voice_style ?? ''}
                    onChange={(event) => setForm({ ...form, voice_style: event.target.value })}
                  />
                </label>
              </section>

              <section className="panel panel--nested">
                <div className="panel__header">
                  <h2>角色画像</h2>
                </div>
                <label className="field field--stack">
                  <span>简介</span>
                  <textarea
                    placeholder="写下这个角色最重要的身份、处境与人物弧线起点。"
                    rows={10}
                    value={form.profile ?? ''}
                    onChange={(event) => setForm({ ...form, profile: event.target.value })}
                  />
                </label>
                <div className="character-note">
                  <Sparkles size={16} />
                  <span>先写最核心的冲突源，再补外貌、背景和习惯，人物会更立得住。</span>
                </div>
              </section>
            </div>

            <div className="tag-grid">
              <TagInputField
                addLabel="添加目标"
                helperText="角色主动追求什么，决定他会推动哪些剧情。"
                label="目标"
                onChange={(goals) => setForm({ ...form, goals })}
                placeholder="输入后回车，例如：拿回家族继承权"
                values={form.goals ?? []}
              />
              <TagInputField
                addLabel="添加恐惧"
                helperText="恐惧会决定人物在哪些场景下退缩或失控。"
                label="恐惧"
                onChange={(fears) => setForm({ ...form, fears })}
                placeholder="例如：害怕再次失去妹妹"
                values={form.fears ?? []}
              />
              <TagInputField
                addLabel="添加秘密"
                helperText="秘密是剧情转折的储备库，越具体越有用。"
                label="秘密"
                onChange={(secrets) => setForm({ ...form, secrets })}
                placeholder="例如：真正放火的人其实是他自己"
                values={form.secrets ?? []}
              />
              <TagInputField
                addLabel="添加约束"
                helperText="约束让角色的行为边界更稳定，便于后续生成。"
                label="约束"
                onChange={(constraints) => setForm({ ...form, constraints })}
                placeholder="例如：绝不主动向他人求助"
                values={form.constraints ?? []}
              />
            </div>

            <div className="form-actions form-actions--end">
              <button className="button" disabled={!canSave} type="submit">
                {selectedCharacterId ? <Save size={17} /> : <Plus size={17} />}
                {selectedCharacterId ? '保存角色' : '创建角色'}
              </button>
            </div>
          </form>
        </section>

        <aside className="panel character-memory-panel">
          <div className="panel__header">
            <h2>角色记忆</h2>
          </div>
          {!selectedCharacterId ? (
            <p className="muted">先创建角色，之后可以把关键记忆、创伤和节点逐条补进来。</p>
          ) : (
            <>
              <form
                className="stack-list"
                onSubmit={(event) => {
                  event.preventDefault();
                  createMemoryMutation.mutate();
                }}
              >
                <textarea
                  placeholder="记录一个会影响后续行为的事实、创伤或执念。"
                  rows={4}
                  value={memoryForm.content}
                  onChange={(event) => setMemoryForm({ ...memoryForm, content: event.target.value })}
                />
                <label className="field">
                  <span>重要度</span>
                  <input
                    min={1}
                    type="number"
                    value={memoryForm.importance ?? 1}
                    onChange={(event) =>
                      setMemoryForm({
                        ...memoryForm,
                        importance: event.target.value ? Number(event.target.value) : undefined,
                      })
                    }
                  />
                </label>
                <button className="button button--ghost" disabled={!memoryForm.content.trim()} type="submit">
                  <Plus size={16} />
                  添加记忆
                </button>
              </form>
              {characterDetailQuery.isLoading || memoriesQuery.isLoading ? <LoadingState /> : null}
              {characterDetailQuery.isError ? <ErrorState message={(characterDetailQuery.error as Error).message} /> : null}
              {memoriesQuery.isError ? <ErrorState message={(memoriesQuery.error as Error).message} /> : null}
              {createMemoryMutation.isError ? <ErrorState message={(createMemoryMutation.error as Error).message} /> : null}
              <div className="memory-list">
                {(memoriesQuery.data ?? []).map((memory) => (
                  <div className="memory-item" key={memory.id}>
                    <p>{memory.content}</p>
                    <small>重要度 {memory.importance ?? 1}</small>
                  </div>
                ))}
              </div>
            </>
          )}
        </aside>
      </div>
    </div>
  );
}

function normalizeCharacterForm(form: CreateCharacterRequest): CreateCharacterRequest {
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

function normalizeCharacterUpdateForm(form: CreateCharacterRequest): UpdateCharacterRequest {
  return {
    name: form.name.trim(),
    role: form.role.trim(),
    profile: form.profile?.trim() ?? '',
    personality: form.personality?.trim() ?? '',
    voice_style: form.voice_style?.trim() ?? '',
    goals: normalizeStringListForUpdate(form.goals),
    fears: normalizeStringListForUpdate(form.fears),
    secrets: normalizeStringListForUpdate(form.secrets),
    constraints: normalizeStringListForUpdate(form.constraints),
  };
}

function normalizeMemoryForm(form: CreateMemoryRequest): CreateMemoryRequest {
  return {
    content: form.content.trim(),
    importance: form.importance,
    note: normalizeOptionalText(form.note),
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

function normalizeStringListForUpdate(values?: string[]) {
  return values?.map((value) => value.trim()).filter(Boolean) ?? [];
}
