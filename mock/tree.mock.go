package mock

import (
	"log"
	"strconv"
	"time"

	"github.com/GoSimplicity/CloudOps/internal/model"
	"gorm.io/gorm"
)

type TreeMock struct {
	db *gorm.DB
}

func NewTreeMock(db *gorm.DB) *TreeMock {
	return &TreeMock{
		db: db,
	}
}

func (t *TreeMock) CreateTreeMock() {
	log.Print("[Tree模块Mock开始]")

	t.createTreeNodeMock()
	t.createEcsMock()
	t.createElbMock()
	t.createRdsMock()

	log.Print("[Tree模块Mock结束]")
}

func (t *TreeMock) createTreeNodeMock() {
	log.Print("[Mock some TreeNodes]")

	// 生成节点的辅助函数
	createNode := func(title string, pid int, level int, isLeaf bool, desc string, user *model.User) (*model.TreeNode, error) {
		node := model.TreeNode{
			Title:  title,
			Pid:    pid,
			Level:  level,
			IsLeaf: isLeaf,
			Desc:   desc,

			OpsAdmins: []*model.User{user},
			RdAdmins:  []*model.User{user},
			RdMembers: []*model.User{user},
		}
		result := t.db.Where("title = ?", node.Title).FirstOrCreate(&node)
		if result.Error != nil {
			return nil, result.Error
		}
		return &node, nil
	}

	// 添加管理员用户为节点负责人
	var admin *model.User
	result := t.db.Where("username = ?", "admin").First(&admin)
	if result.Error != nil {
		log.Printf("获取管理员用户失败: %v\n", result.Error)
		log.Print("[Tree模块Mock结束]")
		return
	}

	// 生成 level = 1 的根节点
	rootNode, err := createNode("Tencent", 0, 1, false, "Tencent yyds", admin)
	if err != nil {
		log.Printf("创建根节点失败: %v\n", err)
		log.Print("[Tree模块Mock结束]")
		return
	}

	// 生成 level = 2 的子节点
	secondNode, err := createNode("WXG", rootNode.ID, 2, false, "", admin)
	if err != nil {
		log.Printf("创建二级节点失败: %v\n", err)
		log.Print("[Tree模块Mock结束]")
		return
	}

	_, err = createNode("CDG", rootNode.ID, 2, false, "", admin)
	if err != nil {
		log.Printf("创建二级节点失败: %v\n", err)
		log.Print("[Tree模块Mock结束]")
		return
	}

	// 生成 level = 3 的子节点
	thirdNode, err := createNode("前端组", secondNode.ID, 3, false, "", admin)
	if err != nil {
		log.Printf("创建三级节点失败: %v\n", err)
		log.Print("[Tree模块Mock结束]")
		return
	}

	_, err = createNode("后端组", secondNode.ID, 3, false, "", admin)
	if err != nil {
		log.Printf("创建三级节点失败: %v\n", err)
		log.Print("[Tree模块Mock结束]")
		return
	}

	// 生成 level = 4 的子节点
	_, err = createNode("好看的前端项目-1", thirdNode.ID, 4, true, "", admin)
	if err != nil {
		log.Printf("创建四级节点失败: %v\n", err)
		log.Print("[Tree模块Mock结束]")
		return
	}

	_, err = createNode("好看的前端项目-2", thirdNode.ID, 4, true, "", admin)
	if err != nil {
		log.Printf("创建四级节点失败: %v\n", err)
		log.Print("[Tree模块Mock结束]")
		return
	}

}

func MockResouceTree(name string) model.ResourceTree {
	return model.ResourceTree{
		InstanceName:     name,
		Hash:             name,
		Vendor:           "阿里云",
		CreateByOrder:    true,
		VpcId:            "vpc-123456",
		ZoneId:           "cn-hangzhou-g",
		Env:              "prod",
		PayType:          "按量付费",
		Status:           "Running",
		Description:      "CentOS 7.4 操作系统",
		Tags:             model.StringList{Items: []string{"tag1", "tag2"}},
		SecurityGroupIds: model.StringList{Items: []string{"sg-123456", "sg-654321"}},
		PrivateIpAddress: model.StringList{Items: []string{"192.168.0.1", "192.168.0.2"}},
		PublicIpAddress:  model.StringList{Items: []string{"1.2.3.4", "5.6.7.8"}},
		IpAddr:           "1.2.3.4",
		CreationTime:     time.Now().Format(time.RFC3339),
		Key:              "example-key",
	}
}

