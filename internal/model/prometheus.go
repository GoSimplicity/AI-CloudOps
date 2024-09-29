package model

import (
	"github.com/prometheus/alertmanager/template"
)

// MonitorAlertEvent 告警事件与相关实体的关系
type MonitorAlertEvent struct {
	Model
	AlertName     string     `json:"alertName" binding:"required,min=1,max=200" gorm:"size:200;comment:告警名称"`                // 告警名称
	Fingerprint   string     `json:"fingerprint" binding:"required,min=1,max=50" gorm:"uniqueIndex;size:100;comment:告警唯一ID"` // 告警唯一ID
	Status        string     `json:"status,omitempty" gorm:"size:50;comment:告警状态（如告警中、已屏蔽、已认领、已恢复）"`                         // 告警状态（如告警中、已屏蔽、已认领、已恢复）
	RuleID        int        `json:"ruleId" gorm:"comment:关联的告警规则ID"`                                                        // 关联的告警规则ID
	SendGroupID   int        `json:"sendGroupId" gorm:"comment:关联的发送组ID"`                                                    // 关联的发送组ID
	EventTimes    int        `json:"eventTimes" gorm:"comment:触发次数"`                                                         // 触发次数
	SilenceID     string     `json:"silenceId,omitempty" gorm:"size:100;comment:AlertManager返回的静默ID"`                        // AlertManager返回的静默ID
	RenLingUserID int        `json:"renLingUserId" gorm:"comment:认领告警的用户ID"`                                                 // 认领告警的用户ID
	Labels        StringList `json:"labels,omitempty" gorm:"type:text;comment:标签组，格式为 key=v"`                                // 标签组，格式为 key=v

	// 前端使用字段
	Key           string            `json:"key" gorm:"-"`                     // 前端表格使用的Key
	AlertRuleName string            `json:"alertRuleName,omitempty" gorm:"-"` // 前端表格显示的告警规则名称
	SendGroupName string            `json:"sendGroupName,omitempty" gorm:"-"` // 前端表格显示的发送组名称
	Alert         template.Alert    `json:"alert,omitempty" gorm:"-"`         // 告警详情
	SendGroup     *MonitorSendGroup `json:"sendGroup,omitempty" gorm:"-"`     // 发送组对象
	RenLingUser   *User             `json:"renLingUser,omitempty" gorm:"-"`   // 认领告警的用户对象
	Rule          *MonitorAlertRule `json:"rule,omitempty" gorm:"-"`          // 告警规则对象
	LabelsM       map[string]string `json:"labelsM,omitempty" gorm:"-"`       // 标签键值对映射
	AnnotationsM  map[string]string `json:"annotationsM,omitempty" gorm:"-"`  // 注释键值对映射
}

// MonitorAlertRule 告警规则的配置
type MonitorAlertRule struct {
	Model
	Name        string     `json:"name" binding:"required,min=1,max=50" gorm:"uniqueIndex;size:100;comment:告警规则名称，支持通配符*进行模糊搜索"` // 告警规则名称，支持通配符*进行模糊搜索
	UserID      int        `json:"userId" gorm:"comment:创建该告警规则的用户ID"`                                                           // 创建该告警规则的用户ID
	PoolID      int        `json:"poolId" gorm:"comment:关联的Prometheus实例池ID"`                                                     // 关联的Prometheus实例池ID
	SendGroupID int        `json:"sendGroupId" gorm:"comment:关联的发送组ID"`                                                          // 关联的发送组ID
	TreeNodeID  int        `json:"treeNodeId" gorm:"comment:绑定的树节点ID"`                                                           // 绑定的树节点ID
	Enable      int        `json:"enable" gorm:"type:int;comment:是否启用告警规则：1启用，2禁用"`                                              // 是否启用告警规则：1启用，2禁用
	Expr        string     `json:"expr" gorm:"type:text;comment:告警规则表达式"`                                                        // 告警规则表达式
	Severity    string     `json:"severity,omitempty" gorm:"size:50;comment:告警级别，如critical、warning"`                             // 告警级别，如critical、warning
	GrafanaLink string     `json:"grafanaLink,omitempty" gorm:"type:text;comment:Grafana大盘链接"`                                   // Grafana大盘链接
	ForTime     string     `json:"forTime,omitempty" gorm:"size:50;comment:持续时间，达到此时间才触发告警"`                                     // 持续时间，达到此时间才触发告警
	Labels      StringList `json:"labels,omitempty" gorm:"type:text;comment:标签组，格式为 key=v"`                                      // 标签组，格式为 key=v
	Annotations StringList `json:"annotations,omitempty" gorm:"type:text;comment:注解，格式为 key=v"`                                  // 注解，格式为 key=v

	// 前端使用字段
	NodePath         string            `json:"nodePath,omitempty" gorm:"-"`         // 节点路径，形式为 a.b.c.d
	TreeNodeIDs      []int             `json:"treeNodeIds,omitempty" gorm:"-"`      // 节点ID整数数组
	Key              string            `json:"key" gorm:"-"`                        // 前端表格的Key
	PoolName         string            `json:"poolName,omitempty" gorm:"-"`         // 前端表格显示的Prometheus实例池名称
	SendGroupName    string            `json:"sendGroupName,omitempty" gorm:"-"`    // 前端表格显示的发送组名称
	CreateUserName   string            `json:"createUserName,omitempty" gorm:"-"`   // 前端表格显示的创建者用户名
	LabelsFront      string            `json:"labelsFront,omitempty" gorm:"-"`      // 前端显示的标签组字符串
	AnnotationsFront string            `json:"annotationsFront,omitempty" gorm:"-"` // 前端显示的注解字符串
	LabelsM          map[string]string `json:"labelsM,omitempty" gorm:"-"`          // 标签键值对映射
	AnnotationsM     map[string]string `json:"annotationsM,omitempty" gorm:"-"`     // 注释键值对映射
}

