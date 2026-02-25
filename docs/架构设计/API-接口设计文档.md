# 接口设计文档 (API Specification)
# 学生会值班管理系统

| 文档信息 | |
|----------|----------|
| 版本号 | v1.0 |
| 创建日期 | 2026-01-29 |
| 最后更新 | 2026-01-29 |
| 文档状态 | 初稿 |
| API版本 | v1 |
| 基础路径 | /api/v1 |

---

## 一、文档概述

### 1.1 目的

本文档定义学生会值班管理系统的 RESTful API 接口规范，包括接口路径、请求方法、参数格式、返回格式及错误码定义。

### 1.2 通用约定

#### 1.2.1 请求格式

- **Content-Type**: `application/json`
- **编码**: UTF-8
- **认证方式**: Bearer Token (JWT)

#### 1.2.2 认证头

需要认证的接口需在请求头中携带：
```
Authorization: Bearer <access_token>
```

#### 1.2.2.1 Cookie 与 CSRF 约定（Refresh Token 推荐 Cookie 模式）

- 当服务端使用 `Set-Cookie` 下发 `refresh_token`（`HttpOnly` + `Secure` + `SameSite=Lax`）时，浏览器会在调用刷新接口时自动携带 Cookie。
- 为降低 CSRF 风险：刷新接口仅用于“换取新 Token”，不允许产生业务副作用；同时建议校验 `Origin/Referer`（同站）并配合 `SameSite=Lax`。
- 非浏览器客户端（如脚本/测试工具）可选择在请求体传 `refresh_token`。

#### 1.2.3 响应格式

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

#### 1.2.4 通用请求参数（分页）

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| page | int | 否 | 1 | 页码 |
| page_size | int | 否 | 20 | 每页数量（最大100） |

#### 1.2.5 HTTP 状态码

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

## 二、认证模块 (Auth)

### 2.1 用户登录

**接口路径**: `POST /api/v1/auth/login`

**权限要求**: 无需认证

> 说明：系统以 **学号（student_id）** 作为唯一登录账号标识；`email` 字段用于通知与联系信息，不参与登录鉴权。

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| student_id | string | 是 | 学号 |
| password | string | 是 | 密码 |
| remember_me | boolean | 否 | 是否记住登录（默认false） |

> 说明：`remember_me` 用于影响 Refresh Token 的有效期策略（例如：默认 24 小时；勾选后 7 天；具体以配置为准）。

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
| refresh_token | string | 刷新令牌（可选；若采用 HttpOnly Cookie 模式，可不返回） |
| expires_in | int | 访问令牌有效期（秒） |
| user | object | 用户信息 |

**Cookie 约定（推荐）：**

- 服务端可通过 `Set-Cookie` 下发 `refresh_token`（`HttpOnly` + `SameSite=Lax` + `Secure`）
- 浏览器端后续刷新 Token 时无需在请求体中传 `refresh_token`（从 Cookie 自动携带）

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

### 2.2 用户登出

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

### 2.3 刷新Token

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
| 11003 | Token无效 |

---

### 2.4 生成邀请链接

**接口路径**: `POST /api/v1/auth/invite`

**权限要求**: 排班管理员

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| expires_days | int | 否 | 有效期天数（默认7天） |

**响应参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| invite_code | string | 邀请码 |
| invite_url | string | 完整邀请链接 |
| expires_at | datetime | 过期时间 |

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

### 2.5 验证邀请码

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
| expires_at | datetime | 过期时间 |

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 11004 | 邀请码无效或已过期 |

---

### 2.6 邀请注册

**接口路径**: `POST /api/v1/auth/register`

**权限要求**: 无需认证（需携带有效邀请码）

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| invite_code | string | 是 | 邀请码 |
| name | string | 是 | 姓名（2-20字符） |
| student_id | string | 是 | 学号 |
| email | string | 是 | 邮箱 |
| password | string | 是 | 密码（8-20字符，含字母和数字） |
| department_id | string | 是 | 部门ID |

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

### 2.7 获取当前用户信息

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
| department | object | 部门信息 |
| duty_required | boolean | 是否需要值班 |
| created_at | datetime | 创建时间 |

