package config

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"testing"
)

// synthWeakSecret 测试专用：运行时生成的短（弱）合成密钥，只用来断言错误信息里不出现它
func synthWeakSecret(t *testing.T, prefix string) string {
	t.Helper()
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return prefix + hex.EncodeToString(b)
}

// P3-8：生产环境多个密钥同时不合格时，报错必须稳定指向同一个变量（按 JWT_KEY1、JWT_KEY2、
// TREE_PASSWORD_ENCRYPTION_KEY 的固定顺序检查），同样的输入每次得到同样的启动失败原因，且不含密钥文本。
func TestProductionWeakKeyErrorIsDeterministic(t *testing.T) {
	t.Setenv("ENV", " Prod ")
	k1, k2, tree := synthWeakSecret(t, "wk1"), synthWeakSecret(t, "wk2"), synthWeakSecret(t, "wk3")
	cfg := &Config{Server: ServerConfig{Port: "8889"}, Log: LogConfig{Dir: "logs"}, MySQL: MySQLConfig{Addr: "db"}, Redis: RedisConfig{Addr: "redis"},
		JWT: JWTConfig{Key1: k1, Key2: k2, Issuer: "fake-issuer-value"}, Tree: TreeConfig{PasswordEncryptionKey: tree}, Mock: MockConfig{Enabled: true}}
	first := cfg.Validate()
	if first == nil || !strings.Contains(first.Error(), "JWT_KEY1") {
		t.Fatalf("三个密钥都弱时应先报 JWT_KEY1，got %v", first)
	}
	for i := 0; i < 200; i++ {
		err := cfg.Validate()
		if err == nil || err.Error() != first.Error() {
			t.Fatalf("第 %d 次校验结果与第一次不同：%v vs %v", i+1, err, first)
		}
	}
	for _, secret := range []string{k1, k2, tree} {
		if strings.Contains(first.Error(), secret) {
			t.Fatalf("错误信息包含了密钥文本")
		}
	}
}
