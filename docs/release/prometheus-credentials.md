# 监控采集凭据读取与更新契约

## API 行为

`GET /api/monitor/scrape_jobs/list` 与 `GET /api/monitor/scrape_jobs/detail/:id` 不再返回 `bearer_token` 字段或原文，始终返回布尔值 `has_bearer_token`。标志只依据数据库中的实际令牌是否非空计算。CA 证书内容、令牌文件路径等原有非秘密字段仍可读取；不会把占位字符串作为令牌返回。

创建请求的 `bearer_token` 保持普通字符串。更新请求改用可选字符串：

| 更新输入 | 行为 |
| --- | --- |
| 省略 `bearer_token` 或 JSON null | 保留当前数据库令牌 |
| `"bearer_token":""` | 清除令牌 |
| `"bearer_token":"新令牌"` | 替换令牌 |

其他配置字段保持完整表单契约，空字符串会实际清除对应字段。令牌切换到文件时必须提交 `bearer_token:""` 与目标 `bearer_token_file`；文件切换到令牌时提交新令牌和 `bearer_token_file:""`。省略令牌同时提交非空文件、而旧记录仍有令牌，会按互斥来源拒绝，数据库不产生半次更新。CA 内容与文件也保持互斥。

前端编辑页应默认“保留已配置令牌”，使用明确的替换和清除操作。读取后不能将空输入或脱敏占位符自动当作新令牌回传。部署时应配套升级具有上述编辑语义的前端；旧客户端将空字符串当“未编辑”的行为不符合新契约。

## 存储与并发

旧明文列和数据库结构不变，没有数据迁移或批量重写。JSON 序列化使用别名类型避免递归，仅隐藏读取响应，不修改 DAO 返回对象和存储值。已有凭据继续参与受控的操作配置生成与导出；没有更改配置导出路由权限或 SDK 全局 Secret 脱敏设置。

服务层先将请求与旧令牌合并并校验。DAO 在事务内重新读取记录，MySQL 使用 `SELECT ... FOR UPDATE` 锁定该行，再基于此刻令牌重做合并与互斥校验；因此省略字段的编辑不会将并发完成的新令牌轮换覆盖成旧快照。显式 map 更新配置字段，可靠写入空串，同时排除创建者等身份字段。当前仍是单工作区、API 级 RBAC，未增加组织隔离承诺。

## 日志与验证

当前通用审计只记录元数据，请求体/响应体为空，访问日志不读取请求体。真实 POST/PUT 的 Gin 中间件回归确认创建字符串、更新指针均未进入审计正文或访问日志。该检查不代表所有数据库错误日志均已安全；GORM SQL/驱动错误日志边界在独立后续模块修复。

先红后绿测试实际复现了旧详情泄漏、无存在标志、清空失败、两种来源无法切换及旧令牌互斥校验缺失，然后验证修复。已执行：

```sh
DEVELOPER_DIR=/Library/Developer/CommandLineTools GOTOOLCHAIN=auto go test -race -tags=smoke ./internal/prometheus/api ./internal/prometheus/service/scrape ./internal/prometheus/cache ./internal/prometheus/dao/scrape
DEVELOPER_DIR=/Library/Developer/CommandLineTools GOTOOLCHAIN=auto go vet ./internal/prometheus/... ./internal/model
```

真实 SQLite 回归覆盖旧数据保持、读取 JSON、保留/清空/替换、token→file、file→token、互斥拒绝与事务回滚；受控握手的并发测试在服务读取旧快照后完成真实轮换，确认更新保留最新值。MySQL 行锁由生产方言原生生成，本阶段未在真实 MySQL 上新增并发验收。已有操作配置经 YAML/HTTPS 的认证测试仍通过。
