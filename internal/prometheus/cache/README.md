# 监控缓存配置来源

## AlertManager

`NewAlertManagerConfigCache` 接收 `ProvideApp` 已有的 typed `*config.Config`，配置为空时返回错误。调用方负责先完成 `Config.Validate`；构造器复制以下三个值，实例运行期间保持这一配置快照：

- `Prometheus.AlertWebhookAddr`：保留完整 HTTP(S) URL、host:port 和发送组参数的既有拼接行为。
- `Prometheus.AlertWebhookFileDir`：决定生成的 receiver `URLFile` 目录。
- `Server.Port`：webhook 地址为空时，继续回退到该实例端口。

修改全局 Viper 或原 Config 对象不会改变已构造实例。需要切换配置时应重新构造组件。

## Prometheus HTTP 服务发现

`NewPrometheusConfigCache` 同样接收已有的 typed `*config.Config`，配置为空时返回错误，并复制 `Prometheus.HttpSdAPI`。该字段继续由配置加载器的 `PROMETHEUS_HTTP_SD_API` 环境变量、原配置键及默认值提供，不再从全局 Viper 另行读取。

生成的 HTTP SD 配置保留该地址的 scheme、host、path 和已有 query，同时以采集任务的 `port`、`tree_node_ids` 覆盖同名参数。构造后的全局配置或原 Config 对象变动不会改变实例使用的发现地址。空地址或非法 URL 仍按原行为跳过 HTTP SD 任务，不替换成其他地址，也不会因此发起网络请求。采集目标认证、TLS 及其他服务发现方式维持原有处理流程。
