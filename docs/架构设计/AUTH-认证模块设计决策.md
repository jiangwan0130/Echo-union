# 认证模块设计决策文档
# Auth Module Design Decision Record

| 文档信息 | |
|----------|------|
| 模块 | 认证模块 (Auth) |
| 作者 | Echo-Union Team |
| 创建日期 | 2026-02-25 |
| 技术栈 | Go 1.24 / Gin / GORM / JWT / PostgreSQL / Redis |

---

## 一、模块职责边界

认证模块负责以下核心关切点，其余（排班、签到等）明确排除在外：

| 职责 | 说明 |
|------|------|
| **身份认证** | 验证"你是谁"——学号+密码登录 |
| **会话管理** | Access Token 颁发、刷新、吊销 |
| **注册准入** | 邀请码机制控制注册入口，防止开放注册 |
| **授权基础** | 将角色/部门信息注入请求上下文，供下游中间件使用 |
| **密码安全** | bcrypt 哈希存储、密码强度校验、修改密码 |

---

## 二、核心技术选型与权衡

### 2.1 为什么选 JWT，而不是 Session？

| 维度 | JWT (选用) | Server-Side Session |
|------|-----------|---------------------|
| 状态 | 无状态，服务端无需存储 | 有状态，需共享存储 (Redis / DB) |
| 水平扩展 | ✅ 天然支持，无会话粘连 | ⚠️ 需要 Redis 等共享存储 |
| 即时吊销 | ⚠️ 需额外黑名单机制 | ✅ 直接从存储删除 |
| 性能 | ✅ 验证纯本地计算，无 IO | ⚠️ 每次请求需查 Redis/DB |
| 负载 | ⚠️ Token 体积较大 (~300B) | ✅ Cookie 仅携带 Session ID |

**决策**：选 JWT。本系统为校园内部工具，并发规模有限，JWT 的无状态优势更匹配 Go 服务的典型部署方式。吊销问题通过 Redis 黑名单补全，见 §3.1。

**面试追问应对**：
> "JWT 最大的问题是无法即时吊销，你怎么解决的？"  
> → 引入 Redis 黑名单存储已吊销 Token 的 JTI（JWT ID），TTL 与 Token 剩余有效期一致，自动清理。代价是 Logout 路径多一次 Redis 写操作，属于可接受的 trade-off。

---

### 2.2 双 Token 策略（Access Token + Refresh Token）

```
Access Token:  短生命周期 (15min)  → 用于 API 鉴权
Refresh Token: 长生命周期 (24h/7d) → 仅用于换取新 Access Token
```

**为什么不用单一长效 Token？**

- 长效 Token 一旦泄漏，攻击窗口无限长
- 双 Token 将"暴露在网络中"的 Token 有效期缩至 15 分钟
- Refresh Token 路径（`/auth/refresh`）流量极低，可施加更严格的速率限制

**remember_me 影响 Refresh Token TTL：**

| 场景 | Refresh Token TTL |
|------|------------------|
| 普通登录 | 24 小时 |
| 勾选"记住我" | 7 天 |

TTL 策略写入 `config.yaml`，运行时可调，无需改代码。

---

### 2.3 Refresh Token 交付：双模式设计

```
登录响应：
  ┌─ Set-Cookie: refresh_token=...; HttpOnly; SameSite=Lax; Path=/api/v1/auth
  └─ JSON body:  { "refresh_token": "..." }   ← 便于 API 测试工具
  
刷新请求：优先读 Cookie → 回退读 request body
```

**HttpOnly Cookie 的安全意义**：
- XSS 攻击无法通过 `document.cookie` 读取 Refresh Token
- `SameSite=Lax` 防止大多数 CSRF 场景（跨站表单提交不携带 Cookie）
- 生产环境设 `Secure=true`，仅 HTTPS 传输

**为什么同时保留 body 模式？**
- curl / Postman 等工具测试场景，不便管理 Cookie
- 脚本集成场景可直接传 token
- 成本：仅多 10 行解析逻辑，无安全增量风险

---

### 2.4 bcrypt 密码哈希

```go
bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
// DefaultCost = 10（约 100ms/次）
```

**为什么选 bcrypt 而不是 SHA256/MD5？**

