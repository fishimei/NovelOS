import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Save } from 'lucide-react';
import { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';

import { getAuthorBible, updateAuthorBible } from '../../api/authorBible';
import { ApiError } from '../../api/http';
import { ErrorState } from '../../components/feedback/ErrorState';
import { LoadingState } from '../../components/feedback/LoadingState';
import { TextArrayField } from '../../components/forms/TextArrayField';
import { WorldStateTable } from '../../components/forms/WorldStateTable';
import type { AuthorBible, UpdateAuthorBibleRequest } from '../../types/api';

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
    retry: (failureCount, error) => !isMissingAuthorBible(error) && failureCount < 3,
  });
  const isMissingBible = isMissingAuthorBible(bibleQuery.error);

  useEffect(() => {
    setForm(emptyBible);
  }, [projectId]);

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
    onSuccess: (savedBible) => {
      queryClient.setQueryData(['authorBible', projectId], savedBible);
      setForm(toAuthorBibleForm(savedBible));
      queryClient.invalidateQueries({ queryKey: ['authorBible', projectId] });
      queryClient.invalidateQueries({ queryKey: ['project', projectId] });
    },
  });

  return (
    <div className="page page--wide page--author-bible">
      <div className="page__header">
        <div>
          <h1>作者圣经</h1>
          <p>把主题、文风、世界规则与禁区写清楚，后续所有生成都会围绕这里的边界运行。</p>
        </div>
        <button className="button" disabled={saveMutation.isPending} onClick={() => saveMutation.mutate()} type="button">
          <Save size={17} />
          保存
        </button>
      </div>

      {bibleQuery.isLoading ? <LoadingState /> : null}
      {bibleQuery.isError && !isMissingBible ? <ErrorState message={(bibleQuery.error as Error).message} /> : null}
      {saveMutation.isError ? <ErrorState message={(saveMutation.error as Error).message} /> : null}

      <section className="form-grid form-grid--hero">
        <label className="field">
          <span>主题</span>
          <input
            placeholder="一句话写清这本书最核心的问题意识"
            value={form.theme ?? ''}
            onChange={(event) => setForm({ ...form, theme: event.target.value })}
          />
        </label>
        <label className="field field--stack field--wide">
          <span>风格指南</span>
          <textarea
            placeholder="语体、叙事距离、句子节奏、禁用表达都写在这里"
            value={form.style_guide ?? ''}
            onChange={(event) => setForm({ ...form, style_guide: event.target.value })}
            rows={5}
          />
        </label>
      </section>

      <section className="editor-section">
        <div className="editor-section__header">
          <div>
            <h2>世界与审美</h2>
            <p>先固定世界如何运转，再规定文本应该呈现出的气味与质感。</p>
          </div>
        </div>
        <div className="editor-section__grid">
          <TextArrayField
            addLabel="添加规则"
            helperText="每条规则都应该足够明确，能直接约束故事推进。"
            label="世界规则"
            onChange={(world_rules) => setForm({ ...form, world_rules })}
            placeholder="例如：超凡力量不能逆转死亡"
            values={form.world_rules ?? []}
          />
          <TextArrayField
            addLabel="添加原则"
            helperText="记录整体审美倾向，例如冷峻、克制、史诗感。"
            label="审美原则"
            onChange={(aesthetic_principles) => setForm({ ...form, aesthetic_principles })}
            placeholder="例如：所有神秘都应保留余味，不做直白解释"
            values={form.aesthetic_principles ?? []}
          />
        </div>
      </section>

      <section className="editor-section">
        <div className="editor-section__header editor-section__header--dashed">
          <div>
            <h2>边界与偏好</h2>
            <p>把必须遵守的底线和可偏向的选择分开写，后续生成会稳定很多。</p>
          </div>
        </div>
        <div className="editor-section__grid">
          <TextArrayField
            addLabel="添加硬约束"
            label="硬约束"
            onChange={(hard_constraints) => setForm({ ...form, hard_constraints })}
            placeholder="例如：第一卷不能出现现代科技解释"
            values={form.hard_constraints ?? []}
          />
          <TextArrayField
            addLabel="添加软偏好"
            label="软偏好"
            onChange={(soft_preferences) => setForm({ ...form, soft_preferences })}
            placeholder="例如：优先采用人物驱动而不是设定讲解"
            values={form.soft_preferences ?? []}
          />
          <TextArrayField
            addLabel="添加禁区"
            helperText="这些内容在风格或价值观上明确不应进入文本。"
            label="禁止动作"
            onChange={(forbidden_moves) => setForm({ ...form, forbidden_moves })}
            placeholder="例如：不能用巧合直接解决主要冲突"
            values={form.forbidden_moves ?? []}
          />
        </div>
      </section>

      <section className="editor-section">
        <div className="editor-section__header editor-section__header--dashed">
          <div>
            <h2>初始世界状态</h2>
            <p>记录故事正式开始前已成立的局势、变量与约束。</p>
          </div>
        </div>
        <WorldStateTable
          value={form.initial_world_state ?? []}
          onChange={(initial_world_state) => setForm({ ...form, initial_world_state })}
        />
      </section>
    </div>
  );
}

function isMissingAuthorBible(error: unknown) {
  return error instanceof ApiError && error.status === 404 && error.code === 'NOT_FOUND';
}

function toAuthorBibleForm(bible: AuthorBible): UpdateAuthorBibleRequest {
  return {
    theme: bible.theme ?? '',
    style_guide: bible.style_guide ?? '',
    world_rules: bible.world_rules ?? [],
    aesthetic_principles: bible.aesthetic_principles ?? [],
    hard_constraints: bible.hard_constraints ?? [],
    soft_preferences: bible.soft_preferences ?? [],
    forbidden_moves: bible.forbidden_moves ?? [],
    initial_world_state: bible.initial_world_state ?? [],
  };
}
