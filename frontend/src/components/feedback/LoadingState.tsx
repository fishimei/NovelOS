// 可复用的加载占位，用于由 query 驱动的页面和面板。
type LoadingStateProps = {
  label?: string;
};

export function LoadingState({ label = '加载中' }: LoadingStateProps) {
  return <div className="loading-state">{label}</div>;
}
