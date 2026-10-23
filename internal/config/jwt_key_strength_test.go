package config

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"regexp"
	"strings"
	"testing"
)

// randomTestKey 测试专用：每次运行临时生成的强密钥（等价于 openssl rand -base64 48）
func randomTestKey() string {
	b := make([]byte, 48)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(b)
}

var allEnvs = []string{"", "development", "test", "production"}

func TestJWTKeyStrengthAllEnvs(t *testing.T) {
	strong1, strong2 := randomTestKey(), randomTestKey()
	cases := []struct {
		name       string
		key1, key2 string
		wantErr    string
	}{
		{"V4-same-1-char", "k", "k", "JWT_KEY1 长度不足"},
		{"V5-same-3-char", "kkk", "kkk", "JWT_KEY1 长度不足"},
		{"key2-short", strong1, "short", "JWT_KEY2 长度不足"},
		{"just-below-min", strings.Repeat("q", MinJWTKeyLength-1), strong2, "JWT_KEY1 长度不足"},
		{"same-strong-keys", strong1, strong1, "不得相同"},
		{"sample-prefix", knownSampleJWTKeyPrefix + randomTestKey(), strong2, "示例值"},
		{"sample-prefix-key2", strong1, knownSampleJWTKeyPrefix + randomTestKey(), "JWT_KEY2 不能使用仓库示例值"},
		{"strong-keys-pass", strong1, strong2, ""},
		{"exact-min-pass", strings.Repeat("q", MinJWTKeyLength), strong2, ""},
	}
	for _, env := range allEnvs {
		for _, tc := range cases {
			t.Run(env+"/"+tc.name, func(t *testing.T) {
				t.Setenv("ENV", env)
				c := validConfigForTest()
				c.Tree.PasswordEncryptionKey = randomTestKey()[:32] // 生产环境要求恰好32字节
				c.JWT.Key1, c.JWT.Key2 = tc.key1, tc.key2
				err := c.Validate()
				if tc.wantErr == "" {
					if err != nil {
						t.Fatalf("合格密钥应通过: %v", err)
					}
					return
				}
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("应被拒并包含 %q, got %v", tc.wantErr, err)
				}
				for _, v := range []string{tc.key1, tc.key2} {
					if len(v) >= 3 && strings.Contains(err.Error(), v) {
						t.Fatalf("错误信息泄露了密钥值: %v", err)
					}
				}
			})
		}
	}
}

// 已知示例值按 sha256 拒绝：临时登记一个随机值的哈希，验证哈希匹配路径
func TestJWTKeyRejectsKnownSampleHash(t *testing.T) {
	sample := randomTestKey()
	sum := sha256.Sum256([]byte(sample))
	h := hex.EncodeToString(sum[:])
	knownSampleJWTKeySHA256[h] = struct{}{}
	t.Cleanup(func() { delete(knownSampleJWTKeySHA256, h) })
	for _, env := range allEnvs {
		t.Setenv("ENV", env)
		c := validConfigForTest()
		c.Tree.PasswordEncryptionKey = randomTestKey()[:32]
		c.JWT.Key2 = sample
		if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "JWT_KEY2 不能使用仓库示例值") {
			t.Fatalf("ENV=%q 示例值应被拒, got %v", env, err)
		}
	}
}

// 拒绝列表只能存 sha256（64 位小写十六进制），不得存明文
func TestKnownSampleJWTKeyListStoresOnlyHashes(t *testing.T) {
	re := regexp.MustCompile(`^[0-9a-f]{64}$`)
	if len(knownSampleJWTKeySHA256) == 0 {
		t.Fatal("示例值拒绝列表不应为空")
	}
	for h := range knownSampleJWTKeySHA256 {
		if !re.MatchString(h) {
			t.Fatalf("拒绝列表条目必须是 sha256 十六进制: %q", h)
		}
	}
}
