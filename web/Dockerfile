# ============================================================
# BlockExplore 前端 Dockerfile
# 使用 nginx 提供静态文件服务
# ============================================================

# 阶段1: 构建前端
FROM node:20-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

# 阶段2: nginx 运行时
FROM nginx:stable-alpine3.23-perl
# 复制构建产物到 nginx 目录
COPY --from=builder /app/dist /usr/share/nginx/html
# 复制 nginx 配置
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
