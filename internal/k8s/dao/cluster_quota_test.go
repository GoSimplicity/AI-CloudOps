package dao

import (
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
)

func TestUpdateClusterPersistsClearedQuota(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.K8sCluster{}); err != nil {
		t.Fatal(err)
	}
	c := &model.K8sCluster{Name: "quota-fixture", Env: model.EnvDev, CpuRequest: "500", CpuLimit: "1000", RestrictNamespace: model.StringList{"team-a"}}
	if err := db.Create(c).Error; err != nil {
		t.Fatal(err)
	}
	d := NewClusterDAO(db, zap.NewNop())
	c.CpuRequest = ""
	c.CpuLimit = ""
	c.RestrictNamespace = model.StringList{}
	if err := d.UpdateCluster(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	got, err := d.GetClusterByID(context.Background(), c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.CpuRequest != "" || got.CpuLimit != "" || len(got.RestrictNamespace) != 0 {
		t.Fatal("clearing quota silently preserved stale limits")
	}
}
