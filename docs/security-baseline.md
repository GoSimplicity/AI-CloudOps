# 1.0 安全基线验证

2026-10-22 对本仓库执行 Go 官方 `govulncheck v1.9.0` 源码调用图分析。
首轮发现 29 个可达漏洞；升级并适配后，可达漏洞为 0。
扫描仍报告导入包中的 3 个、依赖模块中的 1 个不可达漏洞，不等于整个依赖图没有漏洞，
也不替代运行环境、云权限和渗透测试。

修复基线：Go 1.26.10、`golang.org/x/net` 0.60.0、`x/crypto` 0.57.0、
Prometheus 0.311.3、Kubernetes SDK 0.35.3、SPDY stream 0.5.1、
OpenTelemetry SDK/OTLP exporters 1.45.0、gRPC 1.83.1。
PromQL 已按新版本官方源码迁移到实例解析器，保留常规函数默认行为；
回归覆盖聚合、rate、histogram_quantile、无效表达式。

可复现检查：

```sh
GOTOOLCHAIN=go1.26.10 go test ./...
GOTOOLCHAIN=go1.26.10 go test -tags=smoke ./internal/app
GOTOOLCHAIN=go1.26.10 go vet ./...
GOTOOLCHAIN=go1.26.10 go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
```

已修复的应用层边界包括：注册与资料编辑提权、服务账号/终端绕过RBAC、退出会话失效、
JWT算法/签发者/有效期校验、精确CORS与WebSocket来源、生产SSH known_hosts、
请求日志/审计凭据泄漏及大文件截断、服务端路径与身份字段被JSON/query覆盖、
AI网关内部凭据隔离、生产默认密钥/Mock配置校验。

官方依据：[Go漏洞管理](https://go.dev/doc/security/vuln/)、
[Prometheus安全通告](https://pkg.go.dev/vuln/GO-2026-5710)、
[OpenTelemetry安全通告](https://pkg.go.dev/vuln/GO-2026-6505)。

历史 `mock/k8s_mock.go` 曾包含嵌入式客户端私钥和集群地址；当前文件已替换为
无凭据、不可路由的 `example.invalid` 配置。删除当前内容不能撤销 Git 历史中的凭据，
相关集群管理员必须轮换曾公开的客户端证书/密钥并审查访问记录；本次未改写历史或操作真实集群。

运行环境由 `ENV` 统一判定（`pkg/envmode`，SEC-27）：取值去除首尾空白并忽略大小写，`production`/`prod`/`prd` 为生产，`development`/`dev`、`test`/`testing` 为非生产；未设置或无法识别时 `config.Load` 拒绝启动，其他调用方（如 `pkg/ssh` 主机密钥策略、mock 兜底）一律按生产处理。

历史 `mock/user_mock.go` 曾内置 mock 管理员口令（SEC-27）。当前 mock 在生产环境默认关闭，口令只从 `MOCK_ADMIN_PASSWORD` 读取，未设置时不创建管理员。曾经开启过 mock 的环境中，`admin` 可能仍在使用旧的内置口令；删除代码不会修改已有账号，需要由负责人确认并另行修改口令。

启动时会检测用户名为 `admin` 的所有账号是否仍在使用该旧内置口令（源码只保存遮蔽后的字节，运行时在内存中还原并用 bcrypt 比对，用完即清零）：生产环境命中或检查本身读库失败时拒绝启动，非生产环境打印 `[SECURITY]` 告警后继续；不存在 `admin` 时跳过。处置方式：升级前先在当前运行的版本中修改 `admin` 口令（至少12字符）；若已无法启动，由负责人在维护窗口用 bcrypt 生成新口令的哈希，更新用户表中该账号的 `password` 字段后再启动。提示与日志均不包含口令。
