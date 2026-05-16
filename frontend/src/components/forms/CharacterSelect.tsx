// 关系表单使用的角色下拉选择器。
import type { Character } from '../../types/api';

type CharacterSelectProps = {
  label: string;
  value: string;
  characters: Character[];
  onChange: (value: string) => void;
};

export function CharacterSelect({ label, value, characters, onChange }: CharacterSelectProps) {
  return (
    <label className="field">
      <span>{label}</span>
      <select value={value} onChange={(event) => onChange(event.target.value)}>
        <option value="">选择角色</option>
        {characters.map((character) => (
          <option key={character.id} value={character.id}>
            {character.name}
          </option>
        ))}
      </select>
    </label>
  );
}
