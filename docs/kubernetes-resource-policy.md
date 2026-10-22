# 命名空间资源配额

创建或更新集群登记时，集群的 `cpu_request` / `cpu_limit` / `memory_request` / `memory_limit` 对 `restrict_namespace` 指定的每个命名空间分别创建 Kubernetes `ResourceQuota`，限制该命名空间所有 Pod 的资源总量。它们不是单个 Pod 的默认 requests/limits，也不限制整个集群容量。

裸 CPU 数值沿用既有接口单位 millicores（`500` = `500m`）；裸内存数值为 MiB（`1024` = `1024Mi`）。接口也接受显式 Kubernetes quantity 单位。数量不能为负，CPU 精度至少 1m；同时填写 requests 和 limits 时 requests 不能超过 limits。`0` 是零额度，可能阻止新 Pod，空字符串才表示未设置。

空目标列表不会展开为所有命名空间。命名空间必须已经存在且当前凭据有配额管理权限；平台不会自动创建业务命名空间。配置仅操作 `cloudops-quota-<集群ID>` 且带有匹配 `cloudops.io/cluster-id` 标签的配额，遇到同名手工对象会拒绝覆盖。

编辑时显式提交 `restrict_namespace: []` 可撤销所有目标；从目标列表移除命名空间或清空资源值，会清理对应平台配额。省略全部资源字段及命名空间字段会保留原配置。删除集群登记前会清理平台配额；如果清理失败，登记保留以便重试。配额生效后要更换集群连接，应先清除旧配额，避免把旧策略清理到另一个集群。

平台启动读取客户端时不额外写入外部集群；已登记但此前未生效的配置需显式更新应用。

Kubernetes 与数据库没有跨系统事务，多命名空间操作可能部分成功。失败不会被标记为成功；创建时若配额失败，会保留异常状态的集群登记以便修正或清理，避免丢失已部分创建配额的归属；更新失败会恢复原登记配置，远端可能已有部分变更，需重试原操作或检查带平台标签的配额。平台不删除已有业务 Pod。Kubernetes quota controller 和 admission cache 异步收敛，增大额度后放行可能有短暂延迟。

验证包括非法数量、单位转换、同名手工对象保护、幂等更新/清理、取消、不选目标时零 API 副作用、数据库清空值持久化，以及专用 kind 集群上的真实 admission 拒绝/增额放行/撤销。

本地真实测试仅允许专用 loopback kind kubeconfig：

```bash
CLOUDOPS_QUOTA_TEST_KUBECONFIG=/tmp/cloudops-release-kubeconfig \
  GOTOOLCHAIN=go1.26.10 go test -p 2 ./internal/k8s/utils -run TestQuota -count=1
```

依据：[Kubernetes ResourceQuota](https://kubernetes.io/docs/concepts/policy/resource-quotas/)、项目使用的 client-go `ResourceQuotas` API 与 `resource.ParseQuantity` 实现。
