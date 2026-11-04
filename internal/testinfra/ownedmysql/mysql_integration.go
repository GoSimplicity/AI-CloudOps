//go:build integration

// Package ownedmysql supplies disposable local MySQL resources to integration
// tests. It imports no application model, DAO, cache, or application config.
package ownedmysql

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	driver "github.com/go-sql-driver/mysql"
)

const OwnerLabel = "com.cloudops.projection-recovery.owner"

// Fixture connections belong only to this new random schema and server. The
// admin connection is for owned lock observation/KILL tests, never production.
// RuntimeDSNFile is private (0600 in a 0700 directory); never log its contents.
// Resource cleanup captures immutable identity, independently of these fields.
type Fixture struct {
	A, B, Admin                                   *sql.DB
	Address, Schema, User                         string
	RuntimeDSNFile                                string
	Endpoint, ContainerID, OwnerLabel, OwnerToken string
}

func LocalEndpoint(ctx context.Context) (string, error) {
	if os.Getenv("DOCKER_HOST") != "" || os.Getenv("DOCKER_CONTEXT") != "" {
		return "", errors.New("local Docker overrides are forbidden")
	}
	out, err := exec.CommandContext(ctx, "docker", "context", "inspect", "--format", "{{.Endpoints.docker.Host}}").Output()
	if err != nil {
		return "", errors.New("local Docker metadata unavailable")
	}
	endpoint := strings.TrimSpace(string(out))
	parsed, err := url.Parse(endpoint)
	if err != nil || !strings.HasPrefix(endpoint, "unix:///") || parsed.Scheme != "unix" || parsed.Host != "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || !filepath.IsAbs(parsed.Path) {
		return "", errors.New("local Unix Docker endpoint required")
	}
	info, err := os.Stat(parsed.Path)
	if err != nil || info.Mode()&os.ModeSocket == 0 {
		return "", errors.New("local Docker socket unavailable")
	}
	return parsed.String(), nil
}

