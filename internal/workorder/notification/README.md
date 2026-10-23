# 工单通知接收人和通道

实现的通道为 `email` 与 `feishu`。配置可以持久化后再由管理员配置传输，但发送时必须存在已启用通道；不支持的 SMS/Webhook 请求直接拒绝，未知字符串不会推断为邮件。

1.0 自动事件通知仅支持 `trigger_type=immediate`。延迟、定时、重复和条件触发没有执行引擎，写入这些类型/参数会拒绝，历史非即时配置不会自动发送，也不能直接重新启用；可以读取、修改普通元数据或禁用，明确改为即时后再启用。手动测试发送是用户明确发起的一次动作，不代表高级触发已实现。

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
