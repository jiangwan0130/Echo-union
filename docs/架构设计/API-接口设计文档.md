# 接口设计文档 (API Specification)
# 学生会值班管理系统

| 文档信息 | |
|----------|----------|
| 版本号 | v2.0 |
| 创建日期 | 2026-01-29 |
| 最后更新 | 2026-02-26 |
| 文档状态 | 与代码同步 |
| API版本 | v1 |
| 基础路径 | /api/v1 |

---

## 一、文档概述

### 1.1 目的

本文档定义学生会值班管理系统的 RESTful API 接口规范，包括接口路径、请求方法、参数格式、返回格式及错误码定义。

> **Source of Truth**：文档内容以 `backend/internal/api/router/router.go` 实际注册路由为准。

### 1.2 模块实现状态

| 阶段 | 模块 | 状态 |
|------|------|------|
| 一期 | Auth、User、Department、Timetable、Schedule、Semester、TimeSlot、Location、SystemConfig、ScheduleRule、Export | ✅ 已实现 |
| 二期 | Swap（换班）、Check-in（签到）、Notification（通知） | ⏳ Phase 2 - 未实现 |

### 1.3 通用约定

#### 1.3.1 请求格式

- **Content-Type**: `application/json`
- **编码**: UTF-8
- **认证方式**: Bearer Token (JWT)

#### 1.3.2 认证头

需要认证的接口需在请求头中携带：
```
Authorization: Bearer <access_token>
```

#### 1.3.3 Cookie 与 CSRF 约定（Refresh Token 推荐 Cookie 模式）

- 当服务端使用 `Set-Cookie` 下发 `refresh_token`（`HttpOnly` + `Secure` + `SameSite=Lax`）时，浏览器会在调用刷新接口时自动携带 Cookie。
- 为降低 CSRF 风险：刷新接口仅用于"换取新 Token"，不允许产生业务副作用；同时建议校验 `Origin/Referer`（同站）并配合 `SameSite=Lax`。
- 非浏览器客户端（如脚本/测试工具）可选择在请求体传 `refresh_token`。

#### 1.3.4 响应格式

**成功响应：**
```json
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

**分页响应：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [ ... ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total": 100,
      "total_pages": 5
    }
  }
}
```

**错误响应：**
```json
{
  "code": 10001,
  "message": "参数校验失败",
  "details": "邮箱格式不正确"
}
```

#### 1.3.5 通用请求参数（分页）

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| page | int | 否 | 1 | 页码（min=1） |
| page_size | int | 否 | 20 | 每页数量（min=1, max=100） |

#### 1.3.6 HTTP 状态码

| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 201 | 创建成功 |
| 400 | 请求参数错误 |
| 401 | 未授权（未登录/Token失效） |
| 403 | 禁止访问（无权限） |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

---

## 二、健康检查

### 2.0 健康检查

**接口路径**: `GET /health`

**权限要求**: 无需认证

**响应示例：**
```json
{
  "status": "ok"
}
```

---

## 三、认证模块 (Auth)

### 3.1 用户登录

**接口路径**: `POST /api/v1/auth/login`

**权限要求**: 无需认证

> 说明：系统以 **学号（student_id）** 作为唯一登录账号标识；`email` 字段用于通知与联系信息，不参与登录鉴权。

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| student_id | string | 是 | 学号 |
| password | string | 是 | 密码 |
| remember_me | boolean | 否 | 是否记住登录（默认false，影响 Refresh Token 有效期） |

**请求示例：**
```json
{
  "student_id": "2024001",
  "password": "password123",
  "remember_me": true
}
```

**响应参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| access_token | string | 访问令牌 |
| refresh_token | string | 刷新令牌（若采用 HttpOnly Cookie 模式，可不返回） |
| expires_in | int | 访问令牌有效期（秒） |
| user | object | 用户信息 |

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 900,
    "user": {
      "id": "uuid",
      "name": "张三",
      "email": "user@example.com",
      "student_id": "2024001",
      "role": "member",
      "department": {
        "id": "uuid",
        "name": "宣传部"
      }
    }
  }
}
```

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 11001 | 学号或密码错误 |

---

### 3.2 邀请注册

**接口路径**: `POST /api/v1/auth/register`

**权限要求**: 无需认证（需携带有效邀请码）

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| invite_code | string | 是 | 邀请码 |
| name | string | 是 | 姓名（2-20字符） |
| student_id | string | 是 | 学号 |
| email | string | 是 | 邮箱 |
| password | string | 是 | 密码（8-20字符） |
| department_id | string | 是 | 部门ID（UUID） |

**请求示例：**
```json
{
  "invite_code": "ABC123XYZ",
  "name": "张三",
  "student_id": "2024001",
  "email": "zhangsan@example.com",
  "password": "password123",
  "department_id": "uuid"
}
```

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "uuid",
    "name": "张三",
    "email": "zhangsan@example.com"
  }
}
```

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 11004 | 邀请码无效或已过期 |
| 11005 | 邮箱已被注册 |
| 11006 | 学号已被注册 |

---

### 3.3 刷新Token

**接口路径**: `POST /api/v1/auth/refresh`

**权限要求**: 无需认证（需携带 refresh_token：优先 Cookie，其次请求体）

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| refresh_token | string | 否 | 刷新令牌（当未使用 Cookie 模式时必填） |

**响应参数**: 同登录接口

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 11002 | Token已过期 |
| 11003 | Token无效 / 已被吊销 |

---

### 3.4 验证邀请码

**接口路径**: `GET /api/v1/auth/invite/:code`

**权限要求**: 无需认证

**路径参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| code | string | 邀请码 |

