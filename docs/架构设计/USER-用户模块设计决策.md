# 用户模块设计决策文档
# User Module Design Decision Record

| 文档信息 | |
|----------|------|
| 模块 | 用户模块 (User) |
| 作者 | Echo-Union Team |
| 创建日期 | 2026-02-25 |
| 技术栈 | Go 1.24 / Gin / GORM / excelize / PostgreSQL |

---

## 一、模块职责边界

用户模块负责以下核心关切点，并与认证模块明确划分边界：

| 职责 | 说明 | 归属模块 |
|------|------|----------|
| **用户 CRUD** | 查询、更新个人信息 | User 模块 |
| **角色分配** | 管理员变更用户角色 | User 模块 |
| **软删除** | 注销用户（保留历史数据） | User 模块 |
| **批量导入** | 管理员通过 Excel 批量创建成员 | User 模块 |
| **密码重置** | 管理员为他人重置密码（生成临时密码） | User 模块 |
| **自身密码修改** | 用户修改自己的密码 | Auth 模块 |
| **登录 / Token 颁发** | 身份认证与会话管理 | Auth 模块 |
| **邀请码注册** | 新用户通过邀请链接注册 | Auth 模块 |

**边界划分原则**：Auth 模块管"你是谁"，User 模块管"你的信息和权限"。`/auth/me` 和 `/users/me` 均返回当前用户信息，前者由 Auth 服务提供（包含 `created_at` 等详情），后者由 User 服务提供，两者保持独立以避免循环依赖。

---

## 二、核心设计决策与权衡

### 2.1 Leader 权限：后端自动过滤 vs 前端传参控制

**问题**：`部门负责人(leader)` 只能查看本部门成员，如何实现数据隔离？

| 方案 | 说明 | 优点 | 缺点 |
|------|------|------|------|
| **A. 后端自动过滤（选用）** | Service 层检测 `callerRole=="leader"` 时强制覆盖 `department_id` 过滤条件 | 安全可靠，前端无需感知权限逻辑 | Service 需要接收 `callerRole` 和 `callerDeptID` 两个上下文参数 |
| B. 前端传参控制 | 前端调用时传 `department_id=自己的部门` | 接口简单 | **安全漏洞**：leader 可伪造 `department_id` 参数查看其他部门数据 |
| C. 数据库行级安全 | PostgreSQL Row Level Security | 最彻底 | 配置复杂，GORM 支持不友好，杀鸡用牛刀 |

**决策**：选方案 A。权限控制必须在服务端执行，前端传参仅作"提示用途"而非安全边界。

**实现细节**：

```go
func (s *userService) List(ctx context.Context, req *dto.UserListRequest, callerRole, callerDeptID string) (...) {
    filters := &repository.UserListFilters{
        DepartmentID: req.DepartmentID,  // 前端传入的过滤（admin 可用）
        Role:         req.Role,
        Keyword:      req.Keyword,
    }
    // leader 的部门参数强制覆盖，忽略前端传入值
    if callerRole == "leader" {
        filters.DepartmentID = callerDeptID
    }
}
```

**面试追问应对**：
> "如果 leader 通过修改请求参数尝试查看其他部门数据，你如何防止？"  
> → Service 层在处理 List 请求时，一旦检测到调用方角色为 leader，立即用 JWT 中的 `department_id` 声明覆盖请求参数，前端传入的 `department_id` 被完全忽略。JWT 声明在服务端签名，客户端无法伪造。

---

### 2.2 UpdateUser：双角色权限模型

**问题**：更新用户信息应支持"本人修改自己"和"管理员修改任意人"两种场景。

**字段级权限矩阵**：

| 字段 | 本人 | 管理员 |
|------|------|--------|
| `name` | ✅ 可改 | ✅ 可改 |
| `email` | ✅ 可改 | ✅ 可改 |
| `department_id` | ❌ 禁止 | ✅ 可改 |
| `role` | ❌ 禁止 | 通过独立接口 `/role` 操作 |
| `password` | 通过 `/auth/password` | 通过 `/users/:id/reset-password` |

