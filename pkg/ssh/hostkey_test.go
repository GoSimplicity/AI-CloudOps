package ssh

import "testing"

// 生产判断与 runenv 一致：去首尾空白、忽略大小写
func TestHostKeyCallbackRequiresKnownHostsInProductionVariants(t *testing.T) {
	t.Setenv("SSH_KNOWN_HOSTS", "")
	for _, env := range []string{"production", " Production ", "PRODUCTION"} {
		t.Setenv("ENV", env)
		if _, err := HostKeyCallback(); err == nil {
			t.Fatalf("ENV=%q 未配置 SSH_KNOWN_HOSTS 时应报错", env)
		}
	}
	t.Setenv("ENV", "development")
	if _, err := HostKeyCallback(); err != nil {
		t.Fatalf("开发环境不应报错: %v", err)
	}
}
