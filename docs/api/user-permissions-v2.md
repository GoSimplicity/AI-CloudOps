# 用户权限码 v2 契约（SEC-24 / SEC-07）

面向前端（花卷）对齐。本契约描述新增的 `GET /api/user/permissions`，以及它与旧接口 `GET /api/user/codes` 的关系。后端鉴权以本文“匹配规则”为准，`permissions` 与后端 `CheckAuth` 使用同一份匹配代码生成，二者不会出现“前端显示可用、后端拒绝”的差异（超管码与下文列出的例外除外）。

## 1. 接口

| 项目 | 值 |
| --- | --- |
| 方法 / 路径 | `GET /api/user/permissions` |
| 认证 | 需要登录（Bearer JWT，与 `/api/user/codes` 相同） |
| 授权 | 任何已登录用户都可调用，只返回**调用者本人**的权限；不接受 `id` 等参数 |
| 旧接口 | `GET /api/user/codes` **保持不变**：仍返回角色授予的原始 `path` 字符串数组（不带方法、不展开通配、去重），前端可按自己的节奏迁移 |

## 2. 响应

外层仍是统一的 `ApiResponse`（`code` / `data` / `message`），`data` 结构：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `version` | int | 固定为 `2`。后续若格式变化会递增 |
| `super_admin` | bool | 调用者是否持有启用中的内置超级管理员角色（`code=super_admin` 且 `is_system=1`） |
| `permissions` | string[] | 权限码列表，**已排序、已去重**，永不为 `null`（没有权限时是 `[]`） |

### 2.1 权限码格式

```
<METHOD><空格><路由模板>
```

- `METHOD` 只会是 `GET` / `POST` / `PUT` / `DELETE`（大写）。
- `路由模板` 是 gin 注册时的模板，参数保持 `:name` 原样，例如 `PUT /api/tree/cloud/account/:id/status`。
- 前端匹配方式：把“当前要调用的 方法 + 路由模板”拼成同样格式，用 `Set.has()` 精确比较即可。**不要**把真实 ID 代入后再比较，也**不要**把 `:param` 当作通配。
- AIOps 代理（`/api/v1/*path`）下的接口按 AIOps 接口目录展开成具体模板，例如 `GET /api/v1/assistant/session/:session_id`。若角色被授予了整个代理入口，还会额外出现 `<METHOD> /api/v1/*path` 本身。

### 2.2 超管码

超级管理员不展开列表，`permissions` **只包含一个元素** `"*"`（超管码），并且 `super_admin=true`。

- `"*"` 只会在 `super_admin=true` 时出现，且不会与其他权限码混在一起。
- 前端判断建议：`super_admin === true || permissions.includes('*')` 即视为全部可用。
- 选择超管码而不是全量列表的原因：超管在后端是“无条件放行”，全量列表会随路由增减而漂移，也比实际授权更窄（目录外的接口超管同样能访问）。

### 2.3 通配授予的展开

角色里保存的授予（`cl_system_apis.path` + `method`）按下列规则展开到**已注册路由目录**（见第 4 节），只保留方法相同且被覆盖的路由模板：

| 授予写法 | 含义 | 展开结果 |
| --- | --- | --- |
| 普通模板，如 `/api/k8s/clusters/:cluster_id/events/:namespace/:name` | 只覆盖**完全相同**的模板 | 1 条（若该模板已注册） |
| `/*` | 覆盖所有模板 | 该方法下目录内全部路由 |
| 以 `*` 结尾，如 `/api/k8s/*` | 对**模板字符串**做前缀匹配（前缀为去掉末尾 `*` 的部分） | 该方法下所有以 `/api/k8s/` 开头的模板 |
| gin 通配模板，如 `/api/v1/*path` | 与模板完全相同；对 AIOps 代理而言表示整个代理模块 | `<METHOD> /api/v1/*path` 以及该方法下全部 AIOps 目录接口 |
| 其他含 `*` 的写法（开头 `*`、中间 `*`、多个 `*`） | **不再支持**，不覆盖任何模板 | 0 条 |
| 未注册的路径 / 已失效的授予 | 不覆盖任何模板 | 0 条 |

### 2.4 不出现在列表中的接口

