# WebSocket 凭据传输

新版浏览器连接使用固定子协议与认证子协议：

```js
new WebSocket(cleanUrl, ['cloudops-v1', `cloudops-auth.${token}`]);
```

连接 URL 不包含 JWT。服务端只协商并回显固定的 `cloudops-v1`，不把认证值回显为选中的子协议。认证顺序为 Authorization 请求头、唯一认证子协议、历史查询参数；存在重复或空认证子协议时不降级使用查询令牌

兼容历史 `?token=` 的范围仅限真实 WebSocket Upgrade 请求。普通 HTTP API，包括名称含 `execute` 的任务执行接口，继续使用 Bearer 请求头，不能因路径子串误判而要求查询令牌

网关访问日志只记录方法、路径、协议、状态、字节数与耗时，不记录查询字符串、Authorization 或子协议请求头。历史查询令牌格式仍可能进入其他代理的错误诊断，因此浏览器必须完成新协议迁移；不要在监控或调试输出中记录认证头

已执行：凭据解析和登录传输回归，以及 middleware/websocket/DI 的 race 检查通过；Nginx 实际配置检查与重新加载通过。最终真实 Pod exec 升级与断开结果继续记入 `verification.md`
