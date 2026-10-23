//go:build integration

package di

import (
	"context"
	"os"
	"regexp"
	"testing"
	"time"

	k8sDAO "github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	databaseLogger "github.com/GoSimplicity/AI-CloudOps/pkg/logger"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestMySQLStartupMigrationAndYAMLCRUD(t *testing.T) {
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
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	db = db.WithContext(ctx)
	if err := InitTables(db); err != nil {
		t.Fatal("full MySQL startup migration failed")
	}
	for name, table := range declaredPersistentModels(t) {
		if !db.Migrator().HasTable(table) {
			t.Errorf("MySQL missing model %s table %s", name, table)
		}
	}
	templates := k8sDAO.NewYamlTemplateDAO(db, zap.NewNop())
	tasks := k8sDAO.NewYamlTaskDAO(db, zap.NewNop())
	template := &model.K8sYamlTemplate{Name: "isolated-template", Content: "apiVersion: v1\nkind: ConfigMap", ClusterID: 42, UserID: 1}
	if err := templates.CreateYamlTemplate(ctx, template); err != nil {
		t.Fatal("template create failed")
	}
	task := &model.K8sYamlTask{Name: "isolated-task", TemplateID: template.ID, ClusterID: 42, UserID: 1, Variables: model.StringList{"owner=fixture"}, Status: "pending"}
	if err := tasks.CreateYamlTask(ctx, task); err != nil {
		t.Fatal("task create failed")
	}
	account := &model.CloudAccount{Name: "audit-fixture", Provider: model.ProviderAliyun, AccessKey: "fake-encrypted-access", SecretKey: "fake-encrypted-secret"}
	if err := db.Omit("Regions", "CloudResources").Create(account).Error; err != nil {
		t.Fatal("audit account fixture failed")
	}
	region := &model.CloudAccountRegion{CloudAccountID: account.ID, Region: "cn-fixture"}
	if err := db.Omit("CloudAccount").Create(region).Error; err != nil {
		t.Fatal("audit region fixture failed")
	}
	resource := &model.TreeCloudResource{Name: "audit-fixture", InstanceID: "audit-fixture", CloudAccountID: account.ID, CloudAccountRegionID: region.ID, ResourceType: 1}
	if err := db.Omit("CloudAccount", "CloudAccountRegion", "TreeNodes").Create(resource).Error; err != nil {
		t.Fatal("audit resource fixture failed")
	}
	// 模拟旧版本绕过 DAO 写入的私钥审计，并保留普通审计作为对照
	legacy := &model.CloudResourceChangeLog{ResourceID: resource.ID, FieldName: "key", OldValue: "old-sensitive-marker", NewValue: "new-sensitive-marker", ChangeTime: time.Now()}
	ordinary := &model.CloudResourceChangeLog{ResourceID: resource.ID, FieldName: "environment", NewValue: "local", ChangeTime: time.Now()}
	if err := db.Omit("CloudResource").Create(legacy).Error; err != nil {
		t.Fatal("legacy audit fixture failed")
	}
	if err := db.Omit("CloudResource").Create(ordinary).Error; err != nil {
		t.Fatal("ordinary audit fixture failed")
	}
	if err := InitTables(db); err != nil {
		t.Fatal("idempotent migration failed")
	}
	var scrubbed, retained model.CloudResourceChangeLog
	if err := db.First(&scrubbed, legacy.ID).Error; err != nil || scrubbed.NewValue != model.CredentialAuditRedaction || scrubbed.OldValue != model.CredentialAuditRedaction {
		t.Fatal("legacy credential audit not scrubbed")
	}
	if err := db.First(&retained, ordinary.ID).Error; err != nil || retained.NewValue != "local" {
		t.Fatal("ordinary audit changed during migration")
	}
	saved, err := tasks.GetYamlTaskByID(ctx, task.ID, 42)
	if err != nil || saved.TemplateID != template.ID || len(saved.Variables) != 1 || saved.Variables[0] != "owner=fixture" {
		t.Fatal("idempotent migration changed saved YAML task")
	}
	list, err := templates.ListAllYamlTemplates(ctx, &model.YamlTemplateListReq{ClusterID: 42, ListReq: model.ListReq{Page: 1, Size: 10}})
	if err != nil || len(list) != 1 {
		t.Fatal("template database list failed")
	}
	taskList, err := tasks.ListAllYamlTasks(ctx, &model.YamlTaskListReq{ClusterID: 42, ListReq: model.ListReq{Page: 1, Size: 10}})
	if err != nil || len(taskList) != 1 {
		t.Fatal("task database list failed")
	}
	template.Content = "apiVersion: v1\nkind: Secret"
	if err := templates.UpdateYamlTemplate(ctx, template); err != nil {
		t.Fatal("template update failed")
	}
	task.Name = "updated-task"
	if err := tasks.UpdateYamlTask(ctx, task); err != nil {
		t.Fatal("task update failed")
	}
	saved, err = tasks.GetYamlTaskByID(ctx, task.ID, 42)
	if err != nil || saved.Name != "updated-task" {
		t.Fatal("task update did not persist")
	}
	if err := tasks.DeleteYamlTask(ctx, task.ID, 42); err != nil {
		t.Fatal("task delete failed")
	}
	if err := templates.DeleteYamlTemplate(ctx, template.ID, 42); err != nil {
		t.Fatal("template delete failed")
	}
	taskList, err = tasks.ListAllYamlTasks(ctx, &model.YamlTaskListReq{ClusterID: 42})
	if err != nil || len(taskList) != 0 {
		t.Fatal("owned YAML cleanup failed")
	}
}