---

### 2.8 修改密码

**接口路径**: `PUT /api/v1/auth/password`

**权限要求**: 需要认证

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| old_password | string | 是 | 原密码 |
| new_password | string | 是 | 新密码（8-20字符，含字母和数字） |

---

## 三、用户模块 (User)

### 3.1 获取用户列表

**接口路径**: `GET /api/v1/users`

**权限要求**: 排班管理员

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码 |
| page_size | int | 否 | 每页数量 |
| department_id | string | 否 | 部门筛选 |
| role | string | 否 | 角色筛选 |
| keyword | string | 否 | 关键词搜索（姓名/学号） |

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
        },
        "duty_required": true,
        "created_at": "2026-01-01T00:00:00Z"
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

### 3.2 获取用户详情

**接口路径**: `GET /api/v1/users/:id`

**权限要求**: 排班管理员

---

### 3.3 更新用户信息

**接口路径**: `PUT /api/v1/users/:id`

**权限要求**: 排班管理员 或 本人

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 否 | 姓名 |
| email | string | 否 | 邮箱 |
| department_id | string | 否 | 部门ID（仅管理员） |

---

### 3.4 分配角色

**接口路径**: `PUT /api/v1/users/:id/role`

**权限要求**: 排班管理员

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| role | string | 是 | 角色（admin/leader/member） |

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 12002 | 无法修改自己的角色 |

---

### 3.5 批量导入用户

**接口路径**: `POST /api/v1/users/import`

**权限要求**: 排班管理员

**请求格式**: `multipart/form-data`

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file | file | 是 | Excel文件（.xlsx） |

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
| errors | array | 错误详情 |

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

### 3.6 删除用户

**接口路径**: `DELETE /api/v1/users/:id`

**权限要求**: 排班管理员

---

## 四、部门模块 (Department)

### 4.1 获取部门列表

**接口路径**: `GET /api/v1/departments`

**权限要求**: 需要认证

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| include_members | boolean | 否 | 是否包含成员数量 |

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
        "member_count": 15,
        "is_active": true
      }
    ]
  }
}
```

---

### 4.2 创建部门

**接口路径**: `POST /api/v1/departments`

**权限要求**: 排班管理员

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 部门名称 |
| description | string | 否 | 部门描述 |

---

### 4.3 更新部门

**接口路径**: `PUT /api/v1/departments/:id`

**权限要求**: 排班管理员

---

### 4.4 删除部门

**接口路径**: `DELETE /api/v1/departments/:id`

**权限要求**: 排班管理员

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 12003 | 部门下存在成员，无法删除 |

---

### 4.5 获取部门成员

**接口路径**: `GET /api/v1/departments/:id/members`

**权限要求**: 排班管理员 或 本部门负责人

**响应参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| list | array | 成员列表 |
| list[].id | string | 用户ID |
| list[].name | string | 姓名 |
| list[].student_id | string | 学号 |
| list[].duty_required | boolean | 是否需要值班 |
| list[].submit_status | string | 时间表提交状态 |

---

### 4.6 设置值班人员

**接口路径**: `PUT /api/v1/departments/:id/duty-members`

**权限要求**: 排班管理员 或 本部门负责人

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| member_ids | array | 是 | 需要值班的成员ID列表 |

**请求示例：**
```json
{
  "member_ids": ["uuid1", "uuid2", "uuid3"]
}
```

---

## 五、时间表模块 (TimeTable)

### 5.1 导入ICS课表

**接口路径**: `POST /api/v1/timetables/import`

**权限要求**: 值班成员（本人）

**请求参数（文件方式）**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file | file | 是 | ICS文件（最大5MB） |

**请求参数（链接方式）**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| url | string | 是 | ICS链接（支持https/http/webcal） |

**响应参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| imported_count | int | 导入的课程数量 |
| events | array | 导入的事件摘要 |

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
        "weeks": "1-16"
      }
    ]
  }
}
```

---

### 5.2 获取我的时间表

**接口路径**: `GET /api/v1/timetables/me`

