//go:build smoke

package config

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// moduleRoot 从当前包目录向上找 go.mod
func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("找不到 go.mod")
		}
		dir = parent
	}
}

func synthStrongSecret(t *testing.T, n int) string {
	t.Helper()
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(b)
}

func synthHex(t *testing.T, n int) string {
	t.Helper()
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(b)
}

// P3-8 端到端：真实主程序在 ENV=" Prod " 且密钥为弱合成值时，必须 rc=1、stderr 非空、
// 含可读原因（“配置校验失败”+变量名）、不含任何合成密钥文本，且多次运行原因一致。
func TestStartupFailurePrintsReasonWithoutSecrets(t *testing.T) {
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("PATH 中没有 go 工具链")
	}
	bin := filepath.Join(t.TempDir(), "cloudops-e2e")
	build := exec.Command(goBin, "build", "-o", bin, ".")
	build.Dir = moduleRoot(t)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("构建主程序失败：%v\n%s", err, out)
	}
	mysqlPw := "fakepw" + synthHex(t, 4)
	cases := []struct {
		name    string
		secrets map[string]string
		reason  string
	}{
		{"all-weak", map[string]string{
			"JWT_KEY1": "wk1" + synthHex(t, 4), "JWT_KEY2": "wk2" + synthHex(t, 4), "TREE_PASSWORD_ENCRYPTION_KEY": "wk3" + synthHex(t, 4),
		}, "JWT_KEY1"},
		{"tree-weak", map[string]string{
			"JWT_KEY1": synthStrongSecret(t, 48), "JWT_KEY2": synthStrongSecret(t, 48), "TREE_PASSWORD_ENCRYPTION_KEY": "wk3" + synthHex(t, 4),
		}, "TREE_PASSWORD_ENCRYPTION_KEY"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			workDir := t.TempDir() // 无 .env、无 config/ 目录
			env := []string{
				"PATH=" + os.Getenv("PATH"), "HOME=" + workDir, "TMPDIR=" + os.TempDir(),
				"ENV= Prod ", "JWT_ISSUER=fake-issuer-value", "MOCK_ENABLED=false",
				"MYSQL_ADDR=fakeuser:" + mysqlPw + "@tcp(127.0.0.1:9)/fakedb?charset=utf8mb4&parseTime=True&loc=Local",
			}
			for k, v := range tc.secrets {
				env = append(env, k+"="+v)
			}
			var first string
			for i := 0; i < 30; i++ {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				cmd := exec.CommandContext(ctx, bin)
				cmd.Dir, cmd.Env = workDir, env
				var stdout, stderr bytes.Buffer
				cmd.Stdout, cmd.Stderr = &stdout, &stderr
				err := cmd.Run()
				cancel()
				var exitErr *exec.ExitError
				if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
					t.Fatalf("第 %d 次：期望 rc=1，got %v", i+1, err)
				}
				msg := stderr.String()
				if strings.TrimSpace(msg) == "" {
					t.Fatalf("第 %d 次：rc=1 但 stderr 为空", i+1)
				}
				for _, want := range []string{"启动失败", "配置校验失败", tc.reason} {
					if !strings.Contains(msg, want) {
						t.Fatalf("第 %d 次：stderr 缺少 %q：\n%s", i+1, want, msg)
					}
				}
				for _, secret := range append([]string{mysqlPw}, tc.secrets["JWT_KEY1"], tc.secrets["JWT_KEY2"], tc.secrets["TREE_PASSWORD_ENCRYPTION_KEY"]) {
					if strings.Contains(msg, secret) || strings.Contains(stdout.String(), secret) {
						t.Fatalf("第 %d 次：输出包含了合成密钥文本", i+1)
					}
				}
				if i == 0 {
					first = msg
				} else if msg != first {
					t.Fatalf("第 %d 次 stderr 与第 1 次不同：\n第1次：%s\n第%d次：%s", i+1, first, i+1, msg)
				}
			}
		})
	}
}
