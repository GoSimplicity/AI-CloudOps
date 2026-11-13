# 1.0 安全基线验证

2026-11-13 在 `a797990ed88b423d088ffe2f1327ca72ffb883d5` 上用 `govulncheck v1.9.0` 复扫：可达漏洞为 0，导入包 5 个、依赖模块 2 个不可达。这组计数取代 2026-10-22 记录里的 3 个导入包和 1 个模块；下方该日原文保留。

官方记录与模块代理核对后的最小稳定版本：

| 公告 | 修复前 | 选定版本 | 依据 |
| --- | --- | --- | --- |
| GO-2026-6866、GO-2026-6860，`x/net/html` | `golang.org/x/net` v0.60.0 | v0.62.0 | 漏洞库固定版本；v0.63.0 尚不存在 |
| GO-2026-6443，gRPC transport | `google.golang.org/grpc` v1.83.1 | v1.83.2 | 1.83 线的补丁；未跳到 1.84/1.85 |
| GO-2026-5841，`compress/s2` | `github.com/klauspost/compress` v1.18.5 | v1.20.2 | v1.18.7 只覆盖这条包级公告 |
| GO-2026-6865，`compress/fse` 与 `huff0` | 同上 | v1.20.2 | 模块级公告的固定版本；v1.20.2 同时覆盖 s2 |
| GO-2026-4503，edwards25519 | `filippo.io/edwards25519` v1.1.0 | v1.1.1 | 补丁版本；未采用同日的 v1.2.0 |
| GO-2026-5932，`x/crypto/openpgp` | `golang.org/x/crypto` v0.57.0 | v0.59.0 | 漏洞库给出的首个修复是伪版本 `v0.58.1-0.20261009181718-2f1f834330b2`。稳定版 v0.58.0 仍包含 `openpgp`；v0.59.0 已移除该包，且本模块与已选依赖都不导入它 |

`x/crypto` v0.59.0 要求 `x/net` v0.62.0、`x/sys` v0.50.0、`x/term` v0.48.0、`x/text` v0.44.0。`x/text` v0.44.0 再要求 `x/sync` v0.25.0、`x/mod` v0.42.0、`x/tools` v0.52.0。这些是所选模块的传递约束，不是额外功能升级。`go mod tidy` 同时把本模块已经导入的 `go-sql-driver/mysql` 与 `mattn/go-sqlite3` 归入直接依赖，并去掉没有引用的 `github.com/imdario/mergo`；版本没有因此改变。

升级后在同一工作区执行：

```sh
GOTOOLCHAIN=go1.26.11 go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
GOTOOLCHAIN=go1.26.11 go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 -json ./...
```

文本输出只有 `No vulnerabilities found.`，退出码 0。JSON 为 scanner v1.9.0、Go 1.26.11、source/symbol 模式，328 个消息、0 个 finding。上面 7 条公告不再出现，也没有新的可达或不可达公告。这仍只覆盖调用图和当前模块选择，不代替运行环境、云权限和渗透测试。真实 MySQL、Redis 与 Docker 验收这次没有执行。

2026-10-22 对本仓库执行 Go 官方 `govulncheck v1.9.0` 源码调用图分析。
首轮发现 29 个可达漏洞；升级并适配后，可达漏洞为 0。
扫描仍报告导入包中的 3 个、依赖模块中的 1 个不可达漏洞，不等于整个依赖图没有漏洞，
也不替代运行环境、云权限和渗透测试。

修复基线：Go 1.26.11、`golang.org/x/net` 0.60.0、`x/crypto` 0.57.0、
Prometheus 0.311.3、Kubernetes SDK 0.35.3、SPDY stream 0.5.1、
OpenTelemetry SDK/OTLP exporters 1.45.0、gRPC 1.83.1。
PromQL 已按新版本官方源码迁移到实例解析器，保留常规函数默认行为；
回归覆盖聚合、rate、histogram_quantile、无效表达式。

可复现检查：

```sh
GOTOOLCHAIN=go1.26.11 go test ./...
GOTOOLCHAIN=go1.26.11 go test -tags=smoke ./internal/app
GOTOOLCHAIN=go1.26.11 go vet ./...
GOTOOLCHAIN=go1.26.11 go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
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

### 运行环境与弱口令检测的非生产行为 / Non-production behaviour (SEC-27)

**中文**

- `ENV=test` 或 `ENV=testing`（去除首尾空白、忽略大小写）归为测试模式（`envmode.Test`），按非生产处理。仓库没有独立的测试配置文件，测试模式与开发模式一样复用 `config/config.development.yaml`（`internal/config/loader.go` `resolveConfigFile`）。
- 旧内置弱口令检测（`internal/app/legacy_admin_password.go` `checkLegacyAdminPassword`）在非生产环境（development/dev、test/testing）下查库失败时，只打印一条 `[SECURITY]` Warn 并继续启动，不会拒绝启动；生产环境查库失败则拒绝启动（`internal/app/app.go` 中 `Run` 直接返回该错误）。
- 非生产环境命中旧口令同样只告警；但该库一旦迁移到生产环境就会拒绝启动，应尽早修改 `admin` 口令。

**English**

- `ENV=test` or `ENV=testing` (trimmed, case-insensitive) selects test mode (`envmode.Test`), which is treated as non-production. There is no dedicated test config file: test mode reuses `config/config.development.yaml`, exactly like development (`resolveConfigFile` in `internal/config/loader.go`).
- In non-production environments (development/dev, test/testing), if the legacy built-in weak password check (`checkLegacyAdminPassword` in `internal/app/legacy_admin_password.go`) cannot query the database, it only logs a single `[SECURITY]` warning and startup continues. In production the same query failure refuses startup (`Run` in `internal/app/app.go` returns the error).
- A non-production match on the legacy password is also only a warning, but the same database will refuse to start once promoted to production, so change the `admin` password early.
