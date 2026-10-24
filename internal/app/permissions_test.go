//go:build smoke

package app

import (
	"context"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
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

func TestPermissionCatalogNamesServiceAccountToken(t *testing.T) {
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
	handler := func(c *gin.Context) { c.Status(200) }
	engine.GET(model.K8sServiceAccountTokenRoute, handler)
	engine.POST(model.K8sServiceAccountTokenRoute, handler)
	app := &App{db: db, server: engine}
	for i := 0; i < 2; i++ {
		if err := app.bootstrapPermissionCatalog(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	for method, want := range map[int8]string{1: model.K8sServiceAccountTokenPermissionName, 2: model.K8sServiceAccountTokenCreatePermissionName} {
		var rows []model.Api
		db.Where("path = ? AND method = ?", model.K8sServiceAccountTokenRoute, method).Find(&rows)
		if len(rows) != 1 || rows[0].Name != want {
			t.Fatalf("method=%d 权限目录不符: %+v", method, rows)
		}
	}
}

// 升级场景：SEC-29 之前 token 路由已按自动名称登记过；自动名称改为权限码，管理员改过的名称保持不变
func TestPermissionCatalogRenamesAutoNamedServiceAccountToken(t *testing.T) {
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
	handler := func(c *gin.Context) { c.Status(200) }
	engine.GET(model.K8sServiceAccountTokenRoute, handler)
	engine.POST(model.K8sServiceAccountTokenRoute, handler)
	var getRoute gin.RouteInfo
	for _, r := range engine.Routes() {
		if r.Method == "GET" {
			getRoute = r
		}
	}
	legacy := []model.Api{
		{Name: autoPermissionName(getRoute), Path: model.K8sServiceAccountTokenRoute, Method: 1, Description: "GET " + model.K8sServiceAccountTokenRoute, Version: "v1", Category: 2},
		{Name: "admin custom token name", Path: model.K8sServiceAccountTokenRoute, Method: 2, Description: "custom metadata", Version: "v1", Category: 2},
	}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	app := &App{db: db, server: engine}
	if err := app.bootstrapPermissionCatalog(context.Background()); err != nil {
		t.Fatal(err)
	}
	var count int64
	db.Model(&model.Api{}).Count(&count)
	var get, post model.Api
	db.First(&get, legacy[0].ID)
	db.First(&post, legacy[1].ID)
	if count != 2 || get.Name != model.K8sServiceAccountTokenPermissionName || post.Name != "admin custom token name" || post.Description != "custom metadata" {
		t.Fatalf("升级后权限目录不符: count=%d get=%q post=%q/%q", count, get.Name, post.Name, post.Description)
	}
	if get.Description != namedPermissions["GET "+model.K8sServiceAccountTokenRoute].Description {
		t.Fatalf("自动命名的行应同时更新为登记的说明: %q", get.Description)
	}
}

func newPermissionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { pool.Close() })
	if err := db.AutoMigrate(&model.Api{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func newTokenRouteEngine() (*gin.Engine, map[string]gin.RouteInfo) {
	engine := gin.New()
	handler := func(c *gin.Context) { c.Status(200) }
	engine.GET(model.K8sServiceAccountTokenRoute, handler)
	engine.POST(model.K8sServiceAccountTokenRoute, handler)
	routes := map[string]gin.RouteInfo{}
	for _, r := range engine.Routes() {
		routes[r.Method] = r
	}
	return engine, routes
}

// SEC-29 P2：管理员只改了说明（名称仍是自动名）时，启动迁移不能覆盖该行的名称和说明
func TestPermissionCatalogKeepsAdminEditedDescription(t *testing.T) {
	db := newPermissionTestDB(t)
	engine, routes := newTokenRouteEngine()
	legacy := []model.Api{
		// 管理员只改了说明
		{Name: autoPermissionName(routes["GET"]), Path: model.K8sServiceAccountTokenRoute, Method: 1, Description: "运维自定义说明", Version: "v1", Category: 2},
		// 完全是自动生成的行，应被改名
		{Name: autoPermissionName(routes["POST"]), Path: model.K8sServiceAccountTokenRoute, Method: 2, Description: "POST " + model.K8sServiceAccountTokenRoute, Version: "v1", Category: 2},
	}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	app := &App{db: db, server: engine, logger: zap.NewNop()}
	for i := 0; i < 2; i++ {
		if err := app.bootstrapPermissionCatalog(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	var get, post model.Api
	db.First(&get, legacy[0].ID)
	db.First(&post, legacy[1].ID)
	if get.Name != autoPermissionName(routes["GET"]) || get.Description != "运维自定义说明" {
		t.Fatalf("管理员改过说明的行不应被覆盖: %q/%q", get.Name, get.Description)
	}
	wantPost := namedPermissions["POST "+model.K8sServiceAccountTokenRoute]
	if post.Name != wantPost.Name || post.Description != wantPost.Description {
		t.Fatalf("自动命名的行应改为权限码与登记说明: %q/%q", post.Name, post.Description)
	}
	var count int64
	db.Model(&model.Api{}).Count(&count)
	if count != 2 {
		t.Fatalf("不应新增行: %d", count)
	}
}

// SEC-29 P3-1：权限码被其他行占用导致 OnConflict DoNothing 静默跳过时，记 Warn（只含权限码）
func TestPermissionCatalogWarnsWhenNamedInsertSkipped(t *testing.T) {
	db := newPermissionTestDB(t)
	engine, _ := newTokenRouteEngine()
	occupant := model.Api{Name: model.K8sServiceAccountTokenPermissionName, Path: "/api/other/occupant", Method: 1, Description: "occupant", Version: "v1", Category: 2}
	if err := db.Create(&occupant).Error; err != nil {
		t.Fatal(err)
	}
	core, logs := observer.New(zapcore.InfoLevel)
	app := &App{db: db, server: engine, logger: zap.New(core)}
	if err := app.bootstrapPermissionCatalog(context.Background()); err != nil {
		t.Fatal(err)
	}
	var rows []model.Api
	db.Where("path = ?", model.K8sServiceAccountTokenRoute).Find(&rows)
	if len(rows) != 1 || rows[0].Name != model.K8sServiceAccountTokenCreatePermissionName {
		t.Fatalf("前提：GET 行应因权限码冲突被跳过，只插入 POST 行: %+v", rows)
	}
	warns := logs.FilterLevelExact(zapcore.WarnLevel).All()
	if len(warns) != 1 {
		t.Fatalf("应只记一条 Warn: %+v", warns)
	}
	fields := warns[0].ContextMap()
	if len(fields) != 1 || fields["permission"] != model.K8sServiceAccountTokenPermissionName {
		t.Fatalf("Warn 只应包含缺失的权限码: %+v", fields)
	}

	// 全部登记成功时不记 Warn
	db2 := newPermissionTestDB(t)
	core2, logs2 := observer.New(zapcore.InfoLevel)
	app2 := &App{db: db2, server: engine, logger: zap.New(core2)}
	if err := app2.bootstrapPermissionCatalog(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := logs2.FilterLevelExact(zapcore.WarnLevel).Len(); n != 0 {
		t.Fatalf("权限码均已登记时不应告警: %d", n)
	}
}
