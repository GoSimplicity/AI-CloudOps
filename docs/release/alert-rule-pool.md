# 告警规则池实体

告警规则的 pool_id 指向 Prometheus 采集池。规则文件按采集池生成；AlertManager池用于通知路由，两个ID空间不是同一实体。

原保存服务及页面选择器使用AlertManager池，生成器却按ScrapePool查找，导致两表ID偶然相同才掩盖错误。现在创建/更新校验、列表与详情池名、前端分页/搜索/离页选中项详情都使用ScrapePool。pool_id字段名与其它DTO不变，Wire v0.6.0实际重新生成。

## 历史数据

本次不按数字相同自动转换ID，不新增映射表，也不执行数据库迁移。发布前须核对已有规则pool_id是否为真正目标采集池；AlertManager ID不存在于Scrape表时明确拒绝新保存，历史读取不借另一表伪造池名。数字同时存在于两表也不能证明业务意图一致。

## 验证

- 不重合的Alert99/Scrape7内存替身：99被拒，7可创建/更新，实际生成器按7读取且真实rulefmt解析通过。
- nil、错ID及DAO错误不会写规则；列表/详情只显示Scrape名，不猜测迁移。
- Go service/pkg/di通过，新增17子场景；Web namespace9+原表单89/列表37/扫描29共164项通过，应用类型检查通过。
- 新Web测试lint0，页面71→70，其它规则不增；旧FormSafety只替换3处目录API fixture，原断言保留。

规则构造校验、标签/持续时间及发布状态机另阶段处理。本次未连接数据库、Prometheus、AlertManager或生产服务。
