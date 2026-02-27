# Echo Union — 学生会值班管理系统

> 为学校学生会打造的自动化值班管理平台，将排班时间从数天缩短至 10 分钟。

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](https://go.dev/)
[![React](https://img.shields.io/badge/React-19-61DAFB?logo=react)](https://react.dev/)
[![Ant Design](https://img.shields.io/badge/Ant%20Design-6-0170FE?logo=antdesign)](https://ant.design/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

## 功能概览

| 模块 | 说明 | 状态 |
|------|------|------|
| 🔐 认证与权限 | 学号登录、JWT 认证、三级角色（管理员 / 负责人 / 成员） | ✅ |
| 👥 用户管理 | Excel 批量导入、角色分配、密码重置 | ✅ |
| 🏢 部门管理 | 部门 CRUD、值班人员勾选 | ✅ |
| 📅 时间收集 | ICS 课表导入、不可用时间标记、提交进度追踪 | ✅ |
| 🤖 自动排班 | 约束求解自动排班、规则校验、手动调整、排班发布 | ✅ |
| ⚙️ 系统配置 | 学期 / 时间段 / 地点 / 排班规则 / 全局参数管理 | ✅ |
| 📊 数据导出 | 排班表 Excel 导出 | ✅ |
| 🔄 换班管理 | 申请换班、目标确认、管理员审批 | 🔜 二期 |
| ✅ 签到签退 | Web 签到 / 签退、迟到缺席判定、异常记录 | 🔜 二期 |
| 🔔 通知提醒 | 站内通知中心、邮件提醒 | 🔜 二期 |

## 技术栈

**后端**：Go 1.24 · Gin · GORM · PostgreSQL 15 · Redis 7 · JWT

**前端**：React 19 · TypeScript 5 · Vite 6 · Ant Design 6 · Zustand · Tailwind CSS 4

**测试**：Vitest · Playwright · Go testing

**基础设施**：Docker Compose · PostgreSQL 15-alpine · Redis 7-alpine

## 快速开始

### 前置条件

- [Go](https://go.dev/) 1.24+
- [Node.js](https://nodejs.org/) 20+
- [Docker](https://www.docker.com/) & Docker Compose（或手动安装 PostgreSQL 15+ / Redis 7+）

### 1. 启动基础设施

```bash
docker-compose up -d
```

### 2. 启动后端

```bash
cd backend
cp config/config.example.yaml config/config.yaml
# 编辑 config.yaml，至少填写 auth.jwt_secret
go mod tidy
go run cmd/server/main.go
```

后端默认运行在 `http://localhost:8080`，首次启动自动完成数据库迁移。

> **预置测试账号**（密码均为 `admin123`）：
> | 学号 | 角色 | 说明 |
> |------|------|------|
> | `admin` | 管理员 | 系统全部权限 |
> | `leader` | 部门负责人 | 部门管理、查看进度 |
> | `member` | 普通成员 | 提交时间表、查看排班 |

### 3. 启动前端

```bash
cd frontend
npm install
npm run dev
```

前端开发服务器运行在 `http://localhost:5173`，`/api` 请求自动代理到后端。

## 项目结构

```
Echo-union/
├── docker-compose.yaml          # PostgreSQL + Redis 容器编排
├── backend/                     # Go 后端服务
│   ├── cmd/server/main.go       # 应用入口
│   ├── config/                  # 配置管理 (Viper)
│   ├── internal/
│   │   ├── api/
│   │   │   ├── handler/         # HTTP 处理器
│   │   │   ├── middleware/      # JWT / CORS / 日志中间件
│   │   │   └── router/          # 路由注册
│   │   ├── dto/                 # 请求 / 响应数据传输对象
│   │   ├── model/               # 数据库模型 (GORM)
│   │   ├── repository/          # 数据访问层
│   │   └── service/             # 业务逻辑层
│   ├── pkg/                     # 公共工具包 (database/jwt/logger/redis)
│   └── init.sql                 # 数据库 DDL
├── frontend/                    # React 前端应用
│   ├── src/
│   │   ├── components/          # 通用组件
│   │   ├── pages/               # 页面组件
│   │   ├── router/              # 路由配置
│   │   ├── services/            # API 服务层 (Axios)
│   │   ├── stores/              # Zustand 状态管理
│   │   └── types/               # TypeScript 类型定义
│   └── e2e/                     # Playwright E2E 测试
└── docs/                        # 项目文档
    ├── 需求分析/                 # PRD / SRS / 用户故事 / 业务流程
    └── 架构设计/                 # HLD / LLD / API / DB 设计
```

## 架构

```
客户端
  └─► Gin Router
        └─► Middleware (JWT / CORS / Logger)
              └─► Handler
                    └─► Service（业务逻辑）
                          └─► Repository（接口）
                                ├─► PostgreSQL (GORM)
                                └─► Redis（缓存 / Token 黑名单）
```

采用**分层架构 + 依赖注入**，Repository 层基于接口实现，便于单元测试与 Mock。

## 开发指南

### 后端测试

```bash
cd backend

# Service 层单元测试（含 Mock Repository）
go test ./internal/service/... -v

# Repository 集成测试（需数据库连接）
go test ./internal/repository/... -v

# 全部测试
go test ./... -v
```

### 前端测试

```bash
cd frontend

npm run dev              # 启动开发服务器
npm run build            # 生产构建
npm run lint             # ESLint 检查
npm run test:run         # 单元测试（单次运行）
npm run test:coverage    # 测试覆盖率报告
npm run test:e2e         # Playwright E2E 测试
```

### 配置说明

后端配置文件路径：`backend/config/config.yaml`（从 `config.example.yaml` 复制）。
支持以 `ECHO_` 前缀的环境变量覆盖任意配置项。

| 配置段 | 关键字段 | 说明 |
|--------|----------|------|
| `server` | `port`、`cors.allow_origins` | 服务端口与跨域配置 |
| `db` | `host`、`port`、`name`、`user`、`password` | PostgreSQL 连接 |
| `redis` | `addr`、`password` | Redis 连接 |
| `auth` | `jwt_secret`（**必填**）、`access_token_ttl` | JWT 认证配置 |
| `mail` | `smtp_host`、`username`、`password` | SMTP 邮件（二期，可选） |
| `log` | `level`、`format` | 日志级别与格式 |

## 角色说明

| 角色 | 英文标识 | 权限范围 |
|------|----------|---------|
| 管理员 | `admin` | 系统全部功能：用户管理、自动排班、换班审批、系统配置 |
| 部门负责人 | `leader` | 部门成员管理、查看提交进度、查看排班 |
| 成员 | `member` | 提交时间表、查看排班、申请换班、签到签退 |

## 文档索引

| 文档 | 说明 |
|------|------|
| [PRD-值班管理系统](docs/需求分析/PRD-值班管理系统.md) | 产品需求文档 |
| [SRS-软件需求规格说明](docs/需求分析/SRS-软件需求规格说明.md) | 软件需求规格 |
| [HLD-概要设计说明书](docs/架构设计/HLD-概要设计说明书.md) | 系统架构概要设计 |
| [LLD-详细设计说明书](docs/架构设计/LLD-详细设计说明书.md) | 模块详细设计（含二期蓝图） |
| [API-接口设计文档](docs/架构设计/API-接口设计文档.md) | RESTful API 规范 |
| [DB-数据库设计文档](docs/架构设计/DB-数据库设计文档.md) | 数据库表结构设计 |
| [用户故事](docs/需求分析/用户故事-User-Stories.md) | 用户故事与验收标准 |
| [Docker 开发环境指南](DOCKER_SETUP.md) | Docker 环境配置说明 |

## License

[MIT](LICENSE)