**路由权限设计**：

```
PUT /users/:id        — router 层不做角色限制，由 Service 层做细粒度鉴权
PUT /users/:id/role   — router 层 middleware.RoleAuth("admin")
DELETE /users/:id     — router 层 middleware.RoleAuth("admin")
```

**设计权衡**：`PUT /users/:id` 放行所有认证用户，在 Service 层判断 `callerID == targetID` 或 `callerRole == "admin"`。这避免了在路由层写复杂的"本人或管理员"逻辑，路由层只做粗粒度的"已登录"校验。

---

### 2.3 Excel 批量导入：完整实现 vs 简化方案

**原始备选方案**：

| 方案 | 描述 | 选择理由 |
|------|------|----------|
| **A. Excel (.xlsx) 上传（选用）** | 引入 `excelize` 解析，支持中文列名 | API 文档明确定义，业务场景更直观 |
| B. JSON 数组批量创建 | `POST /users/bulk` body 传 JSON 数组 | 技术上更简单，但不符合文档约定，业务人员习惯用 Excel |
| C. CSV 上传 | 解析更简单，无需第三方库 | 不支持中文内容的字符编码，业务人员不习惯 |

**决策**：选方案 A。`excelize` 库成熟稳定（GitHub 16k+星），引入成本低，与 API 文档保持一致。

**导入策略：逐行处理 vs 批量事务**：

| 策略 | 说明 | 优点 | 缺点 |
|------|------|------|------|
| **逐行处理（选用）** | 每行独立校验和写入 | 部分成功、精确报告失败行号 | 少量行间重复查询 |
| 全量事务 | 全部成功才提交，否则全部回滚 | 数据一致性强 | 任意一行失败导致全部丢弃，用户体验差 |

**决策**：逐行处理更符合业务需求——批量导入 50 人时，因第 30 行邮箱重复导致前 29 行全部丢弃，对用户而言不可接受。

**列名解析**：支持中英文双语列头，灵活匹配：

```go
switch strings.ToLower(strings.TrimSpace(h)) {
case "姓名", "name":       idx["name"] = i
case "学号", "student_id": idx["student_id"] = i
case "邮箱", "email":      idx["email"] = i
case "部门", "department": idx["department"] = i
}
```

**导入默认密码规则**：

```
默认密码 = "Ec" + 学号后6位
示例：学号 2024001 → 默认密码 Ec024001
```

| 设计考量 | 说明 |
|----------|------|
| 满足密码强度 | "Ec" 提供字母，后6位学号提供数字，总长度 8 位 |
| 可预测性 | 管理员知道规律，可线下告知员工 |
| 强制首次修改 | 导入时设 `must_change_password=true`，首次登录强制改密 |
| 安全性 | 非随机，存在被猜测风险，但 `must_change_password` 机制限制了暴露窗口 |

**面试追问应对**：
> "批量导入的默认密码用学号推算，是否不安全？"  
> → 有一定风险——学号通常半公开。缓解措施：(1) `must_change_password=true` 强制首次登录即改密；(2) 如安全要求更高，可改为随机临时密码并通过邮件单独发送，但这依赖邮件系统就绪。V1 阶段折中选择。

---

### 2.4 ResetPassword：随机密码 vs 固定规则

**管理员重置他人密码的两种方案**：

| 方案 | 实现 | 优点 | 缺点 |
|------|------|------|------|
| **A. 随机临时密码（选用）** | 生成 8 位加密随机密码，响应体返回 | 每次不同，较安全 | 管理员需将密码传递给用户（线下操作） |
| B. 固定规则（如学号后6位） | 按规则计算，无需返回给管理员 | 用户可自行推算 | 可预测，任何知道规律的人都能猜出 |
| C. 邮件发送重置链接 | 生成 token，发邮件 | 最安全，无需管理员中转 | 依赖邮件模块就绪（V1 未实现） |

