# K8s ServiceAccount token 接口加固（SEC-29）

## 结论

- `GET /api/k8s/serviceaccount/:cluster_id/:namespace/:name/token` 只返回**属于该 SA 的** token Secret：类型必须为 `kubernetes.io/service-account-token`，注解 `kubernetes.io/service-account.name` 必须等于 SA 名称，若有注解 `kubernetes.io/service-account.uid` 则必须等于 SA 的 UID，命名空间一致且 `token` 非空。仅凭 `sa.secrets` 引用挂进来的其他 SA 的 token Secret（借挂）一律不返回。
- 创建/更新 SA 时请求里的 `secrets` 引用**被忽略并记 Warn 日志**（只记 Secret 名称），不再写入集群。
- GET 与 POST（TokenRequest）`/token` 都改为**只认显式授权**：必须按完整路由模板 + 方法授权，`/api/k8s/*`、`/*`、带 `:param` 的模板、具体请求路径都不覆盖。admin 照常放行。
- 两个接口每次调用都**同步**写一条审计，只含元数据、不含 token；审计写入失败时**不返回 token**（HTTP 500）。响应头带 `Cache-Control: no-store`、`Pragma: no-cache`。
- 不涉及数据库表结构变化，没有 DDL。启动时只会向 `cl_system_apis` 补登/改名权限行（见第 3 节）。

## 1. 接口契约变化

### 1.1 GET `/token`（读取 token Secret）

| 场景 | 改造前 | 改造后 |
| --- | --- | --- |
| SA 自己的 token Secret | 返回 token | 不变 |
| `sa.secrets` 引用了其他 SA 的 token Secret | 返回其中第一个 token（越权） | 跳过；只剩借挂的 Secret 时返回 `code=1`、`msg` 含“没有属于自己的token Secret” |
| 注解 name 相同但 uid 不同（同名 SA 删除重建后残留） | 返回旧 token | 跳过 |
| 无 uid 注解 | 返回 | 只校验 name，返回 |
| 审计写入失败 | 无审计 | HTTP 500，不返回 token |

返回结构 `{token, expiration_seconds, created_at, expiration_time}` 不变。

注意：Kubernetes 1.24+ 默认不再自动生成 token Secret，`sa.secrets` 通常为空，此时 GET 一般返回“没有 token Secret”。推荐改用 POST（TokenRequest）签发短期 token。

### 1.2 POST `/token`（TokenRequest）

- 请求体 `{"expiration_seconds": 600}`；`cluster_id`、`namespace`、`service_account_name` 以 URL 为准，请求体里的同名字段会被 URL 值覆盖。
- K8s 返回错误时审计记录真实状态码（403/404/…）。
- **审计写入失败时不返回 token（HTTP 500）。但 token 已由 K8s 签发**，TokenRequest 签发的 token 无法单独吊销，会在 `expiration_seconds` 到期前保持有效（调用方拿不到它）。如需立即失效，只能删除并重建该 SA。

### 1.3 创建 / 更新 SA 的 `secrets` 字段

| 接口 | 改造后行为 |
| --- | --- |
| `POST .../create`、`POST .../create/yaml` | 请求里的 `secrets` 被清空后再创建，`image_pull_secrets` 不受影响 |
| `PUT .../update` | 忽略请求里的 `secrets`，保留集群中该 SA 现有的 `secrets` 引用（原来 Update 是整体替换，不传会清空控制器维护的引用） |
| `PUT .../update/yaml` | 同上；另外 YAML 中的 `metadata.name` / `metadata.namespace` 被强制改为 URL 中的值，不能借 YAML 改到其他 SA |

为什么是“忽略 + 日志”而不是“拒绝”：`secrets` 字段在现有 API 契约和前端类型（`k8s_serviceaccount.ts` 的 `secrets?: string[]`）中都存在，并且 detail YAML 里带有集群维护的 `secrets`，前端/用户把 YAML 原样回传时如果直接拒绝会把正常的编辑请求打失败。忽略后既堵住了“把别人的 token Secret 挂到自己 SA 上”的路径，又不破坏现有调用。日志关键字：`忽略请求中的ServiceAccount secrets引用`（仅含 Secret 名称）。

## 2. 权限码与授权

| 方法 | 路由（`cl_system_apis.path`） | `method` | 权限码（`cl_system_apis.name`） |
| --- | --- | --- | --- |
| GET | `/api/k8s/serviceaccount/:cluster_id/:namespace/:name/token` | 1 | `k8s:serviceaccount:token` |
| POST | `/api/k8s/serviceaccount/:cluster_id/:namespace/:name/token` | 2 | `k8s:serviceaccount:token:create` |

