# SEC-35：存量 admin 行 account_type 由服务账号改为普通用户

> **草稿，不执行，等用户批准。** 本文中的 SQL 仅供评审，未在任何数据库上执行；批准前不得执行 UPDATE。

## 1. 背景

- SEC-27（00cc906）起，`bootstrapAdmin`（`internal/app/bootstrap.go:66`）与 mock 管理员（`mock/user_mock.go:44` `AdminAccountType`）创建的 admin 都使用 `constants.AccountTypeUser`。
- SEC-27 之前的 mock（188e907:`mock/user_mock.go:39`）以 `AdminAccountType = 2` 创建 admin，代码改动不会修改库中已有行，因此曾开启过 mock 的库里 admin 行的 `account_type` 仍为 2。
- 常量（`internal/constants/user.go:38-39`）：`AccountTypeUser = 1`（普通用户），`AccountTypeService = 2`（服务账号）。
- 表名与列名以 model 为准（`internal/model/system_user.go`）：表 `cl_system_users`（`TableName()`），列 `id`、`username`、`account_type`、`updated_at`、`deleted_at`。`deleted_at` 为 `gorm.io/plugin/soft_delete` 的整型软删除列，未删除行为 `0`。
- 生产库为 MySQL（`gorm.io/driver/mysql`），以下 SQL 按 MySQL 语法书写。

## 2. 影响范围

- 仅影响 `cl_system_users` 中 `username = 'admin' AND account_type = 2` 的行，预期为 0 或 1 行（用户名无唯一约束，可能多行，须以核对 SQL 实际结果为准）。
- 不改口令、启用状态、角色、API 关联（`cl_user_apis`）或其他用户；不涉及表结构变更。
- 包括软删除的 admin 行（`deleted_at <> 0`）：草稿默认一并更正，保持与代码一致；如审批人希望只改有效行，在 WHERE 中追加 `AND deleted_at = 0`。
- 没有开启过 mock、或 admin 由 `BOOTSTRAP_ADMIN_PASSWORD` 创建的库，核对结果应为 0 行，无需执行。

## 3. 执行前备份（必须）

1. 按 `docs/release/backup-restore.md` 使用 `scripts/release-backup.py` 生成全库单库备份，并完成恢复核验。
2. 另做一份最小范围的行级备份，便于精确回滚（只保存 id 与原值，不含口令等字段）：

```sql
-- 草稿，不执行，等用户批准
CREATE TABLE cl_system_users_bak_sec35 AS
SELECT id, username, account_type, updated_at, deleted_at
FROM cl_system_users
WHERE username = 'admin' AND account_type = 2;
```

## 4. 只读核对 SQL

```sql
-- 草稿，不执行，等用户批准（只读）
-- 4.1 计数：待更正的 admin 行
SELECT COUNT(*) AS cnt
FROM cl_system_users
WHERE username = 'admin' AND account_type = 2;

-- 4.2 明细：列出所有 admin 行，便于确认同名多行与软删除情况
SELECT id, username, account_type, deleted_at
FROM cl_system_users
WHERE username = 'admin'
ORDER BY id;
```

执行 UPDATE 前记录 4.1 的计数 N 与 4.2 的 id 列表，并与备份表行数核对一致。

## 5. UPDATE 草稿

```sql
-- 草稿，不执行，等用户批准
START TRANSACTION;

UPDATE cl_system_users
SET account_type = 1            -- constants.AccountTypeUser
WHERE username = 'admin' AND account_type = 2;
-- 期望影响行数 = 第4.1节计数 N；不一致则 ROLLBACK

-- 复核：应返回 0
SELECT COUNT(*) FROM cl_system_users WHERE username = 'admin' AND account_type = 2;

COMMIT;   -- 确认影响行数与复核结果后再提交，否则 ROLLBACK;
```

说明：不显式修改 `updated_at`，保持与应用层原样；如审批要求留痕，可追加 `, updated_at = NOW()`（回滚时由备份表恢复）。

## 6. 回滚 SQL

```sql
-- 草稿，不执行，等用户批准
START TRANSACTION;

UPDATE cl_system_users u
JOIN cl_system_users_bak_sec35 b ON b.id = u.id
SET u.account_type = b.account_type,
    u.updated_at   = b.updated_at
WHERE u.username = 'admin';
-- 期望影响行数 = 备份表行数

COMMIT;

-- 确认无需回滚后（观察期结束），再清理备份表：
-- DROP TABLE cl_system_users_bak_sec35;
```

若行级备份不可用，用第 3 节的全库备份在独立库恢复后按 id 比对恢复。

## 7. 对权限的影响

- 权限判断按**用户名**而不是 `account_type`：`internal/middleware/auth.go:155` 只允许 `admin` 调用 `/api/user/signup`，`internal/middleware/auth.go:162` 对 `user.Username == "admin"` 直接放行。
- `internal/middleware/login.go:93-95` 每次请求会用库中的 `account_type` 覆盖 JWT 中的值，但当前非测试代码中没有任何按 `AccountType` 做的权限分支（服务账号已不再自动绕过角色权限，见 README「生产启动安全要求」）。
- 因此本次更正**不改变 admin 的任何权限**，也不需要 admin 重新登录；影响仅限于展示与按 `account_type` 过滤的用户列表（`internal/system/service/user_service.go:146-154`），admin 将从「服务账号」筛选结果移到「普通用户」中。
- 反向风险：若将来引入按 `account_type` 的权限逻辑，未更正的存量 admin 会与新建 admin 行为不一致——这正是 SEC-35 要消除的差异。

## 8. 执行顺序（批准后）

1. 维护窗口内完成第 3 节两份备份；
2. 执行第 4 节只读核对，记录 N 与 id 列表；
3. 执行第 5 节事务，核对影响行数 = N 后提交；
4. 用 admin 登录验证可正常访问用户管理与 `/api/user/signup`；
5. 观察期结束后清理备份表。

**草稿，不执行，等用户批准。**
