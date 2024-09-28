package model

import (
	"database/sql/driver"
	"strings"
)

// ResourceTree 表示 CMDB 中的资源树节点，包含资源的基本信息
type ResourceTree struct {
	InstanceName     string     `json:"instanceName" gorm:"uniqueIndex;type:varchar(100);comment:资源实例名称，支持模糊搜索"` // 资源实例名称，支持模糊搜索
	Hash             string     `json:"hash" gorm:"uniqueIndex;type:varchar(200);comment:用于资源更新的哈希值"`            // 增量更新的哈希值
	Vendor           string     `json:"vendor" gorm:"type:varchar(50);comment:云厂商名称，例：阿里云、华为云、AWS"`              // 云厂商名称
	CreateByOrder    bool       `json:"createByOrder" gorm:"comment:是否由工单创建，工单创建的资源不会被自动更新删除"`                   // 是否由工单创建的标识
	VpcId            string     `json:"vpcId" gorm:"type:varchar(100);comment:专有网络 VPC ID"`                      // 专有网络 VPC ID
	ZoneId           string     `json:"zoneId" gorm:"type:varchar(100);comment:实例所属可用区 ID，如 cn-hangzhou-g"`      // 可用区 ID
	Env              string     `json:"env" gorm:"type:varchar(50);comment:环境标识，如 dev、stage、prod"`               // 环境标识
	PayType          string     `json:"payType" gorm:"type:varchar(50);comment:付费类型，按量付费或包年包月"`                  // 付费类型
	Status           string     `json:"status" gorm:"type:varchar(50);comment:资源状态，如 Running、Stopped"`           // 资源状态
	Description      string     `json:"description" gorm:"type:text;comment:资源描述，如 CentOS 7.4 操作系统"`             // 资源描述
	Tags             StringList `json:"tags" gorm:"type:varchar(500);comment:资源标签集合，用于分类和筛选"`                    // 资源标签
	SecurityGroupIds StringList `json:"securityGroupIds" gorm:"type:varchar(500);comment:安全组 ID 列表"`             // 安全组 ID 列表
	PrivateIpAddress StringList `json:"privateIpAddress" gorm:"type:varchar(500);comment:私有 IP 地址列表"`            // 私有 IP 地址列表
	PublicIpAddress  StringList `json:"publicIpAddress" gorm:"type:varchar(500);comment:公网 IP 地址列表"`             // 公网 IP 地址列表
	IpAddr           string     `json:"ipAddr" gorm:"type:varchar(45);comment:单个公网 IP 地址"`                       // 单个公网 IP 地址
	CreationTime     string     `json:"creationTime" gorm:"type:varchar(30);comment:创建时间，ISO 8601 格式"`           // 创建时间，ISO 8601 格式

	// 前端展示信息
	Key string `json:"key" gorm:"-"` // 前端使用的资源键值
}

