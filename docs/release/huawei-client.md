# 华为云只读计算客户端

本阶段新增华为官方ECS SDK客户端及隔离契约测试。请求使用明确AK/SK、region和对应project ID，调用context会传播到真实HTTP传输；按ID最多100项分批，分页失败、缺失或重复结果不返回部分清单。

SDK的 `all_granted_eps` 最多覆盖100个已授权企业项目，接口本身不能证明全量覆盖。共享同步入口需在接线时禁止无ID增量及全量，直至完成可靠的企业项目范围验证；本阶段客户端尚未开放账户及同步入口，也未使用真实云账号。磁盘容量不使用官方标记无效的字段。

验证：独立提交快照中的 `go test -race ./internal/tree/utils -run '^TestHuawei'` 通过，官方SDK v0.1.221；govulncheck 对该包未发现可达漏洞，详细证据随本地验证记录。
