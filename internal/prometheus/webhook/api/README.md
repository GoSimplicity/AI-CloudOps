# 静默 API 的配置来源

Webhook 启动入口把 `LoadWebhook` 返回的 typed Config 传入现有 `InitWebServer`，再由 Wire 传给 `NewWebHookHandler`。处理器复制 `AlertManagerAPI` 和 `ImFeishu.RequestTimeoutSeconds`，后续修改全局 Viper 或原 Config 不会改变这两个值。

超时继续沿用既有字段与秒单位，要求 1～600 秒，与配置加载器的校验范围一致。nil 配置或非法超时返回错误，不会退化成无限等待。URL 拼接、静默请求体、路由及上游成功后的 DAO 更新流程保持原行为。

本模块仅迁移静默/取消静默的 endpoint 与 timeout。配置加载器仍向全局 Viper 发布其他 webhook 字段，供尚未迁移的组件使用；机器人 token 和业务缓存继续按原流程刷新。
