//go:build smoke

package app

import (
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	workapi "github.com/GoSimplicity/AI-CloudOps/internal/workorder/api"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
)

func TestNotificationEventRoutesAreRegisteredInPermissionCatalog(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	defer pool.Close()
	if err = db.AutoMigrate(&model.Api{}); err != nil {
		t.Fatal(err)
	}
	engine := gin.New()
	workapi.NewNotificationHandler(nil).RegisterRouters(engine)
	app := &App{db: db, server: engine}
	if err = app.bootstrapPermissionCatalog(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		path   string
		method int
	}{{"/api/workorder/notification/events", 1}, {"/api/workorder/notification/events/:id/deliveries", 1}, {"/api/workorder/notification/events/:id/resolutions", 1}, {"/api/workorder/notification/events/:id/resolve", 2}, {"/api/workorder/notification/events/:id/retry", 2}, {model.WorkorderNotificationCancelScheduledRoute, 2}} {
		var count int64
		db.Model(&model.Api{}).Where("path = ? AND method = ?", item.path, item.method).Count(&count)
		if count != 1 {
			t.Fatalf("missing API grant %s", item.path)
		}
	}
}
