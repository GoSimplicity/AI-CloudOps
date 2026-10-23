//go:build smoke

package handler

import (
	"context"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestScriptRuntimeSmoke(t *testing.T) {
	executor := NewScriptExecutor(zap.NewNop())
	for _, tt := range []struct {
		name, scriptType, content string
		wantError                 bool
	}{
		{"bash", "bash", "printf executed", false},
		{"python", "python3", "print('executed')", false},
		{"node", "node", "console.log('executed')", false},
		{"node alias", "javascript", "console.log('executed')", false},
		{"node runtime error", "nodejs", "throw new Error('runtime-failure')", true},
		{"unsupported", "unknown", "", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executor.Execute(context.Background(), &ScriptTask{Type: tt.scriptType, Content: tt.content})
			if (err != nil) != tt.wantError {
				t.Fatalf("unexpected result: %q, %v", output, err)
			}
			if !tt.wantError && output != "executed" {
				t.Fatalf("script did not execute: %q", output)
			}
			if tt.name == "node runtime error" && !strings.Contains(output, "runtime-failure") {
				t.Fatal("runtime error was not captured")
			}
		})
	}
	t.Run("cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := executor.Execute(ctx, &ScriptTask{Type: "node", Content: "setInterval(() => {}, 1000)"}); err == nil {
			t.Fatal("cancelled script succeeded")
		}
	})
}
