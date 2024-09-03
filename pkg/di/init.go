package di

import (
	"github.com/GoSimplicity/CloudOps/internal/model"
	"gorm.io/gorm"
)

func InitTables(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.User{},
		&model.Role{},
		&model.Menu{},
		&model.Api{},
	)
}
