# Prometheus 抓取认证与 HTTP-SD 配置修复

## 行为

静态、HTTP-SD、Kubernetes 三种采集方式统一支持目标的 `bearer_token` / `bearer_token_file` 和 `tls_ca_content` / `tls_ca_file_path`。令牌与令牌文件互斥，内嵌 CA 与 CA 文件互斥；创建、更新和生成阶段按当前 Prometheus common SDK 校验，避免忽略凭据后退回匿名抓取。目标保持 TLS 证书验证。

上述认证仅用于采集指标目标。Kubernetes 服务发现继续使用 kubeconfig 或集群内身份，不把指标令牌混入 API Server 身份；HTTP-SD 也保留自己的 HTTP 客户端配置。文件路径是运行 Prometheus 的环境内路径，需要运营者以只读方式挂载，后端不会读取远端路径文件。

操作配置导出通过 YAML 节点恢复采集认证字段的真实值，供 Prometheus 使用。没有修改 SDK 的全局 `MarshalSecretValue` 开关，普通 SDK 序列化仍脱敏。导出的配置、Redis 缓存和数据库配置正文应视为含凭据的受控资产，不应写入日志、公开下载或提交版本库。读取这些内容仍受现有登录与 API 级 RBAC 控制，当前工作区并不提供组织间数据隔离。

完整生成配置参与缓存摘要，确保只修改作业认证、CA、节点等内容也会更新；不变配置继续跳过写入。HTTP-SD 作业创建与更新现在保存 `tree_node_ids`；URL 使用标准查询编码，并保留配置地址原有查询参数。

## 部署地址

本机开发默认 `http://localhost:8889/api/not_auth/getTreeNodeBindIps`，指向主后端而非告警 webhook 服务。主后端（`main.go`，`SERVER_PORT`）和告警 webhook（`cmd/webhook`，`WEBHOOK_PORT`）是两个不同的进程，默认端口都是 8889；在同一台机器上部署时必须用 `WEBHOOK_PORT` 把 webhook 端口错开，HTTP-SD 地址始终指向主后端端口。发布编排默认 `http://backend:8889/api/not_auth/getTreeNodeBindIps`，通过 `PROMETHEUS_HTTP_SD_API` 注入后端配置生成器。该 URL 必须能从真正运行 Prometheus 的网络访问；外置 Prometheus 应显式配置可达地址。原有 `getBindIps` 与 `getTreeNodeBindIps` 路由都保留。

发布目录当前挂载的 `prometheus.yml` 仍是自身指标采集配置。生成的业务采集配置需要按运维配置分发流程提供给对应 Prometheus，本修复不会将所有数据库作业自动安装到该发布实例。

已有作业若因旧版本遗漏而没有保存树节点 ID，需要重新保存节点选择。存在互斥认证来源的旧作业会在生成时记录字段校验错误并跳过，不会静默选择其中一个来源。凭据详情脱敏与编辑清空的 API 契约另行处理，本阶段未改变详情响应。

## 已执行验证

```sh
CLOUDOPS_PROMTOOL_IMAGE=prom/prometheus:v3.2.1 GOTOOLCHAIN=auto go test -race -tags=smoke ./internal/prometheus/cache ./internal/prometheus/service/scrape ./internal/prometheus/utils ./internal/config
GOTOOLCHAIN=auto go vet ./internal/prometheus/... ./internal/config
```

- 六组本地真实 HTTPS 测试覆盖三种发现模式以及内嵌/文件认证与 CA，经完整生成 YAML、Prometheus 官方解析后成功认证
- 保留已有不可信证书拒绝及 kubeconfig 解析回归
- 缓存回归验证令牌轮换后刷新、不变配置不重写，观察日志确认不含测试令牌
- 服务回归验证节点 ID 保存以及互斥来源在持久化前拒绝
- 使用无网络、只读、运行后删除的独立发布镜像容器，通过实际 `promtool check config` 校验真实生成入口的三种发现配置，未操作已有编排
- Compose 无环境展开解析确认注入地址，未输出环境值

版本依据：项目实际依赖 Prometheus common v0.67.5 与 Prometheus Go 模块 v0.311.3，发布镜像为 v3.2.1；运行时校验直接使用该发布镜像。官方版本源码分别见 [common HTTP 客户端](https://github.com/prometheus/common/blob/v0.67.5/config/http_config.go)、[Secret 序列化](https://github.com/prometheus/common/blob/v0.67.5/config/config.go) 和 [发布镜像依赖](https://github.com/prometheus/prometheus/blob/v3.2.1/go.mod)。

后续已实施凭据读取脱敏与明确的保留/清除/替换契约，见 [prometheus-credentials.md](prometheus-credentials.md)。
