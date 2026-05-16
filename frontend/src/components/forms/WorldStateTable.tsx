// AuthorBible.initial_world_state 的编辑器。每行对应 WorldStateEntryRequest：key 加可选 value/note。
import { Plus, X } from 'lucide-react';

import type { WorldStateEntry } from '../../types/api';

type WorldStateTableProps = {
  value: WorldStateEntry[];
  onChange: (value: WorldStateEntry[]) => void;
};

export function WorldStateTable({ value, onChange }: WorldStateTableProps) {
  const updateEntry = (index: number, patch: Partial<WorldStateEntry>) => {
    onChange(value.map((item, itemIndex) => (itemIndex === index ? { ...item, ...patch } : item)));
  };

  return (
    <div className="table-field">
      <div className="table-field__header">
        <span>世界状态</span>
        <button className="button button--secondary" type="button" onClick={() => onChange([...value, { key: '' }])}>
          <Plus size={16} />
          添加变量
        </button>
      </div>
      <div className="world-table">
        <div className="world-table__head">Key</div>
        <div className="world-table__head">Value</div>
        <div className="world-table__head">Note</div>
        <div className="world-table__head" />
        {value.map((entry, index) => (
          <div className="world-table__row" key={index}>
            <input value={entry.key} onChange={(event) => updateEntry(index, { key: event.target.value })} />
            <input
              value={String(entry.value ?? '')}
              onChange={(event) => updateEntry(index, { value: event.target.value })}
            />
            <input value={entry.note ?? ''} onChange={(event) => updateEntry(index, { note: event.target.value })} />
            <button
              className="icon-button"
              type="button"
              onClick={() => onChange(value.filter((_, itemIndex) => itemIndex !== index))}
              aria-label="移除世界状态"
            >
              <X size={16} />
            </button>
          </div>
        ))}
      </div>
    </div>
  );
}