- bcrypt 内置 salt（每次生成不同哈希），防彩虹表攻击
- Cost factor 可调，随硬件升级提高安全强度
- SHA256 无 salt、无 cost factor，已不适合密码哈希场景

**Cost Factor 权衡**：

| Cost | 耗时（普通服务器） | 适用场景 |
|------|--------------------|----------|
| 10 (default) | ~100ms | 生产环境（本系统选用） |
| 4 (MinCost) | ~1ms | 单元测试（构造测试数据快速） |

测试代码中使用 `bcrypt.MinCost` 避免测试慢。

---

## 三、关键机制详解

### 3.1 Token 黑名单（Redis）

**触发场景：**
1. 用户主动 Logout
2. Refresh Token Rotation（每次刷新，旧 Refresh Token 自动失效）

**数据结构：**
```
Redis Key:   token:blacklist:{jti}
Value:       "1"
TTL:         Token 剩余有效时间（time.Until(claims.ExpiresAt)）
```

**降级策略（fail-open）**：Redis 不可用时，打印 Warn 日志后放行请求，服务不中断。这是"可用性优先"的选择——对内部系统来说，Redis 偶发故障期间少量 Token 无法即时吊销，比服务完全不可用代价更小。

**面试追问应对**：
> "如果 Redis 宕机，已注销的 Token 还能用多久？"  
> → Access Token 最长 15 分钟，Refresh Token 宕机期间理论上可继续使用，但 Refresh Token 生成新 Access Token 的路径也会触发黑名单写入失败（降级放行），实际风险窗口为 Redis 恢复前的 Refresh Token TTL 上限。可通过缩短 Access Token TTL 或熔断策略进一步缩小风险。

---

### 3.2 Token Rotation

每次 RefreshToken 请求：
1. 验证旧 Refresh Token 有效性
2. **将旧 Refresh Token JTI 加入黑名单**
3. 生成新 Access Token + 新 Refresh Token
4. 返回新 Token 对

**防重放攻击**：旧 Refresh Token 立即失效，即使攻击者截获也无法反复使用。

---

### 3.3 邀请码机制

**设计动机**：防止任意人员自行注册，保持成员可控性（学生会场景）。

**实现细节：**

```go
// 9 位大写字母+数字，使用 crypto/rand（加密级随机，非 math/rand）
const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
// 36^9 ≈ 1.01 × 10^14 种组合，暴力枚举不现实
```

**邀请码生命周期：**
```
created → [used / expired]
```
- `used_at` 非空：已被使用，拒绝
- `expires_at < now`：已过期，拒绝
- 注意：已过期的邀请码不需要标记 used，直接通过时间判断

**幂等性处理**：用户注册成功后标记邀请码已使用，若标记失败（Redis/DB 异常），**不回滚用户创建**——因为邀请码是已使用状态的延迟写，比"用户已存在但邀请码未标记"更难处理。

---

### 3.4 密码强度校验

**规则**：8-20 字符，至少包含 1 个字母和 1 个数字。

**Go 正则的 lookahead 限制**：

Go 的 `regexp` 包基于 RE2 引擎，**不支持 lookahead**（`(?=...)`）。常见面试陷阱：

```go
// ❌ 在 Go 中会 panic
var re = regexp.MustCompile(`^(?=.*[a-zA-Z])(?=.*\d).{8,20}$`)

// ✅ 正确做法：拆分为独立检查
var hasLetter = regexp.MustCompile(`[a-zA-Z]`)
var hasDigit  = regexp.MustCompile(`\d`)

func validatePassword(password string) bool {
    if len(password) < 8 || len(password) > 20 {
        return false
    }
    return hasLetter.MatchString(password) && hasDigit.MatchString(password)
}
```

**为什么不用 binding tag 的 `containsrune` 或 `alphanum`？**  
Gin binding 校验在 Handler 层，无法表达"既含字母又含数字"的组合条件，且逻辑复杂时应下沉到 Service 层保持 Handler 薄。

---

## 四、分层架构与低耦合设计

### 4.1 依赖注入链路

