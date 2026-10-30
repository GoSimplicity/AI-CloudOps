# 一致性备份与隔离恢复

`scripts/recovery_backup.py` 默认 dry-run；只有 `--execute` 和显式本机 Docker Unix socket 才实际操作。默认仅接受带本轮 fixture 标签的目标；未来受权操作者使用真实部署时还需显式 `--allow-deployment`。本轮只测试自建、无网络的容器，不对现有或生产服务停写、备份或恢复。

## 停写与快照

操作人员先停止整个部署的 API、后台任务、所有副本及其他写入者，再干净关闭 MySQL/Redis。备份前后均核对 Compose 项目、完整容器 ID、挂载卷、服务镜像与停止状态；数据卷有任何运行中的使用者就拒绝。MySQL 或 Redis 被强杀、非零退出或 OOM 时拒绝快照。必须阻止外部进程在这期间重新启动写入；脚本不能替代该运维约束。

源描述文件为私有 JSON，字段为 version=1、project、containers、volumes、private_config、revisions。containers/volumes 按 mysql、redis、aiops 三个角色列出互不相同的完整容器 ID 和卷名；项目还必须包含停止的 backend/web 服务。private_config 是 name/path 列表，revisions 为 backend/web/aiops 的提交 SHA。容器中的实际卷挂载和镜像 ID会独立验证；提交 SHA 是操作者提供的元数据，不能单独证明镜像由该提交构建。

输出包含 MySQL 物理数据、Redis 整个数据目录（全部数据库/AOF）、AIOps data 卷以及显式提供的私有配置。MySQL 根目录运行时 mysql.sock 不进入备份，其余符号链接、硬链接、设备文件及路径穿越均拒绝。每卷归档和解包数据最多 4 GiB、最多 500000 项；超过上限会失败，不能误报完整备份。目录为700、文件600。归档未加密，须存放在受控且加密的存储介质，HMAC密钥须独立保护。

示例仅用于由用户安排的已授权停写窗口，默认不执行：

```sh
python scripts/recovery_backup.py backup --source /private/source.json --key-file /private/backup.key --output /private/new-backup
# 真正执行还需 --execute --docker-socket /path/to/local/docker.sock --confirm-stopped
# 非fixture目标还需 --allow-deployment
```

## 恢复与副作用核对

恢复先验证私有文件权限、清单 HMAC、逐文件 SHA-256、归档路径和全部所需镜像 ID。随后只创建全新随机标签卷和新私有目录，不覆盖现有卷，不启动应用。输出 receipt 和 restored_quarantined 状态。失败保留 INCOMPLETE、已创建卷和证据，禁止将失败目录作为有效恢复点。

```sh
python scripts/recovery_backup.py restore --bundle /private/new-backup --key-file /private/backup.key --output /private/new-restore
# 真正执行还需 --execute --docker-socket /path/to/local/docker.sock --confirm-quarantine
```

操作人员必须先在隔离网络中用保存的精确版本验证数据库、Redis、知识文档、权限和私有配置，再决定业务恢复。恢复旧 Redis 快照可能回退队列或审批防重墓碑；须逐项核对快照后已产生的外部副作用，禁止直接启动通知/自动修复 worker 或宣称 exactly-once。实际生产启动、切换和回退由用户安排。

## 演练证据

`python scripts/integration/recovery_backup_fixture.py --execute --docker-socket /path/to/local/docker.sock` 只创建带唯一标签、network=none 的专用容器与卷。验证真实 MySQL 行、Redis 两个数据库中的队列/审批状态、知识文件及私有配置从全新卷恢复，并校验标签后清理自有资源。它不代表三仓全部业务端到端或生产RTO验收。

安全单测涵盖篡改、文件权限、路径穿越、链接、目录覆盖和默认 dry-run。真实演练先发现未干净关闭会被拒绝、MySQL socket 链接和标准输入/缓冲偏移问题，修复后恢复核对通过。清单记录快照时刻，恢复回执只报告数据复制时间；RPO取决于真实停写与最后有效快照，RTO还包括应用验证和副作用核对，未虚构生产数值。
