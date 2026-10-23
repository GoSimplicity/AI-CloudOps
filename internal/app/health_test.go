package app

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestReadinessHandler(t *testing.T) {
	secretError := errors.New("mysql://user:password@private-db:3306")
	success := func(context.Context) error { return nil }
	failure := func(context.Context) error { return secretError }
	wait := func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }
	for _, tt := range []struct {
		name            string
		database, cache func(context.Context) error
		canceled        bool
		status          int
	}{
		{"ready", success, success, false, http.StatusOK},
		{"database unavailable", failure, success, false, http.StatusServiceUnavailable},
		{"redis unavailable", success, failure, false, http.StatusServiceUnavailable},
		{"failure cancels other probe", failure, wait, false, http.StatusServiceUnavailable},
		{"request cancelled", wait, wait, true, http.StatusServiceUnavailable},
		{"cancelled success is not ready", success, success, true, http.StatusServiceUnavailable},
		{"shared deadline", wait, wait, false, http.StatusServiceUnavailable},
	} {
		t.Run(tt.name, func(t *testing.T) {
			engine := gin.New()
			engine.GET("/ready", readinessHandler(tt.database, tt.cache))
			request := httptest.NewRequest("GET", "/ready", nil)
			ctx, cancel := context.WithCancel(request.Context())
			defer cancel()
			if tt.canceled {
				cancel()
			}
			response := httptest.NewRecorder()
			started := time.Now()
			engine.ServeHTTP(response, request.WithContext(ctx))
			if response.Code != tt.status {
				t.Fatalf("status = %d, want %d", response.Code, tt.status)
			}
			if strings.Contains(response.Body.String(), "password") || strings.Contains(response.Body.String(), "private-db") {
				t.Fatal("dependency details leaked")
			}
			if elapsed := time.Since(started); elapsed > readinessTimeout+time.Second {
				t.Fatalf("probe exceeded bounded deadline: %v", elapsed)
			}
		})
	}
}

func TestReadinessRecoversWithoutRestart(t *testing.T) {
	unavailable := true
	engine := gin.New()
	engine.GET("/ready", readinessHandler(func(context.Context) error {
		if unavailable {
			return errors.New("temporarily unavailable")
		}
		return nil
	}, func(context.Context) error { return nil }))
	for _, status := range []int{http.StatusServiceUnavailable, http.StatusOK, http.StatusServiceUnavailable} {
		unavailable = status == http.StatusServiceUnavailable
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, httptest.NewRequest("GET", "/ready", nil))
		if response.Code != status {
			t.Fatalf("status = %d, want %d", response.Code, status)
		}
	}
}
