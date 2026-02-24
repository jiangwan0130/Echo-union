# Echo Union — 学生会值班管理系统（后端）

基于 Go + Gin + GORM + PostgreSQL 的值班管理系统后端服务。

## 技术栈

| 类别 | 技术 | 版本 |
|------|------|------|
| 语言 | Go | 1.21+ |
| Web 框架 | Gin | 1.9+ |
| ORM | GORM | 2.x |
| 数据库 | PostgreSQL | 15+ |
| 缓存 | Redis | 7+ |
| 认证 | JWT (golang-jwt) | 5.x |
| 配置 | Viper | 1.x |
| 日志 | Zap | 1.x |

## 项目结构

```
backend/
├── cmd/server/main.go              # 应用入口
├── config/
│   ├── config.go                    # 配置加载（Viper）
│   ├── config.yaml                  # 配置文件（.gitignore 忽略）
│   └── config.example.yaml         # 配置模板
├── internal/
│   ├── api/
│   │   ├── handler/                 # HTTP 处理器（参数校验、响应封装）
│   │   ├── middleware/              # 中间件（JWT、CORS、日志）
│   │   └── router/                  # 路由注册
│   ├── service/                     # 业务逻辑层
│   ├── repository/                  # 数据访问层（接口 + GORM 实现）
│   ├── model/                       # 数据库模型（GORM 结构体）
│   └── dto/                         # 请求/响应数据传输对象
├── pkg/
│   ├── database/                    # 数据库连接初始化
│   ├── logger/                      # 日志初始化
│   ├── jwt/                         # JWT 签发与验证
│   └── response/                    # 统一响应封装
├── init.sql                         # 数据库初始化脚本
├── .env.example                     # 环境变量模板
├── .gitignore
├── go.mod
└── README.md
```

## 快速开始

### 前置条件

- Go 1.21+
- PostgreSQL 15+
- Redis 7+（可选，V1 可暂不依赖）

### 1. 克隆与配置

```bash
cd backend

# 复制配置文件
cp config/config.example.yaml config/config.yaml
# 编辑配置，填写数据库密码、JWT Secret 等
```

### 2. 初始化数据库

```bash
# 创建数据库
createdb echo_union

# 执行初始化脚本
psql -d echo_union -f init.sql
```

### 3. 安装依赖

```bash
go mod tidy
```

### 4. 启动服务

```bash
go run cmd/server/main.go
```

服务启动后访问：
- 健康检查：`GET http://localhost:8080/health`
- API 基础路径：`/api/v1`

## API 概览

| 模块 | 路径前缀 | 说明 |
|------|----------|------|
| 认证 | `/api/v1/auth` | 登录、注册、刷新Token、邀请码 |
| 用户 | `/api/v1/users` | 用户信息、列表管理 |
| 部门 | `/api/v1/departments` | 📝 待实现 |
| 学期 | `/api/v1/semesters` | 📝 待实现 |
| 排班 | `/api/v1/schedules` | 📝 待实现 |
| 换班 | `/api/v1/swaps` | 📝 待实现 |
| 签到 | `/api/v1/duties` | 📝 待实现 |
| 通知 | `/api/v1/notifications` | 📝 待实现 |

## 架构分层

```
请求 → Router → Middleware → Handler → Service → Repository → Database
                                ↑           ↑           ↑
                              DTO层       业务逻辑    GORM 操作
```

- **Handler**：参数校验、调用 Service、封装响应（不含业务逻辑）
- **Service**：核心业务处理、事务管理（不含 SQL/HTTP 细节）
- **Repository**：数据访问抽象，封装 GORM 操作（接口 + 实现分离）
- **Model**：数据库表的 Go 结构体映射

## 许可证

MIT
