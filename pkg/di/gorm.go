package di

import (
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// InitDB 初始化数据库
func InitDB() *gorm.DB {
	addr := viper.GetString("mysql.addr")
	db, err := gorm.Open(mysql.Open(addr), &gorm.Config{
		//Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		panic(err)
	}

	// 初始化表
	if err = InitTables(db); err != nil {
		panic(err)
	}

	return db
}
