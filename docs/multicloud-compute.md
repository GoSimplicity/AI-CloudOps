# 云计算资源接入：阿里云、AWS、腾讯云

这三个厂商都有实际 SDK 调用实现，可验证指定地域的实例读取权限、查询厂商地域列表并同步实例。
新增 AWS/Tencent 的验收依据是官方 SDK 类型/源码、API 文档和离线契约测试；没有真实云账号，
因此没有声明账号权限、网络连通或资源数据已在线验收。所有提供者只调用查询接口，不创建、修改或删除云实例。
全量同步中的删除仅作用于本地资产表。

| 能力 | 阿里云 ECS | AWS EC2 | 腾讯云 CVM |
| --- | --- | --- | --- |
| SDK | Alibaba Cloud SDK Go v1.63.107 | AWS SDK Go v2，EC2 v1.340.0 | CVM v1.3.183；region/common v1.3.194 |
| 凭据验证 | DescribeInstances，MaxResults=10 | DescribeInstances，MaxResults=5 | DescribeInstances，Limit=1 |
| 地域查询 | ECS DescribeRegions | EC2 DescribeRegions，包含opt-in状态 | region 2022-06-27 DescribeRegions，Product=cvm |
| 实例分页 | NextToken，MaxResults=100 | NextToken，未指定ID时MaxResults=1000 | Offset/Limit=100，按实际返回条数推进 |
| 指定实例 | 每批最多100个ID | 每批100个ID，不与MaxResults同时发送 | 每批最多100个ID |
| 取消 | 页间检查，旧SDK请求受timeout限制 | SDK context贯穿请求 | SDK WithContext贯穿请求 |

## 凭据与地域 API

使用已保存账号的长期 AccessKey/SecretKey。AWS 客户端使用显式凭据，不读取本机 profile、环境凭据或实例角色作为替代账号。
AWS `ASIA` 临时凭据需要 session token，目前明确拒绝；腾讯云 STS 临时凭据同样不在本版支持范围。
本版没有声明只有两个字段即可兼容临时凭据，也没有偷偷丢弃 token。

账户创建：`POST /api/tree/cloud/account/create`，包含 `provider`、`access_key`、`secret_key`、
`regions: [{region, region_name, is_default}]`。厂商枚举保持阿里云1、腾讯云2、AWS3、华为云4。
账户更新接受可选的同形 `regions` 数组；不能通过更新切换厂商。

实时地域列表使用：

```text
GET /api/tree/cloud/account/region/available-regions?cloud_account_id=123
```

服务端读取并解密该账号的凭据，使用账号的默认地域调用厂商。只返回
`{regions: [{region, region_name, available}]}`，不回传凭据。URL中的 `access_key` 或 `secret_key` 会被拒绝。
纯 `?provider=1/2/4` 仍提供静态地域参考，用于首次录入，不代表账号已经开通该地域。
AWS实时结果只将 `opt-in-not-required`、`opted-in` 标记可用；腾讯云使用 `RegionState=AVAILABLE`。
腾讯云旧CVM DescribeRegions已停止更新，因此本实现使用官方指向的新region服务。

AWS身份需要 `ec2:DescribeInstances`；实时地域查询另需 `ec2:DescribeRegions`。
腾讯云身份需要允许 CVM DescribeInstances 及地域管理 DescribeRegions 查询。
某一地域验证成功不代表其他地域也有授权。没有配置任何云资源写权限作为前提。

## 同步与字段真实性

`CloudComputeClient` 是可注入的只读边界，统一提供权限验证、地域列表、实例列表；生产工厂选择实际SDK，测试注入替身。
多地域服务按账户真实厂商分派，继续复用本地资产增量/全量同步。

- 任何后续页失败、取消、重复token/实例、腾讯云分页总数变化或未取得进展，都会返回错误而不返回部分清单；失败地域不会进入本地全量清理。
- 全量同步禁止同时筛选实例ID，避免把未查询的本地资产误删。跨地域可以部分成功，但错误不会被伪装成全部成功。
- 更新明确同步云端字段中的空值，例如已解绑的公网IP；不会覆盖本地SSH密码、密钥、用户名、端口、环境或服务树绑定。
- AWS名称来自Name标签（缺省为实例ID），状态/网络/镜像/CPU配置来自DescribeInstances。该API不返回内存容量、磁盘容量或真实账单：容量0表示未知，不表示测得0GiB；费用不推测。
- 腾讯云映射CPU、GiB内存、系统盘加数据盘容量、网络、镜像、标签、状态、PREPAID到期时间。未知计费类型不会伪装成已确认计费方式。
- 本地Currency字段沿用厂商默认元数据，不是账单验收；AWS预留实例折扣、Savings Plans等不能从DescribeInstances推断。

华为云、Azure、GCP仍未实现真实凭据/地域/实例查询，明确返回未实现；没有用隐藏入口替代实现或宣称六云全部可用。

## 离线回归

```sh
GOTOOLCHAIN=go1.26.10 go test ./internal/tree/...
GOTOOLCHAIN=go1.26.10 go test -race ./internal/tree/utils ./internal/tree/service
GOTOOLCHAIN=go1.26.10 go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
```

用例覆盖AWS token与InstanceIds互斥、腾讯云offset及101个ID分批、权限失败、取消、缺页/重复页、
地域状态、字段映射、禁止筛选全量同步、厂商失败不进入DAO清理、accountID凭据边界及本地SSH字段保护。
测试只使用SDK接口替身和测试字符串，不读取部署凭据，不访问真实云。
新增依赖扫描仍为0个可达漏洞；不可达依赖问题仍见安全基线文档。

## 官方依据

- [AWS DescribeInstances](https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeInstances.html)
- [AWS DescribeRegions](https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeRegions.html)
- [AWS EC2 SDK生成源码](https://github.com/aws/aws-sdk-go-v2/tree/main/service/ec2)
- [腾讯云CVM DescribeInstances](https://cloud.tencent.com/document/api/213/15728)
- [旧CVM地域API停止更新说明](https://cloud.tencent.com/document/api/213/15708)
- [新地域管理DescribeRegions](https://cloud.tencent.com/document/product/1596/77930)
- [腾讯云官方Go SDK](https://github.com/TencentCloud/tencentcloud-sdk-go)
- 阿里云逐项核对见 [aliyun-contract-review.md](aliyun-contract-review.md)。
