# Echo Union — 学生会值班管理系统（后端）

基于 Go + Gin + GORM + PostgreSQL 的值班管理系统后端服务。

## 技术栈

| 类别 | 技术 | 版本 |
|------|------|------|
| 语言 | Go | 1.24+ |
| Web 框架 | Gin | 1.9+ |
| ORM | GORM | 2.x |
| 数据库 | PostgreSQL | 15+ |
| 缓存 | Redis | 7+ |
| 认证 | JWT (golang-jwt) | 5.x |
| 配置 | Viper | 1.x |
| 日志 | Zap | 1.x |
| Excel 导出 | excelize | 2.x |
| ICS 解析 | golang-ical | — |

## 主要功能特性

- **邀请码注册机制** — 管理员/部长生成邀请码，用户凭码注册并自动关联部门
- **ICS 课表导入** — 解析 ICS 格式课表文件，自动识别不可用时间段
- **自动排班引擎** — 基于排班规则、成员课表和不可用时间生成排班方案
- **排班冲突验证** — 调整排班时自动校验时间冲突并推荐候选人
- **Excel 排班表导出** — 导出排班结果为 Excel 文件
- **JWT + Redis Token 黑名单** — 安全认证，支持登出令牌立即失效
- **RBAC 权限控制** — 支持 admin / leader / member 三级角色

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
│   ├── errors/                      # 自定义错误类型
│   ├── jwt/                         # JWT 签发与验证
│   ├── logger/                      # 日志初始化
│   ├── redis/                       # Redis 客户端封装
│   └── response/                    # 统一响应封装
├── init.sql                         # 数据库初始化脚本（含种子数据）
├── go.mod
└── README.md
```

## 快速开始

### 前置条件

- Go 1.24+
- PostgreSQL 15+
- Redis 7+（降级模式可运行，但 Token 黑名单等功能需要 Redis）

### 方式一：Docker Compose（推荐）

项目根目录提供了 `docker-compose.yaml`，可一键启动 PostgreSQL + Redis：

```bash
# 在项目根目录
docker-compose up -d

# 然后进入 backend 目录启动服务
cd backend
cp config/config.example.yaml config/config.yaml
# 编辑 config.yaml，确认数据库和 Redis 连接信息
go run cmd/server/main.go
```

### 方式二：手动配置

#### 1. 配置

```bash
cd backend
cp config/config.example.yaml config/config.yaml
# 编辑配置，填写数据库密码、JWT Secret 等
```

#### 2. 初始化数据库

```bash
createdb echo_union
psql -d echo_union -f init.sql
```

#### 3. 安装依赖并启动

```bash
go mod tidy
go run cmd/server/main.go
```

服务启动后访问：
- 健康检查：`GET http://localhost:8080/health`
- API 基础路径：`/api/v1`

## 配置说明

配置文件为 `config/config.yaml`，通过 Viper 加载，支持 `ECHO_` 前缀的环境变量覆盖。

| 配置段 | 说明 | 关键字段 |
|--------|------|----------|
| `server` | 服务配置 | `port`、`base_url`、`cors.allow_origins` |
| `db` | 数据库 | `host`、`port`、`name`、`user`、`password`、`max_open_conns`、`max_idle_conns` |
| `redis` | Redis | `addr`、`password`、`db` |
| `auth` | 认证 | `jwt_secret`、`access_token_ttl`、`refresh_token_ttl_default`、`cookie.*` |
| `mail` | 邮件 | `smtp_host`、`smtp_port`、`username`、`password`、`from` |
| `log` | 日志 | `level`、`format` |
| `feature` | 功能开关 | `oa_import_enabled` |

详见 `config/config.example.yaml` 中的完整模板。

## API 概览

| 模块 | 路径前缀 | 状态 | 说明 |
|------|----------|------|------|
| 认证 | `/api/v1/auth` | ✅ | 登录、注册、刷新 Token、邀请码、登出、修改密码 |
| 用户 | `/api/v1/users` | ✅ | 用户信息、列表管理、角色变更、重置密码、批量导入 |
| 部门 | `/api/v1/departments` | ✅ | CRUD + 部门成员查看、值班成员管理 |
| 学期 | `/api/v1/semesters` | ✅ | CRUD + 当前学期查询、学期激活 |
| 时间段 | `/api/v1/time-slots` | ✅ | 完整 CRUD |
| 地点 | `/api/v1/locations` | ✅ | 完整 CRUD |
| 系统配置 | `/api/v1/system-config` | ✅ | 查看 / 更新系统配置 |
| 排班规则 | `/api/v1/schedule-rules` | ✅ | 查看列表 / 详情 / 更新 |
| 课表时间表 | `/api/v1/timetables` | ✅ | ICS 导入、不可用时间管理、提交、进度查看 |
| 排班 | `/api/v1/schedules` | ✅ | 自动排班、查看、调整、验证、候选人、发布、变更日志 |
| 导出 | `/api/v1/export` | ✅ | 排班表 Excel 导出 |
| 换班 | `/api/v1/swaps` | 📝 | 待实现 |
| 签到 | `/api/v1/duties` | 📝 | 待实现 |
| 通知 | `/api/v1/notifications` | 📝 | 待实现 |

<details>
<summary><strong>详细 API 端点列表</strong>（点击展开）</summary>