**响应参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| valid | boolean | 是否有效 |
| expires_at | string | 过期时间 |

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 11004 | 邀请码无效或已过期 |

---

### 3.5 用户登出

**接口路径**: `POST /api/v1/auth/logout`

**权限要求**: 需要认证

**请求参数**: 无

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```

---

### 3.6 获取当前用户信息

**接口路径**: `GET /api/v1/auth/me`

**权限要求**: 需要认证

**响应参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| id | string | 用户ID |
| name | string | 姓名 |
| email | string | 邮箱 |
| student_id | string | 学号 |
| role | string | 角色（admin/leader/member） |
| department | object | 部门信息 `{id, name}` |
| created_at | string | 创建时间 |

---

### 3.7 修改密码

**接口路径**: `PUT /api/v1/auth/password`

**权限要求**: 需要认证

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| old_password | string | 是 | 原密码 |
| new_password | string | 是 | 新密码（8-20字符） |

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 11001 | 原密码错误 |

---

### 3.8 生成邀请链接

**接口路径**: `POST /api/v1/auth/invite`

**权限要求**: admin 或 leader

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| expires_days | int | 否 | 有效期天数（默认7天） |

**响应参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| invite_code | string | 邀请码 |
| invite_url | string | 完整邀请链接 |
| expires_at | string | 过期时间 |

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "invite_code": "ABC123XYZ",
    "invite_url": "https://example.com/register?code=ABC123XYZ",
    "expires_at": "2026-02-05T00:00:00Z"
  }
}
```

---

## 四、用户模块 (User)

### 4.1 获取当前用户信息

**接口路径**: `GET /api/v1/users/me`

**权限要求**: 需要认证

**响应参数**: 同 `GET /api/v1/auth/me`

---

### 4.2 获取用户列表

**接口路径**: `GET /api/v1/users`

**权限要求**: admin 或 leader

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码 |
| page_size | int | 否 | 每页数量 |
| department_id | string | 否 | 部门筛选（UUID） |
| role | string | 否 | 角色筛选（admin/leader/member） |
| keyword | string | 否 | 关键词搜索（姓名/学号，max=50） |

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": "uuid",
        "name": "张三",
        "email": "zhangsan@example.com",
        "student_id": "2024001",
        "role": "member",
        "department": {
          "id": "uuid",
          "name": "宣传部"
        }
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total": 100,
      "total_pages": 5
    }
  }
}
```

---

### 4.3 获取用户详情

**接口路径**: `GET /api/v1/users/:id`

**权限要求**: admin 或 leader

---

### 4.4 更新用户信息

**接口路径**: `PUT /api/v1/users/:id`

**权限要求**: admin 或 本人（Service 层鉴权）

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 否 | 姓名（2-20字符） |
| email | string | 否 | 邮箱 |
| department_id | string | 否 | 部门ID（UUID，仅管理员） |

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 12001 | 用户不存在 |
| 12004 | 邮箱已被使用 |
| 12005 | 部门不存在 |

---

### 4.5 删除用户

**接口路径**: `DELETE /api/v1/users/:id`

**权限要求**: admin

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 12001 | 用户不存在 |
| 12003 | 无法删除自己 |

---

### 4.6 分配角色

**接口路径**: `PUT /api/v1/users/:id/role`

**权限要求**: admin

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| role | string | 是 | 角色（admin/leader/member） |

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 12001 | 用户不存在 |
| 12002 | 无法修改自己的角色 |

---

### 4.7 重置密码

**接口路径**: `POST /api/v1/users/:id/reset-password`

**权限要求**: admin

**请求参数**: 无

**响应参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| temp_password | string | 临时密码 |

---

### 4.8 批量导入用户

**接口路径**: `POST /api/v1/users/import`

**权限要求**: admin

**请求格式**: `multipart/form-data`

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file | file | 是 | Excel文件（.xlsx，最大5MB） |

**Excel格式要求：**

| 列 | 说明 | 必填 |
|----|------|------|
| 姓名 | 用户姓名 | 是 |
| 学号 | 学号 | 是 |
| 邮箱 | 邮箱地址 | 是 |
| 部门 | 部门名称 | 是 |

**响应参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| total | int | 总行数 |
| success | int | 成功数量 |
| failed | int | 失败数量 |
| errors | array | 错误详情 `[{row, reason}]` |

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total": 50,
    "success": 48,
    "failed": 2,
    "errors": [
      { "row": 10, "reason": "邮箱已存在" },
      { "row": 25, "reason": "部门不存在" }
    ]
  }
}
```

---

## 五、部门模块 (Department)

### 5.1 获取部门列表

**接口路径**: `GET /api/v1/departments`

