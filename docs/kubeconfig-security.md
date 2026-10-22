# 服务端 kubeconfig 接入

平台会执行 Kubernetes API 请求，因此上传的 kubeconfig 不能同时获得运行服务所在主机的命令执行、文件读取或代理配置能力。创建、更新和加载数据库中的已有配置都使用同一解析边界。

支持 HTTPS API、系统信任根或内嵌 `certificate-authority-data`、内嵌 bearer token，以及内嵌客户端证书和私钥。保留 TLS 验证与 `tls-server-name`，支持本地 kind 的安全容器路由。

拒绝以下字段（包括未选中的用户/集群配置）：

- `exec` 和 `auth-provider` 认证插件。
- `tokenFile`、`client-certificate`、`client-key`、`certificate-authority` 服务器文件路径。
- `proxy-url`，代理由部署运行环境管理。
- HTTP、带用户名/密码的 API URL、跳过 TLS 验证。

需要把自己的文件引用转为内嵌内容时，在**凭据持有者自己的受信任机器**上执行：

```bash
umask 077
kubectl config view --raw --flatten --minify > cloudops-kubeconfig.yaml
```

输出含凭据，保持私有，不提交 Git 或粘贴到日志。此命令不会把动态 exec 认证转换成长期静态凭据；这类集群应提供权限适当的服务账号 token 或客户端证书，平台不执行上传文件中的登录程序。

安全解析的错误信息不回显配置内容。已有不安全配置不会继续加载或运行其中程序；可以用相同集群传输身份的内嵌凭据替换。配额的集群身份比较只检查 API 地址、TLS server name 与 CA 数据，允许凭据轮换或 YAML 格式变化，不读取旧凭据指向的文件或执行插件。

依据：[Kubernetes kubeconfig 风险说明](https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/)、项目使用的 client-go `clientcmd` 与认证 transport 实现。
