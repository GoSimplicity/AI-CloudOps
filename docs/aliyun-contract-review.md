# 阿里云 ECS 契约核对

本次核对基于 `alibaba-cloud-sdk-go v1.63.107` 的实际生成代码、官方 ECS
`2014-05-26` API 文档和离线替身测试。**未使用云账号、未连接真实云、未执行资源变更。**
这证明请求/响应处理与已检查的契约一致，不证明真实 RAM 策略、账号余额、地域网络或资源读取已验收。

## 当前范围

阿里云支持 AccessKey/SecretKey 配置、指定区域的 ECS 读取权限检查、区域查询和 ECS 清单同步。
同步只读阿里云 API，新增/更新/删除的是本地资产记录。其他云厂商（腾讯云、AWS、华为云、Azure、GCP）
没有真实凭据验证、区域 API 或资源同步实现；带凭据请求这些区域 API 会明确报未实现。
无凭据区域列表仅为静态选择参考，不代表账号授权、实时库存或地区完整性。
本版未引入 STS 临时凭据/令牌续期，内存沿用本地模型的整数 GB 单位；货币字段是本地默认元数据，不能用作真实账单依据。

## 核对与修复

| 项目 | 官方/SDK契约 | 当前行为 |
| --- | --- | --- |
| 凭据与权限 | `DescribeRegions` 不需要 RAM 操作授权，`DescribeInstances` 需要 `ecs:DescribeInstances` | 用 HTTPS、指定 Region、最多10条实例的只读请求检查实际同步所需权限；失败不会当作验证成功 |
| 实例分页 | 推荐 `MaxResults` + `NextToken`；最多100条；token分页时不能用TotalCount判终止 | 每页100，仅空token结束；短页/TotalCount=0仍继续；重复token或后页失败返回错误，不输出不完整快照 |
| 实例ID筛选 | `InstanceIds` 是最多100个ID的JSON数组 | JSON序列化，超过100个自动分批，各批继续token分页；拒绝空白ID |
| 区域可用性 | `Status` 为 `available` 或 `soldOut` | 如实映射，售罄不再标记可用；指定instance资源类型和中文语言 |
| 实例字段 | 弹性公网地址在 `EipAddress.IpAddress`，到期时间为ISO时间 | 常规公网地址为空时读取EIP，支持RFC3339及SDK既有分钟精度时间 |
| 多区域同步 | 每个区域是独立查询 | 收集并返回失败区域，避免所有查询失败仍记录success |
| 全量同步删除 | 只有完整区域快照才能判断本地已失效资源 | 明确拒绝“全量 + 指定实例ID”，避免删除没有查询的其他资产；指定ID应增量同步 |
| 请求取消 | 此版本生成方法没有context参数；SDK提供连接/读取timeout | 调用前后及页间检查context，连接默认5秒、读取30秒并受调用方deadline约束；主动取消无法立即中断已进入的旧SDK调用，最多等待本次读取超时 |

区域API失败不再静默转成“成功的静态区域列表”。生产部署应按需要给RAM身份授予限定资源范围的
`ecs:DescribeInstances`；某一个区域的读取成功不代表其他区域也有权限。

## 离线验证

```sh
GOTOOLCHAIN=go1.26.10 go test ./internal/tree/...
GOTOOLCHAIN=go1.26.10 go test -race ./internal/tree/utils ./internal/tree/service
```

测试使用注入的ECS接口替身，覆盖短页继续、TotalCount=0、后页错误、重复token、101个ID分批和JSON转义、
取消请求不发出调用、读取权限检查、soldOut映射、EIP与时间格式、过滤全量同步的无副作用拒绝、
区域错误汇总及未实现厂商不伪报验证成功。没有真实网络或云凭据依赖。

## 来源

- [DescribeInstances 官方 API](https://www.alibabacloud.com/help/en/ecs/developer-reference/api-ecs-2014-05-26-describeinstances)
- [DescribeRegions 官方 API](https://www.alibabacloud.com/help/en/ecs/developer-reference/api-ecs-2014-05-26-describeregions)
- [当前 SDK DescribeInstances 源码](https://github.com/aliyun/alibaba-cloud-sdk-go/blob/v1.63.107/services/ecs/describe_instances.go)
- [当前 SDK DescribeRegions 源码](https://github.com/aliyun/alibaba-cloud-sdk-go/blob/v1.63.107/services/ecs/describe_regions.go)
- [当前 SDK 请求超时接口](https://github.com/aliyun/alibaba-cloud-sdk-go/blob/v1.63.107/sdk/requests/acs_request.go)
