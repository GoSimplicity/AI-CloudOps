# Prometheus 主配置发布边界

本阶段保留原单次 job DAO 快照与实际每实例 YAML 摘要哈希，包含 token、discovery 和分片结果。读取/生成失败与确实没有活动任务分开：后者发布能被真实 `config.Load` 接受的空 scrape 配置，停止旧抓取；前者保留最后可用版本。非空但全部无效、nil job、interval/timeout冲突等不会误当成空配置发布。

配置发布、实例成员、数据库保存和旧实例清理成功后才确认池哈希。任何必要读写失败保留重试机会；当前期望IP不因写失败变成stale，删除旧缓存失败不先丢成员索引。恢复后同一快照可再次发布，不会被错误的完成标记跳过。错误仅记录固定阶段和ID，不回显YAML、URL、认证或数据库原文。

这不是Redis/数据库跨系统事务：合法配置可能已部分可见而持久化尚未完成，错误返回后会重试收敛。公开GenerateScrapeConfigs签名与SD、port、Bearer/TLS及字段编码保持，未修改其它生成器。

新增21个纯内存叶场景，覆盖Redis配置/成员/清理/hash、数据库故障恢复、双实例部分失败、空活动列表、坏快照、真实消费校验及日志canary。连28个既有token/discovery/static/auth纯内存场景共49个race通过。

```sh
go test ./...
go vet ./...
go test -race ./internal/prometheus/cache
go test -tags=integration,smoke,schema_smoke -run '^$' ./...
```

全仓test/vet与最后一条只编译已通过；race执行范围以上述49个定向场景为准，不冒称所有集成网络用例跑过。全仓原有本机HTTP/TLS夹具仍由普通test覆盖；没有真实Prometheus、云、通知、主机或生产操作。

## 完成标记补修

首个可见artifact写前先可靠持久化pending:目标哈希，失败不开始发布；只有所有步骤完成后才写纯SHA。失败B已部分可见时，源回A也会重新修复，而不会命中旧A标记误跳过。可信键固定pool.ID且使用独立publication前缀，不与旧Name可控hash键重叠；旧键不删除也不再信任，首次升级重新发布。

record/Prom新增30场景和既有76纯内存场景共106个race、vet通过；覆盖源回退、更名、重建实例、旧键碰撞、pending失败及本实例串行。前置读/生成失败仍保持原完成态，部分写失败必须保持非可信pending；不声称跨系统或跨进程原子性。
