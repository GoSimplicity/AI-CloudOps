# 工单通知接收人和通道

实现的通道为 `email` 与 `feishu`。配置可以持久化后再由管理员配置传输，但发送时必须存在已启用通道；不支持的 SMS/Webhook 请求直接拒绝，未知字符串不会推断为邮件。

自动事件通知支持 `immediate`、`delayed`、`scheduled` 和 `conditional`。延迟以事件创建时间为基准，`delay_minutes` 为 1–10080；定时使用未来 366 天内的 UTC `scheduled_time`，早于事件创建时间的排期记为跳过。条件只接受事件快照中的数字 `status`（1–6）和 `priority`（1–3），多个条件同时匹配；不支持表达式或代码。可选 `repeat_interval`（1–10080 分钟）和 `repeat_count`（2–100，包含首次）必须成对设置。各次重复从前一次确认成功的时间起计算；失败、取消或结果未知时不会自动创建下一次，不追赶积压次数。修改触发类型前可通过 `clear_trigger: true` 显式清除原排期、条件和重复字段，再提交新的字段。历史高级配置默认不执行，须通过同样验证并显式启用或重新提交触发字段后才会生成排期。

工作器首次消费事件时将配置、目标和消息固化为子事件，与父事件计划在同一数据库事务中保存。已生成计划不随配置编辑、禁用或删除而改变；如需停止，应取消尚待处理的子事件。创建计划后父事件可显示 `planned`，具体投递结果请查看子事件和投递记录。排期唯一键为父事件、通知配置和次数；下一次与前次成功状态在同一事务保存。排期创建或完成持久化持续失败会标记 `failed`，已确认投递不会重发；人工核对并修复后，可通过原事件 retry 接口恢复排期，操作写入审计。

`POST /api/workorder/notification/events/:id/cancel` 需要独立精确权限和 `{ "reason": "已核对停止发送" }`，操作人由可信登录身份确定并写入审计。只能取消待处理的排期子事件；发送中、结果未知或已有成功投递时拒绝取消，不能保证撤回在途消息。取消会停止该重复链并清除计划和快照。数据库迁移应在所有服务升级前完成，升级期间停止旧版本 API 和工作器；旧版不递增配置版本，混跑不能保证并发更新安全，也不能正确执行新排期。

通知创建/更新/详情增加 `recipient_addresses`：

```json
{
  "channels": ["email", "feishu"],
  "recipient_types": ["dept", "custom"],
  "recipient_depts": ["12"],
  "recipient_addresses": [
    {"channel": "email", "address": "operator@example.test", "name": "值班邮箱"},
    {"channel": "feishu", "address": "oc_example", "name": "值班群"}
  ]
}
```

`recipient_depts` 保留字符串 ID 数组，只查询指定部门的启用用户，不隐式包含子部门。`custom` 必须有明确地址；地址不能塞进 `recipient_users`。每个地址的 channel 必须存在于通知 channels 中，最多 100 个；name 可省略且不超过 100 字符。邮件为单个纯邮箱，拒绝显示名、多个地址和控制字符；飞书只接受不超过100字符的 `ou_` / `on_` / `oc_` 加字母数字 ID。禁止自定义 URL。

接收人先按用户 ID 去重，再在每个通道按最终规范化地址去重，所以创建人、指定用户、角色、部门和自定义地址指向同一邮箱只发送一次。不同通道不会混用地址。无接收人、缺少有效地址、数据库错误、通道不可用或供应商拒绝均返回错误。多通道发送并非分布式事务：一个通道成功、另一个失败时会返回失败并保留各接收人的发送日志；不要据此推断所有通道都未发送。

更新时省略 `recipient_addresses` 保留原值，明确 `[]` 清空；如果同时仍选 custom，会因缺少地址被拒绝。数据存储是独立 typed JSON 列，不与用户 ID 混存。

`POST /api/workorder/notification/test/send` 请求 `{notification_id,channel,recipient}`。配置只有一个通道时 channel 可省略，多通道时必须指定；测试只发选中的那个通道。手动 `/send` 的 channels 必须是单元素数组。所有测试发送都需要明确地址，不使用默认收件人。

飞书群 ID 通过正式消息 API 的 `receive_id_type=chat_id` 发送，保留 tenant token 认证，不把群 ID 拼接成机器人 webhook。收件人需要满足飞书应用可见范围；群通知需要机器人已加入群。传输成功必须同时符合 HTTP 和业务响应，不接受 `Success=false` 或空响应。访问令牌不写日志，刷新有互斥保护。

## 本地验收

```sh
GOTOOLCHAIN=auto go test -race ./internal/system/... ./internal/workorder/...
```

- `notification/delivery_test.go`：真实本地 SMTP 收件箱收到 MIME 解码后的正文；本地 HTTP 收件箱验证飞书 chat_id、认证头、失败传播，未连接公网。
- `service/notification_integration_test.go`：真实 SQLite 部门/成员及日志，部门用户与自定义地址混合发送到本地 HTTP 收件箱，重复地址只收一次、冻结用户不收、失败状态写入日志。
- `dao/notification_address_test.go`：typed 地址数据库存储/读取/明确清空。

这证明协议和本地持久化链路，不代表真实企业 SMTP 或飞书租户已配置、授权或成功投递。

