package mock

import (
	"log"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
)

type MenuMock struct {
	db *gorm.DB
}

func NewMenuMock(db *gorm.DB) *MenuMock {
	return &MenuMock{
		db: db,
	}
}

func (m *MenuMock) InitMenu() {
	log.Println("[菜单Mock开始]")
	menus := []model.Menu{
		{
			ID:        1,
			Name:      "Dashboard",
			Path:      "/",
			Component: "BasicLayout",
			Meta: model.MetaField{
				Order: -1,
				Title: "page.dashboard.title",
			},
		},
		{
			ID:        2,
			Name:      "Welcome",
			Path:      "/system_welcome",
			Component: "/dashboard/SystemWelcome",
			ParentID:  1,
			Meta: model.MetaField{
				AffixTab: true,
				Icon:     "lucide:area-chart",
				Title:    "欢迎页",
			},
		},
		{
			ID:        3,
			Name:      "用户管理",
			Path:      "/system_user",
			Component: "/dashboard/SystemUser",
			ParentID:  1,
			Meta: model.MetaField{
				Icon:  "lucide:user",
				Title: "用户管理",
			},
		},
		{
			ID:        4,
			Name:      "菜单管理",
			Path:      "/system_menu",
			Component: "/dashboard/SystemMenu",
			ParentID:  1,
			Meta: model.MetaField{
				Icon:  "lucide:menu",
				Title: "菜单管理",
			},
		},
		{
			ID:        5,
			Name:      "接口管理",
			Path:      "/system_api",
			Component: "/dashboard/SystemApi",
			ParentID:  1,
			Meta: model.MetaField{
				Icon:  "lucide:zap",
				Title: "接口管理",
			},
		},
		{
			ID:        6,
			Name:      "角色权限",
			Path:      "/system_role",
			Component: "/dashboard/SystemRole",
			ParentID:  1,
			Meta: model.MetaField{
				Icon:  "lucide:users",
				Title: "角色权限",
			},
		},
		{
			ID:        7,
			Name:      "ServiceTree",
			Path:      "/tree",
			Component: "BasicLayout",
			Meta: model.MetaField{
				Order: 1,
				Title: "page.serviceTree.title",
			},
		},
		{
			ID:        8,
			Name:      "服务树概览",
			Path:      "/tree_overview",
			Component: "/servicetree/TreeOverview",
			ParentID:  7,
			Meta: model.MetaField{
				Icon:  "material-symbols:overview",
				Title: "服务树概览",
			},
		},
		{
			ID:        9,
			Name:      "服务树节点管理",
			Path:      "/tree_node_manager",
			Component: "/servicetree/TreeNodeManager",
			ParentID:  7,
			Meta: model.MetaField{
				Icon:  "fluent-mdl2:task-manager",
				Title: "服务树节点管理",
			},
		},
		{
			ID:        10,
			Name:      "ECS管理",
			Path:      "/ecs_resource_operation",
			Component: "/servicetree/ECSResourceOperation",
			ParentID:  7,
			Meta: model.MetaField{
				Icon:  "mdi:cloud-cog-outline",
				Title: "ECS管理",
			},
		},
		{
			ID: 31,
			Name: "终端管理",
			
		},
		{
			ID:        11,
			Name:      "Prometheus",
			Path:      "/prometheus",
			Component: "BasicLayout",
			Meta: model.MetaField{
				Order: 2,
				Title: "Promethues管理",
			},
		},
		{
			ID:        12,
			Name:      "MonitorScrapePool",
			Path:      "/monitor_pool",
			Component: "/promethues/MonitorScrapePool",
			ParentID:  11,
			Meta: model.MetaField{
				Icon:  "lucide:database",
				Title: "采集池",
			},
		},
		{
			ID:        13,
			Name:      "MonitorScrapeJob",
			Path:      "/monitor_job",
			Component: "/promethues/MonitorScrapeJob",
			ParentID:  11,
			Meta: model.MetaField{
				Icon:  "lucide:list-check",
				Title: "采集任务",
			},
		},
		{
			ID:        14,
			Name:      "MonitorAlert",
			Path:      "/monitor_alert",
			Component: "/promethues/MonitorAlert",
			ParentID:  11,
			Meta: model.MetaField{
				Icon:  "lucide:alert-triangle",
				Title: "alert告警池",
			},
		},
		{
			ID:        15,
			Name:      "MonitorAlertRule",
			Path:      "/monitor_alert_rule",
			Component: "/promethues/MonitorAlertRule",
			ParentID:  11,
			Meta: model.MetaField{
				Icon:  "lucide:badge-alert",
				Title: "告警规则",
			},
		},
		{
			ID:        16,
			Name:      "MonitorAlertEvent",
			Path:      "/monitor_alert_event",
			Component: "/promethues/MonitorAlertEvent",
			ParentID:  11,
			Meta: model.MetaField{
				Icon:  "lucide:bell-ring",
				Title: "告警事件",
			},
		},
		{
			ID:        17,
			Name:      "MonitorAlertRecord",
			Path:      "/monitor_alert_record",
			Component: "/promethues/MonitorAlertRecord",
			ParentID:  11,
			Meta: model.MetaField{
				Icon:  "lucide:box",
				Title: "预聚合",
			},
		},
		{
			ID:        18,
			Name:      "MonitorConfig",
			Path:      "/monitor_config",
			Component: "/promethues/MonitorConfig",
			ParentID:  11,
			Meta: model.MetaField{
				Icon:  "lucide:file-text",
				Title: "配置文件",
			},
		},
		{
			ID:        19,
			Name:      "MonitorOnDutyGroup",
			Path:      "/monitor_onduty_group",
			Component: "/promethues/MonitorOnDutyGroup",
			ParentID:  11,
			Meta: model.MetaField{
				Icon:  "lucide:user-round-minus",
				Title: "值班组",
			},
		},
		{
			ID:        20,
			Name:      "MonitorOnDutyGroupTable",
			Path:      "/monitor_onduty_group_table",
			Component: "/promethues/MonitorOndutyGroupTable",
			ParentID:  11,
			Meta: model.MetaField{
				Icon:       "material-symbols:table-outline",
				Title:      "排班表",
				HideInMenu: true,
			},
		},
		{
			ID:        21,
			Name:      "MonitorSend",
			Path:      "/monitor_send",
			Component: "/promethues/MonitorSend",
			ParentID:  11,
			Meta: model.MetaField{
				Icon:  "lucide:send-horizontal",
				Title: "发送组",
			},
		},
		{
			ID:        22,
			Name:      "K8s",
			Path:      "/k8s",
			Component: "BasicLayout",
			Meta: model.MetaField{
				Order: 3,
				Title: "k8s运维管理",
			},
		},
		{
			ID:        23,
			Name:      "K8sCluster",
			Path:      "/k8s_cluster",
			Component: "/k8s/K8sCluster",
			ParentID:  22,
			Meta: model.MetaField{
				Icon:  "lucide:database",
				Title: "集群管理",
			},
		},
		{
			ID:        24,
			Name:      "K8sNode",
			Path:      "/k8s_node",
			Component: "/k8s/K8sNode",
			ParentID:  22,
			Meta: model.MetaField{
				Icon:       "lucide:list-check",
				Title:      "节点管理",
				HideInMenu: true,
			},
		},
		{
			ID:        25,
			Name:      "K8sPod",
			Path:      "/k8s_pod",
			Component: "/k8s/K8sPod",
			ParentID:  22,
			Meta: model.MetaField{
				Icon:  "lucide:bell-ring",
				Title: "Pod管理",
			},
		},
		{
			ID:        26,
			Name:      "K8sService",
			Path:      "/k8s_service",
			Component: "/k8s/K8sService",
			ParentID:  22,
			Meta: model.MetaField{
				Icon:  "lucide:box",
				Title: "Service管理",
			},
		},
		{
			ID:        27,
			Name:      "K8sDeployment",
			Path:      "/k8s_deployment",
			Component: "/k8s/K8sDeployment",
			ParentID:  22,
			Meta: model.MetaField{
				Icon:  "lucide:file-text",
				Title: "Deployment管理",
			},
		},
		{
			ID:        28,
			Name:      "K8sConfigMap",
			Path:      "/k8s_configmap",
			Component: "/k8s/K8sConfigmap",
			ParentID:  22,
			Meta: model.MetaField{
				Icon:  "lucide:user-round-minus",
				Title: "ConfigMap管理",
			},
		},
		{
			ID:        29,
			Name:      "K8sYamlTemplate",
			Path:      "/k8s_yaml_template",
			Component: "/k8s/K8sYamlTemplate",
			ParentID:  22,
			Meta: model.MetaField{
				Icon:  "material-symbols:table-outline",
				Title: "Yaml模板",
			},
		},
		{
			ID:        30,
			Name:      "K8sYamlTask",
			Path:      "/k8s_yaml_task",
			Component: "/k8s/K8sYamlTask",
			ParentID:  22,
			Meta: model.MetaField{
				Icon:  "lucide:send-horizontal",
				Title: "Yaml任务",
			},
		},
	}

	for _, menu := range menus {
		// 使用FirstOrCreate方法,如果记录存在则跳过,不存在则创建
		result := m.db.Where("id = ?", menu.ID).FirstOrCreate(&menu)
		if result.Error != nil {
			log.Printf("创建菜单失败: %v", result.Error)
			continue
		}

		if result.RowsAffected == 1 {
			log.Printf("创建菜单 [%s] 成功", menu.Name)
		} else {
			log.Printf("菜单 [%s] 已存在,跳过创建", menu.Name)
		}
	}

	log.Println("[菜单Mock结束]")
}
