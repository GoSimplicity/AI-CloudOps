# AI 网关可信用户身份

Go 的 JWT/实时账号/RBAC 校验完成后，AI 代理只从认证 claims 读取正整数用户 ID，注入 `X-CloudOps-User-ID`。缺少认证主体在代理处返回 401，不把浏览器请求降级为系统服务主体

代理丢弃客户端自报的同名身份头，覆盖内部 `X-API-Key`，删除原始 Authorization、Cookie、查询 token 及 `cloudops-auth.*` 认证子协议。固定公开子协议与业务查询参数继续转发，不把浏览器 JWT 传给下游服务

AIOps 仅在服务密钥通过后读取可信身份头，以用户主体约束会话与缓存。无身份头的持钥内部调用属于独立 `service:internal`，不能取得用户会话。AIOps/MCP 不应直接公开给浏览器或不可信网络；内部密钥具有服务间信任权限

AI 的既有 `/api/v1/assistant/stream` 为单独允许的完整 WebSocket 握手路径。新浏览器子协议由 AIOps 选择固定 `cloudops-v1`；原始认证子协议在 Go 已删除，不能选中或回显

已执行 Go race smoke：伪造用户 ID 被覆盖、缺主体被拒绝且不调用下游、SSE 业务参数/流量保留，以及真实反向代理 101/数据帧/固定公开协议。Python 的会话归属与跨用户缓存回归由 AIOps 仓库记录，联合镜像验收在最终验证记录中单独标明
