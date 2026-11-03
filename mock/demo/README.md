# 本地演示数据

独立入口 `cmd/demo-seed`，不会被应用启动、Mock 开关或业务 worker 调用。

```sh
go run ./cmd/demo-seed
```

默认仅打印 88 条 fixture 清单，不读取 DSN、不连接数据库。

| 模块 | 记录 | 覆盖 |
| --- | ---: | --- |
| CMDB | 13 | 4 个树节点、9 台 TEST-NET 主机，开发/预发/生产演示环境、停止/异常状态 |
| 监控 | 19 | 空实例池、禁用采集/告警/记录/通知规则、5 条已恢复告警 |
| 值班 | 3 | 禁用组与已有管理员的成员关联 |
| 工单 | 30 | 分类、表单、草稿流程、禁用模板、六种工单状态、时间线及评论 |
| 系统 | 11 | 团队、普通无 API 授权角色、冻结且无登录密码的成员 |
| 管理审计 | 6 | 显式演示历史记录，无请求凭据或原始数据 |
| 通知配置 | 3 | 全禁用、仅站内信渠道、无事件/队列/投递 |
| Cron | 3 | 全部禁用，示例历史统计，不执行命令 |

## 导入

仅对操作者确认归属的本机开发数据库执行。现有 schema 和管理员应先由正常迁移/引导工具准备；本工具不迁移、不创建数据库、不创建管理员、不改变登录口令。

将已确认的本地 DSN 置于 `CLOUDOPS_DEMO_DSN`，不要把凭据作为命令参数或写进源码：

```sh
ENV=development go run ./cmd/demo-seed --apply --database CONFIRMED_DATABASE --owner-id EXISTING_USER_ID
```

环境必须明确为 `development/dev`；TCP 地址必须是字面量 loopback，拒绝 DNS 名、远程 IP、Unix socket、系统库及多语句/本地文件导入。数据库名需与 DSN 完全相同。单连接 advisory lock 和事务串行保护整次导入，输出只有创建/已存在数量与模块统计，不输出 DSN、凭据或原始数据库错误。

全部实体使用 `demo-local-v1-` 前缀或对应工单序号。重复导入只补不存在的实体，不更新、恢复或删除已有记录；关联仅在实体首次创建时添加，保留后续人工编辑或解绑。命名被删除、归属冲突或原演示任务被启用时直接失败并回滚，不擅自覆盖。改名后的旧演示实体也会保留。

任何已有内置 Cron/通知/同步 worker 的开关不由 seed 管理。首次导入前必须确认独立开发实例的这些任务均已停止，确保不会因已有启动即执行任务触发外部请求。seed 不创建云账号、集群、kubeconfig、外部通知地址、真实 SSH 凭据或通知队列事件。监控池实例为空，采集/规则/发送/值班与 Cron 全部禁用。

## 校验

```sh
go test -race ./mock/demo ./cmd/demo-seed
```

纯单元测试不连接数据库，覆盖 manifest/依赖顺序、所有 fixture 的禁用与空凭据合同、loopback/显式库名及危险 DSN 参数拒绝。实际幂等性须对已确认的隔离本地 MySQL 连续执行两次 `--apply`，第二次应 `created=0`；这与纯单元测试的验证范围不同。

管理员导航由现有启动引导的系统 `super_admin` 角色和权限接口负责，不依赖旧 ApiMock 的通配权限记录；演示角色不会给管理员增加或撤销权限。

收件箱保持为空：真实写入合同需要事件租约及成功投递账本，不用零事件 ID 或孤儿记录伪造投递历史。预览空/未读/已读态应使用隔离前端 API fixture，不创建 outbox、delivery 或 queue 数据。已有 API 目录由启动工具维护，seed 不重复添加目录项或授予角色权限。

禁用配置的安全依据：`notificationDAO.GetActiveNotificationsByEventType` 只选择 `status=1`；`TestSendNotification` 对禁用配置直接拒绝；通知事件 worker 只 claim 已存在的 pending/processing 事件。本 seed 仅直接插入 `status=2`、`is_default=2`、`trigger_type=immediate` 且无定时字段/外部收件地址的 in_app 配置，不调用通知 service、不创建 outbox/delivery/queue。安全单元测试逐条验证上述状态与渠道，修改为启用或外部渠道会被拒绝。
