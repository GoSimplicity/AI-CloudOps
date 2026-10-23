# 1.0 分阶段交付计划

本轮工作区为 `~/project`，三个仓库分别提交推送。每一批变更记录测试结果，远端 CI 成功只代表该流程覆盖的检查通过。历史验收记录不代替本轮环境实测。

## 阶段与验收

| 阶段 | 内容 | 交付证据 | 当前状态 |
| --- | --- | --- | --- |
| 基线与发布入口 | 三仓目录、私有配置、CI、构建工具链 | 工具回归、Compose 检查、远端 CI | 首批已推送 |
| 服务正确性 | 调度幂等、有限输出、TLS、异步 I/O、真实错误传播 | 专项回归、race、完整离线测试 | 持续修复 |
| 前端统一 | 复用 Vben token、共享页面骨架、表格工具栏、主题和移动布局 | 类型检查、组件测试、生产构建、浏览器 | 认证修复已推送，页面迁移进行中 |
| 集成与运行 | 空库初始化、认证、RBAC、工单全流程、AI 网关、持久化恢复 | 隔离 Compose 与真实 HTTP/浏览器结果 | 本轮镜像构建进行中 |
| 发布候选 | 版本清单、备份恢复、升级回滚、配置文档、功能清单 | 与三仓 SHA 对应的验收记录 | 待完成 |

## 测试分层

| 用例 | 类型 | 场景 | 预期 |
| --- | --- | --- | --- |
| 发布工具 | 离线 | 缺少仓库、生成/重建私有配置 | 拒绝错误目录、不覆盖已有密钥、0600 权限 |
| 调度 | 单元及 race | 重复重载、更新、删除、数据库失败 | 单条目、精确注销、失败保留原配置 |
| 任务执行 | smoke/race | 双流大输出、HTTP 超限/超时、解释器运行 | 输出受限、错误可见、脚本真实执行 |
| 监控 | 配置及本机 TLS smoke | SD 加载、未信任证书 | 上游接受配置、拒绝不可信服务端 |
| 前端认证 | 单元 | 并发 401、刷新失败、重试边界 | 共享一次刷新、失败不重放 |
| API 并发 | 离线异步测试 | 慢指标/通知请求、投递失败 | 健康接口可响应、不返回假成功 |
| 业务端到端 | 隔离集成 | 登录、RBAC、工单 CRUD、注销 | 持久化正确、权限拒绝、会话撤销 |

Go 项目遵循现有 `testing` 断言与 smoke 标签约定，无新增测试框架依赖。Go 版本以 `go.mod` 为准，`GOTOOLCHAIN=auto` 可处理本机旧工具链；Node/Python 版本以各仓锁文件与 CI 为准。真实脚本 smoke 需要 Bash、Python3、Node，发布镜像必须包含这些运行时。

## 常用检查

```sh
python3 -m unittest discover -s scripts/tests -v
GOTOOLCHAIN=auto go test ./...
GOTOOLCHAIN=auto go test -race ./internal/cron/scheduler
GOTOOLCHAIN=auto go test -race -tags=smoke ./internal/app ./internal/cron/api ./internal/cron/handler
GOTOOLCHAIN=auto go vet ./...
docker compose -f deploy/release/compose.yaml config --quiet
```

集成脚本在 `scripts/integration/`，执行前确认只连接本轮隔离资源。模型、云、SSH、外部通知分别记录真实服务验收，不将替身测试结果标为真实投递或真实变更成功。外部密钥始终留在私有配置，不进入 Git 或前端变量。
