package config

import (
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

var requiredEnvNames = []string{"JWT_KEY1", "JWT_KEY2", "JWT_ISSUER", "MYSQL_ADDR", "TREE_PASSWORD_ENCRYPTION_KEY"}

func validConfigForTest() *Config {
	return &Config{
		Server: ServerConfig{Port: "8889"},
		Log:    LogConfig{Dir: "logs"},
		MySQL:  MySQLConfig{Addr: "fake-dsn-value"},
		Redis:  RedisConfig{Addr: "redis"},
		JWT:    JWTConfig{Key1: "fake-key1-value", Key2: "fake-key2-value", Issuer: "fake-issuer-value"},
		Tree:   TreeConfig{PasswordEncryptionKey: "fake-tree-key-value"},
	}
}

func TestValidateRequiresSecretEnvVars(t *testing.T) {
	t.Setenv("ENV", "development")
	if err := validConfigForTest().Validate(); err != nil {
		t.Fatalf("完整配置不应失败: %v", err)
	}
	clearers := map[string]func(*Config){
		"JWT_KEY1":                     func(c *Config) { c.JWT.Key1 = "" },
		"JWT_KEY2":                     func(c *Config) { c.JWT.Key2 = "  " },
		"JWT_ISSUER":                   func(c *Config) { c.JWT.Issuer = "" },
		"MYSQL_ADDR":                   func(c *Config) { c.MySQL.Addr = "" },
		"TREE_PASSWORD_ENCRYPTION_KEY": func(c *Config) { c.Tree.PasswordEncryptionKey = "" },
	}
	for name, change := range clearers {
		t.Run(name, func(t *testing.T) {
			c := validConfigForTest()
			change(c)
			err := c.Validate()
			if err == nil || !strings.Contains(err.Error(), name) {
				t.Fatalf("缺少 %s 时应失败并列出变量名, got %v", name, err)
			}
			assertNoValuesLeaked(t, err.Error())
		})
	}
	t.Run("all-missing", func(t *testing.T) {
		c := validConfigForTest()
		for _, change := range clearers {
			change(c)
		}
		err := c.Validate()
		if err == nil {
			t.Fatal("全部缺失时应失败")
		}
		for _, name := range requiredEnvNames {
			if !strings.Contains(err.Error(), name) {
				t.Errorf("错误信息缺少 %s: %v", name, err)
			}
		}
	})
}

// 错误信息中不得出现任何已配置的值
func assertNoValuesLeaked(t *testing.T, msg string) {
	t.Helper()
	if strings.Contains(msg, "fake-") {
		t.Fatalf("错误信息泄露了配置值: %s", msg)
	}
}

func TestLoadFailsFastWithoutSecretEnvVars(t *testing.T) {
	for _, file := range configFiles {
		for _, env := range []string{"development", "production"} {
			t.Run(env+"/"+file, func(t *testing.T) {
				viper.Reset()
				t.Cleanup(viper.Reset)
				t.Setenv("ENV", env)
				for _, name := range requiredEnvNames {
					t.Setenv(name, "")
				}
				t.Setenv("JWT_KEY1", "fake-only-key1-"+strings.Repeat("k", 32))
				_, _, err := loadFromFile(file)
				if err == nil {
					t.Fatal("缺少必需密钥环境变量时 Load 必须失败")
				}
				for _, name := range requiredEnvNames[1:] {
					if !strings.Contains(err.Error(), name) {
						t.Errorf("错误信息缺少 %s: %v", name, err)
					}
				}
				if strings.Contains(err.Error(), "JWT_KEY1") {
					t.Errorf("已提供的 JWT_KEY1 不应被列为缺失: %v", err)
				}
				assertNoValuesLeaked(t, err.Error())
			})
		}
	}
}

func TestNoBuiltInSecretDefaults(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	setDefaults()
	for _, key := range []string{"jwt.key1", "jwt.key2", "jwt.issuer", "mysql.addr", "tree.password_encryption_key", "redis.password"} {
		if viper.GetString(key) != "" {
			t.Errorf("%s 不得有内置默认值", key)
		}
	}
	var walk func(reflect.Type)
	walk = func(rt reflect.Type) {
		for i := 0; i < rt.NumField(); i++ {
			f := rt.Field(i)
			ft := f.Type
			if ft.Kind() == reflect.Ptr {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct && f.Tag.Get("env") == "" {
				walk(ft)
				continue
			}
			def := f.Tag.Get("default")
			for _, name := range requiredEnvNames {
				if f.Tag.Get("env") == name && def != "" {
					t.Errorf("%s 不得声明 default 标签", name)
				}
			}
			if strings.Contains(def, "root:root") || strings.Contains(def, "@tcp(") {
				t.Errorf("%s.%s 的 default 标签包含数据库凭据", rt.Name(), f.Name)
			}
		}
	}
	walk(reflect.TypeOf(Config{}))
	walk(reflect.TypeOf(ExternalConfig{}))
}
