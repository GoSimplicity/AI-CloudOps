# AlertManager 配置来源

`NewAlertManagerConfigCache` 接收 `ProvideApp` 已有的 typed `*config.Config`，配置为空时返回错误。调用方负责先完成 `Config.Validate`；构造器复制以下三个值，实例运行期间保持这一配置快照：

- `Prometheus.AlertWebhookAddr`：保留完整 HTTP(S) URL、host:port 和发送组参数的既有拼接行为。
- `Prometheus.AlertWebhookFileDir`：决定生成的 receiver `URLFile` 目录。
- `Server.Port`：webhook 地址为空时，继续回退到该实例端口。

修改全局 Viper 或原 Config 对象不会改变已构造实例。需要切换配置时应重新构造组件。本次仅收敛 AlertManagerConfigCache 的配置读取；其他缓存组件的配置来源保持原有合同。
