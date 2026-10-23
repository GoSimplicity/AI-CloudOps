package config

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// jwt.issuer 不是密钥：值写在 config/*.yaml 中，两份配置一致，且不属于必需密钥环境变量
func TestJWTIssuerComesFromConfigFile(t *testing.T) {
	var issuers []string
	for _, file := range configFiles {
		v := viper.New()
		v.SetConfigFile(file)
		if err := v.ReadInConfig(); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		issuer := v.GetString("jwt.issuer")
		if strings.TrimSpace(issuer) == "" {
			t.Fatalf("%s: jwt.issuer 应写在配置文件中", file)
		}
		issuers = append(issuers, issuer)
	}
	for _, issuer := range issuers[1:] {
		if issuer != issuers[0] {
			t.Fatal("各配置文件的 jwt.issuer 应保持一致，否则切换环境会使已签发令牌失效")
		}
	}
	for _, required := range requiredSecretEnvVars {
		if required.Env == "JWT_ISSUER" {
			t.Fatal("JWT_ISSUER 不应再是必需的密钥环境变量")
		}
	}
}

func TestJWTIssuerLoadAndEnvOverride(t *testing.T) {
	for _, file := range configFiles {
		t.Run(file, func(t *testing.T) {
			v := viper.New()
			v.SetConfigFile(file)
			if err := v.ReadInConfig(); err != nil {
				t.Fatal(err)
			}
			fromFile := v.GetString("jwt.issuer")

			setRequiredSecretsForTest(t, file)
			t.Setenv("JWT_ISSUER", "")
			viper.Reset()
			t.Cleanup(viper.Reset)
			cfg, _, err := loadFromFile(file)
			if err != nil {
				t.Fatalf("未设置 JWT_ISSUER 时应能用配置文件中的值启动: %v", err)
			}
			if cfg.JWT.Issuer != fromFile {
				t.Fatal("未设置 JWT_ISSUER 时应使用配置文件中的 jwt.issuer")
			}

			t.Setenv("JWT_ISSUER", "fake-override-issuer")
			viper.Reset()
			cfg, _, err = loadFromFile(file)
			if err != nil {
				t.Fatalf("加载配置失败: %v", err)
			}
			if cfg.JWT.Issuer != "fake-override-issuer" {
				t.Fatal("JWT_ISSUER 应能覆盖配置文件中的 jwt.issuer")
			}
		})
	}
}

func TestValidateRejectsEmptyIssuerWithoutListingItAsSecret(t *testing.T) {
	t.Setenv("ENV", "development")
	c := validConfigForTest()
	c.JWT.Issuer = " "
	err := c.Validate()
	if err == nil || !strings.Contains(err.Error(), "jwt.issuer") {
		t.Fatalf("jwt.issuer 为空时应失败, got %v", err)
	}
	if strings.Contains(err.Error(), "JWT_ISSUER") || strings.Contains(err.Error(), "密钥") {
		t.Fatalf("jwt.issuer 不应被当作缺失的密钥环境变量: %v", err)
	}
	assertNoValuesLeaked(t, err.Error())
}

// setRequiredSecretsForTest 用假值设置全部必需密钥环境变量（满足 production 校验）
func setRequiredSecretsForTest(t *testing.T, file string) {
	t.Helper()
	if strings.Contains(file, "production") {
		t.Setenv("ENV", "production")
		t.Setenv("MOCK_ENABLED", "false")
	} else {
		t.Setenv("ENV", "development")
	}
	t.Setenv("JWT_KEY1", "fake-key1-"+strings.Repeat("a", 32))
	t.Setenv("JWT_KEY2", "fake-key2-"+strings.Repeat("b", 32))
	t.Setenv("MYSQL_ADDR", "fake_user:fake_pass@tcp(127.0.0.1:3306)/fake")
	t.Setenv("TREE_PASSWORD_ENCRYPTION_KEY", strings.Repeat("t", 32))
}
