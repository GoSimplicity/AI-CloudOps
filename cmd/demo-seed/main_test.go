package main

import "testing"

func TestLocalConfig(t *testing.T) {
	cases := []struct {
		name, dsn, database string
		valid               bool
	}{
		{"ipv4 loopback", "root@tcp(127.0.0.1:3306)/owned_demo", "owned_demo", true},
		{"ipv6 loopback", "root@tcp([::1]:3306)/owned_demo", "owned_demo", true},
		{"unconfirmed database", "root@tcp(127.0.0.1:3306)/owned_demo", "", false},
		{"wrong database", "root@tcp(127.0.0.1:3306)/owned_demo", "other", false},
		{"remote literal", "root@tcp(192.0.2.11:3306)/owned_demo", "owned_demo", false},
		{"DNS forbidden", "root@tcp(localhost:3306)/owned_demo", "owned_demo", false},
		{"unix forbidden", "root@unix(/tmp/mysql.sock)/owned_demo", "owned_demo", false},
		{"multi statements forbidden", "root@tcp(127.0.0.1:3306)/owned_demo?multiStatements=true", "owned_demo", false},
		{"local file import forbidden", "root@tcp(127.0.0.1:3306)/owned_demo?allowAllFiles=true", "owned_demo", false},
		{"system database", "root@tcp(127.0.0.1:3306)/mysql", "mysql", false},
		{"malformed", "fixture-invalid", "owned_demo", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := localConfig(tc.dsn, tc.database)
			if (err == nil) != tc.valid {
				t.Fatal("unexpected acceptance result")
			}
			if tc.valid && (cfg.Timeout <= 0 || cfg.ReadTimeout <= 0 || cfg.WriteTimeout <= 0 || !cfg.ParseTime) {
				t.Fatal("connection limits not set")
			}
		})
	}
}