**权限要求**: 需要认证

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| include_inactive | boolean | 否 | 是否包含已停用部门 |

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": "uuid",
        "name": "宣传部",
        "description": "负责宣传工作",
        "is_active": true,
        "member_count": 15,
        "created_at": "2026-01-01T00:00:00Z",
        "updated_at": "2026-01-01T00:00:00Z"
      }
    ]
  }
}
```

---

### 5.2 获取部门详情

**接口路径**: `GET /api/v1/departments/:id`

**权限要求**: 需要认证

---

### 5.3 创建部门

**接口路径**: `POST /api/v1/departments`

**权限要求**: admin

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 部门名称（2-50字符） |
| description | string | 否 | 部门描述（max=200） |

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 14002 | 部门名称已存在 |

---

### 5.4 更新部门

**接口路径**: `PUT /api/v1/departments/:id`

**权限要求**: admin

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 否 | 部门名称（2-50字符） |
| description | string | 否 | 部门描述（max=200） |
| is_active | boolean | 否 | 是否启用 |

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 14001 | 部门不存在 |
| 14002 | 部门名称已存在 |

---

### 5.5 删除部门

**接口路径**: `DELETE /api/v1/departments/:id`

**权限要求**: admin

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 14001 | 部门不存在 |
| 14003 | 部门下存在成员，无法删除 |

---

### 5.6 获取部门成员

**接口路径**: `GET /api/v1/departments/:id/members`

**权限要求**: admin 或 leader

**响应参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| list[].user_id | string | 用户ID |
| list[].name | string | 姓名 |
| list[].student_id | string | 学号 |
| list[].email | string | 邮箱 |
| list[].role | string | 角色 |
| list[].duty_required | boolean | 是否需要值班 |
| list[].timetable_status | string | 时间表提交状态 |

---

### 5.7 设置值班人员

**接口路径**: `PUT /api/v1/departments/:id/duty-members`

**权限要求**: admin 或 leader

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| semester_id | string | 是 | 学期ID（UUID） |
| user_ids | array | 是 | 需要值班的成员ID列表（UUID数组，min=1） |

**请求示例：**
```json
{
  "semester_id": "uuid",
  "user_ids": ["uuid1", "uuid2", "uuid3"]
}
```

**响应参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| department_id | string | 部门ID |
| department_name | string | 部门名称 |
| semester_id | string | 学期ID |
| total_set | int | 设置的值班人数 |

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 14001 | 部门不存在 |
| 14004 | 部门已停用 |
| 14005 | 指定用户不属于该部门 |
| 14006 | 指定用户不存在 |
| 14007 | 学期不存在 |

---

## 六、时间表模块 (Timetable)

### 6.1 导入ICS课表

**接口路径**: `POST /api/v1/timetables/import`

**权限要求**: 需要认证

**请求方式一（文件上传）**: `multipart/form-data`

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file | file | 是 | ICS文件 |
| semester_id | string | 否 | 学期ID |

**请求方式二（URL）**: `application/json`

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| url | string | 是 | ICS链接（支持https/http/webcal） |
| semester_id | string | 否 | 学期ID（UUID） |

**响应参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| imported_count | int | 导入的课程数量 |
| events | array | 导入的事件列表 |

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "imported_count": 25,
    "events": [
      {
        "name": "高等数学",
        "day_of_week": 1,
        "start_time": "08:00",
        "end_time": "09:40",
        "weeks": [1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16]
      }
    ]
  }
}
```

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 15000 | ICS 导入失败（请上传 ICS 文件或提供 ICS URL） |

---

### 6.2 获取我的时间表

**接口路径**: `GET /api/v1/timetables/me`

**权限要求**: 需要认证

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| semester_id | string | 否 | 学期ID（UUID，默认当前学期） |

**响应参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| courses | array | 课表数据 |
| unavailable | array | 不可用时间 |
| submit_status | string | 提交状态（not_submitted/submitted） |
| submitted_at | datetime | 提交时间 |

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "courses": [
      {
        "id": "uuid",
        "name": "高等数学",
        "day_of_week": 1,
        "start_time": "08:00",
        "end_time": "09:40",
        "week_type": "all",
        "weeks": [1,2,3,4,5,6,7,8],
        "source": "ics"
      }
    ],
    "unavailable": [
      {
        "id": "uuid",
        "day_of_week": 3,
        "start_time": "19:00",
        "end_time": "21:00",
        "reason": "社团活动",
        "repeat_type": "weekly",
        "week_type": "all"
      }
    ],
    "submit_status": "submitted",
    "submitted_at": "2026-01-25T14:30:00Z"
  }
}
```

---

### 6.3 添加不可用时间

**接口路径**: `POST /api/v1/timetables/unavailable`

**权限要求**: 需要认证

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| day_of_week | int | 是 | 星期几（1-7） |
| start_time | string | 是 | 开始时间（HH:mm） |
| end_time | string | 是 | 结束时间（HH:mm） |
| reason | string | 否 | 原因（max=200） |
| repeat_type | string | 否 | 重复类型：**weekly**（默认） / **biweekly** / **once** |
| specific_date | string | 条件 | 特定日期 YYYY-MM-DD（`repeat_type=once` 时必填） |
| week_type | string | 否 | 周类型：all（默认） / odd / even |
| semester_id | string | 否 | 学期ID（UUID） |

> **repeat_type 联动约束**：
> - `once`：必须指定 `specific_date`，`week_type` 只能为 `all`
> - `weekly`：不可指定 `specific_date`
> - `biweekly`：不可指定 `specific_date`，`week_type` 必须为 `odd` 或 `even`

**请求示例：**
```json
{
  "day_of_week": 3,
  "start_time": "19:00",
  "end_time": "21:00",
  "reason": "社团活动",
  "repeat_type": "weekly",
  "week_type": "all"
}
```

---

### 6.4 更新不可用时间

**接口路径**: `PUT /api/v1/timetables/unavailable/:id`

**权限要求**: 需要认证（本人）

**请求参数（均为可选，部分更新）：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| day_of_week | int | 否 | 星期几（1-7） |
| start_time | string | 否 | 开始时间 |
| end_time | string | 否 | 结束时间 |
| reason | string | 否 | 原因 |
| repeat_type | string | 否 | 重复类型（weekly/biweekly/once） |
| specific_date | string | 否 | 特定日期 |
| week_type | string | 否 | 周类型（all/odd/even） |

---

### 6.5 删除不可用时间

**接口路径**: `DELETE /api/v1/timetables/unavailable/:id`

**权限要求**: 需要认证（本人）

---

### 6.6 提交时间表

**接口路径**: `POST /api/v1/timetables/submit`

**权限要求**: 需要认证

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| semester_id | string | 否 | 学期ID（UUID） |

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "submit_status": "submitted",
    "submitted_at": "2026-01-29T10:30:00Z"
  }
}
```