**权限要求**: 值班成员

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| semester_id | string | 否 | 学期ID（默认当前学期） |

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
        "week_type": "all"
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

### 5.3 添加不可用时间

**接口路径**: `POST /api/v1/timetables/unavailable`

**权限要求**: 值班成员（本人）

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| day_of_week | int | 是 | 星期几（1-7） |
| start_time | string | 是 | 开始时间（HH:mm） |
| end_time | string | 是 | 结束时间（HH:mm） |
| reason | string | 否 | 原因 |
| repeat_type | string | 是 | 重复类型（once/weekly） |
| specific_date | string | 条件 | 特定日期（repeat_type=once时必填） |
| week_type | string | 否 | 周类型（all/odd/even，默认all） |

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

### 5.4 更新不可用时间

**接口路径**: `PUT /api/v1/timetables/unavailable/:id`

**权限要求**: 值班成员（本人）

---

### 5.5 删除不可用时间

**接口路径**: `DELETE /api/v1/timetables/unavailable/:id`

**权限要求**: 值班成员（本人）

---

### 5.6 提交时间表

**接口路径**: `POST /api/v1/timetables/submit`

**权限要求**: 值班成员（本人）

**请求参数**: 无

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

### 5.7 获取提交进度

**接口路径**: `GET /api/v1/timetables/progress`

**权限要求**: 排班管理员

**响应参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| total | int | 需要提交的总人数 |
| submitted | int | 已提交人数 |
| progress | float | 进度百分比 |
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
        "id": "uuid",
        "name": "宣传部",
        "total": 15,
        "submitted": 12,
        "progress": 80.0,
        "not_submitted": [
          { "id": "uuid", "name": "张三" },
          { "id": "uuid", "name": "李四" }
        ]
      }
    ]
  }
}
```

---

### 5.8 获取部门提交进度

**接口路径**: `GET /api/v1/timetables/progress/department/:id`

**权限要求**: 排班管理员 或 本部门负责人

---

## 六、排班模块 (Schedule)

### 6.1 执行自动排班

**接口路径**: `POST /api/v1/schedules/auto`

**权限要求**: 排班管理员

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| semester_id | string | 否 | 学期ID（默认当前学期） |

**响应参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| success | boolean | 是否完全成功 |
| schedule_id | string | 排班表ID |
| total_slots | int | 总时段数 |
| assigned_slots | int | 已分配时段数 |
| unassigned | array | 未分配时段（若有） |
| member_stats | array | 各成员排班次数 |

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "success": true,
    "schedule_id": "uuid",
    "total_slots": 38,
    "assigned_slots": 38,
    "unassigned": [],
    "member_stats": [
      { "member_id": "uuid", "name": "张三", "count": 3 },
      { "member_id": "uuid", "name": "李四", "count": 3 }
    ]
  }
}
```

**部分成功响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "success": false,
    "schedule_id": "uuid",
    "total_slots": 38,
    "assigned_slots": 35,
    "unassigned": [
      {
        "week": 1,
        "day_of_week": 1,
        "time_slot": "08:10-10:05",
        "reason": "无可用人员"
      }
    ],
    "member_stats": [ ... ]
  }
}
```

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 13001 | 提交率未达100%，无法排班 |
| 13002 | 排班正在进行中 |

---

### 6.2 获取排班表

**接口路径**: `GET /api/v1/schedules`

**权限要求**: 需要认证

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| semester_id | string | 否 | 学期ID |
| week | int | 否 | 周次（1或2） |
| department_id | string | 否 | 部门筛选 |
| member_id | string | 否 | 成员筛选 |

**响应参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| schedule_id | string | 排班表ID |
| status | string | 状态（draft/published/archived） |
| items | array | 排班明细 |

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "schedule_id": "uuid",
    "status": "published",
    "items": [
      {
        "id": "uuid",
        "week": 1,
        "day_of_week": 1,
        "date": "2026-02-17",
        "time_slot": {
          "id": "uuid",
          "name": "第一时段",
          "start_time": "08:10",
          "end_time": "10:05"
        },
        "member": {
          "id": "uuid",
          "name": "张三",
          "department": "宣传部"
        },
        "location": "学生会办公室"
      }
    ]
  }
}
```