- `cl_system_apis.name` 有唯一索引，GET 与 POST 不能共用一个名字，所以 POST 用 `:create` 后缀。
- 授权按 `path + method` 精确匹配：GET 授权不覆盖 POST，反之亦然。
- 以下授权都**不再**放行 `/token`：`/api/k8s/*`、`/*`、`/api/k8s/serviceaccount/:cluster_id/:namespace/:name/:action`、换了参数名的模板、具体路径如 `/api/k8s/serviceaccount/1/ops/app/token`、中间通配 `/api/k8s/*/token`。

## 3. 启动时的权限目录变化（无 DDL）

- 新实例：启动时 `bootstrapPermissionCatalog` 按上表名称登记两行。
- 已部署实例：`/token` 路由在 SEC-29 之前就存在，目录里已有按路由自动生成的名称（形如 `GET GetServiceAccountToken-fm 1a2b3c4d`）。启动时**仅当名称仍等于自动生成值**时改成上表的权限码与说明（`UPDATE ... WHERE id = ? AND name = <自动名>`）；管理员改过的名称、说明保持不变。改名失败（例如权限码已被其他行占用）只记 Warn 日志，不影响启动，也不影响授权（授权只看 path + method）。
- 不会自动授予任何角色。

## 4. 审计字段

写入 `cl_system_audit_logs`，同步写入（不经异步队列）：

| 字段 | GET | POST |
| --- | --- | --- |
| `operation_type` | `TOKEN_REVEAL` | `TOKEN_CREATE` |
| `http_method` | `GET` | `POST` |
| `target_type` | `k8s_serviceaccount_token` | 同左 |
| `target_id` | `<cluster_id>/<namespace>/<sa>` | 同左 |
| `status_code` | 200 / 404 / K8s 真实状态码 | 200 / K8s 真实状态码 |
| `user_id`、`ip_address`、`user_agent`、`trace_id`、`endpoint`、`duration`、`error_msg` | 照常 | 照常 |
| `response_body` | 空 | 空 |

`request_body` 为 JSON 元数据（示例值均为虚构）：

```json
{
  "action": "k8s_serviceaccount_token_reveal",
  "user_id": 7,
  "username": "operator",
  "cluster_id": 1,
  "namespace": "ops",
  "service_account": "app",
  "service_account_uid": "00000000-0000-0000-0000-000000000000",
  "secret_name": "app-token-xxxxx",
  "success": true,
  "issued_at": "2026-01-01T00:00:00Z"
}
```

POST 的 `action` 为 `k8s_serviceaccount_token_create`，不含 `secret_name`，含 `expiration_seconds`（请求值）。**任何审计字段都不含 token。** 请求失败（SA 不存在、无合法 token Secret、K8s 拒绝）也会写审计，`success=false`。通用审计中间件对 GET 不记录；对 POST 另记一条 `CREATE` 元数据记录，不记录请求体和响应体。

## 5. 兼容性影响

- **只靠通配符授权（`/api/k8s/*`、`/*` 等）的非 admin 角色将失去 GET/POST `/token`**，返回 403。需要时按第 2 节为角色显式授权对应的 path + method。
- 依赖“借挂 token Secret”拿 token 的用法失效（这正是要堵的越权路径）。
- 创建/更新时传 `secrets` 不再生效，但请求不会失败。
- `/token` 响应不可缓存（`no-store`）。
- 审计库不可用时 `/token` 返回 HTTP 500。

## 6. 回滚

- 回滚代码即可，无需数据库回滚。
- 回滚后 `cl_system_apis` 中残留的行：`k8s:serviceaccount:token`（GET）和 `k8s:serviceaccount:token:create`（POST），或在已部署实例上是被改名后的原行。旧代码启动时按 `(path, method)` 判断已存在，**不会改回自动名称，也不会新增重复行**，残留名称不影响授权。如需恢复原名称，可由 DBA 手动改 `name`；保留不动也无副作用。
- 回滚后通配符授权重新覆盖 `/token`，借挂 token Secret 重新可被读出，审计和 no-store 一并消失。为 SEC-29 新增的显式授权行在旧代码下仍然有效（旧代码同样接受精确授权）。
- 回滚期间已写入的 `TOKEN_REVEAL` / `TOKEN_CREATE` 审计记录保留，旧代码不读取也不受影响。
