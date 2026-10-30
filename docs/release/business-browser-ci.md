# 隔离 CI 浏览器业务验收

三仓集成流水线在本轮新建的 Compose 栈上运行浏览器业务验收；不依赖开发者本机服务，也不支持生产目标。Web 单仓流水线仍负责编译、单测与登录页体验；完整业务需要后端、数据库和 Redis，所以由后端的 `integration.yml` 统一执行。

## 本阶段验收范围

精确选择 Web 三个文件中的五个场景。业务持久化使用真实登录与真实 API；登录过期场景仅注入一次过期错误，随后真实重登录，不伪造业务成功响应：

- `workorder instance completes a real template draft submit and approval lifecycle`：准备本轮唯一分类、表单、流程和模板，通过 UI 创建草稿、提交、审批；确认列表与详情最终状态一致，并逆序清理本轮记录。
- `cron default retry budget and responsive cancellation use real API`：读取真实任务列表，确认创建表单默认重投次数，桌面与移动端打开、取消表单；确认没有发出 Cron 写请求。
- `expired login preserves its dialog and rebuilds current route permissions`：一次性给列表与 refresh 注入 401，确认过期对话框保留；重新登录、profile、权限和列表回到真实后端。
- `cloud account local CRUD preserves regions and never auto-verifies`：通过 UI 分别创建阿里云、AWS、腾讯云账户，全部使用 fake 凭据且从创建提交起就是停用；修改名称、读回地域 ID 和状态，逆序核对归属后清理。云 verify/sync/available-regions 请求在到达后端前中止并使测试失败。
- `disabled HTTP cron persists create and execution-field edits without running`：通过 UI 创建停用 HTTP 任务，再编辑 URL、方法、请求头/体、表达式和超时，API 读回验证后通过 UI 删除。目标固定 `127.0.0.1:9`，始终停用，断言运行次数为零；任何启用或执行写请求都会在后端之前中止。

文件白名单为 `console.spec.ts`、`cloud-account-contract.spec.ts`、`cron-edit-contract.spec.ts`；不是运行整个 E2E 目录。Cron 覆盖创建、编辑持久化、取消和删除，刻意不执行任务，不能视作执行/重投的浏览器验收。工单准备数据通过 API 创建，实例创建、提交和审批通过浏览器完成。现有 Python API 工单验收继续保留。多云用例只验本地账户持久化，不接触真实云、模型、通知、Kubernetes 或 SSH。

## 启动与目标保护

入口为 `scripts/integration/business_browser.py`，没有 URL、密码或 Docker 目标参数。默认拒绝执行，只有同时满足以下条件才会启动浏览器：

1. `GITHUB_ACTIONS=true`、`CLOUDOPS_CI_OWNED_STACK=1`，且 Compose 项目名严格等于 `cloudops-ci-${GITHUB_RUN_ID}`。
2. Docker 使用固定本地 Unix socket `/var/run/docker.sock`；显式远程 daemon/context、自选浏览器 URL、预览服务、用户或配置路径均拒绝。
3. 固定私有配置文件为本仓 `deploy/release/.env`，必须是当前用户持有的普通文件、权限 `0600`、无重复键；网关固定绑定 `127.0.0.1:18080`。
4. 查询本项目容器后，核验 Compose 项目与 service label、运行与健康状态；MySQL、Redis、后端、AIOps、Web、网关必须齐全。仅网关允许发布端口，且只能发布固定回环端口。

配置由该 CI job 在新检出目录中生成。入口要求 AIOps 通知关闭，通知 webhook 与模型密钥保持空值，模型地址固定为容器内不可用的回环地址；Go webhook 的 PRIVATE_ROBOT_APP_ID/SECRET 也必须显式为空，`WEBHOOK_IM_FEISHU_TENANT_ACCESS_TOKEN_API`、`WEBHOOK_IM_FEISHU_PRIVATE_MESSAGE_API`、`WEBHOOK_IM_FEISHU_GROUP_MESSAGE_API` 必须分别为 `http://127.0.0.1:9/open-apis/auth/v3/tenant_access_token/internal`、`http://127.0.0.1:9/open-apis/im/v1/messages`、`http://127.0.0.1:9/open-apis/bot/v2/hook`。不能只关闭 AIOps 就推导另一条 webhook 链路没有后台 HTTP。本轮工单 UI 测试还会在创建业务事件前确认库内没有启用中的工单通知配置，防止外部邮件或飞书投递。测试不执行模型问答、通知投递、云资源、Kubernetes 或 SSH 操作。

该入口是对 GitHub 托管临时 runner 的本轮新建栈施加约束；不提供对任意既有栈的“认领”或改写配置功能。`if: always()` 的 Compose 清理继续仅处理本 job 项目。

## 严格结果门禁与私密输出

仅选择上述五个测试，禁用重试，使用 Chromium、单 worker、固定 hash 路由与本轮随机管理员配置。子进程总上限 720 秒，场景各自还有更短超时。Playwright JSON 报告进入权限为 `0700` 的临时目录，文件为 `0600`；不将原始 stdout、stderr、凭据或报告上传为工件。多云表单填入凭据时不截图。

入口要求五个指定测试恰好各执行一次并通过，报告统计为 5 passed、0 skipped、0 unexpected、0 flaky，且无 runner 错误。密码缺失、测试自身 `test.skip`、测试改名、grep 未选中、预期失败或重试后通过都会失败，不会得到空跑绿灯。

失败时仅输出固定目录中的五个标题、预定义的结果状态和非负整数统计；报告中的未知标题不输出，未知状态或非法计数统一为 `unknown`。即使浏览器进程非零退出，合法 JSON 中的安全状态仍用于定位哪个业务步骤失败。基础设施问题仅输出 `runner_error`、`runner_timeout`、`runner_start_failed`、`invalid_report` 或 `report_oversize` 等固定分类；不输出任何 `error.message`、stack、attachment、stderr 或任意报告字符串。

## 本地验证边界

`python3 -m unittest scripts.tests.test_business_browser -v` 是离线安全回归：覆盖默认拒绝、身份/目标拒绝、私有配置、容器标签和端口校验、跳过/失败/重试报告拒绝、固定目标命令构建，以及非零退出的安全诊断与私密错误文本隔离。全部 Docker、浏览器进程通过 mock 替代，不连接本机已有服务。

本阶段本地运行这些纯单测与精确用例 discovery（不启动浏览器/服务），另有创建状态、URL 合同与真实 SQLite HTTP 持久化回归。五个真实 Chromium 业务结果以包含匹配后端/Web 提交的新 GitHub 三仓集成 job 为准；接线完成或旧版本两例通过不能替代新五例验收。它不替代真实模型、通知、云或 Kubernetes 环境的授权验收，也不代表上线。

旧集成曾发现 Go webhook 在空凭据时仍轮询公共 Token API；该轮已取消。旧版 CI 的功能结果不能作为外发阻断证据。只有包含空/半凭据零 HTTP 修复、上述显式回环配置和新五例的匹配组合，才能重新验收本条隔离边界。
