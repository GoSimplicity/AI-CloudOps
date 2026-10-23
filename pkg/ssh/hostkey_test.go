package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"

	cryptossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func setEnvForHostKey(t *testing.T, raw string, set bool) {
	t.Helper()
	t.Setenv("ENV", raw)
	if !set {
		if err := os.Unsetenv("ENV"); err != nil {
			t.Fatal(err)
		}
	}
}

// 没有 SSH_KNOWN_HOSTS 时：只有明确的开发环境允许跳过主机密钥校验；
// 生产别名、未设置、空白、未知取值都必须报错（强制主机密钥校验）。
func TestHostKeyCallbackWithoutKnownHosts(t *testing.T) {
	t.Setenv("SSH_KNOWN_HOSTS", "")
	for _, tt := range []struct {
		raw     string
		set     bool
		wantErr bool
	}{
		{"production", true, true}, {"prod", true, true}, {"PRD", true, true}, {" Prod\t", true, true},
		{"", false, true}, {"", true, true}, {"  ", true, true},
		{"test", true, true}, {"testing", true, true}, {"staging", true, true},
		{"development", true, false}, {"dev", true, false}, {" DEV ", true, false},
	} {
		setEnvForHostKey(t, tt.raw, tt.set)
		cb, err := HostKeyCallback()
		if (err != nil) != tt.wantErr || (cb == nil) != tt.wantErr {
			t.Errorf("ENV=%q(set=%v): HostKeyCallback() err=%v cb!=nil=%v，期望报错=%v", tt.raw, tt.set, err, cb != nil, tt.wantErr)
		}
	}
}

// 配了 SSH_KNOWN_HOSTS 时任何环境都用它校验：已知主机通过、未知主机拒绝。
func TestHostKeyCallbackUsesKnownHosts(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	key, err := cryptossh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	otherPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	other, err := cryptossh.NewPublicKey(otherPub)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "known_hosts")
	if err := os.WriteFile(file, []byte(knownhosts.Line([]string{"host.example.test:22"}, key)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SSH_KNOWN_HOSTS", file)
	for _, raw := range []string{" PROD ", "dev"} {
		setEnvForHostKey(t, raw, true)
		cb, err := HostKeyCallback()
		if err != nil || cb == nil {
			t.Fatalf("ENV=%q: HostKeyCallback() = %v", raw, err)
		}
		addr := &fakeAddr{"203.0.113.7:22"}
		if err := cb("host.example.test:22", addr, key); err != nil {
			t.Errorf("ENV=%q: 已知主机被拒：%v", raw, err)
		}
		if err := cb("host.example.test:22", addr, other); err == nil {
			t.Errorf("ENV=%q: 主机密钥不符却通过了", raw)
		}
	}
	t.Setenv("SSH_KNOWN_HOSTS", filepath.Join(t.TempDir(), "missing"))
	if _, err := HostKeyCallback(); err == nil {
		t.Error("SSH_KNOWN_HOSTS 指向不存在的文件时应报错")
	}
}

type fakeAddr struct{ s string }

func (a *fakeAddr) Network() string { return "tcp" }
func (a *fakeAddr) String() string  { return a.s }
