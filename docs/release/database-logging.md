# 数据库日志凭据边界

主后端和独立 webhook 的 GORM 连接统一使用 `pkg/logger.NewGORMLogger`。日志只输出固定事件、耗时、行数及错误类型，不输出 SQL、SQL 参数、驱动错误原文、第三方 Info/Warn/Error 的原文或变参。默认沿用主后端 Warn 等级和 200ms 慢查询阈值；`Debug` / `LogMode(Info)` 也只增加相同的安全元数据，不恢复 SQL 正文。webhook 保留 Info 等级的事件可见性。

仅开启 `ParameterizedQueries` 不足以形成此边界：原始 SQL 可能本来就包含字面量，而 MySQL `Duplicate entry` 等驱动错误也可能包含字段值。本实现不尝试通过正则猜测敏感内容，直接不输出这些不受信任的文本。常见取消、超时和未找到使用固定类型标志，其他错误仅记录 Go 类型名称。

数据库连接、ORM 初始化、表初始化及管理员初始化的错误显示保留静态阶段和安全错误类型。安全错误封装仍提供 `Unwrap`，程序可以使用 `errors.Is` / `errors.As` 判断真实原因；普通 `Error()`、stderr 和 webhook 初始化 panic 不展示原始驱动文本。调用方不应再将解包后的原始错误当作可公开日志记录。

这项修改约束 GORM logger 与上述启动边界，不等于所有业务 DAO 自行拼装的日志均已完成审查，也不改变受控操作配置正文的凭据用途。真实数据排障应通过授权查询定位，不能依赖日志保存凭据或完整配置。

已执行先红后绿：

- 使用真实 GORM/SQLite 触发器约束错误、参数化插入、带字面量的查询和极小阈值慢查询，证明默认 logger 即使参数化仍泄漏 canary，修复后固定诊断仍存在且无原文
- Info/Warn/Error/Trace 均使用 canary 覆盖；日志等级克隆、并发使用、静默模式及错误类型回归
- 管理员初始化真实 SQLite 失败先复现 stderr 边界泄漏，再确认阶段可定位、原生错误链保留、取消仍能识别
- webhook 使用真实 MySQL 驱动的无效时区配置，在拨号前产生带 canary 的错误，确认日志和 panic 字符串均不泄漏，无需启动服务或连接数据库

```sh
DEVELOPER_DIR=/Library/Developer/CommandLineTools GOTOOLCHAIN=auto go test -race -tags=smoke ./pkg/logger ./pkg/di ./internal/app
DEVELOPER_DIR=/Library/Developer/CommandLineTools GOTOOLCHAIN=auto go test -race ./internal/prometheus/webhook/di
DEVELOPER_DIR=/Library/Developer/CommandLineTools GOTOOLCHAIN=auto go vet ./pkg/logger ./pkg/di ./internal/app ./internal/prometheus/webhook/di
DEVELOPER_DIR=/Library/Developer/CommandLineTools GOTOOLCHAIN=auto go build ./...
```

webhook Wire 已通过 `go run github.com/google/wire/cmd/wire@v0.7.0 ./internal/prometheus/webhook/di` 实际重新生成，未修改应用依赖版本。
