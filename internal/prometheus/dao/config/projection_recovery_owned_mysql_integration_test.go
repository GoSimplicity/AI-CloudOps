//go:build integration

package config

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/testinfra/ownedmysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func recoveryMySQLLocalEndpoint(ctx context.Context) (string, error) {
	return ownedmysql.LocalEndpoint(ctx)
}

func TestProjectionRecoveryMySQLOwnedRejectsRemoteOverrides(t *testing.T) {
	for _, key := range []string{"DOCKER_HOST", "DOCKER_CONTEXT"} {
		t.Run(key, func(t *testing.T) {
			t.Setenv("DOCKER_HOST", "")
			t.Setenv("DOCKER_CONTEXT", "")
			t.Setenv("PATH", "")
			t.Setenv(key, "remote-must-not-be-used")
			if _, err := recoveryMySQLLocalEndpoint(context.Background()); err == nil {
				t.Fatal("override was accepted before invoking Docker")
			}
		})
	}
}

type recoveryOwnedMySQL struct {
	a, b    *gorm.DB
	admin   *sql.DB
	address string
	schema  string
	user    string
}

func newRecoveryOwnedMySQL(t *testing.T) *recoveryOwnedMySQL {
	t.Helper()
	owned := ownedmysql.Start(t)
	open := func(pool *sql.DB) *gorm.DB {
		t.Helper()
		db, err := gorm.Open(mysql.New(mysql.Config{Conn: pool}), &gorm.Config{Logger: logger.Discard})
		if err != nil {
			t.Fatal("private runtime pool unavailable")
		}
		return db
	}
	f := &recoveryOwnedMySQL{a: open(owned.A), b: open(owned.B), admin: owned.Admin, address: owned.Address, schema: owned.Schema, user: owned.User}
	var aid, bid uint64
	var isolation string
	if f.a.Raw("SELECT CONNECTION_ID()").Scan(&aid).Error != nil || f.b.Raw("SELECT CONNECTION_ID()").Scan(&bid).Error != nil || aid == 0 || bid == 0 || aid == bid {
		t.Fatal("two independent runtime sessions required")
	}
	if f.a.Raw("SELECT @@transaction_isolation").Scan(&isolation).Error != nil || isolation != "REPEATABLE-READ" {
		t.Fatal("owned fixture must report the supported MySQL RR isolation")
	}
	t.Logf("owned runtime connection ids %d/%d; isolation %s", aid, bid, isolation)
	var tables int64
	if f.a.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE()").Scan(&tables).Error != nil || tables != 0 {
		t.Fatal("new fixture schema was not empty")
	}
	if f.a.AutoMigrate(&model.MonitorConfig{}, &model.MonitorConfigNamespaceGuard{}, &model.MonitorConfigPoolControl{}, &model.MonitorConfigPoolOperation{}, &model.MonitorConfigArtifactEvent{}) != nil {
		t.Fatal("private five-model migration failed")
	}
	return f
}
func TestProjectionRecoveryOwnedMySQL(t *testing.T) {
	if os.Getenv("CLOUDOPS_TEST_PROJECTION_RECOVERY_MYSQL") != "true" {
		t.Skip("requires explicit owned recovery MySQL flag")
	}
	fixture := newRecoveryOwnedMySQL(t)
	runRecoveryMySQLCases(t, fixture)
}