**决策**：选方案 A。V1 阶段邮件模块未就绪，随机密码由管理员线下通知用户是可接受的操作模式。

**随机密码生成实现（`generateTempPassword`）**：

```go
// 使用 crypto/rand，非 math/rand
// 保证：至少1个字母 + 至少1个数字 → 满足密码强度要求
// Fisher-Yates 洗牌 → 消除位置偏差
func generateTempPassword(length int) (string, error) {
    // 1. 强制第0位取字母
    // 2. 强制第1位取数字
    // 3. 剩余位随机取全集
    // 4. Fisher-Yates 洗牌打乱顺序
}
```

**为什么用 Fisher-Yates 洗牌？**  
直接"确保第0位是字母"的生成方式会导致密码首位一定是字母，产生可预测的模式偏差。Fisher-Yates 洗牌在 crypto/rand 的支持下提供均匀分布。

---

### 2.5 自保护规则：为什么禁止管理员操作自己？

| 操作 | 自保护规则 | 错误码 |
|------|-----------|--------|
| `PUT /users/:id/role` (self) | 禁止，返回 12002 | `ErrUserSelfRoleChange` |
| `DELETE /users/:id` (self) | 禁止，返回 12003 | `ErrUserSelfDelete` |

**设计动机**：

1. **防止系统锁死**：若系统唯一管理员将自己降级为 member，将无法再创建新管理员（无权执行角色分配），需要直接操作数据库恢复。
2. **防止意外操作**：管理员误删自己账号，影响当前会话和后续操作。
3. **一致性**：操作"自己"和操作"他人"是两类语义不同的操作，分开处理更清晰。

**未解决的边界情况**：若系统存在多个管理员，A 可以删除 B，但无法删除自己。这是合理的——操作自己应使用"修改密码"、"更新资料"等专用接口，而非通用的 User CRUD 接口。

---

### 2.6 软删除而非硬删除

**选用 GORM 软删除（`gorm.DeletedAt`）**：

```go
// 软删除实现
func (r *userRepo) Delete(ctx context.Context, id string, deletedBy string) error {
    return r.db.WithContext(ctx).
        Model(&model.User{}).
        Where("user_id = ?", id).
        Updates(map[string]interface{}{
            "deleted_by": deletedBy,
            "deleted_at": gorm.Expr("NOW()"),
        }).Error
}
```

**为什么选软删除？**

| 考量 | 说明 |
|------|------|
| **历史数据完整性** | 被删除用户的历史值班记录、签到记录需要保留，用于统计和审计 |
| **外键约束** | `duty_records`、`schedules` 等表关联 `user_id`，硬删除会触发 FK 约束错误 |
| **可恢复性** | 误删后可通过清空 `deleted_at` 恢复 |
| **审计追踪** | `deleted_by` 字段记录是谁执行了删除操作 |

**软删除的代价**：查询自动附加 `WHERE deleted_at IS NULL`（GORM 内置），需注意唯一索引需配合 `WHERE deleted_at IS NULL` 条件（`init.sql` 中已按此设计）。

---

## 三、关键机制详解

### 3.1 ListWithFilters：可组合的多条件筛选

用户列表支持四个独立可组合的筛选条件：

```
GET /users?page=1&page_size=20&department_id=uuid&role=member&keyword=张
```

**Repository 层实现**：

```go
type UserListFilters struct {
    DepartmentID string
    Role         string
    Keyword      string
}

func (r *userRepo) ListWithFilters(ctx context.Context, filters *UserListFilters, offset, limit int) (...) {
    db := r.db.WithContext(ctx).Model(&model.User{})
    
    if filters.DepartmentID != "" {
        db = db.Where("department_id = ?", filters.DepartmentID)
    }
    if filters.Role != "" {
        db = db.Where("role = ?", filters.Role)
    }
    if filters.Keyword != "" {
        like := "%" + filters.Keyword + "%"
        db = db.Where("name ILIKE ? OR student_id ILIKE ?", like, like)  // 大小写不敏感
    }
    
    db.Count(&total)  // 带筛选条件的总数
    db.Preload("Department").Offset(offset).Limit(limit).Find(&users)
}
```