---

### 6.3 获取我的排班

**接口路径**: `GET /api/v1/schedules/me`

**权限要求**: 值班成员

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| semester_id | string | 否 | 学期ID |

---

### 6.4 手动调整排班

**接口路径**: `PUT /api/v1/schedules/items/:id`

**权限要求**: 排班管理员

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| member_id | string | 是 | 新值班人员ID |

**响应参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| valid | boolean | 是否校验通过 |
| item | object | 更新后的排班项 |

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 13003 | 排班规则冲突 |
| 13004 | 成员不在排班范围内 |
| 13005 | 成员时间冲突 |

**错误响应示例：**
```json
{
  "code": 13003,
  "message": "排班规则冲突",
  "details": {
    "violations": [
      {
        "rule": "R3",
        "rule_name": "同日部门不重复",
        "message": "该成员部门当天已有人值班"
      }
    ]
  }
}
```

---

### 6.5 校验排班调整

**接口路径**: `POST /api/v1/schedules/items/:id/validate`

**权限要求**: 排班管理员

**说明**: 校验某个调整是否合法，不实际保存

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| member_id | string | 是 | 候选人员ID |

**响应参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| valid | boolean | 是否合法 |
| violations | array | 违反的规则列表 |

---

### 6.6 获取候选人列表

**接口路径**: `GET /api/v1/schedules/items/:id/candidates`

**权限要求**: 排班管理员

**说明**: 获取某个时段可选的候选人列表（已过滤冲突）

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "candidates": [
      {
        "id": "uuid",
        "name": "张三",
        "department": "宣传部",
        "current_count": 2
      }
    ]
  }
}
```

---

### 6.7 发布排班表

**接口路径**: `POST /api/v1/schedules/:id/publish`

**权限要求**: 排班管理员

**请求参数**: 无

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "status": "published",
    "published_at": "2026-01-29T15:00:00Z",
    "notified_count": 120
  }
}
```

---

### 6.8 发布后修改

**接口路径**: `PUT /api/v1/schedules/:id/items/:item_id`

**权限要求**: 排班管理员

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| member_id | string | 是 | 新值班人员ID |
| reason | string | 否 | 修改原因 |

---

### 6.9 获取变更记录

**接口路径**: `GET /api/v1/schedules/:id/change-logs`

**权限要求**: 排班管理员

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": "uuid",
        "item_id": "uuid",
        "time_slot": "周一 08:10-10:05",
        "original_member": { "id": "uuid", "name": "张三" },
        "new_member": { "id": "uuid", "name": "李四" },
        "operator": { "id": "uuid", "name": "管理员" },
        "reason": "张三请假",
        "created_at": "2026-01-29T16:00:00Z"
      }
    ]
  }
}
```

---

## 七、换班模块 (Swap)

> **⚠️ 二期工程内容** — 换班模块计划在二期实现，以下接口为设计预案，尚未实现。

### 7.1 发起换班申请

**接口路径**: `POST /api/v1/swaps`

**权限要求**: 值班成员

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| schedule_item_id | string | 是 | 排班项ID |
| target_member_id | string | 是 | 目标成员ID |
| reason | string | 否 | 换班原因 |

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "uuid",
    "status": "pending",
    "created_at": "2026-01-29T10:00:00Z"
  }
}
```

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 14001 | 已超过换班截止时间 |
| 14002 | 目标成员不在排班范围 |
| 14003 | 目标成员时间冲突 |
| 14004 | 目标成员当日已有排班 |

---

### 7.2 获取换班详情

**接口路径**: `GET /api/v1/swaps/:id`

