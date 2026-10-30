# 三仓集成来源锁

三仓开发集成的前端与AIOps来源由 `integration-sources.json` 固定完整40位Git提交，仓库身份也固定。后端使用当前job实际checkout的提交，流水线记录三仓实际SHA；不再让其他两仓checkout浮动默认分支。源码组合可重现不等于镜像或生产版本已经验收。

`python3 scripts/integration/locked_sources.py` 只读本地清单，不联网解析分支。未知仓库、短SHA、分支、重复键和不支持的格式直接失败。Actions专用输出只在当前job输出文件写入经过校验的SHA；checkout找不到提交时失败，不回退main。

跨仓修复提交并推送后，需要显式更新对应revision，再让后端集成验证该精确组合。不能因某仓main更新而自动换掉待验收组合；CI中的实际SHA记录是判断该次运行覆盖内容的依据。

此来源锁不是新的已验证1.0发布清单。历史 `release-candidate.json` 保持原记录；新的候选还须绑定精确三仓源码、实际构建镜像ID/摘要和本轮验收证据，不能把仅源码锁定标记成已构建、已上线或已验收。
