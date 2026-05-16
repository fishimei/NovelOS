// Vite 开发和构建配置。开发环境中，浏览器发到 /api 的请求会代理到
// localhost:8000 上的 Go 后端。
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8000',
        changeOrigin: true,
      },
    },
  },
});