- 登录后即可访问、不经角色授权的接口：`/api/user/login`、`/api/user/logout`、`/api/user/refresh_token`、`/api/user/profile`、`/api/user/codes`、`/api/user/permissions`、`/api/not_auth/getBindIps`、`/api/not_auth/getTreeNodeBindIps`。
- 仅超管可用的接口，即使角色里授予了也不会出现（后端同样会拒绝非超管）：`POST /api/user/signup`、`/api/department` 下全部接口。
- 非 `/api/` 前缀的路由（`/`、`/health`、`/ready` 等）。

## 3. 示例

### 3.1 普通用户

角色授予：`GET /api/k8s/clusters/:cluster_id/events/:namespace/:name`、`PUT /api/tree/cloud/account/:id/status`、`GET /api/v1/assistant/*`。

```json
{
  "code": 0,
  "data": {
    "version": 2,
    "super_admin": false,
    "permissions": [
      "GET /api/k8s/clusters/:cluster_id/events/:namespace/:name",
      "GET /api/v1/assistant/config",
      "GET /api/v1/assistant/health",
      "GET /api/v1/assistant/info",
      "GET /api/v1/assistant/ready",
      "GET /api/v1/assistant/session/:session_id",
      "GET /api/v1/assistant/stream",
      "PUT /api/tree/cloud/account/:id/status"
    ]
  },
  "message": "请求成功"
}
```

注意：上例**不会**包含 `GET /api/k8s/clusters/:cluster_id/events/nodes/:node_name` 或 `PUT /api/tree/cloud/account/batch/status`，后端也会拒绝这两个请求（这是 SEC-24 修复的授权外溢）。

### 3.2 超级管理员（含 admin）

```json
{
  "code": 0,
  "data": {
    "version": 2,
    "super_admin": true,
    "permissions": ["*"]
  },
  "message": "请求成功"
}
```

### 3.3 没有任何角色（或角色全部禁用）

```json
{
  "code": 0,
  "data": {
    "version": 2,
    "super_admin": false,
    "permissions": []
  },
  "message": "请求成功"
}
```

同一用户调用旧接口 `GET /api/user/codes` 的结果不变，例如 3.1 的用户仍得到：

```json
{
  "code": 0,
  "data": [
    "/api/k8s/clusters/:cluster_id/events/:namespace/:name",
    "/api/tree/cloud/account/:id/status",
    "/api/v1/assistant/*"
  ],
  "message": "请求成功"
}
```

（顺序由数据库决定；admin 的内置超管角色本身不绑定接口，所以 admin 的旧 `codes` 仍是 `[]`，与改动前一致。）

## 4. 后端匹配规则（与 `permissions` 同源）

1. 鉴权对象是 **HTTP 方法 + gin 路由模板**（`c.FullPath()`），不是真实请求路径。`:param` 不再作为通配符参与授权匹配。
2. 角色授予的 `/*` 和末尾 `*` 只对**模板**做前缀匹配。
3. `c.FullPath()` 为空（请求未命中任何路由，即 404）时：非超管直接返回 `403 无权限访问该接口`，不暴露路由是否存在；超管放行，由 gin 返回 404。
4. AIOps 代理入口 `/api/v1/*path` 的 `FullPath` 只有通配模板，后端用内置 AIOps 接口目录把真实路径解析为最具体的目录模板（逐段比较，静态段优先于 `:param`，`:param` 只匹配一个非空且不为 `.`/`..` 的段），再按第 1、2 条比较；解析不到目录模板时，只有覆盖 `/api/v1/*path` 本身的授予（如 `/api/v1/*path`、`/api/*`、`/*`）能放行。
5. 只计入启用中（`status=1`）、未删除的角色及其未删除的接口。
6. 超管（持有启用中的内置超管角色）对所有请求放行；`POST /api/user/signup` 与 `/api/department/*` 只允许超管。
7. 目录来源：运行时 gin 已注册路由（`engine.Routes()`）+ 内置 AIOps 接口目录。之所以不用数据库 `cl_system_apis` 做展开：数据库里可能有已下线或手工录入的条目，展开后会给前端“幽灵权限”；而已注册路由正是 `FullPath` 的取值空间，和后端判定严格一致。

## 5. 前端迁移建议

- 读取 `permissions`，构造 `Set`；按钮/菜单声明自己依赖的 `METHOD 模板`。
- `super_admin` 为真时跳过所有判断。
- 旧 `codes` 中基于 `*` / `/*` 的前端通配逻辑可以在迁移完成后删除。
