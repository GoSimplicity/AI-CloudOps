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
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	databaseLogger "github.com/GoSimplicity/AI-CloudOps/pkg/logger"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitDB(cfg *config.Config, logger *zap.Logger) (*gorm.DB, error) {
	if cfg == nil {
		return nil, fmt.Errorf("配置不能为空")
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	settings := cfg.MySQL.WithDefaults()
	if err := settings.Validate(); err != nil {
		return nil, err
	}
	zone, err := notificationTimeZone(settings.Addr)
	if err != nil {
		return nil, err
	}
	connector, err := mysqlConnector(settings)
	if err != nil {
		return nil, err
	}
	sqlDB := sql.OpenDB(connector)
	configureSQLPool(sqlDB, settings)
	ctx, cancel := context.WithTimeout(context.Background(), settings.StartupTimeout)
	defer cancel()
	db, err := initializeSQLPool(ctx, sqlDB, func(ctx context.Context, pool *sql.DB) (*gorm.DB, error) {
		// 保留驱动版本探测，覆盖其内部Background上下文
		db, err := gorm.Open(mysql.New(mysql.Config{Conn: &startupPool{DB: pool, ctx: ctx}}), &gorm.Config{DisableAutomaticPing: true, Logger: databaseLogger.NewGORMLogger(logger)})
		if err != nil {
			logger.Error("数据库ORM初始化失败", zap.String("error_type", databaseLogger.DatabaseErrorType(err)))
			return nil, databaseLogger.WrapDatabaseError("数据库ORM初始化失败", err)
		}
		db.ConnPool = pool
		db.Statement.ConnPool = pool
		if err := CheckNotificationTimeZone(db.WithContext(ctx), zone); err != nil {
			return nil, err
		}
		// 保持迁移逻辑，使用同一启动期限限制查询
		if err := InitTables(db.WithContext(ctx)); err != nil {
			logger.Error("数据库表初始化失败", zap.String("error_type", databaseLogger.DatabaseErrorType(err)))
			return nil, databaseLogger.WrapDatabaseError("数据库表初始化失败", err)
		}
		return db, nil
	})
	if err != nil {
		logger.Error("数据库初始化失败")
		return nil, err
	}
	logger.Info("数据库初始化成功")
	return db, nil
}

func mysqlConnector(settings config.MySQLConfig) (driver.Connector, error) {
	parsed, err := mysqlDSNConfig(settings)
	if err != nil {
		return nil, err
	}
	connector, err := mysqlDriver.NewConnector(parsed)
	if err != nil {
		return nil, fmt.Errorf("MySQL连接配置无效")
	}
	return connector, nil
}

func mysqlDSNConfig(settings config.MySQLConfig) (*mysqlDriver.Config, error) {
	parsed, err := mysqlDriver.ParseDSN(settings.Addr)
	if err != nil {
		return nil, fmt.Errorf("MySQL DSN配置无效")
	}
	query := ""
	// 与驱动一致，仅在最后一个斜线后的数据库名区域查找查询参数
	databasePart := settings.Addr[strings.LastIndex(settings.Addr, "/")+1:]
	if _, suffix, found := strings.Cut(databasePart, "?"); found {
		query = suffix
	}
	explicitTimeout := false
	for _, parameter := range strings.Split(query, "&") {
		key, _, found := strings.Cut(parameter, "=")
		if found && key == "timeout" {
			explicitTimeout = true
		}
	}
	if !explicitTimeout {
		parsed.Timeout = settings.ConnectTimeout
	}
	return parsed, nil
}

func configureSQLPool(pool *sql.DB, settings config.MySQLConfig) {
	pool.SetMaxOpenConns(settings.MaxOpenConns)
	pool.SetMaxIdleConns(settings.MaxIdleConns)
	pool.SetConnMaxLifetime(settings.ConnMaxLifetime)
	pool.SetConnMaxIdleTime(settings.ConnMaxIdleTime)
}

// startupPool 仅在GORM初始化阶段将版本查询纳入启动期限
type startupPool struct {
	*sql.DB
	ctx context.Context
}

func (p *startupPool) QueryRowContext(_ context.Context, query string, args ...interface{}) *sql.Row {
	return p.DB.QueryRowContext(p.ctx, query, args...)
}

func initializeSQLPool(ctx context.Context, pool *sql.DB, initialize func(context.Context, *sql.DB) (*gorm.DB, error)) (db *gorm.DB, err error) {
	defer func() {
		if err != nil {
			_ = pool.Close()
		}
	}()
	if err = pool.PingContext(ctx); err != nil {
		return nil, databaseLogger.WrapDatabaseError("数据库启动连接检查失败", err)
	}
	return initialize(ctx, pool)
}

func CheckDBHealth(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("数据库连接为空")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取数据库连接失败: %v", err)
	}

	var pingErr error
	for i := 0; i < 3; i++ {
		pingErr = sqlDB.Ping()
		if pingErr == nil {
			return nil
		}
		if i < 2 { // 只在前两次失败后等待
			time.Sleep(10 * time.Second)
		}
	}

	if pingErr != nil {
		return fmt.Errorf("数据库ping失败: %v", pingErr)
	}
	return nil
}

func IsDBAvailable(db *gorm.DB) bool {
	return CheckDBHealth(db) == nil
}
