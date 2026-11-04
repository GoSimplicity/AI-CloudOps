# MySQL 与 Redis 跨进程恢复验收

单独SQL租约与SQLite/Redis协议测试不能替代组合交付。本阶段同时供给新owned mysql:8.4.11和Redis7.4.11实例，使用动态127端口、私有schema/user/文件与精确label清理，不接入演示13306/16379。

```sh
CLOUDOPS_TEST_PROJECTION_COMBINED=true go test -race -json -tags=integration \
  ./internal/prometheus/cache -run '^TestProjectionCombinedOwnedRecovery$' \
  -count=1 -timeout=240s
```

MySQL供给逻辑提取到integration-tag testinfra，不依赖应用model/DAO/cache，不在生产初始化路径中；原SQL测试保留五模型初始化与原12个断言。Redis原setup抽为同包owned helper，旧86场景保持。资源仍由父进程持有，子进程只获runtime私有DSN，不获得admin凭据。

## 真实子进程故障点

第一子进程使用实际MySQL pool/rule DAO、journal及Redis Store生成记录规则，业务配置和operation提交后，Prepare已执行；在第一条实际artifact命令调用前以exit86硬退出，没有走callback或Go测试cleanup。父进程确认原operation为committed、无假ACK且原SQL锁已释放。

第二个OS进程重新建立连接、publisher和独立worker，只运行恢复入口，不启动普通manager/Cron。它恢复同一原始reservation，完成实际Redis正文、成员、ACK与MySQL projected。父进程逐字核对两个raw与接受的MySQL行，并直接验证canonical ACK、legacy SHA及完整成员集合；最终实际publisher/recovery/parent PID分别13566/13570/13537，证明不是同进程对象重建。

子进程使用同一个已编译测试二进制，不嵌套go test编译；进入数据库前核对私有文件、Unix Docker endpoint、完整容器ID/随机label、实际端口映射、schema/user/DSN归属。没有父私有身份时入口不做网络或数据库操作。

## 真断连后零 Redis 命令

第二场景准备实际MySQL租约及Redis fences，捕获原事务连接ID，admin确认目标只属于该owned schema/user后KILL。另一runtime连接实际修改配置并提交。旧handle再调用ApplyArtifacts必须失败；Redis INFO证明EVAL次数未增加，完整相关键DUMP不变，不用健康全局DB或新context续权。

## 收据与边界

另有AM full pending→subset及反向组合，真实MySQL池/发送组改名，worker使用fresh源重签过期pair，验证不拼旧main文件名、subset不改full状态、无generation ping-pong，旧实际TCP请求STALE。它们复用SQLite原生的完整同一断言，避免复制两套准则。

四个组合场景实际race通过。提取后原MySQL12及Redis86全部通过，0fail/skip；全Go/vet/所有标签编译、tagged vet及actionlint通过。首轮扩展AM因前置故障用例故意遗留的坏pending落在单项worker第一页而失败，最终将该破坏性用例排在独立AM验证之后；没有修改生产行为或把坏任务标成功。坏池公平性另有原生专项。

新组合入口接入三仓CI，生产/SQL schema/Store/Lua不改，最新提交仍需远端验证。

本阶段验证的是family4两个测试进程间的journal/worker恢复及AM同进程真实MySQL/Redis组合，不冒称所有family均跨进程，或完整App所有bootstrap/通知/云行为经过OS崩溃演练，也不是生产故障切换、主从切换或性能SLA。子进程诊断只写父私有目录，密钥/DSN/正文不输出。演示服务保持CI验证的67df64e9，未上线。
