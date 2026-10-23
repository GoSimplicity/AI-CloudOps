package config

import (
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestMySQLResourceBounds(t *testing.T) {
	for _, tt := range []struct {
		name     string
		settings MySQLConfig
		invalid  bool
	}{
		{"zero uses finite defaults", MySQLConfig{}, false},
		{"small pool", MySQLConfig{MaxOpenConns: 1, MaxIdleConns: 1}, false},
		{"negative open", MySQLConfig{MaxOpenConns: -1}, true},
		{"excessive open", MySQLConfig{MaxOpenConns: 10001}, true},
		{"idle exceeds pool", MySQLConfig{MaxOpenConns: 2, MaxIdleConns: 3}, true},
		{"negative idle", MySQLConfig{MaxIdleConns: -1}, true},
		{"negative lifetime", MySQLConfig{ConnMaxLifetime: -time.Second}, true},
		{"negative idle lifetime", MySQLConfig{ConnMaxIdleTime: -time.Second}, true},
		{"negative dial timeout", MySQLConfig{ConnectTimeout: -time.Second}, true},
		{"negative startup timeout", MySQLConfig{StartupTimeout: -time.Second}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.settings.Validate(); (err != nil) != tt.invalid {
				t.Fatalf("unexpected validation: %v", err)
			}
		})
	}
	defaults := (MySQLConfig{}).WithDefaults()
	if defaults.MaxOpenConns != 20 || defaults.MaxIdleConns != 5 || defaults.ConnMaxLifetime != 30*time.Minute || defaults.ConnMaxIdleTime != 5*time.Minute || defaults.ConnectTimeout != 5*time.Second || defaults.StartupTimeout != 10*time.Second {
		t.Fatal("resource defaults changed")
	}
}

func TestMySQLSettingsFromEnvironment(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	setDefaults()
	bindEnvVars()
	t.Setenv("MYSQL_MAX_OPEN_CONNS", "4")
	t.Setenv("MYSQL_MAX_IDLE_CONNS", "2")
	t.Setenv("MYSQL_CONN_MAX_LIFETIME", "20m")
	t.Setenv("MYSQL_CONN_MAX_IDLE_TIME", "2m")
	t.Setenv("MYSQL_CONNECT_TIMEOUT", "3s")
	t.Setenv("MYSQL_STARTUP_TIMEOUT", "8s")
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.MySQL.MaxOpenConns != 4 || cfg.MySQL.MaxIdleConns != 2 || cfg.MySQL.ConnMaxLifetime != 20*time.Minute || cfg.MySQL.ConnMaxIdleTime != 2*time.Minute || cfg.MySQL.ConnectTimeout != 3*time.Second || cfg.MySQL.StartupTimeout != 8*time.Second {
		t.Fatal("environment resource settings not loaded")
	}
}
