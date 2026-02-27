# ============================================================
# 前端多阶段构建
# ============================================================

# ── Stage 1: 构建 ──
FROM node:22-alpine AS builder

WORKDIR /build

COPY package.json package-lock.json* ./
RUN npm ci

COPY . .
RUN npm run build

# ── Stage 2: Nginx 托管 ──
FROM nginx:alpine

# 移除默认配置
RUN rm /etc/nginx/conf.d/default.conf

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /build/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