// TreeNode 表示服务树的节点, 包含节点的层级关系和相关资源绑定
type TreeNode struct {
	Model

	Title  string `json:"title" gorm:"uniqueIndex:idx_pid_title;type:varchar(50);comment:节点名称"` // 节点名称
	Pid    int    `json:"pId" gorm:"index;uniqueIndex:idx_pid_title;comment:父节点 ID"`            // 父节点 ID
	Level  int    `json:"level" gorm:"comment:节点层级"`                                            // 节点层级, 用于标识树的深度
	IsLeaf bool   `json:"isLeaf" gorm:"comment:是否为叶子节点"`                                        // 是否为叶子节点
	Desc   string `json:"desc" gorm:"type:text;comment:节点描述"`                                   // 节点描述信息

	// 关联的负责人信息
	OpsAdmins []*User `json:"ops_admins" gorm:"many2many:tree_node_ops_admins;comment:运维负责人列表"` // 运维负责人列表
	RdAdmins  []*User `json:"rd_admins" gorm:"many2many:tree_node_rd_admins;comment:研发负责人列表"`   // 研发负责人列表
	RdMembers []*User `json:"rd_members" gorm:"many2many:tree_node_rd_members;comment:研发工程师列表"` // 研发工程师列表

	// 绑定的资源信息
	BindEcs []*ResourceEcs `json:"bind_ecs" gorm:"many2many:bind_ecs;comment:绑定的 ECS 资源列表"` // 绑定的 ECS 资源列表
	BindElb []*ResourceElb `json:"bind_elb" gorm:"many2many:bind_elb;comment:绑定的 ELB 资源列表"` // 绑定的 ELB 资源列表
	BindRds []*ResourceRds `json:"bind_rds" gorm:"many2many:bind_rds;comment:绑定的 RDS 资源列表"` // 绑定的 RDS 资源列表

	// 前端展示信息
	Key           string      `json:"key" gorm:"-"`             // 节点唯一标识，前端使用
	Label         string      `json:"label" gorm:"-"`           // 节点显示名称，前端使用
	Value         int         `json:"value" gorm:"-"`           // 节点值，前端使用
	OpsAdminUsers StringList  `json:"ops_admin_users" gorm:"-"` // 运维负责人姓名列表，前端使用
	RdAdminUsers  StringList  `json:"rd_admin_users" gorm:"-"`  // 研发负责人姓名列表，前端使用
	RdMemberUsers StringList  `json:"rd_member_users" gorm:"-"` // 研发工程师姓名列表，前端使用
	Children      []*TreeNode `json:"children" gorm:"-"`        // 子节点列表，前端使用

	// 节点统计信息
	EcsNum           int `json:"ecsNum" gorm:"-"`           // 绑定的 ECS 数量
	ElbNum           int `json:"elbNum" gorm:"-"`           // 绑定的 ELB 数量
	RdsNum           int `json:"rdsNum" gorm:"-"`           // 绑定的 RDS 数量
	NodeNum          int `json:"nodeNum" gorm:"-"`          // 子节点数量
	LeafNodeNum      int `json:"leafNodeNum" gorm:"-"`      // 叶子节点数量
	EcsCpuTotal      int `json:"ecsCpuTotal" gorm:"-"`      // 绑定的 ECS 总 CPU 核数
	ElbBandWithTotal int `json:"elbBandWithTotal" gorm:"-"` // 绑定的 ELB 带宽上限
	EcsMemoryTotal   int `json:"ecsMemoryTotal" gorm:"-"`   // 绑定的 ECS 总内存
	EcsDiskTotal     int `json:"ecsDiskTotal" gorm:"-"`     // 绑定的 ECS 总磁盘空间

	// 分类统计数据
	GroupByVendor    []*ChartItem   `json:"groupByVendor" gorm:"-"`    // 按厂商分类统计
	GroupByZoneId    []*ChartItem   `json:"groupByZoneId" gorm:"-"`    // 按可用区分类统计
	GroupByOSName    map[string]int `json:"-" gorm:"-"`                // 按操作系统分类映射
	GroupByVendorElb []*ChartItem   `json:"groupByVendorElb" gorm:"-"` // 按厂商分类统计 ELB
	GroupByVendorRds []*ChartItem   `json:"groupByVendorRds" gorm:"-"` // 按厂商分类统计 RDS

	// 操作权限
	CanAdminNode bool   `json:"canAdminNode" gorm:"-"` // 是否有权限操作此节点
	NodePath     string `json:"nodePath" gorm:"-"`     // 节点路径, 格式如: a.b.c.d
}

// ChartItem 用于前端展示的图表数据结构
type ChartItem struct {
	Name  string `json:"name"`  // 分类名称
	Value int    `json:"value"` // 分类对应的值
}

// ResourceEcs 表示 ECS 资源的结构体, 包含虚拟机的硬件和网络信息
type ResourceEcs struct {
	Model
	ResourceTree

	// 核心资源属性
	OsType            string     `json:"osType" gorm:"type:varchar(50);comment:操作系统类型，例如 win、linux"`           // 操作系统类型
	VmType            int        `json:"vmType" gorm:"default:1;comment:设备类型，1=虚拟设备，2=物理设备"`                   // 设备类型
	InstanceType      string     `json:"instanceType" gorm:"type:varchar(100);comment:实例类型，例：ecs.g8a.2xlarge"` // 实例类型
	Cpu               int        `json:"cpu" gorm:"comment:虚拟 CPU 核数"`                                         // 虚拟 CPU 核数
	Memory            int        `json:"memory" gorm:"comment:内存大小，单位 GiB"`                                    // 内存大小，单位 GiB
	Disk              int        `json:"disk" gorm:"comment:磁盘大小，单位 GiB"`                                      // 磁盘大小，单位 GiB
	OSName            string     `json:"osName" gorm:"type:varchar(100);comment:操作系统名称，例：CentOS 7.4 64 位"`     // 操作系统名称
	ImageId           string     `json:"imageId" gorm:"type:varchar(100);comment:镜像模板 ID"`                     // 镜像模板 ID
	Hostname          string     `json:"hostname" gorm:"type:varchar(100);comment:主机名"`                        // 主机名
	NetworkInterfaces StringList `json:"networkInterfaces" gorm:"type:varchar(500);comment:弹性网卡 ID 集合"`        // 弹性网卡 ID 集合
	DiskIds           StringList `json:"diskIds" gorm:"type:varchar(500);comment:云盘 ID 集合"`                    // 云盘 ID 集合

	// 时间相关字段
	StartTime       string `json:"startTime" gorm:"type:varchar(30);comment:最近启动时间, ISO 8601 标准, UTC+0 时间"`       // 最近启动时间
	AutoReleaseTime string `json:"autoReleaseTime" gorm:"type:varchar(30);comment:自动释放时间, ISO 8601 标准, UTC+0 时间"` // 自动释放时间
	LastInvokedTime string `json:"lastInvokedTime" gorm:"type:varchar(30);comment:最近调用时间, ISO 8601 标准, UTC+0 时间"` // 最近调用时间

	// 多对多关系
	BindNodes []*TreeNode `json:"bind_nodes" gorm:"many2many:bind_ecss;comment:绑定的树节点列表"` // 绑定的树节点
}

