# 内置超级管理员角色与精确鉴权（SEC-07 / SEC-24）迁移说明

## 变更摘要

- SEC-07：去掉所有按用户名 `admin` 判断特权的逻辑，改为判断用户是否持有**启用中的内置超级管理员角色**（`cl_system_roles.code='super_admin' AND is_system=1 AND status=1`）。
- SEC-24：`CheckAuth` 改为“HTTP 方法 + gin 路由模板”精确比较；新增 `GET /api/user/permissions`，契约见 [docs/api/user-permissions-v2.md](../api/user-permissions-v2.md)。

## 表结构

**没有新表、新列、新索引，没有修改任何 GORM 模型 tag。** `pkg/di/init.go` 的 AutoMigrate 不会因本次改动产生新的 DDL。

## 种子数据（启动引导，幂等）

每次启动在权限目录与 Mock 初始化之后执行 `bootstrapSuperAdmin`，在一个事务里：

1. 查找 `code='super_admin' AND is_system=1` 且未删除的角色，按 `id` 取最小的一条；没有则插入：

   | 列 | 值 |
   | --- | --- |
   | `name` | `超级管理员` |
   | `code` | `super_admin` |
   | `description` | `内置超级管理员角色，由启动引导维护，不可修改或删除` |
   | `status` | `1` |
   | `is_system` | `1` |

   该角色**不绑定任何接口**，超管在鉴权中直接放行。
2. 查找所有未删除且用户名**严格等于** `admin` 的用户（Go 侧大小写敏感比较，避免 MySQL 排序规则把 `Admin` 也匹配进来），若尚未绑定该角色则在 `cl_system_user_roles` 插入一行。已绑定时不写库。
3. 若已存在的内置超管角色 `status != 1`（只能是有人直接改库），引导**不会自动重新启用**，只打印告警；此时 admin 不再是超管，需人工处理（见“应急”）。
4. 如果存在 `code='super_admin'` 但 `is_system=0` 的普通角色（历史上手工创建的同名角色），引导**不会提升**它，只告警并另建系统角色；持有那个普通角色的用户不会获得超管权限。

幂等性：重复启动不会产生新行。多副本**同时首次**启动时，因角色表没有唯一索引，极小概率会各插入一条系统超管角色；这不影响正确性（判断依据是 `code + is_system`，任一条都生效），下次启动会统一绑定 `id` 最小的那条，多余的一条可在确认无用户绑定后手工删除。

没有设置 `BOOTSTRAP_ADMIN_PASSWORD` 且未开启 Mock 的空库不会创建 admin，引导只建角色不绑定用户。

## 权限相关的其他行为变化

- 角色接口 `POST /api/role/assign_users`、`POST /api/role/revoke_users`：涉及内置超管角色时，调用者必须是超管，否则 `403`。防止“有分配角色权限的普通用户把自己提升为超管”。
- `PUT /api/role/update/:id`、`DELETE /api/role/delete/:id` 原本就拒绝系统角色；`POST /api/role/create` 原本就强制 `is_system=0`，因此无法通过 API 伪造系统超管角色。
- 由于引导每次启动都会把 `admin` 绑回超管角色，**撤销 admin 的超管角色只在下次重启前有效**。
- 用户名 `admin` 仍是保留名，注册接口拒绝（大小写不敏感），以免被他人注册后在重启时自动获得超管。

## 上线步骤

1. 备份 MySQL（至少 `cl_system_roles`、`cl_system_user_roles`、`cl_system_role_apis`、`cl_system_apis`）。
2. 上线前用只读 SQL 盘点会受精确鉴权影响的授予（在数据库副本或只读账号执行）：

   ```sql
   -- 含 * 但不是“恰好一个且位于末尾”的授予：上线后不再生效
   SELECT id, path, method FROM cl_system_apis
   WHERE deleted_at = 0 AND path LIKE '%*%'
     AND NOT (path LIKE '%*' AND path NOT LIKE '%*%*') ;
   -- 被角色引用、不含 * 也不含 : 的授予：若写的是真实路径（如 /api/tree/cloud/9/status）
   -- 而不是模板，上线后不再生效；请与 gin 路由表人工核对
   SELECT DISTINCT a.id, a.path, a.method FROM cl_system_apis a
   JOIN cl_system_role_apis ra ON ra.api_id = a.id
   WHERE a.deleted_at = 0 AND a.path NOT LIKE '%*%' AND a.path NOT LIKE '%:%' ;
   -- 已存在的同名角色（引导不会提升它们）
   SELECT id, name, code, is_system, status FROM cl_system_roles
   WHERE deleted_at = 0 AND (code = 'super_admin' OR name = '超级管理员');
   ```
   第一条结果里形如 `/api/v1/*path` 的 gin 通配模板仍然有效（与模板完全相等），其余需要改成末尾 `*` 或具体模板。
   第二条结果中，启动时 `bootstrapPermissionCatalog` 自动补录的条目都是路由模板，不受影响；需要关注的是手工录入的真实路径。上线后也可以用受影响用户调用 `GET /api/user/permissions`，列表里缺失的就是不再生效的授予。
3. 部署新版本，启动日志确认无 `初始化超级管理员角色失败`。
4. 验证：以 admin 登录，`GET /api/user/permissions` 返回 `super_admin=true`、`permissions=["*"]`；以普通测试账号验证契约第 3.1 节的拒绝用例。

## 回滚

应用回滚到旧镜像即可，**不需要删除数据**：

- 旧版本按用户名判断 admin，不读取新角色；新增的系统角色和绑定行对旧版本无副作用（旧版本鉴权会遍历该角色的接口列表，而它是空的）。
- 旧版本的“`:param` 匹配任意段”宽松鉴权会随回滚恢复，即 SEC-24 修复的外溢重新出现，回滚需安全负责人知情。
- 如需彻底清理种子数据（可选，先确认没有其他用户绑定该角色）：

  ```sql
  -- 先查
  SELECT r.id, ur.user_id FROM cl_system_roles r
  LEFT JOIN cl_system_user_roles ur ON ur.role_id = r.id
  WHERE r.code = 'super_admin' AND r.is_system = 1 AND r.deleted_at = 0;
  -- 确认后删除（:role_id 为上面查到的 id）
  DELETE FROM cl_system_user_roles WHERE role_id = :role_id;
  DELETE FROM cl_system_role_apis  WHERE role_id = :role_id;
  DELETE FROM cl_system_roles      WHERE id = :role_id AND is_system = 1;
  ```
  注意：只要新版本再次启动，引导会重新创建并绑定。

## 应急

- admin 被误撤销超管：重启服务即可由引导重新绑定；不能重启时，用有权限的 DBA 账号插入 `cl_system_user_roles(user_id, role_id)`。
- 内置超管角色被改库禁用：`UPDATE cl_system_roles SET status = 1 WHERE code='super_admin' AND is_system=1;`，然后让 admin 重新登录（权限每个请求实时读库，无需重启）。
