# 静默 API 的配置来源

Webhook 启动入口把 `LoadWebhook` 返回的 typed Config 传入现有 `InitWebServer`，再由 Wire 传给 `NewWebHookHandler`。处理器复制 `AlertManagerAPI` 和 `ImFeishu.RequestTimeoutSeconds`，后续修改全局 Viper 或原 Config 不会改变这两个值。

超时继续沿用既有字段与秒单位，要求 1～600 秒，与配置加载器的校验范围一致。nil 配置或非法超时返回错误，不会退化成无限等待。URL 拼接、静默请求体、路由及上游成功后的 DAO 更新流程保持原行为。

本模块仅迁移静默/取消静默的 endpoint 与 timeout。配置加载器仍向全局 Viper 发布其他 webhook 字段，供尚未迁移的组件使用；机器人 token 和业务缓存继续按原流程刷新。

## 错误和未确认状态

静默 POST 与取消静默 DELETE 仍只把 HTTP 200 作为上游成功响应，保持原秒级超时与业务响应 envelope。共享 helper 的错误仅保留固定分类或数字 HTTP 状态码，不携带 URL、userinfo、请求内容、响应 body、上游 status 文本或原始 transport/read 错误。目标事件查询与保存日志同样不记录 fingerprint、完整 event 或原始数据库错误。

新返回的 SilenceID 与 DELETE 使用的旧 SilenceID 都必须是 1～100 个 ASCII 字母、数字、连字符或下划线。这个有界单路径段规则兼容既有非 UUID 测试标识；空值、路径字符、百分号编码、空白和控制字符不会被清洗后继续使用。

只有上游成功、返回有效标识且本地保存成功时，静默操作才向客户端返回业务成功。取消静默同样要求上游成功与本地保存成功。上游可能已生效但响应/本地保存失败时，返回“未确认、请人工核对”，这些错误分支不会自行再次调用 POST/DELETE helper，也不会返回业务成功。

本地数据库与 AlertManager 不是原子事务，本阶段没有新增数据库状态、账本或 exactly-once 保证。遇到未确认结果，应先核对 AlertManager 的实际静默状态和本地事件，再决定人工修复；不要因客户端收到失败就重复提交。其他 robot、content 和 HTTP 工具的独立日志问题不包含在本次保证范围。

静默状态使用专用 DAO 条件更新：在上游调用前保存旧 `status/silence_id`，上游成功后仅在旧状态仍匹配时显式写入新值，包括取消静默时的空字符串。更新 0 行时只读核对目标状态；只有已与本次目标完全一致才视为幂等成功，缺行或其他状态都返回本地未确认。因此较晚返回的 DELETE 不会清空另一操作刚保存的新 ID。通用事件更新仍保留升级等其他字段的原合同。

两个并发 POST 仍可能在上游产生多条静默，本地 CAS 不等于分布式 exactly-once。冲突时不重发请求，需人工核对上游与本地状态。
