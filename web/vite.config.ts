// Vite 配置文件
// Vite 是前端构建工具，类似于 Webpack，但更快
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,       // 前端开发服务器端口
    proxy: {
      // 代理 API 请求到后端（解决跨域问题）
      '/api': {
        target: 'http://localhost:8080',  // query-api 服务
        changeOrigin: true,
      },
    },
  },
})