// Start accepts no existing DSN, database, container, or service address. The
// caller must explicitly gate its integration entry before invoking Start.
func Start(t testing.TB) *Fixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	endpoint, err := LocalEndpoint(ctx)
	cancel()
	if err != nil {
		t.Fatal("owned recovery MySQL requires a verified local Docker socket")
	}
	random := func(n int) string {
		t.Helper()
		value := make([]byte, n)
		if _, err := rand.Read(value); err != nil {
			t.Fatal("fixture randomness unavailable")
		}
		return hex.EncodeToString(value)
	}
	token := random(24)
	name := "cloudops-recovery-mysql-" + random(8)
	schema := "cloudops_recovery_" + random(8)
	user := "recovery_" + random(6)
	rootPassword, runtimePassword := random(32), random(32)
	private := t.TempDir()
	if strings.ContainsAny(private, ",\n\r") {
		t.Fatal("private mount path cannot contain Docker mount separators")
	}
	if os.Chmod(private, 0700) != nil {
		t.Fatal("cannot secure private fixture directory")
	}
	write := func(name, body string) string {
		t.Helper()
		p := filepath.Join(private, name)
		if os.WriteFile(p, []byte(body), 0600) != nil {
			t.Fatal("cannot save private fixture input")
		}
		return p
	}
	rootFile := write("root-password", rootPassword)
	runtimeFile := write("runtime-password", runtimePassword)
	envFile := write("mysql.env", "MYSQL_ROOT_PASSWORD_FILE=/run/secrets/root-password\nMYSQL_ROOT_HOST=%\nMYSQL_DATABASE="+schema+"\nMYSQL_USER="+user+"\nMYSQL_PASSWORD_FILE=/run/secrets/runtime-password\n")
	command := func(args ...string) (string, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		out, err := exec.CommandContext(ctx, "docker", append([]string{"--host", endpoint}, args...)...).Output()
		return strings.TrimSpace(string(out)), err
	}
	fullID := ""
	attempted := false
	// Register before start: even an uncertain docker-run result has an owned
	// random-name/label locator. No broad prune or unrelated volume cleanup.
	t.Cleanup(func() {
		if !attempted {
			return
		}
		out, err := command("inspect", "--format", `{{.Id}} {{index .Config.Labels "com.cloudops.projection-recovery.owner"}}`, name)
		if err != nil {
			remaining, probeErr := command("ps", "--all", "--filter", "label="+OwnerLabel+"="+token, "--format", "{{.ID}}")
			if probeErr != nil || remaining != "" {
				t.Error("cannot verify cleanup after an uncertain owned MySQL start")
			}
			return
		}
		parts := strings.Fields(out)
		if len(parts) != 2 || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(parts[0]) || parts[1] != token || fullID != "" && parts[0] != fullID {
			t.Error("refusing mismatched recovery MySQL cleanup")
			return
		}
		if _, err := command("rm", "--force", "--volumes", parts[0]); err != nil {
			t.Error("owned recovery MySQL cleanup failed")
			return
		}
		remaining, err := command("ps", "--all", "--filter", "label="+OwnerLabel+"="+token, "--format", "{{.ID}}")
		if err != nil || remaining != "" {
			t.Error("owned recovery MySQL cleanup was not verified")
		}
	})
	attempted = true
	out, err := command("run", "--detach", "--pull=never", "--name", name, "--label", OwnerLabel+"="+token, "--memory=1g", "--pids-limit=256", "--env-file", envFile, "--mount", "type=bind,src="+rootFile+",dst=/run/secrets/root-password,readonly", "--mount", "type=bind,src="+runtimeFile+",dst=/run/secrets/runtime-password,readonly", "--publish", "127.0.0.1::3306", "mysql:8.4.11")
	if err != nil || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(out) {
		t.Fatal("owned recovery MySQL start failed; raw output withheld")
	}
	fullID = out
	identity, err := command("inspect", "--format", `{{.Id}} {{index .Config.Labels "com.cloudops.projection-recovery.owner"}}`, fullID)
	if err != nil || identity != fullID+" "+token {
		t.Fatal("owned recovery MySQL identity changed")
	}
	address, err := command("port", fullID, "3306/tcp")
	if err != nil {
		t.Fatal("owned loopback port unavailable")
	}
	host, port, err := net.SplitHostPort(address)
	number, portErr := strconv.Atoi(port)
	if err != nil || host != "127.0.0.1" || portErr != nil || number < 1 || number > 65535 || number == 13306 || number == 3306 {
		t.Fatal("owned MySQL must use a fresh dynamic literal loopback port")
	}
	root := driver.NewConfig()
	root.User = "root"
	root.Passwd = rootPassword
	root.Net = "tcp"
	root.Addr = address
	root.DBName = schema
	root.ParseTime = true
	root.Timeout = 2 * time.Second
	root.ReadTimeout = 5 * time.Second
	root.WriteTimeout = 5 * time.Second
	root.Logger = &driver.NopLogger{}
	connector, err := driver.NewConnector(root)
	if err != nil {
		t.Fatal("private admin pool unavailable")
	}
	admin := sql.OpenDB(connector)
	admin.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = admin.Close() })
	readyCtx, readyCancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer readyCancel()
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for admin.PingContext(readyCtx) != nil {
		select {
		case <-readyCtx.Done():
			t.Fatal("owned recovery MySQL readiness timed out")
		case <-ticker.C:
		}
	}
	runtime := *root
	runtime.User = user
	runtime.Passwd = runtimePassword
	runtimeDSNFile := write("runtime.dsn", runtime.FormatDSN())
	openRuntime := func() *sql.DB {
		t.Helper()
		connector, err := driver.NewConnector(&runtime)
		if err != nil {
			t.Fatal("private runtime connector unavailable")
		}
		pool := sql.OpenDB(connector)
		pool.SetMaxOpenConns(1)
		t.Cleanup(func() { _ = pool.Close() })
		pingCtx, pingCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer pingCancel()
		if pool.PingContext(pingCtx) != nil {
			t.Fatal("private runtime connection unavailable")
		}
		return pool
	}
	return &Fixture{A: openRuntime(), B: openRuntime(), Admin: admin, Address: address, Schema: schema, User: user,
		RuntimeDSNFile: runtimeDSNFile, Endpoint: endpoint, ContainerID: fullID, OwnerLabel: OwnerLabel, OwnerToken: token}
}