---

### 6.7 获取全局提交进度

**接口路径**: `GET /api/v1/timetables/progress`

**权限要求**: admin

**响应参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| total | int | 需要提交的总人数 |
| submitted | int | 已提交人数 |
| progress | float | 进度百分比（0-100） |
| departments | array | 各部门进度 |

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total": 120,
    "submitted": 100,
    "progress": 83.33,
    "departments": [
      {
        "department_id": "uuid",
        "department_name": "宣传部",
        "total": 15,
        "submitted": 12,
        "progress": 80.0
      }
    ]
  }
}
```

---

### 6.8 获取部门提交进度

**接口路径**: `GET /api/v1/timetables/progress/department/:id`

**权限要求**: admin 或 leader

**响应参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| department_id | string | 部门ID |
| department_name | string | 部门名称 |
| total | int | 总人数 |
| submitted | int | 已提交人数 |
| progress | float | 进度百分比 |
| members | array | 成员提交状态列表 |

**members 条目：**

| 参数 | 类型 | 说明 |
|------|------|------|
| user_id | string | 用户ID |
| name | string | 姓名 |
| student_id | string | 学号 |
| timetable_status | string | 提交状态 |
| submitted_at | datetime | 提交时间 |

---

## 七、排班模块 (Schedule)

### 7.1 执行自动排班

**接口路径**: `POST /api/v1/schedules/auto`

**权限要求**: admin

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| semester_id | string | 是 | 学期ID（UUID） |

**响应参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| schedule | object | 排班表信息（ScheduleResponse） |
| total_slots | int | 总时段数 |
| filled_slots | int | 已分配时段数 |
| warnings | array | 警告信息 |

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "schedule": {
      "id": "uuid",
      "semester_id": "uuid",
      "status": "draft",
      "items": [ ... ],
      "created_at": "2026-02-01T10:00:00Z",
      "updated_at": "2026-02-01T10:00:00Z"
    },
    "total_slots": 38,
    "filled_slots": 38,
    "warnings": []
  }
}
```

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 13103 | 该学期已存在排班表 |
| 13107 | 课表提交率未达100% |
| 13108 | 无符合条件的排班候选人 |
| 13109 | 无可用时间段 |
| 13111 | 学期不存在 |

---

### 7.2 获取排班表

**接口路径**: `GET /api/v1/schedules`

**权限要求**: 需要认证

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| semester_id | string | 是 | 学期ID |

**响应参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| id | string | 排班表ID |
| semester_id | string | 学期ID |
| semester | object | 学期简要信息 `{id, name}` |
| status | string | 状态（draft/published/need_regen） |
| published_at | string | 发布时间 |
| items | array | 排班明细列表 |
| created_at | string | 创建时间 |
| updated_at | string | 更新时间 |

**排班明细 (items[])：**

| 参数 | 类型 | 说明 |
|------|------|------|
| id | string | 排班项ID |
| schedule_id | string | 排班表ID |
| week_number | int | 周次 |
| time_slot | object | 时间段 `{id, name, day_of_week, start_time, end_time}` |
| member | object | 值班人员 `{id, name, student_id, department}` |
| location | object | 地点 `{id, name}` |
| created_at | string | 创建时间 |
| updated_at | string | 更新时间 |

---

### 7.3 获取我的排班

**接口路径**: `GET /api/v1/schedules/my`

**权限要求**: 需要认证

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| semester_id | string | 是 | 学期ID |

**响应**: 返回 `{ "list": [ScheduleItemResponse, ...] }`

---

### 7.4 手动调整排班项

**接口路径**: `PUT /api/v1/schedules/items/:id`

**权限要求**: admin

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| member_id | string | 否 | 新值班人员ID（UUID） |
| location_id | string | 否 | 新地点ID（UUID） |

**响应**: 返回更新后的 `ScheduleItemResponse`

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 13101 | 排班表不存在 |
| 13102 | 排班项不存在 |
| 13104 | 排班表非草稿状态，不可执行此操作 |
| 13110 | 候选人在该时段不可用 |

---

### 7.5 校验候选人

**接口路径**: `POST /api/v1/schedules/items/:id/validate`

**权限要求**: admin

**说明**: 校验某个调整是否合法，不实际保存

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| member_id | string | 是 | 候选人员ID（UUID） |

**响应参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| valid | boolean | 是否合法 |
| conflicts | array | 冲突原因列表 |

---

### 7.6 获取候选人列表

**接口路径**: `GET /api/v1/schedules/items/:id/candidates`

**权限要求**: admin

**说明**: 获取某个时段可选的候选人列表

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "user_id": "uuid",
        "name": "张三",
        "student_id": "2024001",
        "department": { "id": "uuid", "name": "宣传部" },
        "available": true,
        "conflicts": []
      }
    ]
  }
}
```

---

### 7.7 发布排班表

**接口路径**: `POST /api/v1/schedules/publish`

**权限要求**: admin

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| schedule_id | string | 是 | 排班表ID（UUID） |

**响应**: 返回更新后的 `ScheduleResponse`

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 13101 | 排班表不存在 |
| 13106 | 排班表不可发布 |

---

### 7.8 发布后修改排班项

**接口路径**: `PUT /api/v1/schedules/published/items/:id`

**权限要求**: admin

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| member_id | string | 是 | 新值班人员ID（UUID） |
| reason | string | 是 | 修改原因（2-500字符） |

**响应**: 返回更新后的 `ScheduleItemResponse`

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 13102 | 排班项不存在 |
| 13105 | 排班表非已发布状态 |
| 13110 | 候选人在该时段不可用 |

---

### 7.9 获取变更记录

**接口路径**: `GET /api/v1/schedules/change-logs`

**权限要求**: admin

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| schedule_id | string | 是 | 排班表ID（UUID） |
| page | int | 否 | 页码 |
| page_size | int | 否 | 每页数量 |

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": "uuid",
        "schedule_id": "uuid",
        "schedule_item_id": "uuid",
        "original_member_id": "uuid",
        "original_member_name": "张三",
        "new_member_id": "uuid",
        "new_member_name": "李四",
        "change_type": "manual",
        "reason": "张三请假",
        "operator_id": "uuid",
        "created_at": "2026-01-29T16:00:00Z"
      }
    ],
    "pagination": { ... }
  }
}
```

