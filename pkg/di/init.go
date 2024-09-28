package di

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
)

func InitTables(db *gorm.DB) error {
	return db.AutoMigrate(
		// auth
		&model.User{},
		&model.Role{},
		&model.Menu{},
		&model.Api{},

		// tree
		&model.TreeNode{},
		&model.ResourceEcs{},
		&model.ResourceElb{},
		&model.ResourceRds{},

		// k8s
		&model.K8sCluster{},
		&model.K8sProject{},
		&model.K8sApp{},
		&model.K8sYamlTask{},
		&model.K8sYamlTemplate{},
		&model.K8sInstance{},
		&model.K8sCronjob{},
		&model.K8sNode{},
		&model.K8sPod{},

		// prometheus
		&model.MonitorScrapePool{},
		&model.MonitorScrapeJob{},
		&model.MonitorAlertManagerPool{},
		&model.MonitorAlertRule{},
		&model.MonitorRecordRule{},
		&model.MonitorOnDutyHistory{},
		&model.MonitorOnDutyGroup{},
		&model.MonitorSendGroup{},
		&model.MonitorOnDutyChange{},
		&model.MonitorAlertEvent{},
	)
}