依据：[飞书官方消息接口](https://open.feishu.cn/document/server-docs/im-v1/message/create)、[官方 Go SDK 的接收人类型与响应](https://github.com/larksuite/oapi-sdk-go/blob/v3_main/service/im/v1/model.go)。

## 来源范围与重试

通知同时匹配流程、分类和模板范围；来源在创建工单时从已验证对象保存不可修改的快照，历史未知来源不会命中具体分类或模板。详见[来源匹配](../../../docs/release/notification-scope.md)。

`max_retries=0` 表示仅首次发送，N 表示额外重试 N 次，省略继承默认；`retry_interval` 单位为分钟。同步通知由 Manager 执行，异步通知由实际 Asynq worker 执行单层重试，避免重试次数相乘。批量部分失败不重放整个批次。见[执行与真实 Redis 验证](../../../docs/release/notification-retry-policy.md)。

飞书诊断日志不记录收件地址、消息正文、完整请求 URL 或供应商原始错误正文，仅保留 MessageID、状态及业务错误码等诊断信息。

## 工单事件可靠投递

工单创建、模板创建、编辑、删除、提交、指派、审批、拒绝、取消、完成和退回的事件，在工单变更事务内写入数据库 outbox。事件保存创建时的工单快照；删除通知使用删除前快照。写入失败会回滚本次工单变更。工作器使用有限租约领取、过期后恢复；投递记录以事件、配置、渠道和规范化地址哈希区分，已经确认成功的接收人不会被后续普通重试再次发送。

工作器每次领取只调用一次渠道：飞书明确返回非零业务码时，按配置的额外重试次数和分钟间隔重试；未启用通道、网络中断、超时、未知响应以及SMTP无法确认最终回执等结果按 `uncertain` 停止自动重试。当前SMTP库不提供可验证的拒绝阶段，**所有SMTP传输错误均按 uncertain 处理，不自动重试**；包括表面上的550等拒绝。SMTP底层在上下文取消后可能仍运行，不能认为取消代表未发送。`sending` 遗留在过期租约上时也视为不确定，不自动重发。外部系统接受消息后、数据库记录确认成功之前若进程崩溃，仍无法保证恰好一次，必须先核对供应商发送记录。

事件状态：`pending` 等待、`processing` 领取、`planned` 排期已生成、`success` 有已确认投递、`skipped` 条件未满足或没有匹配配置、`cancelled` 排期已取消、`failed` 失败需处理、`uncertain` 投递结果未知。配置与收件人在**首次消费时**读取；在实际发送前先持久化完整计划（渠道、地址、模板结果与重试参数），之后的重试和人工恢复严格沿用此计划。首次消费前禁用或删除配置可能导致 `skipped`，该状态不是投递成功；事件创建时快照配置不属于当前合同。失败/不确定事件会保留工单快照和投递计划，成功/跳过/取消后清除。

操作员可在独立API权限控制下查看和处理：

- `GET /api/workorder/notification/events?status=uncertain` 查看事件，分页使用 `page`/`size`
- `GET /api/workorder/notification/events/:id/deliveries` 查看渠道、地址哈希、尝试次数和状态；不返回事件快照和消息正文
- `GET /api/workorder/notification/events/:id/resolutions` 查看人工处置审计
- `POST /api/workorder/notification/events/:id/retry` 请求 `{ "reason": "已修复配置并核对" }`，仅允许对零投递记录的 failed 事件重试；单独精确授权并写入审计
- `POST /api/workorder/notification/events/:id/resolve` 请求 `{"delivery_id":1,"action":"delivered|retry","reason":"已核对供应商投递结果"}`；必须选择 `delivered` 或 `retry` 中的一项。只允许处置已终止或不确定的事件；操作者来自JWT，事务内写处置记录。`retry` 表示操作者明确同意重新投递并重置该收件人重试预算；先确认原发送已停止及供应商实际结果，避免重复。reason 请填写核对依据，不填写密钥或消息正文

各GET和POST按方法及路由单独授权；人工处置POST要求精确权限，旧通配符权限不隐式覆盖。现有超级管理员角色按全局规则放行。路由自动登记到系统API目录，不自动授予普通角色。事件快照可能包含工单敏感字段，数据库、备份及访问权限应按工单数据保护；列表接口不会暴露快照。

事件快照仅保存发送所需工单字段，不保留表单内容；事件进入 success/skipped 时立即清除快照与自定义正文。`WORKORDER_EVENT_RETENTION_DAYS` 控制成功或跳过事件的保留天数（默认90，范围1–3650）；工作器分批清理到期的事件及其接收人状态。带人工审计的事件永久保留；pending、processing、failed、uncertain 及其接收人状态不自动删除。数据库备份应遵循同等数据保护与保留策略。

所有 API 和工作器实例必须使用相同且明确的 MySQL DSN `loc`，例如 `loc=UTC` 或 `loc=Asia%2FShanghai`；启动会拒绝 `loc=Local`、缺省值和与数据库时区标记不一致的配置。新空库会自动登记；发现历史通知事件或已有定时配置但没有 `cl_workorder_notification_clock` 标记时，启动会拒绝并要求人工核对，绝不自动转换时间。现有 `loc=Local` 安装必须先停止所有旧版 API 和工作器、备份数据库，确认原机器的真实 IANA 时区以及历史 DATETIME 的语义；可以在确认无需转换时改为相同的明确 IANA 时区，或由操作者按整个数据库的数据合同完成时间转换后改用 UTC。完成核对/迁移后，由有数据库管理权限的操作者显式写入唯一标记（`id=1`、`time_zone` 为选定的时区名），然后统一启动新版实例。不能仅修改 DSN 或混用不同 `loc`，否则历史时间会被错误解释。
