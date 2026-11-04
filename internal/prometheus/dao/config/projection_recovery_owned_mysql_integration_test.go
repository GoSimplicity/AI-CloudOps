//go:build integration

package config

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

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	driver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const recoveryMySQLOwnerLabel = "com.cloudops.projection-recovery.owner"

// This fixture never accepts an existing DSN, port, container or database.
func recoveryMySQLLocalEndpoint(ctx context.Context) (string, error) {
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
func TestProjectionRecoveryMySQLOwnedRejectsRemoteOverrides(t *testing.T) {
	for _, key := range []string{"DOCKER_HOST", "DOCKER_CONTEXT"} {
		t.Run(key, func(t *testing.T) {
			t.Setenv("DOCKER_HOST", "")
			t.Setenv("DOCKER_CONTEXT", "")
			t.Setenv("PATH", "")
			t.Setenv(key, "remote-must-not-be-used")
			if _, err := recoveryMySQLLocalEndpoint(context.Background()); err == nil {
				t.Fatal("override was accepted before invoking Docker")
			}
		})
	}
}

type recoveryOwnedMySQL struct {
	a, b    *gorm.DB
	admin   *sql.DB
	address string
	schema  string
	user    string
}

func newRecoveryOwnedMySQL(t *testing.T) *recoveryOwnedMySQL {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	endpoint, err := recoveryMySQLLocalEndpoint(ctx)
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
			remaining, probeErr := command("ps", "--all", "--filter", "label="+recoveryMySQLOwnerLabel+"="+token, "--format", "{{.ID}}")
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
		remaining, err := command("ps", "--all", "--filter", "label="+recoveryMySQLOwnerLabel+"="+token, "--format", "{{.ID}}")
		if err != nil || remaining != "" {
			t.Error("owned recovery MySQL cleanup was not verified")
		}
	})
	attempted = true
	out, err := command("run", "--detach", "--pull=never", "--name", name, "--label", recoveryMySQLOwnerLabel+"="+token, "--memory=1g", "--pids-limit=256", "--env-file", envFile, "--mount", "type=bind,src="+rootFile+",dst=/run/secrets/root-password,readonly", "--mount", "type=bind,src="+runtimeFile+",dst=/run/secrets/runtime-password,readonly", "--publish", "127.0.0.1::3306", "mysql:8.4.11")
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
	write("runtime.dsn", runtime.FormatDSN())
	open := func() *gorm.DB {
		t.Helper()
		connector, err := driver.NewConnector(&runtime)
		if err != nil {
			t.Fatal("private runtime connector unavailable")
		}
		pool := sql.OpenDB(connector)
		t.Cleanup(func() { _ = pool.Close() })
		db, err := gorm.Open(mysql.New(mysql.Config{Conn: pool}), &gorm.Config{Logger: logger.Discard})
		if err != nil {
			t.Fatal("private runtime pool unavailable")
		}
		pool.SetMaxOpenConns(1)
		return db
	}
	f := &recoveryOwnedMySQL{a: open(), b: open(), admin: admin, address: address, schema: schema, user: user}
	var aid, bid uint64
	var isolation string
	if f.a.Raw("SELECT CONNECTION_ID()").Scan(&aid).Error != nil || f.b.Raw("SELECT CONNECTION_ID()").Scan(&bid).Error != nil || aid == 0 || bid == 0 || aid == bid {
		t.Fatal("two independent runtime sessions required")
	}
	if f.a.Raw("SELECT @@transaction_isolation").Scan(&isolation).Error != nil || isolation != "REPEATABLE-READ" {
		t.Fatal("owned fixture must report the supported MySQL RR isolation")
	}
	t.Logf("owned runtime connection ids %d/%d; isolation %s", aid, bid, isolation)
	var tables int64
	if f.a.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE()").Scan(&tables).Error != nil || tables != 0 {
		t.Fatal("new fixture schema was not empty")
	}
	if f.a.AutoMigrate(&model.MonitorConfig{}, &model.MonitorConfigNamespaceGuard{}, &model.MonitorConfigPoolControl{}, &model.MonitorConfigPoolOperation{}, &model.MonitorConfigArtifactEvent{}) != nil {
		t.Fatal("private five-model migration failed")
	}
	return f
}
func TestProjectionRecoveryOwnedMySQL(t *testing.T) {
	if os.Getenv("CLOUDOPS_TEST_PROJECTION_RECOVERY_MYSQL") != "true" {
		t.Skip("requires explicit owned recovery MySQL flag")
	}
	fixture := newRecoveryOwnedMySQL(t)
	runRecoveryMySQLCases(t, fixture)
}