**注意**：`Count` 和 `Find` 使用同一个带条件的 `db` 实例，确保分页总数与当前筛选结果一致。使用 PostgreSQL 的 `ILIKE` 而非 MySQL 的 `LIKE`，原生支持大小写不敏感搜索。

---

### 3.2 DTO 分层设计

用户模块 DTO 拆分为两个文件，避免单文件臃肿：

```
dto/
├── auth.go     — 认证模块 DTO（LoginRequest, RegisterRequest 等）
├── response.go — 共享响应 DTO（UserResponse, DepartmentResponse, PaginationRequest 等）
└── user.go     — 用户模块专用 DTO（UserListRequest, UpdateUserRequest 等）  ← 新增
```

**`UpdateUserRequest` 使用指针类型字段**：

```go
type UpdateUserRequest struct {
    Name         *string `json:"name"          binding:"omitempty,min=2,max=20"`
    Email        *string `json:"email"         binding:"omitempty,email"`
    DepartmentID *string `json:"department_id" binding:"omitempty,uuid"`
}
```

**为什么用 `*string` 而不是 `string`？**  
区分"客户端未传该字段"（`nil`）和"客户端显式传空字符串"（`""`）。PATCH 语义要求只更新传入的字段——若用 `string`，无法区分"没传"和"传了空值"，会导致意外覆盖。

---

### 3.3 批量导入：Department 名称解析

导入时 Excel 填写的是部门名称（字符串），数据库存的是 `department_id`（UUID）。

**方案对比**：

| 方案 | 描述 | 性能 |
|------|------|------|
| **预加载 Map（选用）** | 一次性加载所有部门，构建 `名称→实体` Map，之后全内存查找 | O(n) 一次 DB 查询 |
| 逐行查库 | 每行导入时按名称查询部门表 | O(rows × DB_RTT) |

```go
func (s *userService) buildDepartmentMap(ctx context.Context) (map[string]*model.Department, error) {
    departments, _ := s.repo.Department.List(ctx)
    m := make(map[string]*model.Department)
    for i := range departments {
        m[departments[i].Name] = &departments[i]
    }
    return m, nil
}
```

部门通常数量极少（≤20），全量加载内存占用可忽略不计，换来显著的查询次数减少。

---

## 四、分层架构与接口设计

### 4.1 Service 接口定义

```go
type UserService interface {
    GetByID(ctx context.Context, id string) (*dto.UserResponse, error)
    List(ctx context.Context, req *dto.UserListRequest, callerRole, callerDeptID string) ([]dto.UserResponse, int64, error)
    Update(ctx context.Context, id string, req *dto.UpdateUserRequest, callerID, callerRole string) (*dto.UserResponse, error)
    Delete(ctx context.Context, id string, callerID string) error
    AssignRole(ctx context.Context, id string, req *dto.AssignRoleRequest, callerID string) error
    ResetPassword(ctx context.Context, id string, callerID string) (*dto.ResetPasswordResponse, error)
    ImportUsers(ctx context.Context, rows []ImportUserRow) (*dto.ImportUserResponse, error)
}
```

**参数设计说明**：`callerID`、`callerRole`、`callerDeptID` 这些"调用方上下文"从 Handler 层从 Gin Context 提取后传入，Service 不依赖 `gin.Context`，保持 Service 层框架无关性，便于单元测试。

### 4.2 Repository 接口扩展

```go
type UserRepository interface {
    Create(ctx context.Context, user *model.User) error
    GetByID(ctx context.Context, id string) (*model.User, error)
    GetByStudentID(ctx context.Context, studentID string) (*model.User, error)
    GetByEmail(ctx context.Context, email string) (*model.User, error)
    Update(ctx context.Context, user *model.User) error
    Delete(ctx context.Context, id string, deletedBy string) error        // 新增：带审计的软删除
    List(ctx context.Context, offset, limit int) ([]model.User, int64, error)
    ListWithFilters(ctx context.Context, filters *UserListFilters, offset, limit int) ([]model.User, int64, error) // 新增：多条件筛选
    BatchCreate(ctx context.Context, users []*model.User) (int, error)    // 新增：批量创建（备用）
}
```