---

### 7.10 范围检测

**接口路径**: `POST /api/v1/schedules/:id/scope-check`

**权限要求**: admin

**说明**: 检测排班表与当前值班人员范围是否发生变化（如有人新增/退出值班）

**响应参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| changed | boolean | 是否有变化 |
| added_users | array | 新增的用户 |
| removed_users | array | 移除的用户 |

---

## 八、学期模块 (Semester)

### 8.1 获取学期列表

**接口路径**: `GET /api/v1/semesters`

**权限要求**: 需要认证

**响应**: 返回 `SemesterResponse` 数组

---

### 8.2 获取当前学期

**接口路径**: `GET /api/v1/semesters/current`

**权限要求**: 需要认证

**响应**: 返回当前激活的 `SemesterResponse`

---

### 8.3 获取学期详情

**接口路径**: `GET /api/v1/semesters/:id`

**权限要求**: 需要认证

---

### 8.4 创建学期

**接口路径**: `POST /api/v1/semesters`

**权限要求**: admin

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 学期名称（2-100字符） |
| start_date | string | 是 | 开始日期（YYYY-MM-DD） |
| end_date | string | 是 | 结束日期（YYYY-MM-DD） |
| first_week_type | string | 是 | 首周类型（odd/even） |

**响应**: 返回 `SemesterResponse`

**SemesterResponse 字段：**

| 参数 | 类型 | 说明 |
|------|------|------|
| id | string | 学期ID |
| name | string | 学期名称 |
| start_date | string | 开始日期 |
| end_date | string | 结束日期 |
| first_week_type | string | 首周类型 |
| is_active | boolean | 是否激活 |
| status | string | 状态 |
| created_at | string | 创建时间 |
| updated_at | string | 更新时间 |

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 14002 | 学期日期无效 |
| 14003 | 学期日期与已有学期重叠 |

---

### 8.5 更新学期

**接口路径**: `PUT /api/v1/semesters/:id`

**权限要求**: admin

**请求参数（均为可选）：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 否 | 学期名称（2-100字符） |
| start_date | string | 否 | 开始日期 |
| end_date | string | 否 | 结束日期 |
| first_week_type | string | 否 | 首周类型（odd/even） |
| status | string | 否 | 状态（active/archived） |

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 14001 | 学期不存在 |
| 14002 | 学期日期无效 |
| 14003 | 学期日期与已有学期重叠 |

---

### 8.6 激活学期

**接口路径**: `PUT /api/v1/semesters/:id/activate`

**权限要求**: admin

**请求参数**: 无

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 14001 | 学期不存在 |

---

### 8.7 删除学期

**接口路径**: `DELETE /api/v1/semesters/:id`

**权限要求**: admin

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 14001 | 学期不存在 |

---

## 九、时间段模块 (TimeSlot)

### 9.1 获取时间段列表

**接口路径**: `GET /api/v1/time-slots`

**权限要求**: 需要认证

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| semester_id | string | 否 | 学期ID（UUID） |
| day_of_week | int | 否 | 星期几（1-5） |

**响应**: 返回 `TimeSlotResponse` 数组

**TimeSlotResponse 字段：**

| 参数 | 类型 | 说明 |
|------|------|------|
| id | string | 时间段ID |
| name | string | 时间段名称 |
| semester_id | string | 关联学期ID |
| semester | object | 学期简要信息 `{id, name}` |
| start_time | string | 开始时间（HH:mm） |
| end_time | string | 结束时间（HH:mm） |
| day_of_week | int | 星期几（1-5） |
| is_active | boolean | 是否启用 |
| created_at | string | 创建时间 |
| updated_at | string | 更新时间 |

---

### 9.2 获取时间段详情

**接口路径**: `GET /api/v1/time-slots/:id`

**权限要求**: 需要认证

---

### 9.3 创建时间段

**接口路径**: `POST /api/v1/time-slots`

**权限要求**: admin

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 时间段名称（2-50字符） |
| semester_id | string | 否 | 关联学期ID（UUID） |
| start_time | string | 是 | 开始时间（HH:mm） |
| end_time | string | 是 | 结束时间（HH:mm） |
| day_of_week | int | 是 | 星期几（1-5） |

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 15002 | 关联的学期不存在 |

---

### 9.4 更新时间段

**接口路径**: `PUT /api/v1/time-slots/:id`

**权限要求**: admin

**请求参数（均为可选）：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 否 | 时间段名称 |
| start_time | string | 否 | 开始时间 |
| end_time | string | 否 | 结束时间 |
| day_of_week | int | 否 | 星期几（1-5） |
| is_active | boolean | 否 | 是否启用 |

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 15001 | 时间段不存在 |

---

### 9.5 删除时间段

**接口路径**: `DELETE /api/v1/time-slots/:id`

**权限要求**: admin

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 15001 | 时间段不存在 |

---

## 十、地点模块 (Location)

### 10.1 获取地点列表

