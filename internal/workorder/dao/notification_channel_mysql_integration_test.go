//go:build integration

package dao

import (
	"os"
	"regexp"
	"testing"
	"time"

	databaseLogger "github.com/GoSimplicity/AI-CloudOps/pkg/logger"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestMySQLNotificationChannelPagination(t *testing.T) {
	dsn := os.Getenv("CLOUDOPS_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("explicit isolated MySQL DSN required")
	}
	parsed, err := mysqlDriver.ParseDSN(dsn)
	if err != nil || !regexp.MustCompile(`^schema_[a-f0-9]{24}$`).MatchString(parsed.DBName) || parsed.User != parsed.DBName {
		t.Fatal("refusing database outside generated isolated fixture")
	}
	parsed.Timeout = 5 * time.Second
	parsed.ReadTimeout = 10 * time.Second
	parsed.WriteTimeout = 10 * time.Second
	db, err := gorm.Open(mysql.Open(parsed.FormatDSN()), &gorm.Config{Logger: databaseLogger.NewGORMLogger(zap.NewNop())})
	if err != nil {
		t.Fatal("isolated MySQL connection failed")
	}
	pool, _ := db.DB()
	pool.SetMaxOpenConns(2)
	defer pool.Close()
	assertNotificationChannelPagination(t, db)
	assertNotificationFieldPresence(t, db)
}
