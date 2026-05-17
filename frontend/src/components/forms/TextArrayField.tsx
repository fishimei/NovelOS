// 可复用的 string[] 字段编辑器，用于 goals、fears、rules、constraints 等 OpenAPI 数组字段。
import { Plus, X } from 'lucide-react';

type TextArrayFieldProps = {
  label: string;
  values: string[];
  onChange: (values: string[]) => void;
  placeholder?: string;
  addLabel?: string;
  helperText?: string;
};

export function TextArrayField({
  label,
  values,
  onChange,
  placeholder,
  addLabel = '添加条目',
  helperText,
}: TextArrayFieldProps) {
  const setValue = (index: number, value: string) => {
    onChange(values.map((item, itemIndex) => (itemIndex === index ? value : item)));
  };

  const removeValue = (index: number) => {
    onChange(values.filter((_, itemIndex) => itemIndex !== index));
  };

  return (
    <section className="array-editor">
      <div className="array-editor__header">
        <div>
          <span>{label}</span>
          {helperText ? <small>{helperText}</small> : null}
        </div>
        <strong>{values.length} 条</strong>
      </div>
      <div className="stack-list array-editor__list">
        {values.map((value, index) => (
          <div className="array-editor__item" key={index}>
            <textarea
              onChange={(event) => setValue(index, event.target.value)}
              placeholder={placeholder}
              rows={2}
              value={value}
            />
            <button className="icon-button" type="button" onClick={() => removeValue(index)} aria-label="移除">
              <X size={16} />
            </button>
          </div>
        ))}
        <button className="button button--ghost" type="button" onClick={() => onChange([...values, ''])}>
          <Plus size={16} />
          {addLabel}
        </button>
      </div>
    </section>
  );
}
