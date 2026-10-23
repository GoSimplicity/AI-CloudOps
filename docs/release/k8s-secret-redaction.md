# K8s Secret 脱敏与 reveal 契约（SEC-25）

## 结论

- `list`、`detail`、`detail/yaml` 三个接口**默认不返回任何 Secret 值**，只返回键名和值的字节数。
- 明文只能通过新接口 `GET /api/k8s/secret/:cluster_id/:namespace/:name/reveal` 获取。每次调用都会同步写一条审计（`cl_system_audit_logs`），审计只含谁、何时、哪个集群/命名空间/Secret、哪些 key，**不含值**。审计写入失败时不返回明文。
- `kubectl.kubernetes.io/last-applied-configuration` 注解可能含整份明文，list/detail 的 `annotations` 和 YAML 中都不再返回它。
- 本次改动不涉及数据库表结构变化（`K8sSecret` 不参与 AutoMigrate，`cl_system_apis`、`cl_system_audit_logs` 的结构未改）。

## 1. list / detail：字段变化

| 字段 | 改造前 | 改造后 |
| --- | --- | --- |
| `data` | `{key: base64(值)}` | `{key: ""}`：键都保留，值一律为空字符串 |
| `string_data` | `{key: 明文}`（通常为空） | `{key: ""}`：键都保留，值一律为空字符串 |
| `data_keys` | 无 | **新增**。`[{key, size}]` 按 key 升序，`size` 为 base64 解码后的值字节数 |
| `string_data_keys` | 无 | **新增**。`[{key, size}]` 按 key 升序，`size` 为字符串的 UTF-8 字节数 |
| `redacted` | 无 | **新增**。固定为 `true`，表示值已被移除 |
| `annotations` | 原样返回 | 不再返回 `kubectl.kubernetes.io/last-applied-configuration`，其余注解照常返回 |
| `data_count` / `size` | 键数 / 总大小 | 不变 |

前端兼容说明：`data` 仍是对象，键集合没变，已有的“按 key 列表展示”逻辑无需修改。如果前端原来会对值做 base64 解码，现在得到的是空字符串，请改为展示 `data_keys[].size` 并提供“查看”按钮调用 reveal。

示例（`GET /api/k8s/secret/1/list` 中的单个 item，或 `GET /api/k8s/secret/1/ops/app-secret/detail` 返回的 `data`）：

```json
{
  "code": 0,
  "data": {
    "name": "app-secret",
    "namespace": "ops",
    "cluster_id": 1,
    "uid": "00000000-0000-0000-0000-000000000000",
    "type": "Opaque",
    "data": { "db-user": "", "db-pass": "" },
    "string_data": { "note": "" },
    "data_keys": [
      { "key": "db-pass", "size": 16 },
      { "key": "db-user", "size": 5 }
    ],
    "string_data_keys": [
      { "key": "note", "size": 9 }
    ],
    "redacted": true,
    "labels": { "app": "demo" },
    "annotations": { "owner": "team-a" },
    "immutable": false,
    "data_count": 3,
    "size": "30 B",
    "age": "240h0m0s",
    "created_at": "2026-10-01T00:00:00Z"
  },
  "message": "请求成功"
}
```

## 2. detail/yaml：字段变化

- `data` 和 `stringData` 下每个键的值都替换为占位符 `<redacted>`，键保留。这个占位符不是合法的 base64，不可能被当作真实值写回集群。
- `metadata.annotations` 中的 `last-applied-configuration` 被移除（改造前已经如此），`resourceVersion`、`uid`、`managedFields` 等系统字段的清理方式不变。

```json
{
  "code": 0,
  "data": {
    "yaml": "data:\n  db-pass: <redacted>\n  db-user: <redacted>\nmetadata:\n  annotations:\n    owner: team-a\n  labels:\n    app: demo\n  name: app-secret\n  namespace: ops\nstringData:\n  note: <redacted>\ntype: Opaque\n"
  },
  "message": "请求成功"
}
```

## 3. 新接口：reveal

`GET /api/k8s/secret/:cluster_id/:namespace/:name/reveal[?keys=k1&keys=k2]`

