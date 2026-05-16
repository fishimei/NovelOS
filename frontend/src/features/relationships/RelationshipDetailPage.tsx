// 关系详情页。编辑关系摘要、锚点、张力点和波动值，暂不展开长期关系视角。
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Save } from 'lucide-react';
import { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';

import { getRelationship, updateRelationship } from '../../api/relationships';
import { ErrorState } from '../../components/feedback/ErrorState';
import { LoadingState } from '../../components/feedback/LoadingState';
import { TextArrayField } from '../../components/forms/TextArrayField';
import type { UpdateRelationshipRequest } from '../../types/api';

const emptyRelationshipForm: UpdateRelationshipRequest = {
  summary: '',
  anchors: [],
  tension_points: [],
  volatility: 0,
};

export function RelationshipDetailPage() {
  const { relationshipId = '' } = useParams();
  const queryClient = useQueryClient();
  const [form, setForm] = useState<UpdateRelationshipRequest>(emptyRelationshipForm);

  const relationshipQuery = useQuery({
    queryKey: ['relationship', relationshipId],
    queryFn: ({ signal }) => getRelationship(relationshipId, signal),
    enabled: Boolean(relationshipId),
  });

  useEffect(() => {
    if (relationshipQuery.data) {
      // 详情页只编辑 OpenAPI 暴露的关系对字段，视角和事件保持只读。
      setForm({
        summary: relationshipQuery.data.pair.summary,
        anchors: relationshipQuery.data.pair.anchors ?? [],
        tension_points: relationshipQuery.data.pair.tension_points ?? [],
        volatility: relationshipQuery.data.pair.volatility ?? 0,
      });
    }
  }, [relationshipQuery.data]);

  const saveMutation = useMutation({
    mutationFn: () => updateRelationship(relationshipId, normalizeRelationshipForm(form)),
    onSuccess: (relationship) => {
      queryClient.invalidateQueries({ queryKey: ['relationship', relationshipId] });
      if (relationship.pair.project_id) {
        queryClient.invalidateQueries({ queryKey: ['relationships', relationship.pair.project_id] });
        queryClient.invalidateQueries({ queryKey: ['project', relationship.pair.project_id] });
      }
    },
  });

  return (
    <main className="detail-layout">
      <section className="page">
        <div className="page__header">
          <div>
            <h1>关系详情</h1>
            <p>
              {relationshipQuery.data?.pair.left_character_id ?? '角色 A'} {'->'}{' '}
              {relationshipQuery.data?.pair.right_character_id ?? '角色 B'}
            </p>
          </div>
          <button className="button" disabled={saveMutation.isPending} onClick={() => saveMutation.mutate()} type="button">
            <Save size={17} />
            保存
          </button>
        </div>

        {relationshipQuery.isLoading ? <LoadingState /> : null}
        {relationshipQuery.isError ? <ErrorState message={(relationshipQuery.error as Error).message} /> : null}
        {saveMutation.isError ? <ErrorState message={(saveMutation.error as Error).message} /> : null}

        <div className="form-grid">
          <label className="field field--stack">
            <span>关系摘要</span>
            <textarea value={form.summary ?? ''} onChange={(event) => setForm({ ...form, summary: event.target.value })} />
          </label>
          <label className="field">
            <span>波动值</span>
            <input
              type="number"
              value={form.volatility ?? 0}
              onChange={(event) => setForm({ ...form, volatility: Number(event.target.value) })}
            />
          </label>
          <TextArrayField label="锚点" values={form.anchors ?? []} onChange={(anchors) => setForm({ ...form, anchors })} />
          <TextArrayField
            label="张力点"
            values={form.tension_points ?? []}
            onChange={(tension_points) => setForm({ ...form, tension_points })}
          />
        </div>
      </section>

      <aside className="context-panel">
        <h2>关系视角</h2>
        <div className="stack-list">
          {(relationshipQuery.data?.views ?? []).map((view) => (
            <article className="memory-item" key={view.id}>
              <strong>
                {view.source_character_id} {'->'} {view.target_character_id}
              </strong>
              {view.public_attitude ? <p>公开态度：{view.public_attitude}</p> : null}
              {view.private_attitude ? <p>私下态度：{view.private_attitude}</p> : null}
            </article>
          ))}
        </div>
      </aside>
    </main>
  );
}

function normalizeRelationshipForm(form: UpdateRelationshipRequest): UpdateRelationshipRequest {
  // PUT 发送完整可编辑关系字段，同时清理列表里的空项。
  return {
    summary: form.summary?.trim() ?? '',
    anchors: normalizeStringListForUpdate(form.anchors),
    tension_points: normalizeStringListForUpdate(form.tension_points),
    volatility: form.volatility ?? 0,
  };
}

function normalizeStringListForUpdate(values?: string[]) {
  return values?.map((value) => value.trim()).filter(Boolean) ?? [];
}
