/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package di

import (
	"fmt"
	"log"
	"time"

	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitDB 初始化数据库（带重试机制）
func InitDB() *gorm.DB {
	addr := viper.GetString("mysql.addr")

	// 数据库连接重试配置
	maxRetries := 5
	retryDelay := time.Second * 2

	var db *gorm.DB
	var err error

	log.Printf("正在连接数据库: %s", addr)

	// 重试连接数据库
	for i := 0; i < maxRetries; i++ {
		db, err = gorm.Open(mysql.Open(addr), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})

		if err == nil {
			log.Printf("数据库连接成功")
			break
		}

		log.Printf("数据库连接失败 (尝试 %d/%d): %v", i+1, maxRetries, err)

		if i < maxRetries-1 {
			log.Printf("等待 %v 后重试...", retryDelay)
			time.Sleep(retryDelay)
			// 增加重试延迟（指数退避）
			retryDelay *= 2
		}
	}

	// 如果所有重试都失败，记录错误但不panic
	if err != nil {
		log.Printf("数据库连接失败，已尝试 %d 次。错误: %v", maxRetries, err)
		log.Printf("程序将以降级模式运行，某些功能可能不可用")
		// 返回一个nil的数据库连接，让调用方处理
		return nil
	}

	// 测试数据库连接
	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("获取sql.DB失败: %v", err)
		return nil
	}

	if err := sqlDB.Ping(); err != nil {
		log.Printf("数据库ping失败: %v", err)
		return nil
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 初始化表
	if err = InitTables(db); err != nil {
		log.Printf("初始化数据库表失败: %v", err)
		// 不panic，让程序继续运行
		return db
	}

	log.Printf("数据库初始化完成")
	return db
}

// InitDBWithFallback 带降级处理的数据库初始化
func InitDBWithFallback() (*gorm.DB, error) {
	db := InitDB()
	if db == nil {
		return nil, fmt.Errorf("数据库连接失败")
	}
	return db, nil
}

// CheckDBHealth 检查数据库健康状态
func CheckDBHealth(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("数据库连接为空")
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取数据库连接失败: %v", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("数据库ping失败: %v", err)
	}

	return nil
}

// IsDBAvailable 检查数据库是否可用
func IsDBAvailable(db *gorm.DB) bool {
	return CheckDBHealth(db) == nil
}