// MonitorSendGroup 发送组的配置
type MonitorSendGroup struct {
	Model
	Name                string     `json:"name" binding:"required,min=1,max=50" gorm:"uniqueIndex;size:100;comment:发送组英文名称，供AlertManager配置文件使用，支持通配符*进行模糊搜索"` // 发送组英文名称，供AlertManager配置文件使用，支持通配符*进行模糊搜索
	NameZh              string     `json:"nameZh" binding:"required,min=1,max=50" gorm:"uniqueIndex;size:100;comment:发送组中文名称，供告警规则选择发送组时使用，支持通配符*进行模糊搜索"`     // 发送组中文名称，供告警规则选择发送组时使用，支持通配符*进行模糊搜索
	Enable              int        `json:"enable" gorm:"type:int;comment:是否启用发送组：1启用，2禁用"`                                                                    // 是否启用发送组：1启用，2禁用
	UserID              int        `json:"userId" gorm:"comment:创建该发送组的用户ID"`                                                                                 // 创建该发送组的用户ID
	PoolID              int        `json:"poolId" gorm:"comment:关联的Prometheus实例池ID"`                                                                          // 关联的Prometheus实例池ID
	OnDutyGroupID       int        `json:"onDutyGroupId" gorm:"comment:值班组ID"`                                                                                // 值班组ID
	StaticReceiveUsers  []*User    `json:"staticReceiveUsers" gorm:"many2many:static_receive_users;comment:静态配置的接收人列表，多对多关系"`                                 // 静态配置的接收人列表，多对多关系
	FeiShuQunRobotToken string     `json:"feiShuQunRobotToken,omitempty" gorm:"size:255;comment:飞书机器人Token，对应IM群"`                                            // 飞书机器人Token，对应IM群
	RepeatInterval      string     `json:"repeatInterval,omitempty" gorm:"size:50;comment:默认重复发送时间"`                                                          // 默认重复发送时间
	SendResolved        int        `json:"sendResolved" gorm:"type:int;comment:是否发送恢复通知：1发送，2不发送"`                                                            // 是否发送恢复通知：1发送，2不发送
	NotifyMethods       StringList `json:"notifyMethods,omitempty" gorm:"type:text;comment:通知方法，如：email, im, phone, sms"`                                     // 通知方法，如：email, im, phone, sms
	NeedUpgrade         int        `json:"needUpgrade" gorm:"type:int;comment:是否需要告警升级：1需要，2不需要"`                                                             // 是否需要告警升级：1需要，2不需要
	FirstUpgradeUsers   []*User    `json:"firstUpgradeUsers" gorm:"many2many:first_upgrade_users;comment:第一升级人列表，多对多关系"`                                      // 第一升级人列表，多对多关系
	UpgradeMinutes      int        `json:"upgradeMinutes,omitempty" gorm:"type:int;comment:告警多久未恢复则升级（分钟）"`                                                   // 告警多久未恢复则升级（分钟）
	SecondUpgradeUsers  []*User    `json:"secondUpgradeUsers" gorm:"many2many:second_upgrade_users;comment:第二升级人列表，多对多关系"`                                    // 第二升级人列表，多对多关系

	// 前端使用字段
	TreeNodeIDs     []int    `json:"treeNodeIds,omitempty" gorm:"-"`     // 节点ID的整数数组
	FirstUserNames  []string `json:"firstUserNames,omitempty" gorm:"-"`  // 第一升级人的用户名数组
	Key             string   `json:"key" gorm:"-"`                       // 前端表格使用的Key
	PoolName        string   `json:"poolName,omitempty" gorm:"-"`        // 前端表格显示的AlertManager实例名称
	OnDutyGroupName string   `json:"onDutyGroupName,omitempty" gorm:"-"` // 前端表格显示的值班组名称
	CreateUserName  string   `json:"createUserName,omitempty" gorm:"-"`  // 前端表格显示的创建者用户名
}

