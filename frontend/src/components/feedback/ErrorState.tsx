// API 查询和写操作失败时使用的通用错误提示。
import { AlertTriangle } from 'lucide-react';

type ErrorStateProps = {
  title?: string;
  message: string;
};

export function ErrorState({ title = '请求失败', message }: ErrorStateProps) {
  return (
    <div className="notice notice--error">
      <AlertTriangle size={18} />
      <div>
        <strong>{title}</strong>
        <p>{message}</p>
      </div>
    </div>
  );
}
