//go:build smoke

package app

import (
	"context"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPermissionCatalogIsIdempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	if err := db.AutoMigrate(&model.Api{}); err != nil {
		t.Fatal(err)
	}
	engine := gin.New()
	handler := func(c *gin.Context) { c.Status(200) }
	engine.GET("/api/user/login", handler)
	engine.GET("/api/item/:id", handler)
	engine.POST("/api/item", handler)
	a := &App{db: db, server: engine}
	if err := a.bootstrapPermissionCatalog(context.Background()); err != nil {
		t.Fatal(err)
	}
	var rows []model.Api
	db.Find(&rows)
	if len(rows) != 2 {
		t.Fatalf("catalog count %d", len(rows))
	}
	db.Model(&model.Api{}).Where("id = ?", rows[0].ID).Update("description", "custom metadata")
	if err := a.bootstrapPermissionCatalog(context.Background()); err != nil {
		t.Fatal(err)
	}
	var count int64
	db.Model(&model.Api{}).Count(&count)
	var first model.Api
	db.First(&first, rows[0].ID)
	if count != 2 || first.Description != "custom metadata" {
		t.Fatal("bootstrap duplicated or overwrote existing permission")
	}
}

func TestAIProxyCatalogSupportsSeparateAssistantAndWorkflowGrants(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	pool.SetMaxOpenConns(1)
	defer pool.Close()
	if err := db.AutoMigrate(&model.Api{}); err != nil {
		t.Fatal(err)
	}
	engine := gin.New()
	engine.Any("/api/v1/*path", func(c *gin.Context) { c.Status(204) })
	app := &App{db: db, server: engine}
	for i := 0; i < 2; i++ {
		if err := app.bootstrapPermissionCatalog(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	for _, grant := range []struct {
		path   string
		method int8
	}{
		{"/api/v1/assistant/query", 2},
		{"/api/v1/assistant/session/:session_id", 1},
		{"/api/v1/autofix/workflow/confirm", 2},
		{"/api/v1/assistant/stream", 1},
	} {
		var count int64
		db.Model(&model.Api{}).Where("path = ? AND method = ?", grant.path, grant.method).Count(&count)
		if count != 1 {
			t.Errorf("缺少可单独授权的AI接口: %s (%d)", grant.path, grant.method)
		}
	}
}

func TestPermissionCatalogNamesSecretReveal(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	pool.SetMaxOpenConns(1)
	defer pool.Close()
	if err := db.AutoMigrate(&model.Api{}); err != nil {
		t.Fatal(err)
	}
	engine := gin.New()
	engine.GET(model.K8sSecretRevealRoute, func(c *gin.Context) { c.Status(200) })
	app := &App{db: db, server: engine}
	if err := app.bootstrapPermissionCatalog(context.Background()); err != nil {
		t.Fatal(err)
	}
	var row model.Api
	if err := db.Where("path = ? AND method = ?", model.K8sSecretRevealRoute, 1).First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Name != model.K8sSecretRevealPermissionName {
		t.Fatalf("权限名=%q", row.Name)
	}
}
