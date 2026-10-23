# 工单通知渠道筛选契约

通知列表使用 `GET /api/workorder/notification/list?page=1&page_size=10&channel=email`。`channel` 可省略，允许 `email`、`feishu`、`sms`、`webhook`；非法值在请求绑定和直接 DAO 调用中均拒绝。分页字段保持 `page_size`，没有添加与通用 `size` 的猜测别名。

筛选在数据库统计总数和 offset/limit 之前执行，返回匹配记录的真实 `total`。与名称、状态、流程/模板/分类及默认配置筛选叠加，响应仍为现有 `items` / `total` 格式。

发布数据库 MySQL 8.4 使用 `JSON_OVERLAPS` 与 JSON 数组类型判断，SQLite 使用 `json_each` 的顶层字符串比较。渠道按大小写敏感的完整数组元素匹配，`email_backup`、`EMAIL` 或嵌套数组中的 `email` 不会被当成 `email`。不使用 LIKE 匹配序列化 JSON，也不只过滤当前页。MySQL 官方语义见 [JSON 搜索函数](https://dev.mysql.com/doc/refman/8.4/en/json-search-functions.html#function_json-overlaps)。

应用写入的 `channels` 一直使用 StringList 的 JSON 字符串数组格式。本次没有数据迁移；缺少合法数组结构的旧/损坏内容在带渠道筛选时不匹配且不会触发 JSON 解析 SQL 错误，省略渠道时仍沿用原有列表读取行为。需要重新保存此类记录才能参与数组成员筛选。

已执行先红后绿：原实现忽略渠道，分页总数与非法枚举测试失败；修复后同一套数据在真实 SQLite 和随机限权 MySQL 库中通过。覆盖三条匹配记录跨四页、混合渠道、所有枚举、状态叠加、大小写/子串/非数组/无效 JSON/嵌套反例及省略渠道。隔离 MySQL 工具确认测试确实 pass 并清理自身数据库和用户为 0。

```sh
DEVELOPER_DIR=/Library/Developer/CommandLineTools GOTOOLCHAIN=auto go test -race ./internal/workorder/dao ./internal/workorder/api ./internal/workorder/service
DEVELOPER_DIR=/Library/Developer/CommandLineTools GOTOOLCHAIN=auto go vet ./internal/workorder/dao ./internal/workorder/api ./internal/workorder/service
DEVELOPER_DIR=/Library/Developer/CommandLineTools python3 scripts/integration/schema_smoke.py --mysql-notification
```

三仓 integration 工作流增加同一真实 MySQL 回归。前端配套由独立负责人维护，发送 `page_size` / 可选 `channel`；本阶段未发送邮件、短信或 webhook，也未调用生产或云 SDK。