// MonitorAlertManagerPool AlertManager 实例池的配置
type MonitorAlertManagerPool struct {
	Model
	Name                  string     `json:"name" binding:"required,min=1,max=50" gorm:"uniqueIndex;size:100;comment:AlertManager实例名称，支持使用通配符*进行模糊搜索"` // AlertManager实例名称，支持使用通配符*进行模糊搜索
	AlertManagerInstances StringList `json:"alertManagerInstances" gorm:"type:text;comment:选择多个AlertManager实例"`                                        // 选择多个AlertManager实例
	UserID                int        `json:"userId" gorm:"comment:创建该实例池的用户ID"`                                                                        // 创建该实例池的用户ID
	ResolveTimeout        string     `json:"resolveTimeout,omitempty" gorm:"size:50;comment:默认恢复时间"`                                                   // 默认恢复时间
	GroupWait             string     `json:"groupWait,omitempty" gorm:"size:50;comment:默认分组第一次等待时间"`                                                   // 默认分组第一次等待时间
	GroupInterval         string     `json:"groupInterval,omitempty" gorm:"size:50;comment:默认分组等待间隔"`                                                  // 默认分组等待间隔
	RepeatInterval        string     `json:"repeatInterval,omitempty" gorm:"size:50;comment:默认重复发送时间"`                                                 // 默认重复发送时间
	GroupBy               StringList `json:"groupBy,omitempty" gorm:"type:text;comment:分组的标签"`                                                         // 分组的标签
	Receiver              string     `json:"receiver,omitempty" gorm:"size:100;comment:兜底接收者"`                                                         // 兜底接收者

	// 前端使用字段
	GroupByFront   string `json:"groupByFront,omitempty" gorm:"-"`   // 前端显示的GroupBy字符串
	Key            string `json:"key" gorm:"-"`                      // 前端表格使用的Key
	CreateUserName string `json:"createUserName,omitempty" gorm:"-"` // 创建者用户名，用于前端展示
}

// MonitorOnDutyChange 值班换班记录
type MonitorOnDutyChange struct {
	Model
	OnDutyGroupID int    `json:"onDutyGroupId" gorm:"uniqueIndex:group_id_date;comment:值班组ID，用于唯一标识值班历史记录"`   // 值班组ID，用于标识换班记录所属的值班组
	UserID        int    `json:"userId" gorm:"comment:创建该换班记录的用户ID"`                                          // 创建该换班记录的用户ID
	DateString    string `json:"dateString" gorm:"uniqueIndex:group_id_date;size:50;comment:计划哪一天进行换班的日期字符串"` // 计划哪一天进行换班的日期字符串
	OnDutyUserID  int    `json:"onDutyUserId" gorm:"comment:换班后值班人员的用户ID"`                                    // 换班后值班人员的用户ID
	OriginUserID  int    `json:"originUserId" gorm:"comment:换班前原定的值班人员用户ID"`                                  // 换班前原定的值班人员用户ID

	// 前端使用字段
	TargetUserName string `json:"targetUserName,omitempty" gorm:"-"` // 前端传递的目标用户名称
	OriginUserName string `json:"originUserName,omitempty" gorm:"-"` // 前端传递的原用户名称
	Key            string `json:"key" gorm:"-"`                      // 前端表格使用的Key
	PoolName       string `json:"poolName,omitempty" gorm:"-"`       // 前端表格使用的池名称
	CreateUserName string `json:"createUserName,omitempty" gorm:"-"` // 前端表格显示的创建者用户名
}