**权限要求**: 相关成员 或 管理员

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "uuid",
    "status": "pending",
    "schedule_item": {
      "id": "uuid",
      "date": "2026-02-17",
      "time_slot": "08:10-10:05",
      "location": "学生会办公室"
    },
    "applicant": {
      "id": "uuid",
      "name": "张三",
      "department": "宣传部"
    },
    "target": {
      "id": "uuid",
      "name": "李四",
      "department": "秘书部"
    },
    "reason": "有事请假",
    "validation": {
      "course_conflict": false,
      "unavailable_conflict": false,
      "same_day_conflict": false
    },
    "created_at": "2026-01-29T10:00:00Z"
  }
}
```

---

### 7.3 获取我的换班列表

**接口路径**: `GET /api/v1/swaps/me`

**权限要求**: 值班成员

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| type | string | 否 | 类型（initiated/received，默认全部） |
| status | string | 否 | 状态筛选 |

---

### 7.4 目标成员响应

**接口路径**: `PUT /api/v1/swaps/:id/respond`

**权限要求**: 目标成员

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| action | string | 是 | 操作（accept/reject） |

---

### 7.5 获取待审核列表

**接口路径**: `GET /api/v1/swaps/pending`

**权限要求**: 排班管理员

---

### 7.6 管理员审批

**接口路径**: `PUT /api/v1/swaps/:id/approve`

**权限要求**: 排班管理员

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| action | string | 是 | 操作（approve/reject） |
| comment | string | 否 | 审批意见 |

---

### 7.7 获取换班记录

**接口路径**: `GET /api/v1/swaps/records`

**权限要求**: 需要认证（成员仅能查看与自己相关的）

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| semester_id | string | 否 | 学期ID |
| status | string | 否 | 状态筛选 |

---

## 八、签到模块 (Duty)

> **⚠️ 二期工程内容** — 签到签退模块计划在二期实现，以下接口为设计预案，尚未实现。

### 8.1 获取今日值班

**接口路径**: `GET /api/v1/duties/today`

**权限要求**: 值班成员

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "has_duty": true,
    "duty": {
      "id": "uuid",
      "schedule_item_id": "uuid",
      "date": "2026-01-29",
      "time_slot": {
        "name": "第一时段",
        "start_time": "08:10",
        "end_time": "10:05"
      },
      "location": "学生会办公室",
      "status": "pending",
      "sign_in_time": null,
      "sign_out_time": null,
      "can_sign_in": true,
      "can_sign_out": false
    }
  }
}
```

---

### 8.2 签到

**接口路径**: `POST /api/v1/duties/:id/sign-in`

**权限要求**: 值班成员（本人）

**请求参数**: 无

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "sign_in_time": "2026-01-29T08:05:00Z",
    "status": "on_duty",
    "is_late": false
  }
}
```

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 15001 | 当前不在签到时间窗口 |
| 15003 | 已签到，请勿重复签到 |

---

### 8.3 签退

**接口路径**: `POST /api/v1/duties/:id/sign-out`

**权限要求**: 值班成员（本人）

**请求参数**: 无

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "sign_out_time": "2026-01-29T10:00:00Z",
    "status": "completed"
  }
}
```

**错误码：**

| 错误码 | 说明 |
|--------|------|
| 15002 | 当前不在签退时间窗口 |
| 15004 | 未签到，无法签退 |

---

### 8.4 补签

**接口路径**: `POST /api/v1/duties/:id/make-up`

**权限要求**: 值班成员（本人，且状态为缺席）

**请求参数**: 无

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "make_up_time": "2026-01-29T08:30:00Z",
    "status": "absent_made_up"
  }
}
```

---

### 8.5 获取我的出勤记录

**接口路径**: `GET /api/v1/duties/me`

**权限要求**: 值班成员

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| semester_id | string | 否 | 学期ID |
| status | string | 否 | 状态筛选 |

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": "uuid",
        "date": "2026-01-29",
        "time_slot": "08:10-10:05",
        "sign_in_time": "2026-01-29T08:05:00Z",
        "sign_out_time": "2026-01-29T10:00:00Z",
        "status": "completed",
        "is_late": false
      }
    ],
    "stats": {
      "total": 10,
      "completed": 8,
      "late": 1,
      "absent": 1
    }
  }
}
```

---

### 8.6 获取异常记录

**接口路径**: `GET /api/v1/duties/abnormal`

