# 临时云凭据与华为地域

> 当前口径（2026-10-30）：本页保留分阶段交付与隔离测试证据；完整能力状态及未验边界见[当前完成度矩阵](readiness-current.md)。不以早期“未接线”描述覆盖后续实现，也不将隔离测试视作真实云验收。

AWS 和腾讯云支持显式临时凭据：类型、访问密钥、会话令牌和UTC到期时间必须成组提供；不完整、过期或混合轮换在SDK调用前拒绝。令牌加密持久化，不出现在普通JSON、URL、日志或审计中；旧记录仍按长期凭据读取。凭据轮换与地域变更在父账户事务中完成并使用版本检查。

华为区域须显式提供各地域的 project_id。由于企业项目查询存在100项上限且无法证明覆盖完整，华为只允许指定实例ID的增量同步，不能无ID增量或全量同步。AWS/腾讯临时凭据现已支持单地域事务化全量对账：地域租约、父账户凭据版本与到期状态会在提交前重新校验；有 TestTemporaryFullSyncAtomicAndExpiredCredentials 和 TestMySQLTemporaryFullSyncAtomic 回归。跨地域本身仍可能部分成功，不承诺跨地域整体事务。账户相关前端字段、整组凭据轮换和同步入口已接入；华为/Azure/GCP的指定ID限制仍不放宽。

升级前须停止旧版服务、备份并在数据库副本演练账户新增字段（credential_type、session_token、credential_expires_at、credential_version）与地域 project_id 的迁移；全部实例使用新代码，不能混跑。旧密文不重写。回滚前必须核对新字段及临时凭据，旧版本不能安全管理新类型。

验证：tree完整race、SDK签名的内存传输和拒绝路径、真实handler/RBAC与日志canary、隔离MySQL旧表迁移/回滚/并发轮换。未调用真实云或生产环境。
