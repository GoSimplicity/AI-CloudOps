package app

import (
	"context"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestInitMockDataRefusedInProduction(t *testing.T) {
	t.Setenv("ENV", "production")
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	defer pool.Close()
	err = (&App{db: db}).initMockData(context.Background())
	if err == nil || !strings.Contains(err.Error(), "生产环境") {
		t.Fatalf("err=%v", err)
	}
	if tables, _ := db.Migrator().GetTables(); len(tables) != 0 {
		t.Fatalf("生产环境不应写入任何 mock 数据: %v", tables)
	}
}