`List` 委托给 `ListWithFilters（filters=nil）`，保持统一查询路径，避免代码重复。

### 4.3 Handler 层的单一关注点

Handler 只负责：
1. 从请求提取参数（`c.Param`, `c.ShouldBindJSON`, `c.FormFile`）
2. 从 Gin Context 提取调用方信息（`c.Get("user_id")`, `c.Get("role")`）
3. 调用 Service
4. 将业务错误映射到 HTTP 状态码和错误码
5. **一切业务逻辑（权限判断、数据校验）都在 Service 层**

---

## 五、错误码体系

用户模块错误码定义在 `12000-12999` 段：

| 错误码 | 含义 | HTTP 状态码 | 触发场景 |
|--------|------|-------------|----------|
| 12001 | 用户不存在 | 404 | `GetByID`、`Update`、`Delete` 等操作目标不存在 |
| 12002 | 无法修改自己的角色 | 400 | `AssignRole` 操作者 == 目标者 |
| 12003 | 无法删除自己 | 400 | `Delete` 操作者 == 目标者 |
| 12004 | 邮箱已被使用 | 400 | `Update` 时邮箱与他人冲突 |
| 12005 | 部门不存在 | 400 | `Update` 时指定了不存在的 `department_id` |
| 10003 | 无权操作 | 403 | 非管理员尝试修改他人信息或更改部门 |
| 10001 | 参数校验失败 | 400 | binding 失败、文件格式错误、文件过大 |

**统一错误处理方法**：

```go
// Handler 层集中处理，避免重复 if-else
func (h *UserHandler) handleUserError(c *gin.Context, err error) {
    switch {
    case errors.Is(err, service.ErrUserNotFound):        response.NotFound(c, 12001, "用户不存在")
    case errors.Is(err, service.ErrUserSelfRoleChange):  response.BadRequest(c, 12002, "无法修改自己的角色")
    case errors.Is(err, service.ErrUserSelfDelete):      response.BadRequest(c, 12003, "无法删除自己")
    case errors.Is(err, service.ErrEmailExists):         response.BadRequest(c, 12004, "邮箱已被使用")
    case errors.Is(err, service.ErrDepartmentNotFound):  response.BadRequest(c, 12005, "部门不存在")
    case errors.Is(err, service.ErrNoPermission):        response.Forbidden(c, 10003, "无权操作")
    default:                                             response.InternalError(c)
    }
}
```

---

## 六、安全设计清单

| 安全措施 | 实现位置 | 说明 |
|----------|----------|------|
| 角色权限中间件 | `middleware/auth.go` | `RoleAuth("admin")` 防止 member 访问管理接口 |
| Service 层细粒度鉴权 | `user_service.go` | 本人/管理员双模式，字段级权限控制 |
| 自保护规则 | `user_service.go` | 禁止管理员降级/删除自己 |
| Leader 数据隔离 | `user_service.go` | 强制覆盖 department_id 过滤，防参数伪造 |
| 软删除 + 审计 | `user_repo.go` | 记录 `deleted_by`，操作可追溯 |
| bcrypt 重置密码 | `user_service.go` | 临时密码经 bcrypt 哈希存储 |
| 加密随机临时密码 | `user_service.go` | `crypto/rand` + Fisher-Yates 洗牌 |
| 强制首次改密 | `user_service.go` | 导入/重置后设 `must_change_password=true` |
| 文件上传限制 | `user_handler.go` | 仅接受 `.xlsx`，限制 5MB 大小 |
| Context 传递 | 全链路 | 每个 DB 调用携带 `ctx`，支持超时取消 |
| 邮箱唯一性校验 | `user_service.go` | Update 时防止邮箱被他人占用 |