**权限要求**: 排班管理员

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| type | string | 否 | 类型（absent/no_sign_out） |
| date_from | string | 否 | 开始日期 |
| date_to | string | 否 | 结束日期 |

---

## 九、通知模块 (Notification)

> **⚠️ 二期工程内容** — 通知模块计划在二期实现，以下接口为设计预案，尚未实现。

### 9.1 获取消息列表

**接口路径**: `GET /api/v1/notifications`

**权限要求**: 需要认证

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| is_read | boolean | 否 | 已读状态筛选 |
| type | string | 否 | 消息类型筛选 |

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": "uuid",
        "type": "schedule_published",
        "title": "排班表已发布",
        "content": "2025-2026学年第二学期排班表已发布，请查看您的值班安排。",
        "is_read": false,
        "created_at": "2026-01-29T15:00:00Z"
      }
    ],
    "unread_count": 5
  }
}
```

---

### 9.2 标记已读

**接口路径**: `PUT /api/v1/notifications/:id/read`

**权限要求**: 需要认证（本人）

---

### 9.3 标记全部已读

**接口路径**: `PUT /api/v1/notifications/read-all`

**权限要求**: 需要认证

---

### 9.4 获取通知偏好

**接口路径**: `GET /api/v1/notifications/preferences`

**权限要求**: 需要认证

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "schedule_published": true,
    "duty_reminder": true,
    "swap_notification": true,
    "absent_notification": true
  }
}
```

---

### 9.5 更新通知偏好

**接口路径**: `PUT /api/v1/notifications/preferences`

**权限要求**: 需要认证

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| schedule_published | boolean | 否 | 排班发布通知 |
| duty_reminder | boolean | 否 | 值班提醒 |
| swap_notification | boolean | 否 | 换班通知 |
| absent_notification | boolean | 否 | 缺席通知 |

---

## 十、配置模块 (Config)

### 10.1 获取当前学期

**接口路径**: `GET /api/v1/config/semester/current`

**权限要求**: 需要认证

---

### 10.2 获取学期列表

**接口路径**: `GET /api/v1/config/semesters`

**权限要求**: 排班管理员

---

### 10.3 创建学期

**接口路径**: `POST /api/v1/config/semesters`

**权限要求**: 排班管理员

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 学期名称 |
| start_date | string | 是 | 开始日期（YYYY-MM-DD） |
| end_date | string | 是 | 结束日期（YYYY-MM-DD） |
| first_week_type | string | 是 | 首周类型（odd/even） |

---

### 10.4 激活学期

**接口路径**: `PUT /api/v1/config/semesters/:id/activate`

**权限要求**: 排班管理员

---

### 10.5 获取时间段配置

**接口路径**: `GET /api/v1/config/time-slots`

**权限要求**: 需要认证

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "weekday": [
      { "id": "uuid", "name": "第一时段", "start_time": "08:10", "end_time": "10:05" },
      { "id": "uuid", "name": "第二时段", "start_time": "10:20", "end_time": "12:15" },
      { "id": "uuid", "name": "第三时段", "start_time": "14:00", "end_time": "16:00" },
      { "id": "uuid", "name": "第四时段", "start_time": "16:10", "end_time": "18:00" }
    ],
    "friday": [
      { "id": "uuid", "name": "第一时段", "start_time": "08:10", "end_time": "10:05" },
      { "id": "uuid", "name": "第二时段", "start_time": "10:20", "end_time": "12:15" },
      { "id": "uuid", "name": "第三时段", "start_time": "14:00", "end_time": "16:00" }
    ]
  }
}
```

---

### 10.6 更新时间段配置

**接口路径**: `PUT /api/v1/config/time-slots`

**权限要求**: 排班管理员

---

### 10.7 获取排班规则配置

**接口路径**: `GET /api/v1/config/rules`

**权限要求**: 排班管理员

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "rules": [
      { "id": "R1", "name": "课表冲突", "enabled": true, "configurable": false },
      { "id": "R2", "name": "不可用时间冲突", "enabled": true, "configurable": false },
      { "id": "R6", "name": "同人同日不重复", "enabled": true, "configurable": false },
      { "id": "R3", "name": "同日部门不重复", "enabled": true, "configurable": true },
      { "id": "R4", "name": "相邻班次部门不重复", "enabled": true, "configurable": true },
      { "id": "R5", "name": "单双周早八不重复", "enabled": true, "configurable": true }
    ]
  }
}
```

