# 资产与 SSH 凭据的显式加密配置

本阶段只收敛 `tree.password_encryption_key` 的运行时依赖，不轮换实际密钥、不重写存量密文、不修改云或 SSH 执行权限。业务加解密不再读取全局 Viper；配置加载层仍可使用既有 typed config 装载方式。

## 调用边界

辅助函数的参数顺序明确区分数据和密钥：

```go
EncryptPassword(plaintext, encryptionKey string)
DecryptPassword(ciphertext, encryptionKey string)
EncryptProviderCredentials(provider, input, encryptionKey)
LoadCloudAccountCredentials(account, encryptionKey)
```

密码函数始终数据在前、key 在后，调用处使用 `req.Password`/密文字段与 `s.passwordEncryptionKey` 等清楚命名，不能交换两个 string 参数。Provider 函数沿用厂商/完整对象在前、显式 key 在最后，不读取宿主环境或其他身份链替代。

本地主机、云资源、云账户、托管资源四个 service，以及 Cron SSHExecutor 的构造器接收 `*config.Config`，只将 `cfg.Tree.PasswordEncryptionKey` 复制到私有 string 字段，不保留 cfg 指针来动态取 key。nil cfg 按其他 typed 构造器风格返回错误。CronHandlers 直接注入已经配置好的 SSHExecutor，不再为了内部 new 执行器引入整个 cfg 或继续保存未使用的本地主机 DAO。Wire provider 和实际生成图同步更新。

终端/SSH 连接工厂与云客户端工厂保持既有明文边界：service/执行器先在当前操作中解密，再传原来的密码、SSH 私钥或完整云凭据；它们不接收、保存或推导该 AES key。

## 旧数据格式完全保留

底层 `pkg/base/crypto.go` 未改动：AES-256-GCM、随机 nonce 前缀、nil AAD 和 `base64.StdEncoding` 保持原样。没有增加 key ID、envelope、第二层包装或新的存储列。

配置生成器输出的 32 个 HEX 字符，既有实现一直按 **32 个 ASCII 字节** 作为 key 使用；本阶段仍然如此，绝不 hex decode 成 16 字节，不 trim、不改大小写、不重新派生 key。其他既有合法 32 字节 key 也没有被强制改成 HEX 格式。错误 key 无法解开旧密文时明确失败，不回显 key、明文、密文或底层私密错误。

空字符串输入继续是 no-op：返回空值且不强求加密 key；非空输入缺 key 或长度不为 32 字节继续失败。构造器只拒 nil cfg，不为了元数据读取或纯 SSH key 认证提前要求密码 cipher key；生产启动的既有配置校验仍单独生效。

已构造实例不受后续全局 Viper 或原 cfg 赋值影响。这是固定依赖快照，不是热轮换能力；重启使用不同 key 仍会导致既有数据无法读取。部署升级与备份恢复必须保留原 key，真实轮换须另有明确数据迁移流程，不能靠改配置完成。

## 验证证据与限制

- 在修改旧 helper 之前，用隔离 Go 进程、合成 key/明文调用旧 Viper 实现生成固定 golden；不读取实际配置或数据，也不在并发测试修改全局 rand.Reader。
- 新 helper 解开旧 golden；新写密文由独立的旧底层解密函数按正确 key 读回，防止 helper 或 service 把两个 string 参数交换。另核对随机 nonce、标准 Base64 长度、opaque HEX 大小写和缺失/错误 key 的安全失败。
- 四个 service 的 Fake/SQLite 回归保留原有凭据、轮换、事务和边界断言。新增 typed 实例在 cfg/global 改动后写入/读回、云/托管 fake factory 收到正确明文的验证。
- SSH fake connection 验证使用构造时的 key、DAO 密文不被覆盖、错误 key 在连接工厂之前失败，以及空 cipher key 不阻止原有纯 SSH key 认证。CronHandlers 验证保留注入的执行器实例。
- 原有全局 key fixtures 改成显式测试输入；仍少量设置 Viper 的新测试仅用于证明生产逻辑不依赖它，不把它作为工作密钥来源。

```sh
go test -race ./internal/tree/... ./internal/cron/... ./pkg/di -count=1 -timeout=180s
go test -tags='integration smoke' ./internal/tree/... ./internal/cron/... -run '^$'
```

第二条仅编译已有集成/冒烟调用者，不执行真实 SSH、MySQL 或云操作。此次使用 fake factory/连接及本地 SQLite，没有连接既有服务、真实云或 SSH，也没有对实际密文和 key 做任何迁移。
