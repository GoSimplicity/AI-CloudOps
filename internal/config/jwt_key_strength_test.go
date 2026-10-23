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

// distinctCharsKey 测试专用：用 n 个不同字符循环拼出长度为 length 的低熵测试串（不是真实密钥形态）
func distinctCharsKey(n, length int) string {
	const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteByte(alphabet[i%n])
	}
	return b.String()
}

// assertRejectedWithoutLeak 断言校验失败，且错误信息不包含任一密钥（或其规范化形式）
func assertRejectedWithoutLeak(t *testing.T, err error, wantErr string, keys ...string) {
	t.Helper()
	if err == nil {
		t.Fatal("应被拒绝, got nil")
	}
	if wantErr != "" && !strings.Contains(err.Error(), wantErr) {
		t.Fatalf("错误信息应包含 %q, got %v", wantErr, err)
	}
	msg := strings.ToLower(err.Error())
	for _, k := range keys {
		if n := normalizeJWTKey(k); len(n) >= 3 && strings.Contains(msg, n) {
			t.Fatalf("错误信息泄露了密钥值: %v", err)
		}
	}
}

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
		// 32 个相同字符：长度达标但只有 1 个不同字符，按低熵拒绝
		{"exact-min-single-char-rejected", strings.Repeat("q", MinJWTKeyLength), strong2, "JWT_KEY1 熵过低"},
		{"exact-min-pass", distinctCharsKey(MinJWTKeyDistinctChars, MinJWTKeyLength), strong2, ""},
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
	sum := sha256.Sum256([]byte(normalizeJWTKey(sample)))
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

// 低熵边界：9 个不同字符拒、10 个通过；两把密钥各自触发
func TestJWTKeyDistinctCharsBoundary(t *testing.T) {
	low := distinctCharsKey(MinJWTKeyDistinctChars-1, MinJWTKeyLength+8)
	ok := distinctCharsKey(MinJWTKeyDistinctChars, MinJWTKeyLength+8)
	if distinctCharCount(low) != 9 || distinctCharCount(ok) != 10 {
		t.Fatalf("测试数据构造错误: %d/%d", distinctCharCount(low), distinctCharCount(ok))
	}
	for _, env := range allEnvs {
		t.Run(env, func(t *testing.T) {
			t.Setenv("ENV", env)
			c := validConfigForTest()
			c.Tree.PasswordEncryptionKey = randomTestKey()[:32]

			c.JWT.Key1, c.JWT.Key2 = low, randomTestKey()
			assertRejectedWithoutLeak(t, c.Validate(), "JWT_KEY1 熵过低", low)

			c.JWT.Key1, c.JWT.Key2 = randomTestKey(), low
			assertRejectedWithoutLeak(t, c.Validate(), "JWT_KEY2 熵过低", low)

			c.JWT.Key1, c.JWT.Key2 = ok, randomTestKey()
			if err := c.Validate(); err != nil {
				t.Fatalf("10 个不同字符应通过 (key1): %v", err)
			}
			c.JWT.Key1, c.JWT.Key2 = randomTestKey(), ok
			if err := c.Validate(); err != nil {
				t.Fatalf("10 个不同字符应通过 (key2): %v", err)
			}
		})
	}
}

// P2-1：空白/控制字符、示例值变体（空白、大小写、前缀多一个字符）、两把密钥仅差空白或大小写，都应被拒且不泄露密钥
func TestJWTKeyNormalizationBypassRejected(t *testing.T) {
	// 临时登记一个随机“示例值”的规范化哈希，不在代码中写示例明文
	sample := randomTestKey()
	sum := sha256.Sum256([]byte(normalizeJWTKey(sample)))
	h := hex.EncodeToString(sum[:])
	knownSampleJWTKeySHA256[h] = struct{}{}
	t.Cleanup(func() { delete(knownSampleJWTKeySHA256, h) })

	base := randomTestKey()
	cases := []struct {
		name       string
		key1, key2 string
		wantErr    string
	}{
		{"a-plus-31-spaces", "a" + strings.Repeat(" ", 31), randomTestKey(), "JWT_KEY1"},
		{"leading-space", " " + randomTestKey(), randomTestKey(), "JWT_KEY1 首尾不能有空白"},
		{"trailing-newline", randomTestKey() + "\n", randomTestKey(), "JWT_KEY1 首尾不能有空白"},
		{"inner-control-char", randomTestKey()[:20] + "\x00" + randomTestKey()[:20], randomTestKey(), "控制字符"},
		{"sample-leading-space", " " + sample, randomTestKey(), "JWT_KEY1"},
		{"sample-leading-tab", randomTestKey(), "\t" + sample, "JWT_KEY2"},
		{"sample-upper", strings.ToUpper(sample), randomTestKey(), "JWT_KEY1 不能使用仓库示例值"},
		{"sample-lower", randomTestKey(), strings.ToLower(sample), "JWT_KEY2 不能使用仓库示例值"},
		{"sample-prefix-one-extra-char", "x" + knownSampleJWTKeyPrefix + randomTestKey(), randomTestKey(), "JWT_KEY1 不能使用仓库示例值"},
		{"sample-prefix-upper-inside", randomTestKey(), randomTestKey()[:8] + strings.ToUpper(knownSampleJWTKeyPrefix) + randomTestKey(), "JWT_KEY2 不能使用仓库示例值"},
		{"key2-equals-key1-plus-space", base, base + " ", "JWT_KEY2 首尾不能有空白"},
		{"key2-equals-upper-key1", base, strings.ToUpper(base), "不得相同"},
	}
	for _, env := range allEnvs {
		for _, tc := range cases {
			t.Run(env+"/"+tc.name, func(t *testing.T) {
				t.Setenv("ENV", env)
				c := validConfigForTest()
				c.Tree.PasswordEncryptionKey = randomTestKey()[:32]
				c.JWT.Key1, c.JWT.Key2 = tc.key1, tc.key2
				assertRejectedWithoutLeak(t, c.Validate(), tc.wantErr, tc.key1, tc.key2)
			})
		}
	}
}
