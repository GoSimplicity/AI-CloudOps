//go:build smoke

package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

func TestCommandOutputSmoke(t *testing.T) {
	executor := NewCommandExecutor(zap.NewNop())
	for _, tt := range []struct {
		name, command        string
		wantError, truncated bool
	}{
		{"success", "printf hello", false, false},
		{"stderr failure", "printf failure >&2; exit 2", true, false},
		{"large output", "printf '%*s' 1100000 x", false, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executor.Execute(context.Background(), &CommandTask{Command: "/bin/bash", Args: model.StringList{"-c", tt.command}})
			if (err != nil) != tt.wantError {
				t.Fatalf("unexpected error: %v", err)
			}
			if strings.Contains(output, outputTruncatedMarker) != tt.truncated {
				t.Fatal("unexpected truncation marker")
			}
			if len(output) > maxTaskOutputBytes+len(outputTruncatedMarker) {
				t.Fatal("unbounded output")
			}
			if !tt.truncated && output != map[bool]string{false: "hello", true: "failure"}[tt.wantError] {
				t.Fatalf("unexpected output: %q", output)
			}
		})
	}
	// 双流输出都超限时，组合结果仍可写入 MySQL TEXT
	combined, err := executor.Execute(context.Background(), &CommandTask{Command: "/bin/bash", Args: model.StringList{"-c", "printf '%*s' 1100000 x; printf '%*s' 1100000 y >&2"}})
	if err != nil || len(combined) > 65535 || strings.Count(combined, outputTruncatedMarker) != 2 {
		t.Fatalf("combined output length %d, error %v", len(combined), err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := executor.Execute(ctx, &CommandTask{Command: "/bin/sleep", Args: model.StringList{"10"}}); err == nil {
		t.Fatal("cancelled command succeeded")
	}
}

func TestScriptOutputSmoke(t *testing.T) {
	output, err := NewScriptExecutor(zap.NewNop()).Execute(context.Background(), &ScriptTask{Type: "bash", Content: "printf '%*s' 1100000 x"})
	if err != nil {
		t.Fatal(err)
	}
	if len(output) > maxTaskOutputBytes+len(outputTruncatedMarker) || !strings.HasSuffix(output, outputTruncatedMarker) {
		t.Fatal("script output was not bounded")
	}
}

func TestHTTPOutputSmoke(t *testing.T) {
	for _, tt := range []struct {
		name         string
		size, status int
		wantError    bool
	}{
		{"normal", 5, http.StatusOK, false},
		{"exact limit", maxTaskOutputBytes, http.StatusOK, false},
		{"oversized", maxTaskOutputBytes + 1, http.StatusOK, true},
		{"upstream failure", 5, http.StatusInternalServerError, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(strings.Repeat("a", tt.size)))
			}))
			t.Cleanup(server.Close)
			output, err := NewHTTPExecutor(zap.NewNop()).Execute(context.Background(), &HTTPTask{Method: "GET", URL: server.URL})
			if (err != nil) != tt.wantError {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(output) > maxTaskOutputBytes+len(outputTruncatedMarker) {
				t.Fatal("unbounded HTTP output")
			}
			if strings.HasSuffix(output, outputTruncatedMarker) != (tt.size > maxTaskOutputBytes) {
				t.Fatal("unexpected truncation marker")
			}
		})
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	t.Cleanup(server.Close)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := NewHTTPExecutor(zap.NewNop()).Execute(ctx, &HTTPTask{Method: "GET", URL: server.URL}); err == nil {
		t.Fatal("timed out request succeeded")
	}
}
