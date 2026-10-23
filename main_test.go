package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// 测试专用标记，代表 .env 中的密钥值；不是真实密钥
const dotEnvMarker = "dotenv-marker-value-for-test-only"

func writeDotEnv(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadDotEnvMissingFileContinues(t *testing.T) {
	if err := loadDotEnv(filepath.Join(t.TempDir(), ".env")); err != nil {
		t.Fatalf(".env 不存在时应继续: %v", err)
	}
}

func TestLoadDotEnvValidFile(t *testing.T) {
	t.Setenv("DOTENV_TEST_VALID", "")
	os.Unsetenv("DOTENV_TEST_VALID")
	if err := loadDotEnv(writeDotEnv(t, "DOTENV_TEST_VALID=ok\n")); err != nil {
		t.Fatalf("合法 .env 应加载成功: %v", err)
	}
	if os.Getenv("DOTENV_TEST_VALID") != "ok" {
		t.Fatal("合法 .env 中的变量未生效")
	}
}

func TestLoadDotEnvParseErrorHidesValue(t *testing.T) {
	for name, content := range map[string]string{
		// 这些输入下 godotenv 原始错误都会回显出错内容
		"unclosed-double-quote": "JWT_KEY1=\"" + dotEnvMarker + "\n",
		"unclosed-single-quote": "JWT_KEY1='" + dotEnvMarker + "\n",
		"bare-value-line":       "A=1\n" + dotEnvMarker + "\n",
		"bad-prefix":            "!" + dotEnvMarker + "=x\n",
	} {
		t.Run(name, func(t *testing.T) {
			err := loadDotEnv(writeDotEnv(t, content))
			if !errors.Is(err, errDotEnvInvalid) {
				t.Fatalf("格式错误的 .env 必须返回固定错误, got %v", err)
			}
			if strings.Contains(err.Error(), dotEnvMarker) {
				t.Fatalf("错误信息泄露了 .env 原值: %v", err)
			}
		})
	}
}

// 进程级：.env 格式错误时以非零码退出，stderr 只有固定文案
func TestMainExitsNonZeroOnInvalidDotEnv(t *testing.T) {
	if dir := os.Getenv("CLOUDOPS_TEST_MAIN_DIR"); dir != "" {
		if err := os.Chdir(dir); err != nil {
			os.Exit(3)
		}
		os.Args = []string{os.Args[0]}
		main()
		os.Exit(0)
	}
	path := writeDotEnv(t, "JWT_KEY1=\""+dotEnvMarker+"\n")
	cmd := exec.Command(os.Args[0], "-test.run=^TestMainExitsNonZeroOnInvalidDotEnv$")
	cmd.Env = append(os.Environ(), "CLOUDOPS_TEST_MAIN_DIR="+filepath.Dir(path))
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
		t.Fatalf("期望退出码 1, got %v; stderr=%q", err, stderr.String())
	}
	out := stdout.String() + stderr.String()
	if strings.Contains(out, dotEnvMarker) {
		t.Fatal("输出泄露了 .env 原值")
	}
	if !strings.Contains(stderr.String(), errDotEnvInvalid.Error()) {
		t.Fatalf("stderr 应为固定文案, got %q", stderr.String())
	}
}
