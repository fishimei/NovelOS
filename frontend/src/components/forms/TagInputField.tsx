import { Plus, X } from 'lucide-react';
import { KeyboardEvent, useState } from 'react';

type TagInputFieldProps = {
  label: string;
  values: string[];
  onChange: (values: string[]) => void;
  placeholder?: string;
  addLabel?: string;
  helperText?: string;
};

export function TagInputField({
  label,
  values,
  onChange,
  placeholder,
  addLabel = '添加条目',
  helperText,
}: TagInputFieldProps) {
  const [draft, setDraft] = useState('');

  const commitDraft = () => {
    const next = draft.trim();
    if (!next) {
      return;
    }

    onChange([...values, next]);
    setDraft('');
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'Enter' || event.key === ',') {
      event.preventDefault();
      commitDraft();
    }
  };

  return (
    <div className="tag-field">
      <div className="tag-field__header">
        <div>
          <span>{label}</span>
          {helperText ? <small>{helperText}</small> : null}
        </div>
        <strong>{values.length} 条</strong>
      </div>
      <div className="tag-cluster">
        {values.length === 0 ? <p className="tag-field__empty">还没有内容，先写最关键的 1 到 3 条。</p> : null}
        {values.map((value, index) => (
          <button
            aria-label={`移除${label}`}
            className="tag-pill"
            key={`${label}-${index}-${value}`}
            onClick={() => onChange(values.filter((_, itemIndex) => itemIndex !== index))}
            type="button"
          >
            <span>{value}</span>
            <X size={14} />
          </button>
        ))}
      </div>
      <div className="tag-input-row">
        <input
          onChange={(event) => setDraft(event.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          value={draft}
        />
        <button className="button button--ghost" onClick={commitDraft} type="button">
          <Plus size={16} />
          {addLabel}
        </button>
      </div>
    </div>
  );
}