```
main.go
  │
  ├─ config.Load()           ← 配置
  ├─ database.NewDB()        ← 数据库
  ├─ redis.NewClient()       ← Redis（可 nil 降级）
  ├─ jwt.NewManager()        ← JWT 工具类
  │
  ├─ repository.NewRepository(db)
  │     ├─ UserRepo
  │     ├─ DepartmentRepo
  │     └─ InviteCodeRepo
  │
  ├─ service.NewService(cfg, repo, jwtMgr, rdb, logger)
  │     ├─ AuthService  ← 业务逻辑
  │     └─ UserService
  │
  └─ handler.NewHandler(cfg, svc)
        ├─ AuthHandler  ← HTTP 关注点
        └─ UserHandler
```

**每层只依赖其直接下层的接口，不跨层调用。**

### 4.2 接口驱动，便于 Mock 测试

```go
// Service 层暴露接口
type AuthService interface {
    Login(ctx context.Context, req *dto.LoginRequest) (*dto.TokenResponse, error)
    Register(ctx context.Context, req *dto.RegisterRequest) (*dto.RegisterResponse, error)
    RefreshToken(ctx context.Context, refreshToken string) (*dto.TokenResponse, error)
    // ...
}

// Repository 层同样接口化
type UserRepository interface {
    GetByStudentID(ctx context.Context, studentID string) (*model.User, error)
    // ...
}
```

**单元测试使用 struct mock，不需要 testify/mock 等框架**——直接实现接口，保持零外部依赖。

---

## 五、错误码体系

认证模块错误码集中在 `11000-11999`：

| 错误码 | 含义 | HTTP | 触发场景 |
|--------|------|------|----------|
| 10001 | 参数校验失败 | 400 | binding 失败、弱密码 |
| 10002 | 未认证 | 401 | 无 Token / Token 无效 |
| 11001 | 学号或密码错误 | 401 | 登录失败、原密码错误 |
| 11002 | Token 已过期 | 401 | Refresh 时 Token 过期 |
| 11003 | Token 无效/已吊销 | 401 | 非法 Token / 黑名单命中 |
| 11004 | 邀请码无效或已过期 | 400 | 注册时邀请码非法 |
| 11005 | 邮箱已被注册 | 400 | 注册重复邮箱 |
| 11006 | 学号已被注册 | 400 | 注册重复学号 |

**设计原则**：
- 登录失败统一返回 `11001`（学号或密码错误），**不区分"用户不存在"和"密码错误"**——防止用户枚举攻击（User Enumeration）。
- Token 类错误统一 HTTP 401，让前端可以统一拦截做跳转。

---

## 六、安全设计清单

| 安全措施 | 实现位置 | 说明 |
|----------|----------|------|
| bcrypt 密码哈希 | `auth_service.go` | DefaultCost=10，含 salt |
| JWT HS256 签名 | `pkg/jwt/jwt.go` | secret 从 config 注入，不硬编码 |
| Token 黑名单 | `pkg/redis/redis.go` | Logout 和 Token Rotation |
| HttpOnly Cookie | `auth_handler.go` | Refresh Token 防 XSS |
| SameSite=Lax | `auth_handler.go` | 防 CSRF |
| 用户枚举防护 | `auth_service.go` | 登录错误信息统一，不暴露用户是否存在 |
| 密码强度校验 | `auth_service.go` | 至少含字母+数字，8-20 字符 |
| Context 传递 | 全链路 | 每个 DB/Redis 调用携带 `ctx`，支持超时取消 |
| Token 类型校验 | `middleware/auth.go` | Refresh Token 不能直接用于 API 认证 |
| HTTPS（生产） | `config.yaml` | `cookie.secure=true` 确保仅 HTTPS 传输 |

---

## 七、可扩展性设计

### 7.1 当前未实现但预留的扩展点

| 功能 | 当前状态 | 扩展方式 |
|------|----------|----------|
| 速率限制 (Rate Limiting) | 未实现 | 中间件层接入 Redis 计数器 |
| 登录日志/审计 | 未实现 | Service 层 Login 成功后异步写审计表 |
| 多因素认证 (MFA) | 未实现 | Service 层增加 TOTP 验证步骤 |
| OAuth 第三方登录 | 未实现 | 新增 `oauth_service.go` |
| Token 轮转限频 | 未实现 | Redis 记录同 userID 的刷新次数 |

