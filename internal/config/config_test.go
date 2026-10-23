package config

import (
	"strings"
	"testing"
)

func TestProductionValidation(t *testing.T) {
	t.Setenv("ENV", "production")
	for _, tt := range []struct {
		name    string
		change  func(*Config)
		invalid bool
	}{
		{"valid", func(*Config) {}, false},
		{"short jwt", func(c *Config) { c.JWT.Key1 = "short" }, true},
		{"sample jwt", func(c *Config) { c.JWT.Key1 = "ebe3vx" + strings.Repeat("x", 26) }, true},
		{"same keys", func(c *Config) { c.JWT.Key2 = c.JWT.Key1 }, true},
		{"mock", func(c *Config) { c.Mock.Enabled = true }, true},
		{"missing service key", func(c *Config) { c.AIOps = AIOpsConfig{BaseURL: "http://aiops:8080"} }, true},
		{"invalid target", func(c *Config) { c.AIOps.BaseURL = "file:///tmp/socket" }, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{Server: ServerConfig{Port: "8889"}, Log: LogConfig{Dir: "logs"}, MySQL: MySQLConfig{Addr: "db"}, Redis: RedisConfig{Addr: "redis"}, JWT: JWTConfig{Key1: strings.Repeat("a", 32), Key2: strings.Repeat("b", 32), Issuer: "test-issuer"}, Tree: TreeConfig{PasswordEncryptionKey: strings.Repeat("c", 32)}}
			tt.change(c)
			if err := c.Validate(); (err != nil) != tt.invalid {
				t.Fatalf("unexpected validation result: %v", err)
			}
		})
	}
}
