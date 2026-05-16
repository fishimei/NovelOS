// 共享的 TanStack Query 客户端。默认配置偏保守：查询短时间保鲜，查询失败重试一次，写操作不重试。
import { QueryClient } from '@tanstack/react-query';

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      staleTime: 20_000,
      refetchOnWindowFocus: false,
    },
    mutations: {
      retry: 0,
    },
  },
});
