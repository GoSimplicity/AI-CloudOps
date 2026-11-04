# 配置键与生成名称的无损宽化

v4只扩宽三个已有表，不增加表、回填或删除数据，也不改Redis物理key。v1/v2/v3发布SQL与执行包装保持原字节和校验和。

| 字段 | 旧容量 | 当前容量 |
| --- | --- | --- |
| metadata name | 100字符 | 255字符 |
| metadata instance_ip | 45字符 | 128字符 |
| namespace guard instance_key | 180字节 | 512字节 |
| artifact event instance_key | 180字节 | 512字节 |

45字符合法IPv6加`_1`就有47字符，最大正组ID会到65；合法API池名50字符生成webhook名称可到124，历史物理池名100字符时可到174。新容量覆盖这些实际命名需求，不依赖压缩IPv6、裁剪名称或改变既有owner命名规则。

Go共享常量按Unicode码点限制key128、Name255，二进制列预留每字符四字节。Create/Update DTO同步上限，Update key的omitempty及DAO struct省略零值规则保持；旧45个emoji、自定义SQL键、大小写/空格字节差异和跨kind合法性保留。宽度不赋予Redis投影权限，`pool:7:ips`仍不能作为成员SET的正文操作授权。

## 迁移保护

持锁连接先验证metadata表默认字符属性与四个源列的完整发布属性，全部通过后才执行三个ALTER。varchar列继承已确认一致的utf8mb4/utf8mb4_0900_ai_ci默认值，NOT NULL、comment和索引不变；guard/event保持VARBINARY。不接受静默collation转换或意外源结构漂移。

初次原生全组检查实际捕获迁移显式列collation与GORM继承默认的SHOW CREATE表达差异；修复增加表默认属性校验并采用同值继承，原结构硬比对保留。Apply前置拒绝保证零ALTER，但迁移引擎可能已记录started/failed意图；不能宣称没有history写入或自动重试dirty。

已有完整干净版本库在停写后显式执行pending迁移。源码更改与测试不表示已对任何生产库执行；本地预览仍固定0618/schema2，未应用本v4。

## 已执行验收

五包race、vet、标签编译及全Go测试通过。隔离MySQL8.4.11/RR实际六子场景通过：fresh+clock、V3→V4保留全部数据/ID/key/name/guard/event/NULL和[]/索引/账本原值，以及dirty、drift、源column collation和table-default漂移拒绝。47/65key与106Name在V3实际拒绝，V4接受；128emoji/255Name与129/256拒绝在真实数据库验证，SQLite不替代容量证明。

原失败的TestCurrentSchemaMatchesManifest已独立转绿；完整internal/dbschema与cmd/schema两组Docker原生验收通过。临时库、账号和DSN清理，未上线。前端真实Form长度与权限修复另在Web阶段验收，不以仅改formRules宣称实际submit已生效。
