# 已知 49 表升级的物理备份与隔离恢复证据

`scripts/integration/known_upgrade_rollback.py` 专门验证 E04 的回退数据路径：确切 a8f1700675bef0fe950bce5018893b9a2366f3b4 旧结构和代表数据，先经已发布的 `recovery_backup.py` 完成真实物理备份与 HMAC 认证，再通过实际 `schema upgrade-a8f1700` CLI 升至 58 张业务表，最后恢复到全新隔离卷，用旧模型只读 checker 检查恢复后的 49 表与原数据。

这不是在升级库上执行反向 DDL，也不会启动旧整站、worker、通知轮询、云客户端或 SSH。此证据不能证明生产 rollback 已验收、整站 RTO、应用兼容性或外部副作用已恢复。

## 默认不执行

```sh
python3 scripts/integration/known_upgrade_rollback.py
```

默认只输出 dry-run，不构造 Docker runner、不读取凭据或源数据、不创建资源。实际运行必须显式授权这个新 fixture：

```sh
python3 scripts/integration/known_upgrade_rollback.py --execute --go /path/to/go1.26.10/bin/go
```

需要本机固定 `/var/run/docker.sock`、已缓存的 `mysql:8.4.11` 和 `redis:7.4.11-alpine`、源 Git 对象及离线 Go 模块缓存。构建强制 `GOTOOLCHAIN=local`、`GOPROXY=off`、`GOSUMDB=off`、`CGO_ENABLED=0`，不下载模块或工具链。Docker 环境覆盖一律拒绝，不支持远程 daemon、任意旧容器、旧卷或用户指定数据库。

## 资源边界

项目名固定为随机 `cloudops-fixture-<32 hex>`，另生成随机 owner。每个容器与卷都携带 owner、project 和 fixture 标签。所有运行容器使用 `--network none`、`--pull=never`，数据服务不发布 TCP 端口；升级 CLI 与旧 checker 都复制到自有 MySQL 容器，通过私密 Unix DSN 文件连接。

原备份工具保持不变。新 fixture 只包装它的 Docker argv，为临时 archive/restore helper 追加本轮名称与标签，并固定本地 socket。helper 镜像声明的 `/data` 用已有明确挂载或 tmpfs 覆盖，避免产生无标签匿名卷。镜像先在本地解析为 immutable image ID，物理备份/恢复沿用这些 ID。

backend/web 是已退出的无业务占位容器，AIOps 仅有数据卷和知识文件；Redis 只有合成队列和审批标记。实际应用从未启动，队列没有消费者。恢复卷保持工具生成的 quarantine 状态，只启动隔离 MySQL/Redis 以执行数据检查。

## 数据与检查

旧结构来自冻结 49 表 snapshot，源文件 provenance 与本地 Git archive 核对。只抽取旧 `go.mod`、`go.sum`、model 和旧初始化源文件用于 provenance；checker 不引用 DI 或初始化器。

checker 编译在旧模块中，使用旧 model 与 GORM/SQL 只读事务。它核对完整 49 表定义、列顺序、索引/外键和全表行指纹；读取软删除记录，检查 SQL NULL、旧通知队列，并验证合成访问凭据/密码的 AES-GCM 密文字节可解密。合成密钥与凭据只存在本轮私有文件及数据卷，不输出原值。

升级使用新 CLI 的真实已知来源方案，先确认 CLI 的 source/checksum 与冻结 snapshot 一致。备份证明绑定真实备份 manifest 的 SHA256；manifest 的 HMAC 和每个物理归档的散列仍由原工具验证。升级完成后核验 58 业务表和 schema 版本，随后从备份恢复至新卷，旧 checker 的完整数据指纹必须与升级前一致。

另外检查 Redis 多数据库中的队列、审批与禁止重放标记、知识文件字节，以及私有配置完整字节一致。没有自动重放任务或通知，也没有宣称恢复外部系统状态。

## 证据与失败保留

证据目录为新建 `0700` 私有目录，文件 `0600`。保留源快照、旧源、两个工具二进制、合成 key、备份 HMAC key、descriptor、完整物理 bundle、恢复 receipt 与 `result.json`。证据包含私密材料，不应作为公开 CI artifact 上传。

结果记录 immutable image ID、工具散列、原始/恢复后的旧模型 checker 结果，以及原工具的 `data_copy_seconds`。该时间覆盖校验、创建恢复卷和数据复制，不包含旧整站启动，不能作为整站 RTO。

任一失败保留证据和本轮资源，输出固定阶段，不回显 SQL、DSN、密钥或底层命令结果。全部验证成功后，清理前再次核验所有本轮资源的 owner/project 标签，之后只移除自己的容器与卷，私有证据目录继续保留。

## 离线安全测试

```sh
python3 -m unittest scripts.tests.test_known_upgrade_rollback -v
```

覆盖 dry-run 零副作用、Docker override 拒绝、固定本地 socket、helper 标签与匿名卷阻断、未知/外国资源拒绝、清理前核签、失败证据保留、旧 archive 路径与 provenance 约束，以及旧 checker 不调用应用或表初始化器。

## 本机实测记录（2026-10-30）

完整链路已实际通过：49 张旧表与 12 条代表行经物理备份，实际升级 CLI 确认 58 张业务表，再恢复到新卷。旧 model checker 的完整结构和数据指纹在升级前与恢复后相同，合成密文解密、软删除/NULL、Redis 队列/审批/禁止重放标记、知识文件及私有配置检查均通过。原恢复工具记录的数据复制时间为 **2.472 秒**；这不包含应用启动，不是整站 RTO。

联调中两次 fixture 构建问题均被校验阻断：MySQL seed 客户端未显式 UTF8 时破坏了 COMMENT 字节；没有 `docker run -i` 时知识文件 stdin 被丢弃。现已固定 `--default-character-set=utf8mb4`、显式 Unix DSN charset，并在种子写入后立即读回知识字节；带 stdin 的 run/exec 必须有 `-i`。失败诊断与物理签名归档保存在本机私有证据目录，未掩盖为首次成功。

最终成功组及这两组已归档的失败 fixture 都在 owner/project 再核验后移除，私有证据未删除。没有启动旧整站、worker 或外部服务。
