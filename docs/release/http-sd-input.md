# 公共 HTTP-SD 输入边界

`/api/not_auth/getBindIps` 与已发布别名 `/api/not_auth/getTreeNodeBindIps` 继续提供 Prometheus HTTP 服务发现，但不走用户登录。请求必须携带 `Authorization: Bearer <PROMETHEUS_HTTP_SD_TOKEN>`，比较使用 SHA-256 摘要和常量时间比较。令牌来自环境变量或配置 `prometheus.httpSdToken`，不写进代码和日志。只有 `ENV` 明确为 `development`/`dev` 且令牌未配置时，才只接受 TCP 对端是本机回环地址的直连，不信任 `X-Forwarded-For` 或 `X-Real-IP`，并在启动时打印告警。生产环境（`production`/`prod`/`prd`）以及未设置、空白、`test` 等无法识别的环境，未配置令牌会拒绝启动。判定与 `pkg/envmode.IsProduction` 一致。非空但不合法的令牌在所有环境都会拒绝启动。升级后必须重新生成一次 Prometheus 配置，否则已下发的 http_sd 抓取会返回 401。成功响应仍是原始 target-group JSON 数组，不包裹业务 API envelope；响应仅包含目标地址和现有发现标签，不返回树节点关联用户、密码或资源密钥。

输入现在保序去重：`tree_node_ids=7,7,8` 只分别查询节点 7、8 一次，目标地址也保持原有去重行为，标签中的节点 ID 统一为 `7,8`。空白和空片段仍可忽略；非整数、非正整数或全部为空的请求返回 HTTP 400，避免旧行为悄悄执行部分查询。

限制为每次最多 100 个不同节点、1000 个原始非空 ID 条目、解码后的 ID 参数 4096 字节、完整 URL 查询串 8192 字节，端口保持 1～65535。服务层也执行节点校验与限量，直接调用无法绕过边界；参数错误在任何树节点读取前拒绝。需要发现超过 100 个节点的作业应拆分成多个抓取作业或 HTTP-SD 配置。该限制约束单次请求扇出，不替代部署入口的请求限流或网络访问控制。

IPv6 地址使用标准 `net.JoinHostPort`，例如 `2001:db8::1` 与端口 9100 生成 `[2001:db8::1]:9100`。请求取消会停止后续节点读取。

已执行：

```sh
GOTOOLCHAIN=auto go test -race ./internal/not_auth/...
GOTOOLCHAIN=auto go vet ./internal/not_auth/...
```

真实 Gin/httptest 回归验证双路由、raw target-group 协议、ID 去重及所有请求边界；服务回归验证查询次数、100 节点边界、无效输入零 DAO 调用、稳定顺序、IPv6 及请求取消。测试不连接外部服务器或已有数据库。
