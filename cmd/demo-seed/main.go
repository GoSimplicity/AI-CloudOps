// demo-seed is an opt-in local development tool, never an application service.
package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/netip"
	"os"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/mock/demo"
	"github.com/GoSimplicity/AI-CloudOps/pkg/envmode"
	driver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func localConfig(raw, database string) (*driver.Config, error) {
	invalid := errors.New("demo seed requires a literal loopback TCP DSN and matching explicit database")
	cfg, err := driver.ParseDSN(raw)
	if err != nil || database == "" || cfg.DBName != database || cfg.Net != "tcp" || cfg.MultiStatements || cfg.AllowAllFiles || cfg.AllowCleartextPasswords || cfg.AllowFallbackToPlaintext {
		return nil, invalid
	}
	host, port, err := net.SplitHostPort(cfg.Addr)
	if err != nil || port == "" {
		return nil, invalid
	}
	ip, err := netip.ParseAddr(host)
	if err != nil || !ip.IsLoopback() {
		return nil, invalid
	}
	switch strings.ToLower(cfg.DBName) {
	case "mysql", "sys", "information_schema", "performance_schema":
		return nil, invalid
	}
	cfg.Logger = &driver.NopLogger{}
	cfg.ParseTime = true
	cfg.Timeout = 3 * time.Second
	cfg.ReadTimeout = 10 * time.Second
	cfg.WriteTimeout = 10 * time.Second
	return cfg, nil
}
func run() error {
	apply := flag.Bool("apply", false, "insert missing local demo fixtures (default: print plan only)")
	database := flag.String("database", "", "explicit locally owned database name, required for apply")
	owner := flag.Int("owner-id", 0, "existing local user ID, required for apply")
	flag.Parse()
	if !*apply {
		return json.NewEncoder(os.Stdout).Encode(demo.Plan())
	}
	if envmode.IsProduction() {
		return errors.New("demo seed requires ENV=development")
	}
	if *owner <= 0 {
		return errors.New("demo seed requires an existing positive owner ID")
	}
	raw := os.Getenv("CLOUDOPS_DEMO_DSN")
	_ = os.Unsetenv("CLOUDOPS_DEMO_DSN")
	cfg, err := localConfig(raw, *database)
	if err != nil {
		return err
	}
	connector, err := driver.NewConnector(cfg)
	if err != nil {
		return errors.New("local demo connector rejected")
	}
	pool := sql.OpenDB(connector)
	defer pool.Close()
	pool.SetMaxOpenConns(1)
	pool.SetMaxIdleConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	if err := pool.PingContext(ctx); err != nil {
		return errors.New("local demo database is not available")
	}
	// One pinned connection owns the advisory lock throughout the transaction.
	connection, err := pool.Conn(ctx)
	if err != nil {
		return errors.New("local demo connection failed")
	}
	defer connection.Close()
	databaseHash := sha256.Sum256([]byte(*database))
	lockName := fmt.Sprintf("cloudops-demo-local-v1:%x", databaseHash[:16])
	var locked int
	if err := connection.QueryRowContext(ctx, "SELECT GET_LOCK(?, 0)", lockName).Scan(&locked); err != nil || locked != 1 {
		return errors.New("another local demo seed is running")
	}
	defer func() {
		release, stop := context.WithTimeout(context.Background(), 2*time.Second)
		defer stop()
		_, _ = connection.ExecContext(release, "SELECT RELEASE_LOCK(?)", lockName)
	}()
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: connection, SkipInitializeWithVersion: true}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent), DisableAutomaticPing: true})
	if err != nil {
		return errors.New("local demo database initialization failed")
	}
	report, err := demo.Apply(ctx, db, *owner)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(report)
}
