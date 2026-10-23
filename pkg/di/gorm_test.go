package di

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"gorm.io/gorm"
)

func TestMySQLDSNTimeoutPolicy(t *testing.T) {
	for _, tt := range []struct {
		name, dsn            string
		timeout, read, write time.Duration
		invalid              bool
	}{
		{name: "default dial", dsn: "u:p@tcp(localhost:3306)/db", timeout: 5 * time.Second},
		{name: "explicit timeouts", dsn: "u:p@tcp(localhost:3306)/db?timeout=9s&readTimeout=7s&writeTimeout=4s", timeout: 9 * time.Second, read: 7 * time.Second, write: 4 * time.Second},
		{name: "explicit zero", dsn: "u:p@tcp(localhost:3306)/db?timeout=0s"},
		{name: "question in password", dsn: "u:p?timeout=0s@tcp(localhost:3306)/db", timeout: 5 * time.Second},
		{name: "encoded key remains driver parameter", dsn: "u:p@tcp(localhost:3306)/db?time%6fut=0s", timeout: 5 * time.Second},
		{name: "bare key does not disable default", dsn: "u:p@tcp(localhost:3306)/db?timeout", timeout: 5 * time.Second},
		{name: "invalid DSN", dsn: "secret-password-invalid", invalid: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := mysqlDSNConfig(config.MySQLConfig{Addr: tt.dsn, ConnectTimeout: 5 * time.Second})
			if (err != nil) != tt.invalid {
				t.Fatal("unexpected DSN validation result")
			}
			if err != nil {
				if strings.Contains(err.Error(), tt.dsn) {
					t.Fatal("DSN leaked")
				}
				return
			}
			if parsed.Timeout != tt.timeout || parsed.ReadTimeout != tt.read || parsed.WriteTimeout != tt.write {
				t.Fatal("explicit timeout policy changed")
			}
		})
	}
}

type memoryConnector struct {
	opened, closed atomic.Int32
	pingFailure    bool
	waitForContext bool
}

func (c *memoryConnector) Connect(context.Context) (driver.Conn, error) {
	c.opened.Add(1)
	return &memorySQLConn{owner: c}, nil
}
func (c *memoryConnector) Driver() driver.Driver { return memorySQLDriver{connector: c} }

type memorySQLDriver struct{ connector *memoryConnector }

func (d memorySQLDriver) Open(string) (driver.Conn, error) {
	return d.connector.Connect(context.Background())
}

type memorySQLConn struct{ owner *memoryConnector }

func (c *memorySQLConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("not needed") }
func (c *memorySQLConn) Close() error                        { c.owner.closed.Add(1); return nil }
func (c *memorySQLConn) Begin() (driver.Tx, error)           { return nil, errors.New("not needed") }
func (c *memorySQLConn) Ping(ctx context.Context) error {
	if c.owner.waitForContext {
		<-ctx.Done()
		return ctx.Err()
	}
	if c.owner.pingFailure {
		return errors.New("driver-private-credential-error")
	}
	return ctx.Err()
}
func (c *memorySQLConn) QueryContext(ctx context.Context, _ string, _ []driver.NamedValue) (driver.Rows, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestSQLPoolStartupClosesFailures(t *testing.T) {
	for _, tt := range []struct {
		name                                        string
		pingFailure, initializationFailure, timeout bool
	}{
		{name: "success"},
		{name: "ping failure", pingFailure: true},
		{name: "schema or ORM failure", initializationFailure: true},
		{name: "startup timeout", timeout: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			connector := &memoryConnector{pingFailure: tt.pingFailure, waitForContext: tt.timeout}
			pool := sql.OpenDB(connector)
			t.Cleanup(func() { _ = pool.Close() })
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
			defer cancel()
			called := false
			_, err := initializeSQLPool(ctx, pool, func(context.Context, *sql.DB) (*gorm.DB, error) {
				called = true
				if tt.initializationFailure {
					return nil, errors.New("initialization failed")
				}
				return &gorm.DB{}, nil
			})
			failure := tt.pingFailure || tt.initializationFailure || tt.timeout
			if (err != nil) != failure {
				t.Fatal("unexpected initialization result")
			}
			if called == (tt.pingFailure || tt.timeout) {
				t.Fatal("unexpected initializer invocation")
			}
			if failure {
				if connector.opened.Load() != connector.closed.Load() {
					t.Fatal("startup failure leaked connections")
				}
				if pool.PingContext(context.Background()) == nil {
					t.Fatal("failed startup pool remains open")
				}
				if strings.Contains(err.Error(), "driver-private") {
					t.Fatal("driver error leaked")
				}
			} else if pool.PingContext(context.Background()) != nil {
				t.Fatal("successful pool was closed")
			}
		})
	}
}

func TestSQLPoolLimitsConnectionAcquisition(t *testing.T) {
	connector := &memoryConnector{}
	pool := sql.OpenDB(connector)
	t.Cleanup(func() { _ = pool.Close() })
	configureSQLPool(pool, config.MySQLConfig{MaxOpenConns: 1, MaxIdleConns: 1, ConnMaxLifetime: time.Minute, ConnMaxIdleTime: time.Minute})
	held, err := pool.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := pool.Conn(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("pool did not bound acquisition: %v", err)
	}
	if pool.Stats().MaxOpenConnections != 1 || connector.opened.Load() != 1 {
		t.Fatal("connection pool exceeded configured capacity")
	}
}

func TestGORMVersionQueryUsesStartupDeadline(t *testing.T) {
	connector := &memoryConnector{}
	pool := sql.OpenDB(connector)
	t.Cleanup(func() { _ = pool.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	startup := &startupPool{DB: pool, ctx: ctx}
	var version string
	err := startup.QueryRowContext(context.Background(), "SELECT VERSION()").Scan(&version)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("startup version probe escaped deadline: %v", err)
	}
}

func TestSQLPoolReleasesExcessIdleConnections(t *testing.T) {
	connector := &memoryConnector{}
	pool := sql.OpenDB(connector)
	t.Cleanup(func() { _ = pool.Close() })
	configureSQLPool(pool, config.MySQLConfig{MaxOpenConns: 3, MaxIdleConns: 1, ConnMaxLifetime: time.Minute, ConnMaxIdleTime: time.Minute})
	first, err := pool.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	second, err := pool.Conn(context.Background())
	if err != nil {
		_ = first.Close()
		t.Fatal(err)
	}
	_ = first.Close()
	_ = second.Close()
	if pool.Stats().Idle != 1 || connector.closed.Load() != 1 {
		t.Fatal("excess idle connections were retained")
	}
}
