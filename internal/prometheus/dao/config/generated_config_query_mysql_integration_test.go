//go:build integration

package config

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	driver "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Opt-in only: provision a NEW EMPTY loopback schema and confirm its exact name
// separately. This fixture creates/drops only its own cl_monitor_configs table.
func TestGeneratedConfigQueriesNativeMySQL(t *testing.T) {
	path := os.Getenv("CLOUDOPS_CONFIG_QUERY_MYSQL_DSN_FILE")
	if path == "" {
		t.Skip("private native MySQL fixture not configured")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > 16384 {
		t.Fatal("fixture requires a small owner-only regular DSN file")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal("fixture DSN unreadable")
	}
	cfg, err := driver.ParseDSN(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatal("fixture DSN invalid")
	}
	if cfg.Net != "tcp" || cfg.Addr != "127.0.0.1:13306" || !regexp.MustCompile(`^cloudops_config_query_[a-z0-9_]+$`).MatchString(cfg.DBName) || cfg.DBName != os.Getenv("CLOUDOPS_CONFIG_QUERY_MYSQL_DATABASE") {
		t.Fatal("fixture requires loopback and separately confirmed private schema")
	}
	cfg.ParseTime = true
	db, err := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal("fixture connection failed")
	}
	conn, err := db.DB()
	if err != nil {
		t.Fatal("fixture pool unavailable")
	}
	defer conn.Close()
	var count int64
	if err := db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ?", cfg.DBName).Scan(&count).Error; err != nil || count != 0 {
		t.Fatal("fixture schema must be empty")
	}
	if err := db.AutoMigrate(&model.MonitorConfig{}); err != nil {
		t.Fatal("fixture migration failed")
	}
	defer func() {
		if err := db.Migrator().DropTable(&model.MonitorConfig{}); err != nil {
			t.Error("fixture cleanup failed")
		}
	}()
	// A PAD SPACE, case-insensitive column deliberately stresses both byte alias
	// cases. The private query must override it; the ordinary IN API must not.
	if err := db.Exec("ALTER TABLE cl_monitor_configs MODIFY instance_ip VARCHAR(45) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL").Error; err != nil {
		t.Fatal("fixture weak-collation setup failed")
	}
	dao := NewMonitorConfigDAO(zap.NewNop(), db).(*monitorConfigDAO)
	exerciseGeneratedConfigQueries(t, dao, db)
}