**接口路径**: `GET /api/v1/locations`

**权限要求**: 需要认证

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| include_inactive | boolean | 否 | 是否包含已停用地点 |

**响应**: 返回 `LocationResponse` 数组

**LocationResponse 字段：**

| 参数 | 类型 | 说明 |
|------|------|------|
| id | string | 地点ID |
| name | string | 地点名称 |
| address | string | 地址 |
| is_default | boolean | 是否默认地点 |
| is_active | boolean | 是否启用 |
| created_at | string | 创建时间 |
| updated_at | string | 更新时间 |

---

### 10.2 获取地点详情

**接口路径**: `GET /api/v1/locations/:id`

**权限要求**: 需要认证

---

### 10.3 创建地点

**接口路径**: `POST /api/v1/locations`

**权限要求**: admin

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 地点名称（2-100字符） |
| address | string | 否 | 地址（max=200） |
| is_default | boolean | 否 | 是否为默认地点 |

---

### 10.4 更新地点

**接口路径**: `PUT /api/v1/locations/:id`

**权限要求**: admin

**请求参数（均为可选）：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 否 | 地点名称 |
| address | string | 否 | 地址 |
| is_default | boolean | 否 | 是否默认 |
| is_active | boolean | 否 | 是否启用 |

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 16001 | 地点不存在 |

---

### 10.5 删除地点

**接口路径**: `DELETE /api/v1/locations/:id`

**权限要求**: admin

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 16001 | 地点不存在 |

---

## 十一、系统配置模块 (SystemConfig)

### 11.1 获取系统配置

**接口路径**: `GET /api/v1/system-config`

**权限要求**: 需要认证

**响应参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| swap_deadline_hours | int | 换班截止时间（小时） |
| duty_reminder_time | string | 值班提醒时间 |
| default_location | string | 默认值班地点 |
| sign_in_window_minutes | int | 签到窗口时间（分钟） |
| sign_out_window_minutes | int | 签退窗口时间（分钟） |
| updated_at | string | 更新时间 |

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "swap_deadline_hours": 24,
    "duty_reminder_time": "09:00",
    "default_location": "学生会办公室",
    "sign_in_window_minutes": 15,
    "sign_out_window_minutes": 15,
    "updated_at": "2026-01-29T00:00:00Z"
  }
}
```

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 17001 | 系统配置未初始化 |

---

### 11.2 更新系统配置

**接口路径**: `PUT /api/v1/system-config`

**权限要求**: admin

**请求参数（均为可选）：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| swap_deadline_hours | int | 否 | 换班截止时间（1-168小时） |
| duty_reminder_time | string | 否 | 值班提醒时间 |
| default_location | string | 否 | 默认值班地点（1-200字符） |
| sign_in_window_minutes | int | 否 | 签到窗口时间（1-60分钟） |
| sign_out_window_minutes | int | 否 | 签退窗口时间（1-60分钟） |

---

## 十二、排班规则模块 (ScheduleRule)

### 12.1 获取排班规则列表

**接口路径**: `GET /api/v1/schedule-rules`

**权限要求**: 需要认证

**响应参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| list[].id | string | 规则ID |
| list[].rule_code | string | 规则代码（如 R1, R2, R3...） |
| list[].rule_name | string | 规则名称 |
| list[].description | string | 规则描述 |
| list[].is_enabled | boolean | 是否启用 |
| list[].is_configurable | boolean | 是否可配置（false 为固定规则） |
| list[].created_at | string | 创建时间 |
| list[].updated_at | string | 更新时间 |

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      { "id": "uuid", "rule_code": "R1", "rule_name": "课表冲突", "is_enabled": true, "is_configurable": false },
      { "id": "uuid", "rule_code": "R2", "rule_name": "不可用时间冲突", "is_enabled": true, "is_configurable": false },
      { "id": "uuid", "rule_code": "R3", "rule_name": "同日部门不重复", "is_enabled": true, "is_configurable": true },
      { "id": "uuid", "rule_code": "R4", "rule_name": "相邻班次部门不重复", "is_enabled": true, "is_configurable": true },
      { "id": "uuid", "rule_code": "R5", "rule_name": "单双周早八不重复", "is_enabled": true, "is_configurable": true },
      { "id": "uuid", "rule_code": "R6", "rule_name": "同人同日不重复", "is_enabled": true, "is_configurable": false }
    ]
  }
}
```

---

### 12.2 获取排班规则详情

**接口路径**: `GET /api/v1/schedule-rules/:id`

**权限要求**: 需要认证

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 18001 | 排班规则不存在 |

---

### 12.3 更新排班规则

**接口路径**: `PUT /api/v1/schedule-rules/:id`

**权限要求**: admin

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| is_enabled | boolean | 否 | 是否启用 |

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 18001 | 排班规则不存在 |
| 18002 | 该规则不可配置 |

---

## 十三、导出模块 (Export)

### 13.1 导出排班表

**接口路径**: `GET /api/v1/export/schedule`

**权限要求**: admin 或 leader

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| semester_id | string | 是 | 学期ID（UUID） |

**响应**: Excel 文件下载（`application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`）

**响应头：**
```
Content-Disposition: attachment; filename*=UTF-8''<encoded_filename>.xlsx
Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
```

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 16001 | 该学期暂无排班表 |
| 16002 | 排班表中无排班项 |

---

### 13.2 导出签到统计

> **⚠️ Phase 2 - 未实现** — 签到统计导出依赖签到模块，计划在二期实现。

**接口路径**: `GET /api/v1/export/duty-stats`（预案）

---

## 十四、换班模块 (Swap)

> **⚠️ Phase 2 - 未实现** — 换班模块计划在二期实现，以下接口为设计预案，尚未注册路由。

