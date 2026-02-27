# ============================================================
# 后端多阶段构建
# ============================================================

# ── Stage 1: 编译 ──
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

# 先复制依赖文件，利用 Docker 缓存层
COPY go.mod go.sum ./
RUN go mod download

# 复制源码并编译
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/server ./cmd/server

# ── Stage 2: 运行 ──
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/server .
COPY config/config.example.yaml ./config/config.yaml

EXPOSE 8080

ENTRYPOINT ["./server"]
