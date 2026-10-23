package ssh

import (
	"strings"
	"testing"
)

// HostKeyCallback 与 config.IsProduction 复用 pkg/envmode：别名、大小写/空白、未设置与未知取值都按生产处理。
func TestHostKeyCallbackSharesEnvmodeDecision(t *testing.T) {
	for _, tt := range []struct {
		env        string
		production bool
	}{
		{"production", true},
		{"Production", true},
		{" production", true},
		{"prod", true},
		{"prd", true},
		{"PROD", true},
		{"", true},
		{"staging", true},
		{"development", false},
		{"Dev", false},
		{"test", false},
	} {
		t.Run(tt.env, func(t *testing.T) {
			t.Setenv("ENV", tt.env)
			t.Setenv("SSH_KNOWN_HOSTS", "")
			cb, err := HostKeyCallback()
			if tt.production {
				if err == nil || cb != nil || !strings.Contains(err.Error(), "SSH_KNOWN_HOSTS") {
					t.Fatalf("ENV=%q 应要求 SSH_KNOWN_HOSTS, cb=%v err=%v", tt.env, cb != nil, err)
				}
				return
			}
			if err != nil || cb == nil {
				t.Fatalf("ENV=%q 开发/测试环境应允许跳过校验: %v", tt.env, err)
			}
		})
	}
}
