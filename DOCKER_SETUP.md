# Docker 本地开发环境指南

## 前置要求

- Docker Desktop 已安装并运行
- Docker Compose 已安装（Docker Desktop 已包含）

## 快速启动

### 1. 启动容器

```bash
cd c:\Project\Echo-union

# 启动 PostgreSQL 和 Redis
docker-compose up -d

# 查看容器运行状态
docker-compose ps
```

### 2. 验证服务

```bash
# 测试 PostgreSQL
docker exec echo_union_postgres psql -U echo_union -d echo_union -c "SELECT version();"

# 测试 Redis
docker exec echo_union_redis redis-cli -a redis_password ping
```

### 3. 查看日志

```bash
# 查看所有日志
docker-compose logs -f

# 仅查看 Redis 日志
docker-compose logs -f redis

# 仅查看 PostgreSQL 日志
docker-compose logs -f postgres
```

## 配置信息（可用于 config.yaml）

```yaml
server:
  port: 8080
  base_url: http://localhost:8080

db:
  host: localhost           # 或服务名 postgres（在容器内）
  port: 5432
  name: echo_union
  user: echo_union
  password: echo_union_password
  sslmode: disable
  timezone: Asia/Shanghai

redis:
  addr: localhost:6379      # 或服务名 redis:6379（在容器内）
  password: redis_password
  db: 0
```

## 管理命令

```bash
# 停止容器（但保留数据）
docker-compose down

# 停止容器并删除数据
docker-compose down -v

# 进入 PostgreSQL 命令行
docker exec -it echo_union_postgres psql -U echo_union -d echo_union

# 进入 Redis 命令行
docker exec -it echo_union_redis redis-cli -a redis_password
```

## 常见问题

### Q: 端口被占用
```bash
# 修改 docker-compose.yaml 中的端口映射
# 例如将 5432:5432 改为 5433:5432
```

### Q: 如何重置数据库
```bash
docker-compose down -v
docker-compose up -d
```

### Q: 后端无法连接数据库
确保后端容器与 postgres/redis 在同一网络，或使用 `host.docker.internal` 访问主机服务。

---

**PS:** 如果后端也要容器化，可以创建 `Dockerfile` 并在 docker-compose 中加入后端服务。
