//go:build smoke

package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type readyRedis struct{ redis.Cmdable }

func (readyRedis) Ping(ctx context.Context) *redis.StatusCmd {
	return redis.NewStatusResult("PONG", ctx.Err())
}

func TestHealthRoutesObserveClosedDatabase(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	app := &App{db: db, redis: readyRedis{}, server: gin.New()}
	app.registerHealthRoutes()
	for _, stage := range []struct {
		closeDB bool
		path    string
		status  int
	}{
		{false, "/ready", http.StatusOK},
		{true, "/ready", http.StatusServiceUnavailable},
		{false, "/health", http.StatusOK},
	} {
		if stage.closeDB {
			if err := sqlDB.Close(); err != nil {
				t.Fatal(err)
			}
		}
		response := httptest.NewRecorder()
		app.server.ServeHTTP(response, httptest.NewRequest("GET", stage.path, nil))
		if response.Code != stage.status {
			t.Fatalf("%s status = %d, want %d", stage.path, response.Code, stage.status)
		}
	}
}
