# SSH 连接与公钥安装边界

生产环境要求 `SSH_KNOWN_HOSTS` 指向运维预配并核验的 OpenSSH 主机公钥库。默认发布模板不挂载开发者 SSH 凭据，也不会信任首次遇到的主机

需要启用时，在私有部署 `.env` 中设置 `SSH_KNOWN_HOSTS_FILE` 为现有主机公钥文件的绝对路径；通过独立可信渠道核对指纹后，再叠加可选模板：

```sh
docker compose -f deploy/release/compose.yaml -f deploy/release/ssh.yaml config --quiet
docker compose -f deploy/release/compose.yaml -f deploy/release/ssh.yaml up -d --wait
```

后续编排操作保持同一组 `-f` 文件，避免重建时移除挂载。挂载只读且缺失时不自动创建目录。不要把未知服务器的 `ssh-keyscan` 输出未经核对直接当作可信身份

安装公钥要求单行有效的 `authorized_keys` 格式，最大 16 KiB。合法选项与注释按原文保留并以 shell 字面量写入，不能通过注释中的引号、美元符号或分号插入命令。多行与无效密钥在建立执行会话之前拒绝；目录/文件权限分别设为 0700/0600

连接期限同时覆盖 TCP 拨号和 SSH 握手，避免仅设置 `ssh.ClientConfig.Timeout` 造成静默服务器无限占用资源；连接成功后清除握手期限，不截断正常会话。地址使用 `net.JoinHostPort`，支持 IPv6。日志仅记录命令/输出长度，不写入原始内容

已执行：

```sh
GOTOOLCHAIN=auto go test -race -tags=smoke ./pkg/ssh ./internal/tree/ssh
```

覆盖合法/无效/多行/超限公钥、真实临时 shell 注释注入拒绝与权限、静默本机 TCP 服务的握手期限及失败连接回收。全部操作仅作用于临时目录和本机临时端口，尚不代表真实远程主机、Web 终端或远程权限配置验收完成