// MonitorOnDutyGroup 值班组的配置
type MonitorOnDutyGroup struct {
	Model
	Name                      string  `json:"name" binding:"required,min=1,max=50" gorm:"uniqueIndex;size:100;comment:值班组名称，供AlertManager配置文件使用，支持通配符*进行模糊搜索"` // 值班组名称，供AlertManager配置文件使用，支持通配符*进行模糊搜索
	UserID                    int     `json:"userId" gorm:"comment:创建该值班组的用户ID"`                                                                               // 创建该值班组的用户ID
	Members                   []*User `json:"members" gorm:"many2many:monitor_onDuty_users;comment:值班组成员列表，多对多关系"`                                             // 值班组成员列表，多对多关系
	ShiftDays                 int     `json:"shiftDays,omitempty" gorm:"type:int;comment:轮班周期，以天为单位"`                                                          // 轮班周期，以天为单位
	ImRobotToken              string  `json:"imRobotToken,omitempty" gorm:"size:255;comment:IM机器人Token，用于指定对应的IM群"`                                            // IM机器人Token，用于指定对应的IM群
	YesterdayNormalDutyUserID int     `json:"yesterdayNormalDutyUserId" gorm:"comment:昨天的正常排班值班人ID，由cron任务设置"`                                                 // 昨天的正常排班值班人ID，由cron任务设置

	// 前端使用字段
	TodayDutyUser  *User    `json:"todayDutyUser,omitempty" gorm:"-"`  // 今天的值班用户
	UserNames      []string `json:"userNames,omitempty" gorm:"-"`      // 值班组中所有成员的用户名列表
	Key            string   `json:"key" gorm:"-"`                      // 前端表格使用的Key
	CreateUserName string   `json:"createUserName,omitempty" gorm:"-"` // 创建者用户名，用于前端展示
}

// MonitorOnDutyHistory 值班历史记录
type MonitorOnDutyHistory struct {
	Model
	OnDutyGroupID int    `json:"onDutyGroupId" gorm:"uniqueIndex:group_id_date;comment:值班组ID，用于唯一标识值班历史记录"` // 值班组ID，用于唯一标识值班历史记录
	DateString    string `json:"dateString" gorm:"uniqueIndex:group_id_date;size:50;comment:具体哪一天的日期字符串"`   // 具体哪一天的日期字符串
	OnDutyUserID  int    `json:"onDutyUserId" gorm:"comment:当天值班人员的用户ID"`                                   // 当天值班人员的用户ID
	OriginUserID  int    `json:"originUserId" gorm:"comment:原计划的值班人员用户ID"`                                  // 原计划的值班人员用户ID

	// 前端使用字段
	Key            string `json:"key" gorm:"-"`                      // 前端表格使用的Key
	PoolName       string `json:"poolName,omitempty" gorm:"-"`       // 前端表格使用的池名称
	CreateUserName string `json:"createUserName,omitempty" gorm:"-"` // 前端表格显示的创建者用户名
}

