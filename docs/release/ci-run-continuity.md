# 持续提交的CI运行完整性

2026-11-04核对三个仓库远端主分支及Actions：当前Go8ca03bc2的Simple CI、gitleaks和三仓集成成功；Web70b7699的Verify/gitleaks成功，AIOpsba1876a的Verify/gitleaks成功。历史失败记录不代表当前源码仍失败，也没有删除或隐藏旧记录。

后端Simple CI及Three-repository integration仍使用分支级并发组和无条件cancel-in-progress。连续阶段push会取消前一阶段已运行的检查，最近51ef/a3d/883等集成记录可见cancelled。Web、AIOps及三仓gitleaks已使用每push独立策略，只有这两个后端工作流不一致。

现在push与手动运行按run_id分组，每个阶段独立完成；PR同ref仍替换过时head。按[GitHub并发说明](https://docs.github.com/en/actions/how-tos/write-workflows/choose-when-workflows-run/control-workflow-concurrency)，仅关闭cancel-in-progress仍会替换同组旧pending，因此同时区分运行组。没有修改测试、触发条件、权限、依赖、隔离资源或发布任务，也没有触发Release、tag或上线。

两个工作流通过actionlint；解析前后YAML并移除concurrency后，所有剩余内容一致。修复后的远端运行结果需在本提交触发后独立核对，不能拿上述旧成功作为新配置验收。

首次修复运行的[Simple CI 41765525322](https://github.com/GoSimplicity/AI-CloudOps/actions/runs/41765525322)复现真实失败：来源锁测试仍断言旧的分支级group字面值。已同步该契约为每push独立、PR条件取消，保留锁文件验证与Web/AIOps完整SHA checkout断言；没有删除失败测试。完整171项release tooling测试及actionlint通过，随后新运行继续核对。

## 修复后的远端验收

修复提交638c1bda的[三仓集成41765868496](https://github.com/GoSimplicity/AI-CloudOps/actions/runs/41765868496)及Simple CI/gitleaks成功。后续d478ac7的[三仓集成41766503939](https://github.com/GoSimplicity/AI-CloudOps/actions/runs/41766503939)、Simple CI及gitleaks也成功；两个同分支运行独立完成，后push没有取消前run。

两次集成均核对实际checkout的Web813fb305、AIOpsba1876a，5项owned浏览器业务检查零跳过、失败或重试。Web93decdd自身Verify成功，166文件3619项单测及生产构建登录启动检查通过；这项Web独立结果不代表上述集成锁已覆盖93decdd。历史失败与取消记录保留。

## v2模型与冻结v1采用夹具的边界

1145f07c的Simple CI/gitleaks成功，三仓集成[41773617342](https://github.com/GoSimplicity/AI-CloudOps/actions/runs/41773617342)在`TestLegacyCurrentMySQL`失败：夹具仍使用现代InitTables（现在包含v2锁表），却要求sole-v1采用校验接受它。现代模型与当前迁移一致性检查本身已通过。

修复仅让该旧库夹具从冻结v1 Apply创建无账本58表，并明确检查无v2锁表；原有只读数据指纹、时间戳、权限可见性、未知表/视图、结构与数据拒绝全部保留。没有放宽validator、修改v2 SQL或校验和。实际本机独立MySQL容器运行完整`TestLegacyCurrentMySQL`通过，资源由既有所有权清理完成；修复提交的远端集成仍需独立核对。

## 现代CLI的原生fixture完整版本目标

后续[34fb集成41776112209](https://github.com/GoSimplicity/AI-CloudOps/actions/runs/41776112209)已通过上面的旧库结构校验，继续暴露两个CLI原生夹具的sole-v1假设：fresh与adopt夹具在只落v1后直接调用要求完整CurrentManifest的现代时区初始化。

已让fresh调用完整当前目标、计数由版本和表清单推导；adopt仍从冻结v1源开始并保留原审计证明，先验证pending及clock拒绝无写，再显式pending迁移后验证时区。完整`go test -race -tags=integration ./cmd/schema -count=1 -timeout=180s`已在自有Docker MySQL上通过，原拒绝、dirty、无账本、只读账号和时区断言均保留。没有改生产校验、v1或v2输入。新提交远端结果继续独立跟踪。
