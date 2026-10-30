//go:build integration

package service

import (
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	treeUtils "github.com/GoSimplicity/AI-CloudOps/internal/tree/utils"
	databaseLogger "github.com/GoSimplicity/AI-CloudOps/pkg/logger"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestMySQLProviderCredentialMigration(t *testing.T) {
	dsn := os.Getenv("CLOUDOPS_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("explicit isolated MySQL DSN required")
	}
	parsed, err := mysqlDriver.ParseDSN(dsn)
	if err != nil || !regexp.MustCompile(`^schema_[a-f0-9]{24}$`).MatchString(parsed.DBName) || parsed.User != parsed.DBName {
		t.Fatal("refusing database outside isolated fixture")
	}
	parsed.Timeout = 5 * time.Second
	parsed.ReadTimeout = 10 * time.Second
	parsed.WriteTimeout = 10 * time.Second
	db, err := gorm.Open(mysql.Open(parsed.FormatDSN()), &gorm.Config{Logger: databaseLogger.NewGORMLogger(zap.NewNop())})
	if err != nil {
		t.Fatal("isolated mysql connection failed")
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := db.AutoMigrate(&model.CloudAccount{}, &model.CloudAccountRegion{}, &model.TreeCloudResource{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Migrator().DropColumn(&model.CloudAccount{}, "ProviderCredentials"); err != nil {
		t.Fatal(err)
	}
	encryptionKey := strings.Repeat("k", 32)
	access, err := treeUtils.EncryptPassword("legacy-access-canary", encryptionKey)
	if err != nil {
		t.Fatal(err)
	}
	secret, err := treeUtils.EncryptPassword("legacy-secret-canary", encryptionKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO cl_tree_cloud_account (name,provider,access_key,secret_key,status,create_user_id,create_user_name,created_at,updated_at,deleted_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", "legacy-provider-fixture", 3, access, secret, 1, 1, "fixture", time.Now().UTC(), time.Now().UTC(), 0).Error; err != nil {
		t.Fatal("legacy insert failed")
	}
	if err := db.AutoMigrate(&model.CloudAccount{}, &model.CloudAccountRegion{}, &model.TreeCloudResource{}); err != nil {
		t.Fatal("provider migration failed")
	}
	var legacy model.CloudAccount
	if err := db.Where("name = ?", "legacy-provider-fixture").First(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	credentials, err := treeUtils.LoadCloudAccountCredentials(&legacy, encryptionKey)
	if err != nil || credentials.AccessKey != "legacy-access-canary" || credentials.SecretKey != "legacy-secret-canary" {
		t.Fatal("legacy credentials no longer readable")
	}
	assertProviderCredentials(t, db)
}
