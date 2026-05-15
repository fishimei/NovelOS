import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Save } from 'lucide-react';
import { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';

import { getAuthorBible, updateAuthorBible } from '../../api/authorBible';
import { ErrorState } from '../../components/feedback/ErrorState';
import { LoadingState } from '../../components/feedback/LoadingState';
import { TextArrayField } from '../../components/forms/TextArrayField';
import { WorldStateTable } from '../../components/forms/WorldStateTable';
import type { UpdateAuthorBibleRequest } from '../../types/api';

const emptyBible: UpdateAuthorBibleRequest = {
  theme: '',
  style_guide: '',
  world_rules: [],
  aesthetic_principles: [],
  hard_constraints: [],
  soft_preferences: [],
  forbidden_moves: [],
  initial_world_state: [],
};

export function AuthorBiblePage() {
  const { projectId = '' } = useParams();
  const queryClient = useQueryClient();
  const [form, setForm] = useState<UpdateAuthorBibleRequest>(emptyBible);

  const bibleQuery = useQuery({
    queryKey: ['authorBible', projectId],
    queryFn: ({ signal }) => getAuthorBible(projectId, signal),
    enabled: Boolean(projectId),
  });

  useEffect(() => {
    if (bibleQuery.data) {
      setForm({
        theme: bibleQuery.data.theme ?? '',
        style_guide: bibleQuery.data.style_guide ?? '',
        world_rules: bibleQuery.data.world_rules ?? [],
        aesthetic_principles: bibleQuery.data.aesthetic_principles ?? [],
        hard_constraints: bibleQuery.data.hard_constraints ?? [],
        soft_preferences: bibleQuery.data.soft_preferences ?? [],
        forbidden_moves: bibleQuery.data.forbidden_moves ?? [],
        initial_world_state: bibleQuery.data.initial_world_state ?? [],
      });
    }
  }, [bibleQuery.data]);

  const saveMutation = useMutation({
    mutationFn: () => updateAuthorBible(projectId, form),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['authorBible', projectId] });
      queryClient.invalidateQueries({ queryKey: ['project', projectId] });
    },
  });

  return (
    <div className="page">
      <div className="page__header">
        <div>
          <h1>作者圣经</h1>
          <p>维护项目的主题、叙事风格、世界规则和创作约束。</p>
        </div>
        <button className="button" disabled={saveMutation.isPending} onClick={() => saveMutation.mutate()} type="button">
          <Save size={17} />
          保存
        </button>
      </div>

      {bibleQuery.isLoading ? <LoadingState /> : null}
      {bibleQuery.isError ? <ErrorState message={(bibleQuery.error as Error).message} /> : null}
      {saveMutation.isError ? <ErrorState message={(saveMutation.error as Error).message} /> : null}

      <div className="form-grid">
        <label className="field">
          <span>主题</span>
          <input value={form.theme ?? ''} onChange={(event) => setForm({ ...form, theme: event.target.value })} />
        </label>
        <label className="field">
          <span>风格指南</span>
          <textarea
            value={form.style_guide ?? ''}
            onChange={(event) => setForm({ ...form, style_guide: event.target.value })}
            rows={5}
          />
        </label>
        <TextArrayField
          label="世界规则"
          values={form.world_rules ?? []}
          onChange={(world_rules) => setForm({ ...form, world_rules })}
        />
        <TextArrayField
          label="审美原则"
          values={form.aesthetic_principles ?? []}
          onChange={(aesthetic_principles) => setForm({ ...form, aesthetic_principles })}
        />
        <TextArrayField
          label="硬约束"
          values={form.hard_constraints ?? []}
          onChange={(hard_constraints) => setForm({ ...form, hard_constraints })}
        />
        <TextArrayField
          label="软偏好"
          values={form.soft_preferences ?? []}
          onChange={(soft_preferences) => setForm({ ...form, soft_preferences })}
        />
        <TextArrayField
          label="禁用走法"
          values={form.forbidden_moves ?? []}
          onChange={(forbidden_moves) => setForm({ ...form, forbidden_moves })}
        />
        <WorldStateTable
          value={form.initial_world_state ?? []}
          onChange={(initial_world_state) => setForm({ ...form, initial_world_state })}
        />
      </div>
    </div>
  );
}
