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
			ID:       6,
			Name:     "权限管理",
			Path:     "/system_permission",
			ParentID: 1,
			Meta: model.MetaField{
				Icon:  "lucide:shield",
				Title: "权限管理",
			},
		},
		{
			ID:        7,
			Name:      "角色权限",
			Path:      "/system_role",
			Component: "/dashboard/SystemRole",
			ParentID:  6,
			Meta: model.MetaField{
				Icon:  "lucide:users",
				Title: "角色权限",
			},
		},
		{
			ID:        8,
			Name:      "用户权限",
			Path:      "/system_user_role",
			Component: "/dashboard/SystemUserRole",
			ParentID:  6,
			Meta: model.MetaField{
				Icon:  "lucide:user-cog",
				Title: "用户权限",
			},
		},
		{
			ID:        9,
			Name:      "ServiceTree",
			Path:      "/tree",
			Component: "BasicLayout",
			Meta: model.MetaField{
				Order: 1,
				Title: "page.serviceTree.title",
			},
		},
		{
			ID:        10,
			Name:      "服务树概览",
			Path:      "/tree_overview",
			Component: "/servicetree/TreeOverview",
			ParentID:  9,
			Meta: model.MetaField{
				Icon:  "material-symbols:overview",
				Title: "服务树概览",
			},
		},
		{
			ID:        11,
			Name:      "服务树节点管理",
			Path:      "/tree_node_manager",
			Component: "/servicetree/TreeNodeManager",
			ParentID:  9,
			Meta: model.MetaField{
				Icon:  "fluent-mdl2:task-manager",
				Title: "服务树节点管理",
			},
		},
		{
			ID:        12,
			Name:      "ECS管理",
			Path:      "/ecs_resource_operation",
			Component: "/servicetree/ECSResourceOperation",
			ParentID:  9,
			Meta: model.MetaField{
				Icon:  "mdi:cloud-cog-outline",
				Title: "ECS管理",
			},
		},
		{
			ID:        13,
			Name:      "Prometheus",
			Path:      "/prometheus",
			Component: "BasicLayout",
			Meta: model.MetaField{
				Order: 2,
				Title: "Promethues管理",
			},
		},
		{
			ID:        14,
			Name:      "MonitorScrapePool",
			Path:      "/monitor_pool",
			Component: "/promethues/MonitorScrapePool",
			ParentID:  13,
			Meta: model.MetaField{
				Icon:  "lucide:database",
				Title: "采集池",
			},
		},
		{
			ID:        15,
			Name:      "MonitorScrapeJob",
			Path:      "/monitor_job",
			Component: "/promethues/MonitorScrapeJob",
			ParentID:  13,
			Meta: model.MetaField{
				Icon:  "lucide:list-check",
				Title: "采集任务",
			},
		},
		{
			ID:        16,
			Name:      "MonitorAlert",
			Path:      "/monitor_alert",
			Component: "/promethues/MonitorAlert",
			ParentID:  13,
			Meta: model.MetaField{
				Icon:  "lucide:alert-triangle",
				Title: "alert告警池",
			},
		},
		{
			ID:        17,
			Name:      "MonitorAlertRule",
			Path:      "/monitor_alert_rule",
			Component: "/promethues/MonitorAlertRule",
			ParentID:  13,
			Meta: model.MetaField{
				Icon:  "lucide:badge-alert",
				Title: "告警规则",
			},
		},
		{
			ID:        18,
			Name:      "MonitorAlertEvent",
			Path:      "/monitor_alert_event",
			Component: "/promethues/MonitorAlertEvent",
			ParentID:  13,
			Meta: model.MetaField{
				Icon:  "lucide:bell-ring",
				Title: "告警事件",
			},
		},
		{
			ID:        19,
			Name:      "MonitorAlertRecord",
			Path:      "/monitor_alert_record",
			Component: "/promethues/MonitorAlertRecord",
			ParentID:  13,
			Meta: model.MetaField{
				Icon:  "lucide:box",
				Title: "预聚合",
			},
		},
		{
			ID:        20,
			Name:      "MonitorConfig",
			Path:      "/monitor_config",
			Component: "/promethues/MonitorConfig",
			ParentID:  13,
			Meta: model.MetaField{
				Icon:  "lucide:file-text",
				Title: "配置文件",
			},
		},
		{
			ID:        21,
			Name:      "MonitorOnDutyGroup",
			Path:      "/monitor_onduty_group",
			Component: "/promethues/MonitorOnDutyGroup",
			ParentID:  13,
			Meta: model.MetaField{
				Icon:  "lucide:user-round-minus",
				Title: "值班组",
			},
		},
		{
			ID:        22,
			Name:      "MonitorOnDutyGroupTable",
			Path:      "/monitor_onduty_group_table",
			Component: "/promethues/MonitorOndutyGroupTable",
			ParentID:  13,
			Meta: model.MetaField{
				Icon:       "material-symbols:table-outline",
				Title:      "排班表",
				HideInMenu: true,
			},
		},
		{
			ID:        23,
			Name:      "MonitorSend",
			Path:      "/monitor_send",
			Component: "/promethues/MonitorSend",
			ParentID:  13,
			Meta: model.MetaField{
				Icon:  "lucide:send-horizontal",
				Title: "发送组",
			},
		},
		{
			ID:        24,
			Name:      "K8s",
			Path:      "/k8s",
			Component: "BasicLayout",
			Meta: model.MetaField{
				Order: 3,
				Title: "k8s运维管理",
			},
		},
		{
			ID:        25,
			Name:      "K8sCluster",
			Path:      "/k8s_cluster",
			Component: "/k8s/K8sCluster",
			ParentID:  24,
			Meta: model.MetaField{
				Icon:  "lucide:database",
				Title: "集群管理",
			},
		},
		{
			ID:        26,
			Name:      "K8sNode",
			Path:      "/k8s_node",
			Component: "/k8s/K8sNode",
			ParentID:  24,
			Meta: model.MetaField{
				Icon:       "lucide:list-check",
				Title:      "节点管理",
				HideInMenu: true,
			},
		},
		{
			ID:        27,
			Name:      "K8sPod",
			Path:      "/k8s_pod",
			Component: "/k8s/K8sPod",
			ParentID:  24,
			Meta: model.MetaField{
				Icon:  "lucide:bell-ring",
				Title: "Pod管理",
			},
		},
		{
			ID:        28,
			Name:      "K8sService",
			Path:      "/k8s_service",
			Component: "/k8s/K8sService",
			ParentID:  24,
			Meta: model.MetaField{
				Icon:  "lucide:box",
				Title: "Service管理",
			},
		},
		{
			ID:        29,
			Name:      "K8sDeployment",
			Path:      "/k8s_deployment",
			Component: "/k8s/K8sDeployment",
			ParentID:  24,
			Meta: model.MetaField{
				Icon:  "lucide:file-text",
				Title: "Deployment管理",
			},
		},
		{
			ID:        30,
			Name:      "K8sConfigMap",
			Path:      "/k8s_configmap",
			Component: "/k8s/K8sConfigmap",
			ParentID:  24,
			Meta: model.MetaField{
				Icon:  "lucide:user-round-minus",
				Title: "ConfigMap管理",
			},
		},
		{
			ID:        31,
			Name:      "K8sYamlTemplate",
			Path:      "/k8s_yaml_template",
			Component: "/k8s/K8sYamlTemplate",
			ParentID:  24,
			Meta: model.MetaField{
				Icon:  "material-symbols:table-outline",
				Title: "Yaml模板",
			},
		},
		{
			ID:        32,
			Name:      "K8sYamlTask",
			Path:      "/k8s_yaml_task",
			Component: "/k8s/K8sYamlTask",
			ParentID:  24,
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