### 7.2 为什么 Redis Client 传 nil 而不是空接口？

```go
// main.go
rdb, err = redis.NewClient(&cfg.Redis, logger)
if err != nil {
    logger.Warn("Redis 连接失败，Token 黑名单功能将不可用")
    rdb = nil  // 显式 nil，下层通过 if rdb != nil 判断
}
```

**权衡**：使用 Optional Pattern（可为 nil 的具体类型）而不是 Null Object Pattern。优点是调用方代码更直观（`if rdb != nil`），缺点是需要在多处做 nil check。对于"可降级的可选依赖"，nil check 是更惯用的 Go 风格。

---

## 八、测试策略

### 8.1 单元测试（28 个，全部通过）

| 覆盖场景 | 测试数量 |
|----------|---------|
| Login（成功/密码错/用户不存在/RememberMe） | 4 |
| Register（成功/邀请码失效/过期/重复学号/弱密码） | 5 |
| RefreshToken（成功/无效/用 AccessToken 刷新） | 3 |
| GenerateInvite（成功/默认天数） | 2 |
| ValidateInvite（有效/已过期） | 2 |
| ChangePassword（成功/原密码错/弱新密码） | 3 |
| GetCurrentUser（成功/用户不存在） | 2 |
| JWT（生成解析/RememberMe TTL/错误密钥/过期） | 6 |
| **合计** | **28** |

### 8.2 Mock 策略

```go
// 不依赖任何 Mock 框架，直接实现 interface
type mockUserRepo struct {
    users map[string]*model.User
}

func (m *mockUserRepo) GetByStudentID(_ context.Context, id string) (*model.User, error) {
    if u, ok := m.users[id]; ok {
        return u, nil
    }
    return nil, gorm.ErrRecordNotFound
}
```

**优点**：零框架依赖，测试即文档，类型安全。  
**适用场景**：接口方法数量少（≤10）时。方法多时考虑 `testify/mock` 或 `gomock`。

### 8.3 端到端验证结果

在真实 PostgreSQL + Redis 环境中验证全部接口:

```
✅ POST /auth/login           → 200 含 access_token + refresh_token
✅ GET  /auth/me              → 200 返回用户详情
✅ POST /auth/invite          → 200 生成 9 位邀请码
✅ GET  /auth/invite/:code    → 200 valid=true
✅ POST /auth/register        → 201 新用户创建成功
✅ POST /auth/refresh         → 200 新 Token 对
✅ PUT  /auth/password        → 200 新密码可登录
✅ POST /auth/logout          → 200 旧 Token 随即 401
✅ 弱密码注册                 → 400 拒绝
✅ 无效邀请码                 → 400 拒绝
```

---

## 九、面试高频问题与回答策略

**Q1: 你的系统如何防止 SQL 注入？**  
→ 全程使用 GORM ORM，参数化查询，Go 驱动层自动转义，不拼接 SQL 字符串。

**Q2: JWT secret 泄漏了怎么办？**  
→ 立即更换 `jwt_secret` 配置并重启服务。所有旧 Token 因签名验证失败立即失效，相当于全局强制登出。

**Q3: 你如何设计密码重置功能？**（未实现，但可扩展）  
→ 生成有时限的随机 token 存 Redis，发邮件给用户，点击链接后校验 token，允许设置新密码，token 立即销毁（单次使用）。

**Q4: Access Token 为什么是 15 分钟？**  
→ Trade-off：越短越安全，越长用户体验越好。15 分钟是业界常见默认值（参考 AWS Cognito、Auth0 默认配置），配合透明的 Refresh 机制对用户无感知。

**Q5: 你的架构如何保证高内聚低耦合？**  
→ 接口隔离（每层依赖接口而非具体实现），单一职责（Handler 只处理 HTTP，Service 只处理业务，Repo 只处理数据），依赖注入（通过构造函数传入依赖，便于 Mock）。

**Q6: 如果要支持微服务架构，认证模块怎么演进？**  
→ 将 JWT 验证逻辑提取为独立 Auth Service，其他服务通过调用 Auth Service 或本地验证公钥（RSA 签名，改用 RS256）来鉴权，避免每个服务都持有 shared secret。

---

*文档版本 v1.0 | 2026-02-25*
