# 部门管理契约

部门是单父级组织单元；用户只属于一个部门，`department_id=0` 表示未分配。用户查询返回 ID，不嵌套部门对象。组织管理接口只允许已认证的 `admin`，并继续经过系统 JWT/RBAC 中间件。

| 方法与路径 | 请求 / 返回 |
| --- | --- |
| GET `/api/department/list` | `page/size/search`，返回 `data.items` 与 `data.total` |
| GET `/api/department/:id/detail` | 返回部门对象 |
| POST `/api/department/create` | `{name,parent_id,description}`，返回部门对象 |
| PUT `/api/department/:id/update` | 相同字段全量更新，返回部门对象 |
| DELETE `/api/department/:id/delete` | 有成员或子部门时拒绝 |
| PUT `/api/department/user/:id` | `{department_id:正整数或0}`；0 清空归属 |

部门对象包含 `id/name/parent_id/description/created_at/updated_at`。名称去首尾空白后 1–100 字符、描述不超过 500 字符、根部门 `parent_id=0`；同一上级下不可重名，上级不得为自身或子部门。写操作使用事务；数据库唯一索引保障同级名称唯一，删除、创建子部门和用户分配通过部门行锁协调。

部门归属不属于个人资料编辑字段，`PUT /api/user/profile/update/:id` 若携带 `department_id` 会拒绝，包括管理员请求；使用独立分配端点。注册用户默认未分配。不要在前端通过隐藏字段模拟权限。

启动迁移增加 `cl_system_departments` 与 `cl_system_users.department_id`；既有用户保持未分配，不猜测原组织关系。

验证：`GOTOOLCHAIN=auto go test -race ./internal/system/...`。`api/department_handler_test.go` 使用真实本地 HTTP 服务及隔离 SQLite，验证权限、持久化、环、非空删除、分配与清空。测试身份注入只存在测试代码，不是生产认证路径。

依据：[GORM 事务](https://gorm.io/docs/transactions.html)、[GORM Update 零值语义](https://gorm.io/docs/update.html)。