| 接口 | 方法 | 路径 | 说明 |
|------|------|------|------|
| 发起换班申请 | POST | /api/v1/swaps | 值班成员发起 |
| 获取换班详情 | GET | /api/v1/swaps/:id | 相关成员/管理员 |
| 获取我的换班列表 | GET | /api/v1/swaps/me | 值班成员 |
| 目标成员响应 | PUT | /api/v1/swaps/:id/respond | accept/reject |
| 获取待审核列表 | GET | /api/v1/swaps/pending | 管理员 |
| 管理员审批 | PUT | /api/v1/swaps/:id/approve | approve/reject |
| 获取换班记录 | GET | /api/v1/swaps/records | 需要认证 |

---

## 十五、签到模块 (Check-in)

> **⚠️ Phase 2 - 未实现** — 签到签退模块计划在二期实现，以下接口为设计预案，尚未注册路由。

| 接口 | 方法 | 路径 | 说明 |
|------|------|------|------|
| 获取今日值班 | GET | /api/v1/duties/today | 值班成员 |
| 签到 | POST | /api/v1/duties/:id/sign-in | 本人 |
| 签退 | POST | /api/v1/duties/:id/sign-out | 本人 |
| 补签 | POST | /api/v1/duties/:id/make-up | 本人 |
| 获取我的出勤记录 | GET | /api/v1/duties/me | 值班成员 |
| 获取异常记录 | GET | /api/v1/duties/abnormal | 管理员 |

---

## 十六、通知模块 (Notification)

> **⚠️ Phase 2 - 未实现** — 通知模块计划在二期实现，以下接口为设计预案，尚未注册路由。

| 接口 | 方法 | 路径 | 说明 |
|------|------|------|------|
| 获取消息列表 | GET | /api/v1/notifications | 需要认证 |
| 标记已读 | PUT | /api/v1/notifications/:id/read | 本人 |
| 标记全部已读 | PUT | /api/v1/notifications/read-all | 需要认证 |
| 获取通知偏好 | GET | /api/v1/notifications/preferences | 需要认证 |
| 更新通知偏好 | PUT | /api/v1/notifications/preferences | 需要认证 |

---

## 十七、路由总览

### 一期已实现路由

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | /health | 无 | 健康检查 |
| **Auth** | | | |
| POST | /api/v1/auth/login | 无 | 用户登录 |
| POST | /api/v1/auth/register | 无 | 邀请注册 |
| POST | /api/v1/auth/refresh | 无 | 刷新Token |
| GET | /api/v1/auth/invite/:code | 无 | 验证邀请码 |
| POST | /api/v1/auth/logout | 认证 | 用户登出 |
| GET | /api/v1/auth/me | 认证 | 获取当前用户 |
| PUT | /api/v1/auth/password | 认证 | 修改密码 |
| POST | /api/v1/auth/invite | admin/leader | 生成邀请链接 |
| **User** | | | |
| GET | /api/v1/users/me | 认证 | 获取当前用户 |
| GET | /api/v1/users | admin/leader | 用户列表 |
| GET | /api/v1/users/:id | admin/leader | 用户详情 |
| PUT | /api/v1/users/:id | 认证 | 更新用户（admin或本人） |
| DELETE | /api/v1/users/:id | admin | 删除用户 |
| PUT | /api/v1/users/:id/role | admin | 分配角色 |
| POST | /api/v1/users/:id/reset-password | admin | 重置密码 |
| POST | /api/v1/users/import | admin | 批量导入 |
| **Department** | | | |
| GET | /api/v1/departments | 认证 | 部门列表 |
| GET | /api/v1/departments/:id | 认证 | 部门详情 |
| POST | /api/v1/departments | admin | 创建部门 |
| PUT | /api/v1/departments/:id | admin | 更新部门 |
| DELETE | /api/v1/departments/:id | admin | 删除部门 |
| GET | /api/v1/departments/:id/members | admin/leader | 部门成员 |
| PUT | /api/v1/departments/:id/duty-members | admin/leader | 设置值班人员 |
| **Semester** | | | |
| GET | /api/v1/semesters | 认证 | 学期列表 |
| GET | /api/v1/semesters/current | 认证 | 当前学期 |
| GET | /api/v1/semesters/:id | 认证 | 学期详情 |
| POST | /api/v1/semesters | admin | 创建学期 |
| PUT | /api/v1/semesters/:id | admin | 更新学期 |
| PUT | /api/v1/semesters/:id/activate | admin | 激活学期 |
| DELETE | /api/v1/semesters/:id | admin | 删除学期 |
| **TimeSlot** | | | |
| GET | /api/v1/time-slots | 认证 | 时间段列表 |
| GET | /api/v1/time-slots/:id | 认证 | 时间段详情 |
| POST | /api/v1/time-slots | admin | 创建时间段 |
| PUT | /api/v1/time-slots/:id | admin | 更新时间段 |
| DELETE | /api/v1/time-slots/:id | admin | 删除时间段 |
| **Location** | | | |
| GET | /api/v1/locations | 认证 | 地点列表 |
| GET | /api/v1/locations/:id | 认证 | 地点详情 |
| POST | /api/v1/locations | admin | 创建地点 |
| PUT | /api/v1/locations/:id | admin | 更新地点 |
| DELETE | /api/v1/locations/:id | admin | 删除地点 |
| **SystemConfig** | | | |
| GET | /api/v1/system-config | 认证 | 获取系统配置 |
| PUT | /api/v1/system-config | admin | 更新系统配置 |
| **ScheduleRule** | | | |
| GET | /api/v1/schedule-rules | 认证 | 排班规则列表 |
| GET | /api/v1/schedule-rules/:id | 认证 | 排班规则详情 |
| PUT | /api/v1/schedule-rules/:id | admin | 更新排班规则 |
| **Timetable** | | | |
| POST | /api/v1/timetables/import | 认证 | 导入ICS课表 |
| GET | /api/v1/timetables/me | 认证 | 我的时间表 |
| POST | /api/v1/timetables/unavailable | 认证 | 添加不可用时间 |
| PUT | /api/v1/timetables/unavailable/:id | 认证 | 更新不可用时间 |
| DELETE | /api/v1/timetables/unavailable/:id | 认证 | 删除不可用时间 |
| POST | /api/v1/timetables/submit | 认证 | 提交时间表 |
| GET | /api/v1/timetables/progress | admin | 全局提交进度 |
| GET | /api/v1/timetables/progress/department/:id | admin/leader | 部门提交进度 |
| **Schedule** | | | |
| POST | /api/v1/schedules/auto | admin | 自动排班 |
| GET | /api/v1/schedules | 认证 | 获取排班表 |
| GET | /api/v1/schedules/my | 认证 | 我的排班 |
| PUT | /api/v1/schedules/items/:id | admin | 手动调整排班 |
| POST | /api/v1/schedules/items/:id/validate | admin | 校验候选人 |
| GET | /api/v1/schedules/items/:id/candidates | admin | 候选人列表 |
| POST | /api/v1/schedules/publish | admin | 发布排班表 |
| PUT | /api/v1/schedules/published/items/:id | admin | 发布后修改 |
| GET | /api/v1/schedules/change-logs | admin | 变更记录 |
| POST | /api/v1/schedules/:id/scope-check | admin | 范围检测 |
| **Export** | | | |
| GET | /api/v1/export/schedule | admin/leader | 导出排班表 |