---

## 七、可扩展性设计

### 7.1 当前未实现但预留的扩展点

| 功能 | 当前状态 | 扩展方式 |
|------|----------|----------|
| `duty_required` 字段 | 未在用户列表返回 | 等排班学期模块就绪后，`ListWithFilters` 联表 `user_semester_assignments` 查询 |
| 邮件发送重置密码 | 未实现 | `ResetPassword` 可在生成临时密码后调用 Notification Service 发邮件 |
| 用户操作审计日志 | 未实现 | Service 层在 Create/Update/Delete 成功后异步写审计表 |
| 批量删除 | 未实现 | Repository 层增加 `BatchDelete(ids []string, deletedBy string)` |
| 用户搜索全文索引 | 未实现 | 当前 `ILIKE` 查询无索引；如数据量大，可对 `name`、`student_id` 加 GIN 全文索引 |
| 头像上传 | 未实现 | User 模型增加 `avatar_url` 字段，上传至对象存储（OSS/MinIO） |
| 组织架构树 | 未实现 | Department 模型增加 `parent_id`，支持多层级结构 |

### 7.2 BatchCreate 的保留逻辑

`BatchCreate` 接口已定义但当前导入逻辑不直接使用（改用逐行 `Create` 以精确报告错误行号）。保留 `BatchCreate` 接口有以下用途：

- 未来支持"全成功才导入"的严格模式
- 初始化种子数据脚本可使用
- 数据迁移场景的批量写入

---

## 八、测试策略

### 8.1 单元测试覆盖（28 个，全部通过）

| 测试场景 | 数量 |
|----------|------|
| `GetByID`（成功 / 不存在） | 2 |
| `List`（admin 全量 / leader 自动过滤 / 角色筛选 / 关键词搜索） | 4 |
| `Update`（本人改自己 / admin 改部门 / 非admin不能改部门 / 不能改他人 / 邮箱重复 / 用户不存在） | 6 |
| `Delete`（成功 / 自删保护 / 用户不存在） | 3 |
| `AssignRole`（成功 / 自改保护 / 用户不存在） | 3 |
| `ResetPassword`（成功验证哈希+MustChange / 用户不存在） | 2 |
| `ImportUsers`（全成功 / 部门不存在 / 学号重复 / 空字段 / 混合场景） | 5 |
| `generateTempPassword`（20次迭代，验证长度+字母+数字） | 1 |
| Auth 模块回归测试（无变化，一并验证） | 20 |
| **合计** | **48** |

### 8.2 Mock 策略：扩展现有 Mock 结构

用户模块测试复用了 Auth 模块已有的 Mock Repo，在原有 `mockUserRepo` 基础上扩展：

```go
// 新增方法以满足扩展后的 UserRepository 接口
func (m *mockUserRepo) Delete(_ context.Context, id string, _ string) error {
    for key, u := range m.users {
        if u.UserID == id { delete(m.users, key) }
    }
    return nil
}

func (m *mockUserRepo) ListWithFilters(_ context.Context, filters *repository.UserListFilters, offset, limit int) (...) {
    // 内存过滤：DepartmentID + Role + Keyword（contains 匹配）
}
```

**共享 Mock 的取舍**：

| 方式 | 优点 | 缺点 |
|------|------|------|
| **共享 Mock（选用）** | 单一 mock 文件，维护成本低 | Mock 变更影响所有测试 |
| 每个测试文件独立 Mock | 隔离性强 | 大量代码重复 |

本项目 Repository 接口方法数量适中（≤10 个），共享 Mock 是合理选择。若接口超过 20 个方法，应考虑 `gomock` 按需生成。

### 8.3 端到端验证结果（真实 PostgreSQL + Redis）