// MonitorRecordRule 记录规则的配置
type MonitorRecordRule struct {
	Model
	Name       string `json:"name" binding:"required,min=1,max=50" gorm:"uniqueIndex;size:100;comment:记录规则名称，支持使用通配符*进行模糊搜索"`      // 记录规则名称，支持使用通配符*进行模糊搜索
	RecordName string `json:"recordName" binding:"required,min=1,max=500" gorm:"uniqueIndex;size:500;comment:记录名称，支持使用通配符*进行模糊搜索"` // 记录名称，支持使用通配符*进行模糊搜索
	UserID     int    `json:"userId" gorm:"comment:创建该记录规则的用户ID"`                                                                  // 创建该记录规则的用户ID
	PoolID     int    `json:"poolId" gorm:"comment:关联的Prometheus实例池ID"`                                                            // 关联的Prometheus实例池ID
	TreeNodeID int    `json:"treeNodeId" gorm:"comment:绑定的树节点ID"`                                                                  // 绑定的树节点ID
	Enable     int    `json:"enable" gorm:"type:int;comment:是否启用记录规则：1启用，2禁用"`                                                     // 是否启用记录规则：1启用，2禁用
	ForTime    string `json:"forTime,omitempty" gorm:"size:50;comment:持续时间，达到此时间才触发记录规则"`                                          // 持续时间，达到此时间才触发记录规则
	Expr       string `json:"expr" gorm:"type:text;comment:记录规则表达式"`                                                               // 记录规则表达式

	// 前端使用字段
	NodePath         string            `json:"nodePath,omitempty" gorm:"-"`         // 节点路径，形式为 a.b.c.d
	TreeNodeIDs      []int             `json:"treeNodeIds,omitempty" gorm:"-"`      // 节点ID整数数组
	Key              string            `json:"key" gorm:"-"`                        // 前端表格的Key
	PoolName         string            `json:"poolName,omitempty" gorm:"-"`         // 前端表格显示的池名称
	SendGroupName    string            `json:"sendGroupName,omitempty" gorm:"-"`    // 前端表格显示的发送组名称
	CreateUserName   string            `json:"createUserName,omitempty" gorm:"-"`   // 前端表格显示的创建者用户名
	LabelsFront      string            `json:"labelsFront,omitempty" gorm:"-"`      // 前端显示的标签组字符串
	AnnotationsFront string            `json:"annotationsFront,omitempty" gorm:"-"` // 前端显示的注解字符串
	LabelsM          map[string]string `json:"labelsM,omitempty" gorm:"-"`          // 标签键值对映射
	AnnotationsM     map[string]string `json:"annotationsM,omitempty" gorm:"-"`     // 注释键值对映射
}

// MonitorScrapeJob 监控采集任务的配置
type MonitorScrapeJob struct {
	Model
	Name                     string     `json:"name" binding:"required,min=1,max=50" gorm:"uniqueIndex;size:100;comment:采集任务名称，支持使用通配符*进行模糊搜索"` // 采集任务名称，支持使用通配符*进行模糊搜索
	UserID                   int        `json:"userId" gorm:"comment:任务关联的用户ID"`                                                                // 任务关联的用户ID
	Enable                   int        `json:"enable" gorm:"type:int;comment:是否启用采集任务：1为启用，2为禁用"`                                              // 是否启用采集任务：1为启用，2为禁用
	ServiceDiscoveryType     string     `json:"serviceDiscoveryType,omitempty" gorm:"size:50;comment:服务发现类型，支持 k8s 或 http"`                     // 服务发现类型，支持 "k8s" 或 "http"
	MetricsPath              string     `json:"metricsPath,omitempty" gorm:"size:255;comment:监控采集的路径"`                                          // 监控采集的路径
	Scheme                   string     `json:"scheme,omitempty" gorm:"size:10;comment:监控采集的协议方案（如 http 或 https）"`                              // 监控采集的协议方案（如 http 或 https）
	ScrapeInterval           int        `json:"scrapeInterval,omitempty" gorm:"default:30;type:int;comment:采集的时间间隔（秒）"`                         // 采集的时间间隔（秒）
	ScrapeTimeout            int        `json:"scrapeTimeout,omitempty" gorm:"default:10;type:int;comment:采集的超时时间（秒）"`                          // 采集的超时时间（秒）
	PoolID                   int        `json:"poolId" gorm:"comment:关联的采集池ID"`                                                                 // 关联的采集池ID
	RelabelConfigsYamlString string     `json:"relabelConfigsYamlString,omitempty" gorm:"type:text;comment:relabel配置的YAML字符串"`                  // relabel配置的YAML字符串
	RefreshInterval          int        `json:"refreshInterval,omitempty" gorm:"type:int;comment:刷新目标的时间间隔（针对服务树http类型，秒）"`                     // 刷新目标的时间间隔（针对服务树http类型，秒）
	Port                     int        `json:"port,omitempty" gorm:"type:int;comment:端口号（针对服务树服务发现接口）"`                                        // 端口号（针对服务树服务发现接口）
	TreeNodeIDs              StringList `json:"treeNodeIds,omitempty" gorm:"type:text;comment:服务树接口绑定的树节点ID列表，用于获取IP列表"`                        // 服务树接口绑定的树节点ID列表，用于获取IP列表
	KubeConfigFilePath       string     `json:"kubeConfigFilePath,omitempty" gorm:"size:255;comment:连接apiServer的Kubernetes配置文件路径"`              // 连接apiServer的Kubernetes配置文件路径
	TlsCaFilePath            string     `json:"tlsCaFilePath,omitempty" gorm:"size:255;comment:TLS CA证书文件路径"`                                   // TLS CA证书文件路径
	TlsCaContent             string     `json:"tlsCaContent,omitempty" gorm:"type:text;comment:TLS CA证书内容"`                                     // TLS CA证书内容
	BearerToken              string     `json:"bearerToken,omitempty" gorm:"type:text;comment:鉴权Token内容"`                                       // 鉴权Token内容
	BearerTokenFile          string     `json:"bearerTokenFile,omitempty" gorm:"size:255;comment:鉴权Token文件路径"`                                  // 鉴权Token文件路径
	KubernetesSdRole         string     `json:"kubernetesSdRole,omitempty" gorm:"size:50;comment:Kubernetes服务发现角色"`                             // Kubernetes服务发现角色

	// 前端使用字段
	TreeNodeIDIns  []int  `json:"treeNodeIdIns,omitempty" gorm:"-"`  // 树节点ID的整数列表
	Key            string `json:"key" gorm:"-"`                      // 前端表格使用的Key
	PoolName       string `json:"poolName,omitempty" gorm:"-"`       // 采集池的名称，用于前端展示
	CreateUserName string `json:"createUserName,omitempty" gorm:"-"` // 创建用户的名称，用于前端展示
}

