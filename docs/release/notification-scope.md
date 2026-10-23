# 工单通知来源、范围与历史配置边界

日期：2026-10-23

## 来源快照与迁移

工单实例新增可空 `template_id`、`category_id` 列，由现有 `InitTables` 对 `WorkorderInstance` 的 `AutoMigrate` 注册添加；本批没有独立数据回填。上线账号须具备现有迁移所需的表结构修改权限

- 普通创建从实际验证通过的已发布表单读取分类，保存分类快照；没有模板来源
- 从模板创建保存已验证模板的 ID 和模板自身分类；模板分类为空时保持空，不回退到流程或表单分类
- 创建请求 DTO 不接受这两个来源字段，客户端自报同名字段不能伪造来源
- 两列使用 GORM `<-:create`，普通实例更新不能改写；之后修改模板、表单分类也不改变已创建实例的快照
- 历史实例保持 `NULL`，不从当前可变的模板或流程推断历史来源

旧版本程序可忽略新增可空列。回滚程序不要求删除列；若主动删除则丢失本批之后记录的来源，不能通过再次迁移恢复。升级前应按现有流程备份数据库

## 通知范围

中央处理入口在读取收件人和调用发送器之前检查范围。配置的 `process_id`、`category_id`、`template_id` 逐项使用 AND：未配置的一项不限制，配置的一项必须为正 ID 且与实例来源一致

历史实例的分类或模板为 `NULL` 时，不匹配对应的具体范围；未限制来源的全局配置仍可匹配。分类来源按上述创建路径定义，不使用流程分类替代，也不声称实现多组织或租户隔离

## 即时通知与历史高级参数

1.0 自动通知执行即时触发。即使历史记录的 `trigger_type` 为 `immediate`，只要仍带有 `scheduled_time`、`repeat_interval` 或非空 `trigger_condition`，也在收件人查询之前拒绝自动发送，避免将未实现的调度或条件配置当成立即发送

更新请求显式设置启用状态时，对旧记录与本次字段合并后的最终配置执行同样校验；只提交 `status` 或将触发类型改成 `immediate` 不能绕过旧参数。历史配置仍允许停用和普通元数据编辑

显式空条件对象 `{}` 可以清除条件。现有更新 DTO 无法用 `null` 清除已有的定时或重复指针字段；这类记录不能重新启用，应新建不含高级参数的即时通知配置。本批不增加高级调度能力

## MySQL 幂等更新

MySQL 默认按实际变更行数返回 `RowsAffected`。相同值且更新时间戳相同时，现存记录也可能返回 0。通知 DAO 在零变更时使用现有软删除作用域核对 ID 是否仍存在：存在则更新成功，不存在或已软删除才返回不存在错误

真实隔离 MySQL 回归固定 GORM 时钟，确认底层同值 `UPDATE` 确实返回 0，并验证 DAO 连续两次同值写成功、记录保留、缺失 ID 与软删除记录仍被拒绝

## 已执行验证与复验命令

已执行并通过以下三包完整 race 检查；SQLite 测试验证真实创建与持久化、来源不可伪造和不可改写、迁移幂等、历史未知来源、模板空分类不回退。范围与激活表驱动测试覆盖 12 个处理入口场景和 8 个激活场景，不匹配范围及历史高级参数均为零收件人查询、零发送

```sh
DEVELOPER_DIR=/Library/Developer/CommandLineTools GOTOOLCHAIN=auto go test -race ./internal/workorder/service ./internal/workorder/dao ./internal/workorder/notification
```

以下两项也已在本机 `cloudops` 编排的健康 MySQL 容器中实际通过。复用既有隔离工具创建随机 `schema_<nonce>` 数据库及仅获该库权限的同名临时用户；不使用业务库。每项结束后均核对自有数据库和用户已清理

```sh
DEVELOPER_DIR=/Library/Developer/CommandLineTools python3 - <<'PY'
import importlib.util

spec = importlib.util.spec_from_file_location(
    "schema_smoke", "scripts/integration/schema_smoke.py"
)
smoke = importlib.util.module_from_spec(spec)
spec.loader.exec_module(smoke)
smoke.isolated_mysql(
    package="./internal/workorder/service",
    test="TestMySQLInstanceNotificationSource",
)
smoke.isolated_mysql(
    package="./internal/workorder/dao",
    test="TestMySQLNotificationIdempotentUpdate",
)
PY
```

对应测试文件：

- `internal/workorder/service/notification_scope_test.go`
- `internal/workorder/service/instance_notification_source_test.go`
- `internal/workorder/service/instance_notification_source_mysql_integration_test.go`
- `internal/workorder/dao/notification_idempotent_mysql_integration_test.go`

上述验证不向真实邮件、飞书、Webhook 或云服务发送请求，不代表高级调度已实现。重试执行合同见 [notification-retry-policy.md](notification-retry-policy.md)
