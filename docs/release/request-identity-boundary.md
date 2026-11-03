# 实体路径ID与虚拟父节点边界

2026-11-03 修复一个绑定前置条件漏洞：`BindRequest`已保留非零URI/claims字段，但旧路径解析允许实体0；URI0预填后不属于受信快照，请求体可填另一个合法ID并到达业务服务。真实Config更新与K8s回滚handler在修改前均已复现这一错误目标调用。

`GetParamID`及`GetCustomParamID`现在只接受正整数。独立审查基线134处普通调用和183处自定义调用，后者均为`cluster_id`；它们指向真实实体。原有非零路径的authoritative绑定规则不变，body/query仍不能覆盖URI目标。缺失、零、负数、非整数及溢出在绑定/业务执行前失败；保留合法正数和带前导零的正数旧格式。

树子节点查询的父0是真实虚拟根，改为显式`GetParentParamID`，允许非负整数；不能把该例外用于实体更新/删除。部门根、用户解绑部门等body/query零值合同未改变。树Move.NewParentID=0的required绑定矛盾属于已发现的另一个独立问题，本阶段未顺带修改DTO或权限。

另补5处工单解析失败只return的分支：评论Update/Delete/Get/Tree和时间线Detail现在返回规范HTTP400，不再空200。其它handler沿用原有失败envelope/status合同，未以本阶段统一改写全仓HTTP协议。

| 验证 | 结果 |
| --- | --- |
| 新53个表组场景 | base解析10、真实Config7、K8s回滚7、树父4、工单25全部通过；无真实网络/DB调用的新增用例 |
| 正常路径对照 | URI17/7不被body/query99/999覆盖；真实工单handler保留JWT操作者7 |
| 虚拟树根 | URI父0读取根子节点；负父ID拒绝，query不能替换父ID |
| `go test ./...`、`go vet ./...` | 所有Go写入方冻结后的全仓通过 |
| 相关5包race | base、Prom API、Tree API、K8s API、Workorder API通过 |
| integration/smoke/schema_smoke标签 | 全仓只编译通过，不当作已执行外部集成 |

使用既有Go1.26.10、Gin与仓库接口recorder，无新增依赖；新增测试只执行内存httptest handler，不调用真实K8s回滚。既有完整API包回归沿用其隔离设施，未连接生产或上线。

```sh
go test ./...
go vet ./...
go test -race ./pkg/base ./internal/prometheus/api ./internal/tree/api ./internal/k8s/api ./internal/workorder/api
go test -tags=integration,smoke,schema_smoke -run '^$' ./...
```

完整回归源码：`pkg/base/param_id_test.go`、`internal/prometheus/api/route_identity_test.go`、`internal/k8s/api/cluster_path_identity_test.go`、`internal/tree/api/tree_parent_id_test.go`、`internal/workorder/api/path_id_errors_test.go`。本阶段没有变更数据表/迁移、远程环境或发布镜像。
