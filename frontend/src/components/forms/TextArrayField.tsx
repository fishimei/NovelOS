// 可复用的 string[] 字段编辑器，用于 goals、fears、rules、constraints 等 OpenAPI 数组字段。
import { Plus, X } from 'lucide-react';

type TextArrayFieldProps = {
  label: string;
  values: string[];
  onChange: (values: string[]) => void;
  placeholder?: string;
};

export function TextArrayField({ label, values, onChange, placeholder }: TextArrayFieldProps) {
  const setValue = (index: number, value: string) => {
    onChange(values.map((item, itemIndex) => (itemIndex === index ? value : item)));
  };

  const removeValue = (index: number) => {
    onChange(values.filter((_, itemIndex) => itemIndex !== index));
  };

  return (
    <label className="field field--stack">
      <span>{label}</span>
      <div className="stack-list">
        {values.map((value, index) => (
          <div className="inline-row" key={index}>
            <input value={value} onChange={(event) => setValue(index, event.target.value)} placeholder={placeholder} />
            <button className="icon-button" type="button" onClick={() => removeValue(index)} aria-label="移除">
              <X size={16} />
            </button>
          </div>
        ))}
        <button className="button button--secondary" type="button" onClick={() => onChange([...values, ''])}>
          <Plus size={16} />
          添加
        </button>
      </div>
    </label>
  );
}