// MonitorScrapePool 采集池的配置
type MonitorScrapePool struct {
	Model
	Name                  string     `json:"name" binding:"required,min=1,max=50" gorm:"uniqueIndex;size:100;comment:采集池名称，支持使用通配符*进行模糊搜索"`        // 采集池名称，支持使用通配符*进行模糊搜索
	PrometheusInstances   StringList `json:"prometheusInstances,omitempty" gorm:"type:text;comment:选择多个Prometheus实例"`                              // 选择多个Prometheus实例
	AlertManagerInstances StringList `json:"alertManagerInstances,omitempty" gorm:"type:text;comment:选择多个AlertManager实例"`                          // 选择多个AlertManager实例
	UserID                int        `json:"userId" gorm:"comment:创建该采集池的用户ID"`                                                                    // 创建该采集池的用户ID
	ScrapeInterval        int        `json:"scrapeInterval,omitempty" gorm:"default:30;type:int;comment:采集间隔（秒）"`                                  // 采集间隔（秒）
	ScrapeTimeout         int        `json:"scrapeTimeout,omitempty" gorm:"default:10;type:int;comment:采集超时时间（秒）"`                                 // 采集超时时间（秒）
	ExternalLabels        StringList `json:"externalLabels,omitempty" gorm:"type:text;comment:remote_write时添加的标签组，格式为 key=v，例如 scrape_ip=1.1.1.1"` // remote_write时添加的标签组，格式为 key=v，例如 scrape_ip=1.1.1.1
	SupportAlert          int        `json:"supportAlert" gorm:"type:int;comment:是否支持告警：1支持，2不支持"`                                                 // 是否支持告警：1支持，2不支持
	SupportRecord         int        `json:"supportRecord" gorm:"type:int;comment:是否支持预聚合：1支持，2不支持"`                                               // 是否支持预聚合：1支持，2不支持
	RemoteReadUrl         string     `json:"remoteReadUrl,omitempty" gorm:"size:255;comment:远程读取的地址"`                                              // 远程读取的地址
	AlertManagerUrl       string     `json:"alertManagerUrl,omitempty" gorm:"size:255;comment:AlertManager的地址"`                                    // AlertManager的地址
	RuleFilePath          string     `json:"ruleFilePath,omitempty" gorm:"size:255;comment:规则文件路径"`                                                // 规则文件路径
	RecordFilePath        string     `json:"recordFilePath,omitempty" gorm:"size:255;comment:记录文件路径"`                                              // 记录文件路径
	RemoteWriteUrl        string     `json:"remoteWriteUrl,omitempty" gorm:"size:255;comment:远程写入的地址"`                                             // 远程写入的地址
	RemoteTimeoutSeconds  int        `json:"remoteTimeoutSeconds,omitempty" gorm:"default:5;type:int;comment:远程写入的超时时间（秒）"`                        // 远程写入的超时时间（秒）

	// 前端使用字段
	ExternalLabelsFront string `json:"externalLabelsFront,omitempty" gorm:"-"` // 前端显示的ExternalLabels字符串
	Key                 string `json:"key" gorm:"-"`                           // 前端表格使用的Key
	CreateUserName      string `json:"createUserName,omitempty" gorm:"-"`      // 创建者用户名，用于前端展示
}