```
✅ GET  /users/me                        → 200 返回当前用户信息
✅ GET  /users                           → 200 返回分页列表（含部门关联）
✅ GET  /users?role=leader               → 200 角色筛选 1 条
✅ GET  /users/:id                       → 200 用户详情
✅ PUT  /users/:id                       → 200 更新 name + email 成功
✅ PUT  /users/:id/role                  → 200 角色变更 member → leader
✅ POST /users/:id/reset-password        → 200 返回 8 位临时密码，MustChange=true
✅ POST /users/import (Excel 3行)        → 200 success=2, failed=1（部门不存在行精确报告）
✅ DELETE /users/:id                     → 200 软删除成功，列表总数减1
✅ PUT  /users/:id/role (self)           → 400 code=12002 自改保护触发
✅ DELETE /users/:id (self)              → 400 code=12003 自删保护触发
```

---

## 九、面试高频问题与回答策略

**Q1: 用户导入时如何防止并发下的学号重复？**  
→ 数据库层面，`users` 表有唯一索引 `uk_users_student_id UNIQUE (student_id) WHERE deleted_at IS NULL`。即使 Service 层并发检查通过，最终由数据库唯一约束兜底，返回 DB 错误并在错误报告中标注对应行号。V1 场景下并发导入概率极低，此方案足够。高并发场景可引入分布式锁或 PostgreSQL 的 `INSERT ... ON CONFLICT DO NOTHING`。

**Q2: `ILIKE` 搜索在大数据量下的性能问题如何解决？**  
→ `ILIKE '%keyword%'` 前缀通配无法使用 B-Tree 索引，O(n) 全表扫描。解决方案：(1) 数据量小（≤1万）时可接受；(2) 数据量大时对 `name`、`student_id` 建 PostgreSQL GIN 全文索引（`to_tsvector`），或引入 Elasticsearch；(3) 限制搜索关键词长度（已有 max=50 校验）减少扫描范围。

**Q3: 为什么 `UpdateUserRequest` 用指针字段？**  
→ 实现 PATCH 语义：区分"未提供"（`nil`）和"提供为空"（`""`）。使用 `string` 类型时，省略的字段默认为 `""` 会意外清空已有数据。指针字段配合 `binding:"omitempty"` 标签，只校验非 nil 的字段。

**Q4: 软删除后如何查询被删除的用户（如审计需求）？**  
→ GORM 软删除自动在所有查询附加 `WHERE deleted_at IS NULL`。查询被删除记录需使用 `db.Unscoped().Where(...)`，绕过软删除过滤。可在 Repository 增加 `GetDeletedByID` 等专用方法供审计功能使用。

**Q5: 如果要支持 SSO 单点登录，用户模块如何演进？**  
→ 用户表增加 `external_id`（SSO 提供商的用户 ID）和 `auth_provider`（如 `google`, `dingtalk`）字段。注册流程增加 SSO 回调处理，首次 SSO 登录自动创建用户（无需邀请码），后续通过 `external_id` 关联。现有密码字段可置空或保留供本地登录备用。

**Q6: leader 角色的权限边界设计有哪些其他方案？**  
→ 本系统选择了"隐式过滤"（自动限定部门），另外两种常见方案：(1) RBAC 资源级权限表——`permissions` 表定义细粒度操作权限，灵活但维护复杂度高；(2) ABAC 属性级访问控制——基于用户属性+资源属性+环境条件动态判断，适合权限规则经常变化的场景。本系统角色固定为 3 类，隐式过滤足够且实现简单。

**Q7: 如何保证用户列表的分页总数与实际返回数量一致（避免条件竞争）？**  
→ `Count` 和 `Find` 使用同一个带条件的 GORM `db` 对象实例，查询条件完全相同。在 `Count` 和 `Find` 之间若有并发写入，理论上可能出现不一致（幻读）。解决方案：(1) 接受最终一致性（本系统选用，后台管理场景可接受）；(2) 在事务中执行两个查询；(3) 使用 `SELECT COUNT(*)` 子查询和 `LIMIT/OFFSET` 在同一 SQL 完成（一次 RTT）。

---

*文档版本 v1.0 | 2026-02-25*
