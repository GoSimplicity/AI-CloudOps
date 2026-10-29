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
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestMySQLTemporaryCloudCredentials(t *testing.T) {
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
	// 生产连接允许命名时区，验证凭据到期值按真实时刻读取而非当作 UTC 墙钟。
	parsed.Loc, err = time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(mysql.Open(parsed.FormatDSN()), &gorm.Config{Logger: databaseLogger.NewGORMLogger(zap.NewNop())})
	if err != nil {
		t.Fatal("isolated MySQL connection failed")
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := db.AutoMigrate(&model.CloudAccount{}, &model.CloudAccountRegion{}, &model.TreeCloudResource{}); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"CredentialType", "SessionToken", "CredentialExpiresAt", "CredentialVersion"} {
		if err := db.Migrator().DropColumn(&model.CloudAccount{}, field); err != nil {
			t.Fatal(err)
		}
	}
	oldKey := viper.Get("tree.password_encryption_key")
	viper.Set("tree.password_encryption_key", strings.Repeat("k", 32))
	defer viper.Set("tree.password_encryption_key", oldKey)
	access, err := treeUtils.EncryptPassword("legacy-access-fixture")
	if err != nil {
		t.Fatal(err)
	}
	secret, err := treeUtils.EncryptPassword("legacy-secret-fixture")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO cl_tree_cloud_account (name, provider, access_key, secret_key, status, create_user_id, create_user_name, created_at, updated_at, deleted_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", "legacy-migration-fixture", 3, access, secret, 1, 1, "fixture", time.Now().UTC(), time.Now().UTC(), 0).Error; err != nil {
		t.Fatal("legacy insert failed")
	}
	if err := db.AutoMigrate(&model.CloudAccount{}, &model.CloudAccountRegion{}, &model.TreeCloudResource{}); err != nil {
		t.Fatal("legacy schema migration failed")
	}
	var legacy model.CloudAccount
	if err := db.Where("name = ?", "legacy-migration-fixture").First(&legacy).Error; err != nil {
		t.Fatal("legacy record could not be read")
	}
	credential, err := treeUtils.LoadCloudAccountCredentials(&legacy)
	if err != nil || credential.AccessKey != "legacy-access-fixture" || credential.SecretKey != "legacy-secret-fixture" || credential.SessionToken != "" {
		t.Fatal("legacy ciphertext no longer works")
	}
	assertTemporaryCredentials(t, db)
}
