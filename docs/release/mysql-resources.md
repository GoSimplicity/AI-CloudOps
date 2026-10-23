# MySQL 连接资源与启动期限

后端使用现有 MySQL 驱动和 GORM，沿用现有建表流程。资源参数支持环境变量或配置文件 `mysql` 节点，环境变量优先

| 环境变量 | 配置键 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `MYSQL_MAX_OPEN_CONNS` | `max_open_conns` | `20` | 单个后端实例的最大打开连接数 |
| `MYSQL_MAX_IDLE_CONNS` | `max_idle_conns` | `5` | 最大保留空闲连接数，不得超过最大打开数 |
| `MYSQL_CONN_MAX_LIFETIME` | `conn_max_lifetime` | `30m` | 连接最大使用寿命 |
| `MYSQL_CONN_MAX_IDLE_TIME` | `conn_max_idle_time` | `5m` | 空闲连接最大保留时间 |
| `MYSQL_CONNECT_TIMEOUT` | `connect_timeout` | `5s` | DSN 未指定 `timeout` 时的拨号期限 |
| `MYSQL_STARTUP_TIMEOUT` | `startup_timeout` | `10s` | 启动 Ping、版本检测与现有建表流程共享期限 |

上述资源参数为零时使用默认值，不表示无限连接或无限等待；负数被拒绝。打开连接数限制为 1–10000，空闲数需在 1 至最大打开数之间。降低最大连接数时应同时调整空闲数，例如 `MYSQL_MAX_OPEN_CONNS=4`、`MYSQL_MAX_IDLE_CONNS=2`

每个副本都持有自己的连接池。按“实例数 × 最大打开数”计算总预算，并为管理、备份及其他应用保留 MySQL 连接余量。观察连接等待和服务器负载后再逐步调整。生命周期应短于数据库或中间代理的连接回收周期

`MYSQL_ADDR` 中已有的 `timeout`、`readTimeout`、`writeTimeout` 保持原值，包括显式 `timeout=0s`；启动操作仍受共享启动期限约束。运行期未显式配置读写超时时保持驱动原有行为，应由业务请求上下文控制期限。对耗时建表或高延迟数据库，按实际初始化耗时调大 `MYSQL_STARTUP_TIMEOUT`，例如 `60s`，不要通过关闭池上限解决启动失败

启动先创建自有 `sql.DB` 并配置池，再执行有界 Ping。GORM 原有的 MySQL 版本探测仍会运行，并受同一启动上下文约束；完成后恢复实际 `sql.DB` 供业务使用。Ping、ORM 或表初始化失败会关闭自有连接池。启动错误与日志不打印 DSN 或原始驱动错误；请结合数据库侧日志排查认证和网络问题

已执行验证：

```sh
GOTOOLCHAIN=auto go test -race ./internal/config ./pkg/di
GOTOOLCHAIN=auto go vet ./internal/config ./pkg/di
```

测试使用隔离的内存 `database/sql` 驱动，无真实数据库凭据或外部网络连接。覆盖配置边界、环境变量加载、显式 DSN 超时保留、密码内问号不被误判为查询参数、连接池容量与空闲回收、启动失败关闭、有界 Ping 及版本查询期限。生产 MySQL 建表仍需独立部署验收
