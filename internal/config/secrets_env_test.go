package config

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
)

var configFiles = []string{"../../config/config.production.yaml", "../../config/config.development.yaml"}

// 敏感字段 -> 注入用环境变量
var secretFields = map[string]string{
	"jwt.key1":                       "JWT_KEY1",
	"jwt.key2":                       "JWT_KEY2",
	"jwt.issuer":                     "JWT_ISSUER",
	"redis.password":                 "REDIS_PASSWORD",
	"mysql.addr":                     "MYSQL_ADDR",
	"tree.password_encryption_key":   "TREE_PASSWORD_ENCRYPTION_KEY",
	"notification.email.username":    "NOTIFICATION_EMAIL_USERNAME",
	"notification.email.password":    "NOTIFICATION_EMAIL_PASSWORD",
	"notification.feishu.app_id":     "NOTIFICATION_FEISHU_APP_ID",
	"notification.feishu.app_secret": "NOTIFICATION_FEISHU_APP_SECRET",
}

func TestConfigFilesContainNoSecrets(t *testing.T) {
	for _, file := range configFiles {
		v := viper.New()
		v.SetConfigFile(file)
		if err := v.ReadInConfig(); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		for key := range secretFields {
			if !v.IsSet(key) {
				t.Errorf("%s: 缺少占位字段 %s", file, key)
				continue
			}
			if v.GetString(key) != "" {
				t.Errorf("%s: %s 不应在配置文件中写入值", file, key) // 不输出值本身
			}
		}
	}
}

func TestSecretsInjectedFromEnvironment(t *testing.T) {
	for _, file := range configFiles {
		t.Run(file, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			if strings.Contains(file, "production") {
				t.Setenv("ENV", "production")
				t.Setenv("MOCK_ENABLED", "false")
			} else {
				t.Setenv("ENV", "development")
			}
			want := map[string]string{}
			for key, env := range secretFields {
				value := "fake-" + strings.ReplaceAll(key, ".", "-") + "-" + strings.Repeat("x", 32)
				want[key] = value
				t.Setenv(env, value)
			}
			want["tree.password_encryption_key"] = strings.Repeat("t", 32)
			t.Setenv("TREE_PASSWORD_ENCRYPTION_KEY", want["tree.password_encryption_key"])
			want["mysql.addr"] = "fake_user:fake_pass@tcp(127.0.0.1:3306)/fake"
			t.Setenv("MYSQL_ADDR", want["mysql.addr"])

			cfg, _, err := loadFromFile(file)
			if err != nil {
				t.Fatalf("加载配置失败: %v", err)
			}
			got := map[string]string{
				"jwt.key1":                       cfg.JWT.Key1,
				"jwt.key2":                       cfg.JWT.Key2,
				"jwt.issuer":                     cfg.JWT.Issuer,
				"redis.password":                 cfg.Redis.Password,
				"mysql.addr":                     cfg.MySQL.Addr,
				"tree.password_encryption_key":   cfg.Tree.PasswordEncryptionKey,
				"notification.email.username":    cfg.Notification.GetEmail().GetUsername(),
				"notification.email.password":    cfg.Notification.GetEmail().GetPassword(),
				"notification.feishu.app_id":     cfg.Notification.GetFeishu().GetAppID(),
				"notification.feishu.app_secret": cfg.Notification.GetFeishu().GetAppSecret(),
			}
			for key, value := range want {
				if got[key] != value {
					t.Errorf("%s 未从环境变量 %s 注入", key, secretFields[key])
				}
			}
		})
	}
}
