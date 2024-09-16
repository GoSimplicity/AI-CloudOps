package di

import (
	"github.com/GoSimplicity/CloudOps/internal/model"
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
	)
}