- `keys` 可选，可以重复传，也可以用逗号分隔。为空时返回全部键；指定的键不存在时报错，同样会写审计。同一个键同时出现在 `data` 和 `stringData` 中时，两处的值都会返回，`keys` 中该键只出现一次。
- `data` 的编码与改造前 detail 一致（base64），前端原有的解码逻辑可以直接复用。
- 响应头带 `Cache-Control: no-store`、`Pragma: no-cache`。
- 同步审计写入失败时返回 **HTTP 500**（`code=1`，`message` 为“写入审计失败，已拒绝返回Secret明文”），不含任何值。其他失败（Secret 不存在、键不存在等）沿用项目惯例：HTTP 200、`code=1`。
- 权限码：接口目录（`cl_system_apis`）中的名称为 `k8s:secret:reveal`，路径为 `/api/k8s/secret/:cluster_id/:namespace/:name/reveal`，方法为 GET。**只有显式授予这一条才生效**：授权记录的 `path` 必须与上面的路由模板逐字相等、方法为 GET。`/api/k8s/*`、`/*`、`/api/k8s/*/reveal` 这类通配符授权，`/api/k8s/secret/:cluster_id/:namespace/:name/:action`、`/api/k8s/secret/:a/:b/:c/reveal` 这类参数化授权（不含 `*`，参数名或段不同），具体请求路径（如 `/api/k8s/secret/1/ops/app/reveal`），以及 detail 权限，都不覆盖 reveal。用户名为 `admin` 的账号与其他接口一样直接放行。

```json
{
  "code": 0,
  "data": {
    "cluster_id": 1,
    "namespace": "ops",
    "name": "app-secret",
    "type": "Opaque",
    "data": { "db-pass": "<base64 明文>", "db-user": "<base64 明文>" },
    "string_data": null,
    "keys": ["db-pass", "db-user"]
  },
  "message": "请求成功"
}
```

### 审计记录

由 reveal 接口在 service 层同步写入，不依赖审计中间件（中间件对 GET 请求不记录，也从不记录请求体或响应体）。

| 列 | 值 |
| --- | --- |
| `user_id` | 操作者 ID |
| `ip_address` / `user_agent` / `trace_id` | 请求元数据（表结构要求 IP 非空） |
| `http_method` / `endpoint` | `GET` / 实际请求路径 |
| `status_code` | 本次 reveal 的实际结果：成功 `200`；指定的键不存在 `400`；读取 Secret 失败时取 K8s API 返回的状态码（如 `404`、`403`），无法识别时 `500` |
| `operation_type` | `REVEAL`（已加入 `GetAuditTypes`） |
| `target_type` / `target_id` | `k8s_secret` / `集群ID/命名空间/名称` |
| `request_body` | 见下方 JSON，只含元数据和键名 |
| `response_body` | 始终为空 |
| `error_msg` | 失败原因（不含值） |
| `created_at` | 写入时间 |

```json
{
  "action": "k8s_secret_reveal",
  "user_id": 7,
  "username": "alice",
  "cluster_id": 1,
  "namespace": "ops",
  "name": "app-secret",
  "keys": ["db-pass", "db-user"],
  "success": true,
  "revealed_at": "2026-10-23T15:00:00Z"
}
```

## 4. 回写保护（保证前端改造前不丢数据）

- `PUT .../update`：如果请求 `data` 中某个键的值为空，而集群中该键当前有非空值，就保留原值。这样前端把脱敏后的详情原样回填提交时，真实值不会被清空。要删除一个键，请在请求里去掉它。`string_data` 按同样规则处理：某个键的值为空、而集群中该键当前有非空值时，这个键不会作为 `stringData` 提交（否则 K8s 合并 `stringData` 时会把 `data` 中的值覆盖成空），而是在 `data` 中保留原值；如果请求 `data` 对同名键给了非空新值，以请求的新值为准。集群中不存在的键按原样写入。副作用是：通过表单无法把一个已有非空值的键改成空值（`data` 和 `string_data` 都不行），需要时请改用 YAML 更新，在 YAML 中写 `""`。请求没有携带 `last-applied-configuration` 注解时，保留集群中的原注解。
- **SEC-29 同批次调整（SEC-25 P3）**：
  - `data` 缺省与空对象的区别：请求**没有 `data` 字段**或为 `"data": null` 时，视为“不修改 data”，**保留集群现有的全部 `data` 键和值**（改造前会把 `data` 整体清空）；请求带 **`"data": {}`** 时表示**显式清空**，集群中的 `data` 键全部删除。
  - `string_data` 空串的恢复范围收窄：某个键在 `string_data` 中为空串、集群中有非空原值时，**只有请求 `data` 也包含该键**（或请求根本没带 `data`，即整体保留）才恢复原值；请求 `data` 不包含该键时，该键视为已被删除，不会被恢复，也不会被写成空值（从 `stringData` 中移除，更新后集群中不再有该键）。
  - 示例：集群 `data` 为 `{a, b}`，请求 `{"data": {"a": ""}, "string_data": {"a": "", "b": ""}}` → 更新后 `a` 保留原值、`b` 被删除；请求 `{"string_data": {"a": ""}}`（无 `data`）→ `a`、`b` 都保留；请求 `{"data": {}}` → `a`、`b` 都删除。
