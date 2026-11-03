# 服务树移根与父节点存在性

2026-11-03 修复API已承诺的移到根功能：服务本来允许非负父ID，但DTO使用非指针int与required，将合法显式`new_parent_id: 0`当缺失拒绝。现在用`*int`的required/gte=0区分“未提供/null”和“明确虚拟根0”，JSON字段/数字格式不变；URI实体node ID仍须正数且不能被body/query替换。

指针单改仍不够：Gin把空表单字符串转为0且分配指针，可能把空选择误当移根。MoveNode局部补表单字段存在/非空检查，用Gin缓存保持既有urlencoded/query/multipart绑定优先级，并保留原始解析错误，避免坏编码缓存后被二次绑定当成可用值。JSON验证保持原严格整数合同，不扩XML或全局binder。

- 显式0及正父ID到达业务一次；缺失/null/负数/小数/字符串/溢出及空/坏编码表单均400且不调用服务。
- 原服务的self与祖先cycle防护、DAO更新、部门根/解绑等其它字段、schema均未改；没有将虚拟父0重新放开为实体URI0。
- 47个真实Gin/httptest Recorder场景覆盖JSON17、三form/query各9、坏编码2、业务拒绝1。两轮红测试保留了原0误拒及pointer空form误接受的证据。
- API/utils/base测试及race、model/API/utils/base vet、Tree全包编译通过；主代理最终Go全仓test/vet、integration/smoke/schema_smoke只编译通过。新增测试全内存Fake，无真实DB/TCP、真实节点移动或上线。

代码和完整HTTP测试：`internal/model/tree_node.go`、`internal/tree/api/tree_node_handler.go`、`internal/tree/api/tree_move_root_test.go`。
