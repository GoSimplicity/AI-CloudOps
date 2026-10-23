# W1：JWT 配置收紧、令牌有效期与 webhook 默认端口

本说明覆盖 `sched/w1-mantou` 分支上的 JWT 密钥校验、访问令牌有效期、签发者（iss）要求、
命令行定时任务环境变量规则和告警 webhook 默认端口调整。

## 1. 访问令牌有效期：约 60 小时 → 30 分钟

- `JWT_EXPIRATION` 单位是**分钟**（`pkg/di/jwt.go` 映射为 `ExpirationMinutes`，`pkg/jwt` 按 `time.Minute` 计算）。
- 旧默认值 `3600` 实际是 3600 分钟 = **60 小时**，不是 1 小时。
- 现在代码默认值（`internal/config/loader.go`、`JWTConfig` 结构体标签）和 `config/config.development.yaml`、
  `config/config.production.yaml` 都是 `30`，即 30 分钟。
- 刷新令牌有效期不变（7 天）。前端（`AI-CloudOps-web`）已开启 `enableRefreshToken`：收到 401 后调用
  `POST /api/user/refresh_token` 换取新访问令牌并重放原请求，刷新失败才回到登录页。普通 HTTP 请求对用户无感。
- 若部署环境显式设置了 `JWT_EXPIRATION`，以环境变量为准；请勿按秒填写。
- **上限 1440 分钟（24 小时）**：`Config.Validate` 中超过 1440 时，`ENV=production` 拒绝启动并报错；
  其他环境在启动时记一条告警日志（标准错误输出，`[配置告警]` 开头）后继续启动。环境判断沿用现有的 `ENV == "production"`。
  因此按旧习惯填 `3600` 的生产环境升级后会启动失败，需改为 `30`。

## 2. 旧刷新令牌失效，所有用户需重新登录一次

签发者（iss）校验上线后，之前签发的**不带 iss 的旧刷新令牌全部失效**。升级后每个用户的下一次续期会失败，
需要**重新登录一次**；之后签发的令牌都带 iss，正常续期。建议在发布公告中提前告知。

此外 `pkg/jwt.NewJWTHandler` 现在在 issuer 为空（或全是空白）时返回 `ErrEmptyIssuer`，后端启动失败，
不再签发没有 iss 的令牌。`jwt.issuer` 取自 `config/*.yaml`，可用 `JWT_ISSUER` 覆盖；修改该值同样会使已签发令牌全部失效。

## 3. JWT 密钥校验（所有环境，含开发环境）

`JWT_KEY1`、`JWT_KEY2` 在启动时按以下顺序逐把校验，不满足即拒绝启动：

1. 首尾不能有空白、不能包含控制字符，也不能包含 Unicode 格式字符（`unicode.Cf`，如零宽空格 U+200B、BOM U+FEFF、
   从右到左覆盖 U+202E）。**不会自动去除**，请检查 `.env`/编排的注入方式（例如多余的换行或空格）。
2. 至少 32 **字节**（按 UTF-8 字节数计，不是字符数）。
3. 示例值检查（尽力而为）：比较前统一去首尾空白并转小写，然后 (a) 只要包含仓库历史示例密钥的一个**公共前缀片段**即拒绝
   （大小写变化、前后加字符同样拒绝）；(b) 与已登记的历史示例/默认值的 sha256 完全相同即拒绝。
   这不是对所有示例值的完整匹配：不含该片段、也未登记哈希的其他示例或泄露值无法识别，仍需人工确保使用新生成的密钥。
4. **低熵检查**：不同字符少于 10 个即拒绝（例如 32 个相同字符）。
5. 两把密钥去首尾空白、忽略大小写后不得相同。

所有报错只包含变量名和原因，不会输出密钥内容。推荐用 `openssl rand -base64 48` 分别生成两把密钥
（README 中已标注为「第一次启动必做」）。用 hex 时请用 `openssl rand -hex 32`（输出 64 个十六进制字符，即 64 字节）。

> 本次没有轮换任何现有密钥。已部署环境若使用低熵或示例派生的密钥，升级后会启动失败，需要运维更换密钥；
> 更换 JWT 密钥会使已签发令牌全部失效。

## 4. 告警 webhook 默认端口：8888 → 8889

- `cmd/webhook` 的默认端口（`internal/config/loader.go` 的 `webhook.port` 默认值和 `WebhookConfig.Port` 结构体标签）
  从 `8888` 改为 `8889`，与 `config/webhook.yaml` 中已有的 `port: "8889"` 一致。
- `WEBHOOK_PORT` 环境变量仍可覆盖。
- `docs/release/prometheus-scrape-auth.md` 中的 8888 描述已同步更新。
- 注意：主后端（`main.go`，`SERVER_PORT`）和告警 webhook（`cmd/webhook`，`WEBHOOK_PORT`）是**两个独立进程**，
  默认端口都是 8889。部署在同一台机器（或共享网络命名空间）时，必须用 `WEBHOOK_PORT` 把 webhook 端口错开，否则后启动的进程会端口冲突。

**待效能对齐**：`env.example` 本次未改动（由效能同事修改），以下几处需要对齐：

| 位置 | 内容 | 说明 |
| --- | --- | --- |
| `env.example:19` | `JWT_EXPIRATION=3600` | 单位已明确为分钟，3600 即有效期 **60 小时**，且超过 1440 上限，`ENV=production` 会拒绝启动；建议改为 `30` |
| `env.example:41` | `PROMETHEUS_HTTP_SD_API` 指向 `localhost:8888` | HTTP-SD 接口由主后端提供，应为 8889 |
| `env.example:78` | `WEBHOOK_PORT=8888` | 显式覆盖了新默认值 8889 |

从 `env.example` 复制 `.env` 的环境在对齐前仍会使用 8888，`JWT_EXPIRATION` 为 3600 分钟。

## 5. 命令行定时任务环境变量

见 `README.md`「生产启动安全要求」一节。要点：

- 只对命令行任务（类型 2）校验 `environment`；HTTP、SSH 任务不再因该字段被拦截。
- key 和 value 都为空的行保存前丢弃、执行时跳过。
- 只拒绝以下情况：key 为空但 value 非空、key 含 `=` 或 NUL，以及拒绝名单中的变量名。
- 拒绝名单新增 `BASH_FUNC_*`、`MALLOC_*`（前缀），以及 `SHELLOPTS`、`BASHOPTS`、`PS4`、`PROMPT_COMMAND`、`GCONV_PATH`、`GLIBC_TUNABLES`、`LOCPATH`。
- 变量名先按白名单格式 `^[A-Za-z_][A-Za-z0-9_]*$` 校验（去首尾空白后），不匹配即拒绝，再过拒绝名单；
  可拦掉 `BASH_FUNC_echo%%`、夹带零宽字符的 `L\u200bD_PRELOAD` 等变体。
- `PYTHONPATH`、`NODE_OPTIONS`、`PERL5OPT` 有正当用途，暂不禁止，但同样能让解释器加载任意代码，需与命令同等审查。
- 这是纵深防御：类型 2 任务本身就能执行任意命令。

存量数据核查使用只读 SQL 草稿 `docs/release/w1-cron-env-audit.md`。**需要用户批准后才能执行**。

## 验证命令

```sh
gofmt -l .
go build ./... && go vet ./... && go test ./...
go test -race ./...
go test -race -tags=smoke ./internal/app
wire check ./pkg/di
```
