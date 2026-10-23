# AI 网关可信用户身份

Go 的 JWT/实时账号/RBAC 校验完成后，AI 代理只从认证 claims 读取正整数用户 ID，注入 `X-CloudOps-User-ID`。缺少认证主体在代理处返回 401，不把浏览器请求降级为系统服务主体

代理丢弃客户端自报的同名身份头，覆盖内部 `X-API-Key`，删除原始 Authorization、Cookie、查询 token 及 `cloudops-auth.*` 认证子协议。固定公开子协议与业务查询参数继续转发，不把浏览器 JWT 传给下游服务

AIOps 仅在服务密钥通过后读取可信身份头，以用户主体约束会话与缓存。无身份头的持钥内部调用属于独立 `service:internal`，不能取得用户会话。AIOps/MCP 不应直接公开给浏览器或不可信网络；内部密钥具有服务间信任权限

AI 的既有 `/api/v1/assistant/stream` 为单独允许的完整 WebSocket 握手路径。新浏览器子协议由 AIOps 选择固定 `cloudops-v1`；原始认证子协议在 Go 已删除，不能选中或回显

启动权限目录包含 AIOps 的逐方法、逐路径条目，管理员可以分别授予 `POST /api/v1/assistant/query`、`GET /api/v1/assistant/session/:session_id` 和 `GET /api/v1/assistant/stream`，而不授予 AutoFix 确认修复或通知接口。参数只匹配一个非空路径段，不能覆盖子路径；会话归属仍由 AIOps 校验。目录只新增缺失记录，不给任何角色自动授权，也不覆盖已有备注

既有 `/api/v1/*path` 授权仍明确代表对应 HTTP 方法的整个 AI 模块，包括修复与通知。升级不会静默撤销既有角色权限；管理员应先为助手角色选择所需细分接口，再移除宽泛授权。单独授予助手接口不会自动赋予工作流确认权限。普通助手内部工具策略与高权限内部 MCP 通道的边界由 AIOps 独立执行

`internal/app/aiops_permissions.json` 与实际 AIOps HTTP/WebSocket 路由在三仓集成 CI 中对账，包含未进入 OpenAPI 的别名。修改 AIOps 路由时需要同步该目录；检查只导入本地应用并读取路由，不调用模型、通知或基础设施

已执行 Go race smoke：伪造用户 ID 被覆盖、缺主体被拒绝且不调用下游、SSE 业务参数/流量保留，以及真实反向代理 101/数据帧/固定公开协议。Python 的会话归属与跨用户缓存回归由 AIOps 仓库记录，联合镜像验收在最终验证记录中单独标明

真实隔离网关验收（Go 7f59b8a、AIOps/MCP b943bc5）：从实际权限目录给唯一临时普通角色仅授予 info 和 session 参数 GET，info 返回成功、未知会话返回下游 404、越界子路径和 POST query/confirm/notify 全部在网关拒绝 403；撤权后 info 立即 403。未调用模型或变更基础设施，角色、账号和管理员会话均清理。该流程已纳入 `account_access.py --rbac` 和三仓 CI

普通助手只读策略镜像另行执行真实硅基流动原生 tool_calls 验收：2 次模型请求、1 次 MCP calculate 调用，计算器实际结果及模型最终回答均为14。该验收只允许 calculate，没有部署、修复或通知调用
