# SSH密码加密后的存储边界

本地主机创建/修改及云资源修改现在在DAO写入前检查实际密文长度，最多500个ASCII字符，与现有 `varchar(500)` 一致；超限返回固定“SSH密码超出加密存储限制”，不回显明文、密文或key，也不调用云资源字段变更审计DAO。该缺口在typed crypto重构之前已存在，不属于9b50db02引入的回归。

AES-256-GCM的12字节nonce前缀、16字节tag与标准Base64完全保留，不改Model、schema、原密文或密钥。当前格式下347个UTF8原始字节恰好产生500字符密文，348字节产生504字符；判断依据是实际加密结果，不能把Unicode字符数当作字节预算。

空密码仍沿原路径表示未提供/保留；原有纯key认证路径不额外要求AES key。不新增mask识别规则，已有短字符串写入行为保持；云账户AK/SK已有的密文预算检查不在本次修改范围。

新增Fake DAO测试覆盖三条写路径的347/348 ASCII与UTF8多字节边界、短mask字符串、空值及key-only，验证接受值能由独立旧底层解密器读回、超限不调用写DAO且原值不变。测试通过完成信号同步其Fake变更审计DAO调用。没有真实SSH、云服务或存量数据库访问。

```sh
go test -race ./internal/tree/... -count=1 -timeout=180s
go test -tags='integration smoke' ./internal/tree/... -run '^$'
```

第二条仅编译相关集成/冒烟测试，不执行其外部协议场景。