- `PUT .../update/yaml`：值仍为 `<redacted>` 的键保持集群中的原值，用户只需修改需要变更的键。占位符对应的键在集群中不存在时拒绝更新。
- `POST .../create/yaml`：YAML 中含 `<redacted>` 时拒绝创建。
- **SEC-26 P2：YAML 解析与校验收紧**（`update/yaml` 与 `create/yaml` 都适用）：
  - 占位符的识别和恢复改为在类型化的 Secret 对象上进行，字段名按 encoding/json 不区分大小写的规则匹配（apiserver/kubectl 是区分大小写的）：`Data`、`DATA`、`StringData`、`stringdata`、`STRINGDATA` 等写法与 `data`/`stringData` 一样处理（更新时恢复原值，创建时拒绝）。改造前这些写法会绕过检查，把 `<redacted>` 原样写入集群。
  - 只接受**单个 YAML 文档**。含 `---` 分隔的第二个文档（包括末尾多出一个只有 `---` 的空文档）一律拒绝。开头的 `---` 和末尾的 `...` 仍可使用。
  - 拒绝**重复键**：任意层级完全相同的重复键，以及顶层字段、`metadata` 下只差大小写的字段（如同时写 `stringData` 和 `StringData`，或 `name` 和 `Name`）。使用合并键 `<<` 后又显式覆盖其中某个键的写法也会因为重复键被拒绝。同一映射中转成字符串后相同的键（如 `1` 与 `"1"`、`true` 与 `"true"`、`yes` 与 `"true"`、`1.5` 与 `"1.5"`）也按重复键拒绝，改造前以哪个为准取决于遍历顺序、结果不确定。
  - 下面这些写法会被视为仍含占位符：`data` 中 base64 解码后（含折行写法，比如 `|` 块里分两行）为 `<redacted>` 或其首尾带空白、NUL（`\0`）、零宽字符（U+200B、U+200C、U+200D、U+FEFF）的变体；`stringData` 中首尾带上述字符的 `<redacted>`（如 `"<redacted> "`、`"\u200b<redacted>"`）。这些字符只在首尾时去掉，夹在中间的不算占位符。更新时报错要求填写真实值，创建时拒绝。
  - YAML 库给出的解析错误按白名单改写，只保留错误类别和行号（如 `解析YAML失败: line 3: 映射键必须是字符串等标量`），不再带出任何键或值；本模块自己的校验报错（只差大小写的字段、非法 base64、仍为占位符）只包含字段名和键名。不回显 Secret 的值。
  - 入口大小上限 **1 MiB（1048576 字节）**：`update/yaml`、`create/yaml` 提交的 YAML 超过上限时在任何解析之前拒绝，报错为通用的 `YAML 超过 1 MiB 上限`。上限按整个 YAML 文本计算，`data` 中的值是 base64（约膨胀 4/3），因此通过 YAML 能提交的 Secret 实际数据量小于 Kubernetes 自身的 1 MiB 上限。
  - 更新时如果发生了占位符恢复，提交给集群的 YAML 由解析后的对象重新生成，字段顺序和格式会规范化（未识别的顶层字段会被丢弃，与改造前写入集群的结果一致）。
  - **已知行为差异**：因为字段名按 encoding/json 不区分大小写匹配，单独写的 `StringData:`（没有标准的 `stringData:`）也会被当作 `stringData` 处理；而 apiserver/kubectl 区分大小写，会把它当作未知字段。按 TL 决定保留这一兼容行为，前提是任何情况下都不会把占位符写回集群：更新时其中的 `<redacted>` 恢复为原值，创建时拒绝。
  - **升级风险**（TL 已接受）：
    - 使用合并键 `<<` 后再显式覆盖其中某个键的 YAML，改造前能提交，现在按重复键拒绝，需要去掉覆盖或展开合并键。
    - 末尾多出一个只有 `---` 的空文档（常见于模板拼接或 `kubectl get -o yaml` 结果拼接）的 YAML，改造前能提交，现在按多文档拒绝，需要删掉末尾的 `---`。

## 5. 前端待办

“查看值”“复制”“编辑”三个入口需要明文时，改为先调用 reveal（可以按 `keys` 只取需要的键），不再读取 detail/yaml 中的值。

## 6. 回滚

本次没有数据库迁移，回退提交即可恢复旧行为。回退后，reveal 接口在 `cl_system_apis` 中登记的那一行会留下，但不会产生影响；如需清理，删除 `name = 'k8s:secret:reveal'` 的记录及其在 `cl_system_role_apis` 中的关联即可。
