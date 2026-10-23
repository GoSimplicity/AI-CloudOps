package config

import (
	"strings"
	"testing"
)

func prodTestConfig() *Config {
	c := validConfigForTest()
	c.Tree.PasswordEncryptionKey = randomTestKey()[:32]
	return c
}

// ENV 去首尾空白、忽略大小写后等于 production 即按生产环境校验
func TestProductionEnvDetectionTrimAndFold(t *testing.T) {
	for _, env := range []string{"production", " Production ", "PRODUCTION", "\tproduction\n"} {
		t.Run(env, func(t *testing.T) {
			t.Setenv("ENV", env)
			c := prodTestConfig()
			c.Mock.Enabled = true // 仅生产环境拒绝
			if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "MOCK_ENABLED") {
				t.Fatalf("ENV=%q 应按生产环境校验, got %v", env, err)
			}
			c = prodTestConfig()
			c.JWT.Expiration = MaxJWTExpirationMinutes + 1
			if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "JWT_EXPIRATION") {
				t.Fatalf("ENV=%q 下 1441 分钟应拒绝启动, got %v", env, err)
			}
		})
	}
	for _, env := range []string{"", "development", "prod", "production-x"} {
		t.Run("non-prod/"+env, func(t *testing.T) {
			t.Setenv("ENV", env)
			c := validConfigForTest()
			c.Mock.Enabled = true
			if err := c.Validate(); err != nil {
				t.Fatalf("ENV=%q 不应按生产环境校验: %v", env, err)
			}
		})
	}
}

// JWT_EXPIRATION <= 0：告警“将使用默认值 30”，不报错（所有环境）
func TestJWTExpirationNonPositiveWarnsDefault30(t *testing.T) {
	for _, env := range allEnvs {
		for _, exp := range []int64{0, -5} {
			t.Run(env, func(t *testing.T) {
				t.Setenv("ENV", env)
				warns := captureConfigWarn(t)
				c := prodTestConfig()
				c.JWT.Expiration = exp
				if err := c.Validate(); err != nil {
					t.Fatalf("JWT_EXPIRATION=%d 不应报错: %v", exp, err)
				}
				if len(*warns) != 1 || !strings.Contains((*warns)[0], "将使用默认值 30") {
					t.Fatalf("JWT_EXPIRATION=%d 应告警一次并提示默认值 30, got %v", exp, *warns)
				}
			})
		}
	}
}

// 生产环境：TREE_PASSWORD_ENCRYPTION_KEY、JWT_ISSUER、AIOPS_API_KEY 拒绝首尾空白、控制字符、格式字符，且不泄露值
func TestProductionSecretsRejectSpaceAndControlChars(t *testing.T) {
	type field struct {
		name string
		set  func(c *Config, v string)
		good func() string
	}
	fields := []field{
		{"TREE_PASSWORD_ENCRYPTION_KEY", func(c *Config, v string) { c.Tree.PasswordEncryptionKey = v }, func() string { return randomTestKey()[:31] }},
		{"JWT_ISSUER", func(c *Config, v string) { c.JWT.Issuer = v }, func() string { return "fake-issuer-" + randomTestKey()[:8] }},
		{"AIOPS_API_KEY", func(c *Config, v string) { c.AIOps.BaseURL = "https://aiops.invalid"; c.AIOps.APIKey = v }, func() string { return randomTestKey() }},
	}
	variants := map[string]func(string) string{
		"leading-space":    func(s string) string { return " " + s },
		"trailing-newline": func(s string) string { return s + "\n" },
		"inner-nul":        func(s string) string { return s[:4] + "\x00" + s[4:] },
		"inner-tab":        func(s string) string { return s[:4] + "\t" + s[4:] },
		"zwsp-U+200B":      func(s string) string { return s[:4] + "\u200B" + s[4:] },
		"bom-U+FEFF":       func(s string) string { return "\uFEFF" + s },
		"rlo-U+202E":       func(s string) string { return s[:4] + "\u202E" + s[4:] },
	}
	for _, env := range []string{"production", " Production ", "PRODUCTION"} {
		for _, f := range fields {
			for vname, mk := range variants {
				t.Run(env+"/"+f.name+"/"+vname, func(t *testing.T) {
					t.Setenv("ENV", env)
					c := prodTestConfig()
					bad := mk(f.good())
					f.set(c, bad)
					err := c.Validate()
					assertRejectedWithoutLeak(t, err, f.name+" 首尾不能有空白", bad)
					if strings.Contains(err.Error(), strings.TrimSpace(bad)) {
						t.Fatalf("错误信息泄露了值: %v", err)
					}
				})
			}
		}
	}
	// 合法值在生产环境通过
	t.Setenv("ENV", "production")
	c := prodTestConfig()
	c.AIOps.BaseURL, c.AIOps.APIKey = "https://aiops.invalid", randomTestKey()
	if err := c.Validate(); err != nil {
		t.Fatalf("合法生产配置应通过: %v", err)
	}
	// 非生产环境不做这三项检查（行为不变）
	t.Setenv("ENV", "development")
	c = validConfigForTest()
	c.Tree.PasswordEncryptionKey = " fake-tree-key-value"
	if err := c.Validate(); err != nil {
		t.Fatalf("非生产环境不应检查 TREE_PASSWORD_ENCRYPTION_KEY 首尾空白: %v", err)
	}
}
