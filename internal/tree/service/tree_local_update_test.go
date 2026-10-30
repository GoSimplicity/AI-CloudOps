package service

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	databaseLogger "github.com/GoSimplicity/AI-CloudOps/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type rotatedLocalDAO struct {
	dao.TreeLocalDAO
	db *gorm.DB
}

func (d rotatedLocalDAO) GetByID(ctx context.Context, id int) (*model.TreeLocalResource, error) {
	local, err := d.TreeLocalDAO.GetByID(ctx, id)
	if err == nil {
		err = d.db.Model(&model.TreeLocalResource{}).Where("id = ?", id).Update("key", "newer-private-key").Error
	}
	return local, err
}

func TestLocalMetadataUpdatePreservesCredentialsAndRuntimeFields(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "local.db")), &gorm.Config{Logger: databaseLogger.NewGORMLogger(zap.NewNop())})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	defer pool.Close()
	if err := db.Migrator().CreateTable(&model.TreeLocalResource{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE cl_tree_node_local (tree_node_id INTEGER, tree_local_resource_id INTEGER)").Error; err != nil {
		t.Fatal(err)
	}
	local := &model.TreeLocalResource{Name: "fixture", Description: "clear me", Environment: "test", Tags: model.KeyValueList{{Key: "team", Value: "ops"}}, IpAddr: "192.0.2.5", Port: 22, Username: "root", AuthMode: model.AuthModeKey, Key: "old-private-key", Status: model.RUNNING, Cpu: 4}
	if err := db.Omit("TreeNodes").Create(local).Error; err != nil {
		t.Fatal(err)
	}
	store := dao.NewTreeLocalDAO(db, zap.NewNop())
	service, err := NewTreeLocalService(&config.Config{}, zap.NewNop(), rotatedLocalDAO{TreeLocalDAO: store, db: db})
	if err != nil {
		t.Fatal(err)
	}
	var request model.UpdateTreeLocalResourceReq
	if err := json.Unmarshal([]byte(`{"description":"","tags":[],"username":"operator"}`), &request); err != nil {
		t.Fatal(err)
	}
	request.ID = local.ID
	if err := service.UpdateTreeLocal(context.Background(), &request); err != nil {
		t.Fatal(err)
	}
	var saved model.TreeLocalResource
	if err := db.First(&saved, local.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.Description != "" || len(saved.Tags) != 0 || saved.Username != "operator" {
		t.Fatal("显式清空标签和描述、修改用户名没有持久化")
	}
	if saved.Key != "newer-private-key" || saved.Status != model.RUNNING || saved.Cpu != 4 || saved.Name != "fixture" || saved.Environment != "test" || saved.AuthMode != model.AuthModeKey {
		t.Fatal("局部元数据更新覆盖了新凭据、运行状态或未提供的字段")
	}
	encoded, err := json.Marshal(saved)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "private-key") || strings.Contains(string(encoded), `"key":`) {
		t.Fatal("主机列表或详情序列化包含私钥")
	}
}