// EcsBuyWorkOrder 表示购买 ECS 工作订单的结构体
type EcsBuyWorkOrder struct {
	Vendor         string `json:"vendor" gorm:"type:varchar(50);comment:云厂商名称, 例: 阿里云"`                    // 云厂商名称
	Num            int    `json:"num" gorm:"comment:购买的 ECS 实例数量"`                                         // 购买的 ECS 实例数量
	BindLeafNodeId int    `json:"bindLeafNodeId" gorm:"comment:绑定的叶子节点 ID"`                                // 绑定的叶子节点 ID
	InstanceType   string `json:"instance_type" gorm:"type:varchar(100);comment:实例类型, 例: ecs.g8a.2xlarge"` // 实例类型
	Hostnames      string `json:"hostnames" gorm:"type:text;comment:主机名, 支持多条记录, 用 \\n 分隔"`                // 主机名, 支持多条记录, 用 \n 分隔
}

// ResourceElb 表示负载均衡器资源的结构体
type ResourceElb struct {
	Model
	ResourceTree

	// 负载均衡器的核心属性
	LoadBalancerType   string      `json:"loadBalancerType" gorm:"type:varchar(50);comment:负载均衡类型, 例: nlb, alb, clb"` // 负载均衡类型, 如 nlb, alb, clb
	BandwidthCapacity  int         `json:"bandwidthCapacity" gorm:"comment:带宽容量上限, 单位 Mb, 例: 50"`                     // 带宽容量上限, 单位 Mb
	AddressType        string      `json:"addressType" gorm:"type:varchar(50);comment:地址类型, 公网或内网"`                   // 地址类型, 支持公网或内网
	DNSName            string      `json:"dnsName" gorm:"type:varchar(255);comment:DNS 解析地址"`                         // DNS 解析地址
	BandwidthPackageId string      `json:"bandwidthPackageId" gorm:"type:varchar(100);comment:绑定的带宽包 ID"`             // 绑定的带宽包 ID
	CrossZoneEnabled   bool        `json:"crossZoneEnabled" gorm:"comment:是否启用跨可用区"`                                  // 是否启用跨可用区
	BindNodes          []*TreeNode `json:"bind_nodes" gorm:"many2many:bind_elbs;comment:绑定的树节点列表"`                    // 绑定的树节点
}

// ResourceRds 表示 RDS 资源的结构体
type ResourceRds struct {
	Model
	ResourceTree

	// RDS 的核心属性
	Engine            string `json:"engine" gorm:"type:varchar(50);comment:数据库引擎类型, 例: mysql, mariadb, postgresql"`           // 数据库引擎类型
	DBInstanceNetType string `json:"dbInstanceNetType" gorm:"type:varchar(50);comment:实例网络类型, 例: Internet(外网), Intranet(内网)"` // 实例网络类型
	DBInstanceClass   string `json:"dbInstanceClass" gorm:"type:varchar(100);comment:实例规格, 例: rds.mys2.small"`                // 实例规格
	DBInstanceType    string `json:"dbInstanceType" gorm:"type:varchar(50);comment:实例类型, 例: Primary(主实例), Readonly(只读实例)"`    // 实例类型
	EngineVersion     string `json:"engineVersion" gorm:"type:varchar(50);comment:数据库版本, 例: 8.0, 5.7"`                        // 数据库版本
	MasterInstanceId  string `json:"masterInstanceId" gorm:"type:varchar(100);comment:主实例 ID"`                                // 主实例 ID
	DBInstanceStatus  string `json:"dbInstanceStatus" gorm:"type:varchar(50);comment:实例状态"`                                   // 实例状态
	ReplicateId       string `json:"replicateId" gorm:"type:varchar(100);comment:复制实例 ID"`                                    // 复制实例 ID

	// 多对多关系
	BindNodes []*TreeNode `json:"bind_nodes" gorm:"many2many:bind_rdss;comment:绑定的树节点列表"` // 绑定的树节点
}

type BindResourceReq struct {
	NodeId      int   `json:"nodId" `
	ResourceIds []int `json:"resource_ids" binding:"required,min=1"`
}

// StringList 封装了 []string 类型，用于与数据库中的逗号分隔字符串进行转换
type StringList []string

func (m *StringList) Scan(val interface{}) error {
	s := val.([]uint8)
	ss := strings.Split(string(s), "|")
	*m = ss
	return nil
}

func (m StringList) Value() (driver.Value, error) {
	str := strings.Join(m, "|")
	return str, nil
}