### 认证 `/api/v1/auth`

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| POST | `/auth/login` | 公开 | 登录 |
| POST | `/auth/register` | 公开 | 注册（需邀请码） |
| POST | `/auth/refresh` | 公开 | 刷新 Token |
| GET | `/auth/invite/:code` | 公开 | 验证邀请码 |
| POST | `/auth/logout` | 登录用户 | 登出 |
| GET | `/auth/me` | 登录用户 | 获取当前用户信息 |
| PUT | `/auth/password` | 登录用户 | 修改密码 |
| POST | `/auth/invite` | admin/leader | 生成邀请码 |

### 用户 `/api/v1/users`

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/users/me` | 登录用户 | 获取个人信息 |
| GET | `/users` | admin/leader | 用户列表 |
| GET | `/users/:id` | admin/leader | 用户详情 |
| PUT | `/users/:id` | Service 层鉴权 | 更新用户信息 |
| DELETE | `/users/:id` | admin | 删除用户 |
| PUT | `/users/:id/role` | admin | 变更角色 |
| POST | `/users/:id/reset-password` | admin | 重置密码 |
| POST | `/users/import` | admin | 批量导入用户 |

### 部门 `/api/v1/departments`

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/departments` | 登录用户 | 部门列表 |
| GET | `/departments/:id` | 登录用户 | 部门详情 |
| POST | `/departments` | admin | 创建部门 |
| PUT | `/departments/:id` | admin | 更新部门 |
| DELETE | `/departments/:id` | admin | 删除部门 |
| GET | `/departments/:id/members` | admin/leader | 部门成员列表 |
| PUT | `/departments/:id/duty-members` | admin/leader | 更新值班成员 |

### 学期 `/api/v1/semesters`

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/semesters` | 登录用户 | 学期列表 |
| GET | `/semesters/current` | 登录用户 | 当前学期 |
| GET | `/semesters/:id` | 登录用户 | 学期详情 |
| POST | `/semesters` | admin | 创建学期 |
| PUT | `/semesters/:id` | admin | 更新学期 |
| PUT | `/semesters/:id/activate` | admin | 激活学期 |
| DELETE | `/semesters/:id` | admin | 删除学期 |

### 时间段 `/api/v1/time-slots`

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/time-slots` | 登录用户 | 时间段列表 |
| GET | `/time-slots/:id` | 登录用户 | 时间段详情 |
| POST | `/time-slots` | admin | 创建时间段 |
| PUT | `/time-slots/:id` | admin | 更新时间段 |
| DELETE | `/time-slots/:id` | admin | 删除时间段 |

### 地点 `/api/v1/locations`

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/locations` | 登录用户 | 地点列表 |
| GET | `/locations/:id` | 登录用户 | 地点详情 |
| POST | `/locations` | admin | 创建地点 |
| PUT | `/locations/:id` | admin | 更新地点 |
| DELETE | `/locations/:id` | admin | 删除地点 |

### 系统配置 `/api/v1/system-config`

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/system-config` | 登录用户 | 查看系统配置 |
| PUT | `/system-config` | admin | 更新系统配置 |

### 排班规则 `/api/v1/schedule-rules`

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/schedule-rules` | 登录用户 | 排班规则列表 |
| GET | `/schedule-rules/:id` | 登录用户 | 排班规则详情 |
| PUT | `/schedule-rules/:id` | admin | 更新排班规则 |

### 课表/时间表 `/api/v1/timetables`

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| POST | `/timetables/import` | 登录用户 | 导入 ICS 课表 |
| GET | `/timetables/me` | 登录用户 | 查看个人课表 |
| POST | `/timetables/unavailable` | 登录用户 | 添加不可用时间 |
| PUT | `/timetables/unavailable/:id` | 登录用户 | 更新不可用时间 |
| DELETE | `/timetables/unavailable/:id` | 登录用户 | 删除不可用时间 |
| POST | `/timetables/submit` | 登录用户 | 提交课表 |
| GET | `/timetables/progress` | admin | 全局课表提交进度 |
| GET | `/timetables/progress/department/:id` | admin/leader | 部门课表提交进度 |

### 排班 `/api/v1/schedules`

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| POST | `/schedules/auto` | admin | 自动排班 |
| GET | `/schedules` | 登录用户 | 排班列表 |
| GET | `/schedules/my` | 登录用户 | 我的排班 |
| PUT | `/schedules/items/:id` | admin | 调整排班项 |
| POST | `/schedules/items/:id/validate` | admin | 验证排班项 |
| GET | `/schedules/items/:id/candidates` | admin | 候选人列表 |
| POST | `/schedules/publish` | admin | 发布排班 |
| PUT | `/schedules/published/items/:id` | admin | 调整已发布排班项 |
| GET | `/schedules/change-logs` | admin | 排班变更日志 |
| POST | `/schedules/:id/scope-check` | admin | 排班范围检查 |

### 导出 `/api/v1/export`

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/export/schedule` | admin/leader | 导出排班表（Excel） |

</details>

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
- **DTO**：请求/响应的数据传输对象，Handler 与 Service 之间的契约

## 测试

```bash
# 运行所有单元测试
go test ./internal/service/... -v

# 运行集成测试（需要数据库连接）
go test ./internal/repository/... -v
```

## 许可证

MIT
