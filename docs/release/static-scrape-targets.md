# Static采集地址与端口

2026-11-03 修复前端/生成器合同脱节：界面按独立`ip_address`和`port`保存，但Static生成器只使用ip_address，裸地址配置的非默认端口被忽略。现在生成的`__address__`遵循：

- 已含`host:port`（包括括号IPv6）的原地址优先且逐字保留，忽略分离port提示字段；这保留已有API、已存记录及TLS随机端口用例。
- 裸IP/主机且分离port>0时，用标准net.JoinHostPort构造目标。裸/方括号IPv6正确保留方括号。
- 分离port=0不强制改成9090；无显式端口时继续交由协议默认处理，IPv6补必要方括号。已有`host:0`也原样保留，不宣称其实际可达。
- 只有需要使用分离port时才校验0..65535；非法分离值不发布该Static target。空主机、首尾空白、空/非法IP括号不被改写成意外的本地目标。没有引入DNS/HTTP可达性探测。

持久化字段和API编码不变，不迁移旧数据、不改变HTTP/K8s SD逻辑；裸地址以前错误地走协议默认端口，修复后会按已配置port采集。地址内已明确带端口的配置不变。

前端另完成真正Form验证，并兼容已有主机/IP、host:port及IPv6目标；错误值由固定文案拒绝，不把旧IPv4-only未执行规则直接激活成编辑回归。前端不预先改写输入地址或自动触发真实采集。

## 验证

17个生成器表组场景覆盖裸IP/主机、嵌入端口优先、IPv6、0合同及非法分离端口/空主机，修改前7个场景红、修复后通过。新增本地TLS闭环从裸host+独立随机port生成配置，经过真正Prometheus YAML解析后使用生成地址、CA、Bearer和metrics path访问本轮自建TLS fixture成功；不是绕过生成地址直接请求硬编码URL。既有嵌入随机端口TLS回归保留。

缓存包race、最终Go全仓test/vet通过；新增格式用例无网络，TLS用例仅本地临时server，未接触真实Prometheus/K8s/云或生产。无新增依赖、数据表或镜像发布。

```sh
go test -race ./internal/prometheus/cache
go test ./...
go vet ./...
```

代码及回归：`internal/prometheus/cache/prom_cache.go`、`static_target_port_test.go`、`prom_auth_test.go`。