---

### 10.8 更新排班规则配置

**接口路径**: `PUT /api/v1/config/rules`

**权限要求**: 排班管理员

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| rules | array | 是 | 规则配置数组 |
| rules[].id | string | 是 | 规则ID |
| rules[].enabled | boolean | 是 | 是否启用 |

---

### 10.9 获取系统设置

**接口路径**: `GET /api/v1/config/settings`

**权限要求**: 排班管理员

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "swap_deadline_hours": 24,
    "duty_reminder_time": "09:00",
    "default_location": "学生会办公室"
  }
}
```

---

### 10.10 更新系统设置

**接口路径**: `PUT /api/v1/config/settings`

**权限要求**: 排班管理员

---

## 十一、导出模块 (Export)

### 11.1 导出排班表

**接口路径**: `GET /api/v1/exports/schedule`

**权限要求**: 排班管理员

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| semester_id | string | 否 | 学期ID |

**响应**: Excel 文件下载

---

### 11.2 导出签到统计

> **⚠️ 二期工程内容** — 签到统计导出依赖签到模块，计划在二期实现。

**接口路径**: `GET /api/v1/exports/duty-stats`

**权限要求**: 排班管理员

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| semester_id | string | 否 | 学期ID |
| date_from | string | 否 | 开始日期 |
| date_to | string | 否 | 结束日期 |

**响应**: Excel 文件下载

---

## 十二、错误码汇总

| 错误码 | 含义 | HTTP状态码 |
|--------|------|------------|
| 0 | 成功 | 200 |
| **10000-10999** | **通用错误** | |
| 10001 | 参数校验失败 | 400 |
| 10002 | 未授权（未登录） | 401 |
| 10003 | 禁止访问（无权限） | 403 |
| 10004 | 资源不存在 | 404 |
| 10005 | 服务器内部错误 | 500 |
| **11000-11999** | **认证错误** | |
| 11001 | 学号或密码错误 | 401 |
| 11002 | Token已过期 | 401 |
| 11003 | Token无效 | 401 |
| 11004 | 邀请码无效或已过期 | 400 |
| 11005 | 邮箱已被注册 | 400 |
| 11006 | 学号已被注册 | 400 |
| **12000-12999** | **用户错误** | |
| 12001 | 用户不存在 | 404 |
| 12002 | 无法修改自己的角色 | 400 |
| 12003 | 部门下存在成员，无法删除 | 400 |
| **13000-13999** | **排班错误** | |
| 13001 | 提交率未达100%，无法排班 | 400 |
| 13002 | 排班正在进行中 | 400 |
| 13003 | 排班规则冲突 | 400 |
| 13004 | 成员不在排班范围内 | 400 |
| 13005 | 成员时间冲突 | 400 |
| 13006 | 需要重新排班 | 400 |
| **14000-14999** | **换班错误** | |
| 14001 | 已超过换班截止时间 | 400 |
| 14002 | 目标成员不在排班范围 | 400 |
| 14003 | 目标成员时间冲突 | 400 |
| 14004 | 目标成员当日已有排班 | 400 |
| 14005 | 换班申请不存在 | 404 |
| 14006 | 换班状态不允许此操作 | 400 |
| **15000-15999** | **签到错误** | |
| 15001 | 当前不在签到时间窗口 | 400 |
| 15002 | 当前不在签退时间窗口 | 400 |
| 15003 | 已签到，请勿重复签到 | 400 |
| 15004 | 未签到，无法签退 | 400 |
| 15005 | 值班记录不存在 | 404 |

---

## 版本历史

| 版本 | 日期 | 修改内容 | 修改人 |
|------|------|----------|--------|
| v1.0 | 2026-01-29 | 初稿 | 系统架构师 |

