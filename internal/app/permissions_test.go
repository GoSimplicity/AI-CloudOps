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