---

## 十八、错误码汇总

| 错误码 | 含义 | HTTP状态码 |
|--------|------|------------|
| 0 | 成功 | 200 |
| **10000-10999** | **通用错误** | |
| 10001 | 参数校验失败 | 400 |
| 10002 | 未授权（未登录） | 401 |
| 10003 | 禁止访问（无权限） | 403 |
| 10004 | 资源不存在 | 404 |
| 10005 | 服务器内部错误 | 500 |
| **11000-11999** | **认证错误 (Auth)** | |
| 11001 | 学号或密码错误 / 原密码错误 | 401 |
| 11002 | Token已过期 | 401 |
| 11003 | Token无效 / 已被吊销 | 401 |
| 11004 | 邀请码无效或已过期 | 400 |
| 11005 | 邮箱已被注册 | 400 |
| 11006 | 学号已被注册 | 400 |
| **12000-12999** | **用户错误 (User)** | |
| 12001 | 用户不存在 | 404 |
| 12002 | 无法修改自己的角色 | 400 |
| 12003 | 无法删除自己 | 400 |
| 12004 | 邮箱已被使用 | 400 |
| 12005 | 部门不存在 | 400 |
| **13000-13999** | **排班错误 (Schedule)** | |
| 13101 | 排班表不存在 | 404 |
| 13102 | 排班项不存在 | 404 |
| 13103 | 该学期已存在排班表 | 400 |
| 13104 | 排班表非草稿状态，不可执行此操作 | 400 |
| 13105 | 排班表非已发布状态 | 400 |
| 13106 | 排班表不可发布 | 400 |
| 13107 | 课表提交率未达100% | 400 |
| 13108 | 无符合条件的排班候选人 | 400 |
| 13109 | 无可用时间段 | 400 |
| 13110 | 候选人在该时段不可用 | 400 |
| 13111 | 学期不存在 | 404 |
| **14000-14999** | **部门 / 学期错误 (Department & Semester)** | |
| 14001 | 部门不存在 / 学期不存在 | 404 |
| 14002 | 部门名称已存在 / 学期日期无效 | 400 |
| 14003 | 部门下存在成员，无法删除 / 学期日期重叠 | 400 |
| 14004 | 部门已停用 | 400 |
| 14005 | 指定用户不属于该部门 | 400 |
| 14006 | 指定用户不存在 | 404 |
| 14007 | 学期不存在（部门上下文） | 404 |
| **15000-15999** | **时间表 / 时间段错误 (Timetable & TimeSlot)** | |
| 15000 | ICS 导入失败 | 400 |
| 15001 | 时间段不存在 | 404 |
| 15002 | 关联的学期不存在 | 400 |
| 15008 | 时间表资源不存在 | 404 |
| 15010 | 学期不存在（时间表上下文） | 404 |
| **16000-16999** | **地点 / 导出错误 (Location & Export)** | |
| 16001 | 地点不存在 / 该学期暂无排班表 | 404 |
| 16002 | 排班表中无排班项 | 400 |
| **17000-17999** | **系统配置错误 (SystemConfig)** | |
| 17001 | 系统配置未初始化 | 404 |
| **18000-18999** | **排班规则错误 (ScheduleRule)** | |
| 18001 | 排班规则不存在 | 404 |
| 18002 | 该规则不可配置 | 400 |

---

## 版本历史

| 版本 | 日期 | 修改内容 | 修改人 |
|------|------|----------|--------|
| v1.0 | 2026-01-29 | 初稿 | 系统架构师 |
| v2.0 | 2026-02-26 | 以 router.go 为准全面同步：拆分配置模块为独立资源路由（Semester/TimeSlot/Location/SystemConfig/ScheduleRule）、修正排班模块路由、新增 Location 模块、新增 Users/me 和 reset-password 端点、导出路径修正、错误码按实际代码更新、不可用时间新增 biweekly 支持、二期模块统一标记 | 文档同步助手 |
