package model

import (
	"gorm.io/gorm"
)

func InitTables(db *gorm.DB) error {
	return db.AutoMigrate(
		&User{},
		&Role{},
		&Menu{},
		&Api{},
	)
}
