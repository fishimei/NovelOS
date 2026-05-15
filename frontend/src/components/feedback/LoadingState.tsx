type LoadingStateProps = {
  label?: string;
};

export function LoadingState({ label = '加载中' }: LoadingStateProps) {
  return <div className="loading-state">{label}</div>;
}