func (t *TreeMock) createEcsMock() {
	log.Print("[Mock some ECS]")

	// 清空 resource_ecs、bind_ecss 表
	t.db.Exec("DELETE FROM bind_ecss")
	t.db.Exec("DELETE FROM resource_ecs")

	// 生成 ECS 的辅助函数
	createEcs := func(node *model.TreeNode, cnt string) (*model.ResourceEcs, error) {
		ecs := model.ResourceEcs{
			ResourceTree: MockResouceTree("ECS" + cnt),

			OsType:            "linux",
			VmType:            1,
			InstanceType:      "ecs.g8a.2xlarge",
			Cpu:               4,
			Memory:            16,
			Disk:              100,
			OSName:            "CentOS 7.4 64 位",
			ImageId:           "img-123456",
			Hostname:          "example-hostname",
			NetworkInterfaces: model.StringList{Items: []string{"eni-123456", "eni-654321"}},
			DiskIds:           model.StringList{Items: []string{"disk-123456", "disk-654321"}},

			BindNodes: []*model.TreeNode{node},
		}
		result := t.db.Where("instance_name = ?", ecs.ResourceTree.InstanceName).FirstOrCreate(&ecs)
		if result.Error != nil {
			return nil, result.Error
		}
		return &ecs, nil
	}

	// 获取子节点，绑定ECS资源
	var nodes []*model.TreeNode
	result := t.db.Where("is_leaf = ?", 1).Find(&nodes)
	if result.Error != nil {
		log.Printf("获取子节点失败: %v\n", result.Error)
		log.Print("[Tree模块Mock结束]")
		return
	}

	num := 24
	for i := 1; i <= num; i++ {
		cnt := strconv.Itoa(i)
		_, err := createEcs(nodes[i%len(nodes)], "-"+cnt)
		if err != nil {
			log.Printf("创建 ECS 失败: %v\n", err)
			log.Print("[Tree模块Mock结束]")
			return
		}
	}

}

func (t *TreeMock) createElbMock() {
	log.Print("[Mock some ELB]")

	// 清空 resource_elbs、bind_elbs 表
	t.db.Exec("DELETE FROM bind_elbs")
	t.db.Exec("DELETE FROM resource_elbs")

	createElb := func(node *model.TreeNode, cnt string) (*model.ResourceElb, error) {
		elb := model.ResourceElb{
			ResourceTree: MockResouceTree("ELB" + cnt),

			LoadBalancerType:   "nlb",
			BandwidthCapacity:  100,
			AddressType:        "intranet",
			DNSName:            "example-dns",
			BandwidthPackageId: "bwp-123456",
			CrossZoneEnabled:   true,

			BindNodes: []*model.TreeNode{node},
		}
		result := t.db.Where("instance_name = ?", elb.ResourceTree.InstanceName).FirstOrCreate(&elb)
		if result.Error != nil {
			return nil, result.Error
		}
		return &elb, nil
	}

	// 获取子节点，绑定ELB资源
	var nodes []*model.TreeNode
	result := t.db.Where("is_leaf = ?", 1).Find(&nodes)
	if result.Error != nil {
		log.Printf("获取子节点失败: %v\n", result.Error)
		log.Print("[Tree模块Mock结束]")
		return
	}

	num := 24
	for i := 1; i <= num; i++ {
		cnt := strconv.Itoa(i)
		_, err := createElb(nodes[i%len(nodes)], "-"+cnt)
		if err != nil {
			log.Printf("创建 ELB 失败: %v\n", err)
			log.Print("[Tree模块Mock结束]")
			return
		}
	}
}

func (t *TreeMock) createRdsMock() {
	log.Print("[Mock some RDS]")

	// 清空 resource_rds、bind_rdss 表
	t.db.Exec("DELETE FROM bind_rdss")
	t.db.Exec("DELETE FROM resource_rds")

	createRds := func(node *model.TreeNode, cnt string) (*model.ResourceRds, error) {
		rds := model.ResourceRds{
			ResourceTree: MockResouceTree("RDS" + cnt),

			Engine:            "MySQL",
			DBInstanceNetType: "Internet",
			DBInstanceClass:   "rds.mysql.s1.small",
			DBInstanceType:    "SSD",
			EngineVersion:     "5.7",
			MasterInstanceId:  "rds-123456",
			DBInstanceStatus:  "Running",
			ReplicateId:       "rds-654321",

			BindNodes: []*model.TreeNode{node},
		}
		result := t.db.Where("instance_name = ?", rds.ResourceTree.InstanceName).FirstOrCreate(&rds)
		if result.Error != nil {
			return nil, result.Error
		}
		return &rds, nil
	}

	// 获取子节点，绑定RDS资源
	var nodes []*model.TreeNode
	result := t.db.Where("is_leaf = ?", 1).Find(&nodes)
	if result.Error != nil {
		log.Printf("获取子节点失败: %v\n", result.Error)
		log.Print("[Tree模块Mock结束]")
		return
	}

	num := 24
	for i := 1; i <= num; i++ {
		cnt := strconv.Itoa(i)
		_, err := createRds(nodes[i%len(nodes)], "-"+cnt)
		if err != nil {
			log.Printf("创建 RDS 失败: %v\n", err)
			log.Print("[Tree模块Mock结束]")
			return
		}
	}
}
