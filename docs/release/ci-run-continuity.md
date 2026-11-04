# 持续提交的CI运行完整性

2026-11-04核对三个仓库远端主分支及Actions：当前Go8ca03bc2的Simple CI、gitleaks和三仓集成成功；Web70b7699的Verify/gitleaks成功，AIOpsba1876a的Verify/gitleaks成功。历史失败记录不代表当前源码仍失败，也没有删除或隐藏旧记录。

后端Simple CI及Three-repository integration仍使用分支级并发组和无条件cancel-in-progress。连续阶段push会取消前一阶段已运行的检查，最近51ef/a3d/883等集成记录可见cancelled。Web、AIOps及三仓gitleaks已使用每push独立策略，只有这两个后端工作流不一致。

现在push与手动运行按run_id分组，每个阶段独立完成；PR同ref仍替换过时head。按[GitHub并发说明](https://docs.github.com/en/actions/how-tos/write-workflows/choose-when-workflows-run/control-workflow-concurrency)，仅关闭cancel-in-progress仍会替换同组旧pending，因此同时区分运行组。没有修改测试、触发条件、权限、依赖、隔离资源或发布任务，也没有触发Release、tag或上线。

两个工作流通过actionlint；解析前后YAML并移除concurrency后，所有剩余内容一致。修复后的远端运行结果需在本提交触发后独立核对，不能拿上述旧成功作为新配置验收。
