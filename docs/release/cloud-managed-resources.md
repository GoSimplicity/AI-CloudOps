# 非计算云资源合同

首批引入阿里云 RDS、传统 CLB 和 Tair/Redis/Memcache 的独立只读 DTO 与指定 ID 查询，字段不混用 ECS 的 SSH 或计算实例数据。每次提供 1 至 100 个 ID，逐项校验地域、ID、数量和分页；缺失、重复、超出范围或上游错误均整体失败。没有查询权限覆盖证明时不做全量删除。最低权限分别为 `rds:DescribeDBInstances`、`slb:DescribeLoadBalancers` 和 `kvstore:DescribeInstances`。

本阶段只有 SDK 合同及本地假传输测试，没有 API、数据库对账、前端入口或真实云验收；ALB/NLB 与其他云厂商也尚未接入。旧版 SDK 单次读取的超时上限为 30 秒，取消可能不会立即生效。后续增量同步必须使用独立资源表与地域租约，并保留本地维护字段。
