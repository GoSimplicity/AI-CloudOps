// Command schema inspects and explicitly initializes versioned MySQL schemas.
// It never loads application .env files or uses runtime database credentials.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/GoSimplicity/AI-CloudOps/internal/dbschema"
	"github.com/GoSimplicity/AI-CloudOps/pkg/schema"
	"github.com/go-sql-driver/mysql"
	"golang.org/x/sys/unix"
)

const credentialEnvironment = "CLOUDOPS_SCHEMA_DSN"

var errCredentials = errors.New("dedicated schema credentials are missing or invalid")

type options struct {
	command              string
	dsnFile              string
	timeout              time.Duration
	confirm              bool
	confirmClock         bool
	zone                 string
	database             string
	runtimeName          string
	runtimeHost          string
	migratorName         string
	migratorHost         string
	section              string
	format               string
	confirmLegacy        bool
	noteFile             string
	upgradeExecute       bool
	upgradeNoteFile      string
	backupProofFile      string
	attestHistoricalTime bool
}

type schemaDatabase interface {
	Verify(context.Context, []schema.Migration) error
	PreflightFresh(context.Context) error
	Migrate(context.Context, []schema.Migration) error
	VerifyClock(context.Context, string) error
	InitializeClock(context.Context, string) error
	ValidateLegacy(context.Context) error
	AdoptBaseline(context.Context, []schema.Migration, string) error
	UpgradeBaseline(context.Context, []schema.Migration, schema.UpgradeEvidence) error
	Close() error
}

type dependencies struct {
	manifest func() ([]schema.Migration, error)
	getenv   func(string) string
	open     func(string, time.Duration) (schemaDatabase, error)
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Stdout, os.Stderr, dependencies{
		manifest: dbschema.Manifest,
		getenv:   os.Getenv,
		open:     openDatabase,
	}))
}

func parseOptions(args []string) (options, error) {
	result := options{command: "plan", timeout: 2 * time.Minute}
	if len(args) == 0 {
		return result, nil
	}
	result.command = args[0]
	if result.command != "plan" && result.command != "verify" && result.command != "migrate" && result.command != "init-clock" && result.command != "verify-clock" && result.command != "roles" && result.command != "preflight-legacy" && result.command != "adopt" && result.command != knownUpgradeCommand {
		return options{}, errors.New("expected plan, verify, migrate, clock, roles, legacy inspection/adoption or upgrade-a8f1700")
	}
	flags := flag.NewFlagSet("schema", flag.ContinueOnError)
	// flag errors can include supplied values. Credentials must never appear in
	// diagnostics, including accidentally pasted DSNs in unsupported arguments.
	flags.SetOutput(io.Discard)
	flags.StringVar(&result.dsnFile, "dsn-file", "", "private DSN file")
	flags.DurationVar(&result.timeout, "timeout", result.timeout, "operation deadline")
	flags.BoolVar(&result.confirm, "confirm-new-database", false, "allow fresh baseline DDL")
	flags.BoolVar(&result.confirmClock, "confirm-no-writers", false, "confirm every database writer and migration job is stopped")
	flags.StringVar(&result.zone, "notification-timezone", "", "explicit UTC or IANA timezone")
	flags.StringVar(&result.database, "database", "", "target database name for offline GRANT plan")
	flags.StringVar(&result.runtimeName, "runtime-role", "", "existing runtime role name")
	flags.StringVar(&result.runtimeHost, "runtime-role-host", "", "explicit runtime role host")
	flags.StringVar(&result.migratorName, "migrator-role", "", "existing migrator role name")
	flags.StringVar(&result.migratorHost, "migrator-role-host", "", "explicit migrator role host")
	flags.StringVar(&result.section, "section", "", "roles plan section: migrator, runtime or all")
	flags.StringVar(&result.format, "format", "", "roles plan output: sql or json")
	flags.BoolVar(&result.confirmLegacy, "confirm-legacy-schema", false, "confirm verified existing version-1 schema adoption")
	flags.StringVar(&result.noteFile, "adoption-note-file", "", "owner-only single-line adoption audit note file")
	flags.BoolVar(&result.upgradeExecute, "execute", false, "execute the explicit supported-source upgrade instead of its offline plan")
	flags.StringVar(&result.upgradeNoteFile, "upgrade-note-file", "", "owner-only single-line upgrade audit note")
	flags.StringVar(&result.backupProofFile, "backup-proof-file", "", "owner-only single-line JSON backup attestation")
	flags.BoolVar(&result.attestHistoricalTime, "attest-historical-time", false, "explicitly attest unchanged historical DATETIME interpretation")
	if err := flags.Parse(args[1:]); err != nil || flags.NArg() != 0 {
		return options{}, errors.New("invalid command options; use --help")
	}
	if result.timeout < time.Second || result.timeout > 30*time.Minute {
		return options{}, errors.New("timeout must be between 1s and 30m")
	}
	if result.command == "migrate" && !result.confirm {
		return options{}, errors.New("migrate requires --confirm-new-database")
	}
	if result.command != "migrate" && result.confirm {
		return options{}, errors.New("confirmation is only valid for migrate")
	}
	if result.command == "init-clock" || result.command == "verify-clock" || (result.command == knownUpgradeCommand && result.upgradeExecute) {
		if dbschema.ValidateClockZone(result.zone) != nil {
			return options{}, errors.New("clock commands require an explicit valid notification timezone")
		}
		if result.command == "init-clock" && !result.confirmClock {
			return options{}, errors.New("init-clock requires --confirm-no-writers")
		}
	} else if result.zone != "" {
		return options{}, errors.New("notification-timezone is only valid for clock commands")
	}
	if result.command != "init-clock" && result.command != "adopt" && result.command != knownUpgradeCommand && result.confirmClock {
		return options{}, errors.New("confirm-no-writers is only valid for init-clock or adopt")
	}
	if result.command == "plan" && result.dsnFile != "" {
		return options{}, errors.New("plan is offline and does not accept credentials")
	}
	if result.command == "adopt" {
		if !result.confirmLegacy || !result.confirmClock || result.noteFile == "" {
			return options{}, errors.New("adopt requires --confirm-legacy-schema, --confirm-no-writers and --adoption-note-file")
		}
	} else if result.confirmLegacy || result.noteFile != "" {
		return options{}, errors.New("adoption options are only valid for adopt")
	}
	if result.command == knownUpgradeCommand {
		if result.upgradeExecute {
			if !result.confirmClock || result.upgradeNoteFile == "" || result.backupProofFile == "" {
				return options{}, errors.New("upgrade execution requires --confirm-no-writers, --upgrade-note-file, --backup-proof-file and --notification-timezone")
			}
		} else if result.confirmClock || result.upgradeNoteFile != "" || result.backupProofFile != "" || result.attestHistoricalTime || result.dsnFile != "" {
			return options{}, errors.New("upgrade plan is offline; execution options require --execute")
		}
	} else if result.upgradeExecute || result.upgradeNoteFile != "" || result.backupProofFile != "" || result.attestHistoricalTime {
		return options{}, errors.New("upgrade options are only valid for upgrade-a8f1700")
	}
	if result.command == "roles" {
		if result.section == "" {
			result.section = "all"
		}
		if result.format == "" {
			result.format = "sql"
		}
		if (result.section != "all" && result.section != "migrator" && result.section != "runtime") || (result.format != "sql" && result.format != "json") || (result.format == "json" && result.section == "all") {
			return options{}, errors.New("roles requires a valid section and format; JSON requires migrator or runtime")
		}
		if result.dsnFile != "" {
			return options{}, errors.New("roles is offline and does not accept credentials")
		}
		if _, err := BuildRolePlan(result.database, RoleAccount{result.runtimeName, result.runtimeHost}, RoleAccount{result.migratorName, result.migratorHost}); err != nil {
			return options{}, errRolePlan
		}
	} else if result.database != "" || result.runtimeName != "" || result.runtimeHost != "" || result.migratorName != "" || result.migratorHost != "" || result.section != "" || result.format != "" {
		return options{}, errors.New("role options are only valid for roles")
	}
	return result, nil
}

func run(ctx context.Context, args []string, out, diagnostics io.Writer, deps dependencies) int {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprintln(out, "Usage: schema [plan | verify | migrate --confirm-new-database | verify-clock --notification-timezone ZONE | init-clock --confirm-no-writers --notification-timezone ZONE] [--dsn-file PATH] [--timeout 2m]")
		fmt.Fprintln(out, "Known source: upgrade-a8f1700 is an offline plan; --execute requires no-writers, private note/backup attestation and explicit notification timezone. Historical times need separate attestation.")
		fmt.Fprintln(out, "Legacy inspection: preflight-legacy. Adoption: adopt --confirm-legacy-schema --confirm-no-writers --adoption-note-file PATH. These never convert historical timestamps.")
		fmt.Fprintln(out, "Default plan and roles are offline. roles requires --database, --runtime-role, --runtime-role-host, --migrator-role and --migrator-role-host. Database commands use a private DSN file or CLOUDOPS_SCHEMA_DSN; no application .env is loaded.")
		return 0
	}
	config, err := parseOptions(args)
	if err != nil {
		fmt.Fprintln(diagnostics, err.Error())
		return 2
	}
	manifest, err := deps.manifest()
	if err != nil || schema.Validate(manifest) != nil {
		fmt.Fprintln(diagnostics, "embedded migration manifest is invalid")
		return 1
	}
	if config.command == knownUpgradeCommand && !config.upgradeExecute {
		plan, err := knownUpgradePlan()
		if err != nil {
			fmt.Fprintln(diagnostics, "immutable known-source upgrade inputs are invalid")
			return 1
		}
		fmt.Fprintln(out, "Offline known-source upgrade plan only; no credentials read or database connected. 15 frozen schema/repair statements plus explicit timezone registration.")
		fmt.Fprintf(out, "source=%s source_checksum=%s implementation_checksum=%s target_checksum=%s\n", plan.Source, plan.SourceChecksum, plan.ImplementationChecksum, manifest[0].Checksum)
		fmt.Fprintln(out, "Backup SHA and operator attestation do not prove automatic full-stack restore. Only the exact source schema is supported.")
		return 0
	}
	if config.command == "roles" {
		plan, err := BuildRolePlan(config.database, RoleAccount{config.runtimeName, config.runtimeHost}, RoleAccount{config.migratorName, config.migratorHost})
		if err != nil {
			fmt.Fprintln(diagnostics, "invalid offline role plan")
			return 2
		}
		if config.format == "json" {
			statements := plan.Migrator
			if config.section == "runtime" {
				statements = plan.Runtime
			}
			statements = append([]string(nil), statements...)
			for index, statement := range statements {
				statements[index] = strings.TrimSuffix(statement, ";")
			}
			payload := struct {
				Version    int      `json:"version"`
				Database   string   `json:"database"`
				Section    string   `json:"section"`
				Statements []string `json:"statements"`
			}{1, config.database, config.section, statements}
			if json.NewEncoder(out).Encode(payload) != nil {
				fmt.Fprintln(diagnostics, "role plan output failed")
				return 1
			}
			return 0
		}
		fmt.Fprintln(out, "-- Offline GRANT plan only. No users, credentials or permissions have been changed.")
		if config.section == "all" || config.section == "migrator" {
			fmt.Fprintln(out, "-- Migrator: apply before explicit migration to separately provisioned roles.")
			for _, statement := range plan.Migrator {
				fmt.Fprintln(out, strings.TrimSuffix(statement, ";")+";")
			}
		}
		if config.section == "all" || config.section == "runtime" {
			fmt.Fprintln(out, "-- Runtime: apply only after all baseline tables exist. Review and remove inherited broad grants separately.")
			for _, statement := range plan.Runtime {
				fmt.Fprintln(out, strings.TrimSuffix(statement, ";")+";")
			}
		}
		return 0
	}
	if config.command == "plan" {
		fmt.Fprintln(out, "Manifest only: database state has not been inspected; no changes executed.")
		for _, migration := range manifest {
			fmt.Fprintf(out, "version=%d name=%s checksum=%s\n", migration.Version, migration.Name, migration.Checksum)
		}
		return 0
	}
	var adoptionNote string
	if config.command == "adopt" {
		var err error
		adoptionNote, err = readCredentials(config.noteFile, func(string) string { return "" })
		if err != nil || len(adoptionNote) > 1000 || !utf8.ValidString(adoptionNote) {
			fmt.Fprintln(diagnostics, "adoption requires a valid owner-only note file of at most 1000 UTF-8 bytes")
			return 2
		}
	}
	var upgradeEvidence schema.UpgradeEvidence
	if config.command == knownUpgradeCommand {
		var err error
		upgradeEvidence, err = readUpgradeEvidence(config)
		if err != nil {
			fmt.Fprintln(diagnostics, errUpgradeEvidence.Error())
			return 2
		}
	}
	dsn, err := readCredentials(config.dsnFile, deps.getenv)
	if err != nil {
		fmt.Fprintln(diagnostics, errCredentials.Error())
		return 2
	}
	db, err := deps.open(dsn, config.timeout)
	if err != nil {
		fmt.Fprintln(diagnostics, "database connection configuration is invalid")
		return 1
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(ctx, config.timeout)
	defer cancel()
	if config.command == knownUpgradeCommand {
		if err := db.UpgradeBaseline(ctx, manifest, upgradeEvidence); err != nil {
			if errors.Is(err, schema.ErrPreflight) {
				fmt.Fprintln(diagnostics, "known-source preflight rejected: inspect exact 49-table source, direct table/trigger metadata privileges, zero triggers, disabled unsupported providers and historical-time attestation; no automatic repair is performed")
			} else {
				fmt.Fprintln(diagnostics, safeError(err))
			}
			return 1
		}
		if err := db.Verify(ctx, manifest); err != nil {
			fmt.Fprintln(diagnostics, "upgrade returned but schema history verification failed; inspect before retry")
			return 1
		}
		if err := db.VerifyClock(ctx, config.zone); err != nil {
			fmt.Fprintln(diagnostics, "upgrade schema is present but notification timezone verification failed; inspect before retry")
			return 1
		}
		fmt.Fprintln(out, "Version-1 schema history and explicit timezone verified. Inspect ledger upgrade provenance; backup attestation does not certify full-stack recovery. Historical timestamps and legacy queues were not converted or replayed.")
		return 0
	}
	if config.command == "preflight-legacy" {
		if err := db.ValidateLegacy(ctx); err != nil {
			fmt.Fprintln(diagnostics, safeError(err))
			return 1
		}
		fmt.Fprintln(out, "Existing schema and repair postconditions match version 1. No ledger or application data changed; adoption has not been performed.")
		return 0
	}
	if config.command == "adopt" {
		if err := db.AdoptBaseline(ctx, manifest, adoptionNote); err != nil {
			fmt.Fprintln(diagnostics, safeError(err))
			return 1
		}
		if err := db.Verify(ctx, manifest); err != nil {
			fmt.Fprintln(diagnostics, "adoption returned but history verification failed; inspect state before retry")
			return 1
		}
		fmt.Fprintln(out, "Schema history verified. Existing business data and notification timezone have not been converted. Inspect ledger adoption audit separately.")
		return 0
	}
	verification := db.Verify(ctx, manifest)
	if config.command == "init-clock" || config.command == "verify-clock" {
		if verification != nil {
			fmt.Fprintln(diagnostics, safeError(verification))
			return 1
		}
		if config.command == "init-clock" {
			if err := db.InitializeClock(ctx, config.zone); err != nil {
				fmt.Fprintln(diagnostics, safeError(err))
				return 1
			}
		}
		if err := db.VerifyClock(ctx, config.zone); err != nil {
			fmt.Fprintln(diagnostics, safeError(err))
			return 1
		}
		fmt.Fprintln(out, "Notification timezone marker verified. Historical timestamps have not been converted.")
		return 0
	}
	if config.command == "verify" || verification == nil {
		if verification != nil {
			fmt.Fprintln(diagnostics, safeError(verification))
			return 1
		}
		fmt.Fprintln(out, "Schema history matches the embedded manifest. No changes executed.")
		return 0
	}
	// Dirty/drift outcomes must never flow into migration automatically. A missing
	// ledger is expected for a new database, but cannot establish freshness alone.
	if !errors.Is(verification, schema.ErrPreflight) && !errors.Is(verification, schema.ErrPending) {
		fmt.Fprintln(diagnostics, safeError(verification))
		return 1
	}
	if err := db.PreflightFresh(ctx); err != nil {
		fmt.Fprintln(diagnostics, safeError(err))
		return 1
	}
	if err := db.Migrate(ctx, manifest); err != nil {
		fmt.Fprintln(diagnostics, safeError(err))
		return 1
	}
	if err := db.Verify(ctx, manifest); err != nil {
		fmt.Fprintln(diagnostics, "migration returned but verification failed; inspect the database before retry")
		return 1
	}
	fmt.Fprintln(out, "Fresh baseline applied and schema history verified. Application startup is a separate step.")
	return 0
}

func safeError(err error) string {
	switch {
	case errors.Is(err, dbschema.ErrLegacyStructure):
		return "legacy schema differs from this release; no adoption or automatic repair is permitted"
	case errors.Is(err, dbschema.ErrLegacyData):
		return "legacy data does not satisfy release repair postconditions; no changes were executed"
	case errors.Is(err, dbschema.ErrLegacyVisibility):
		return "legacy inspection requires proven complete schema metadata visibility"
	case errors.Is(err, schema.ErrAdoption):
		return "invalid explicit baseline adoption request"
	case errors.Is(err, dbschema.ErrClockInvalidZone):
		return "notification timezone must be explicit UTC or a valid IANA timezone"
	case errors.Is(err, dbschema.ErrClockMissing):
		return "notification timezone marker is missing; explicit initialization is required"
	case errors.Is(err, dbschema.ErrClockMismatch):
		return "notification timezone differs from the recorded marker; no automatic conversion is permitted"
	case errors.Is(err, dbschema.ErrClockHistory):
		return "notification history exists without a timezone marker; operator reconciliation is required"
	case errors.Is(err, schema.ErrUncertain):
		return "schema operation outcome is uncertain; inspect the database before retry"
	case errors.Is(err, schema.ErrDirty):
		return "schema history is dirty; reconcile partially committed DDL before retry"
	case errors.Is(err, schema.ErrDrift):
		return "schema history differs from this release; no automatic repair is permitted"
	case errors.Is(err, dbschema.ErrLegacy):
		return "unversioned database is not empty; explicit legacy adoption is required"
	case errors.Is(err, schema.ErrLocked):
		return "schema migration lock is unavailable"
	case errors.Is(err, schema.ErrPending):
		return "schema migrations are pending; verify does not execute migrations"
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return "schema operation was canceled or timed out; inspect state before retrying a write"
	default:
		return "schema inspection failed; check database access and history without exposing credentials"
	}
}

func readCredentials(path string, getenv func(string) string) (string, error) {
	if path == "" {
		value := strings.TrimSpace(getenv(credentialEnvironment))
		if value == "" || len(value) > 64*1024 || strings.ContainsAny(value, "\r\n\x00") {
			return "", errCredentials
		}
		return value, nil
	}
	before, err := os.Lstat(path)
	if err != nil || !privateFile(before) {
		return "", errCredentials
	}
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return "", errCredentials
	}
	file := os.NewFile(uintptr(fd), "schema credentials")
	defer file.Close()
	after, err := file.Stat()
	if err != nil || !os.SameFile(before, after) || !privateFile(after) {
		return "", errCredentials
	}
	content, err := io.ReadAll(io.LimitReader(file, 64*1024+1))
	if err != nil || len(content) > 64*1024 {
		return "", errCredentials
	}
	value := strings.TrimSpace(string(content))
	if value == "" || strings.ContainsAny(value, "\r\n\x00") {
		return "", errCredentials
	}
	return value, nil
}

func privateFile(info os.FileInfo) bool {
	if info == nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uint32(os.Geteuid())
}

type sqlDatabase struct{ db *sql.DB }

func (db sqlDatabase) Verify(ctx context.Context, manifest []schema.Migration) error {
	return schema.Verify(ctx, db.db, manifest)
}
func (db sqlDatabase) PreflightFresh(ctx context.Context) error {
	conn, err := db.db.Conn(ctx)
	if err != nil {
		return schema.ErrPreflight
	}
	defer conn.Close()
	return dbschema.PreflightFresh(ctx, conn)
}
func (db sqlDatabase) Migrate(ctx context.Context, manifest []schema.Migration) error {
	return schema.Migrate(ctx, db.db, manifest)
}
func (db sqlDatabase) VerifyClock(ctx context.Context, zone string) error {
	return dbschema.VerifyClock(ctx, db.db, zone)
}
func (db sqlDatabase) InitializeClock(ctx context.Context, zone string) error {
	return dbschema.InitializeFreshClock(ctx, db.db, zone)
}
func (db sqlDatabase) ValidateLegacy(ctx context.Context) error {
	return schema.WithMigrationLock(ctx, db.db, func(conn *sql.Conn) error { return dbschema.ValidateLegacyCurrent(ctx, conn) })
}
func (db sqlDatabase) AdoptBaseline(ctx context.Context, manifest []schema.Migration, note string) error {
	return schema.AdoptBaseline(ctx, db.db, manifest, dbschema.ValidateLegacyCurrent, note)
}
func (db sqlDatabase) UpgradeBaseline(ctx context.Context, manifest []schema.Migration, evidence schema.UpgradeEvidence) error {
	plan, err := knownUpgradePlan()
	if err != nil {
		return err
	}
	return schema.UpgradeBaseline(ctx, db.db, manifest, plan, evidence)
}
func (db sqlDatabase) Close() error { return db.db.Close() }

func databaseConfig(dsn string, deadline time.Duration) (*mysql.Config, error) {
	config, err := mysql.ParseDSN(dsn)
	if err != nil || config.DBName == "" || config.MultiStatements || config.AllowAllFiles || config.AllowCleartextPasswords || config.AllowFallbackToPlaintext {
		return nil, errCredentials
	}
	config.Logger = &mysql.NopLogger{}
	// Both context cancellation and transport deadlines bound failed handshakes
	// and stalled SQL I/O. Keep stricter operator-provided timeout settings.
	capDuration := func(value, limit time.Duration) time.Duration {
		if limit > deadline {
			limit = deadline
		}
		if value <= 0 || value > limit {
			return limit
		}
		return value
	}
	config.Timeout = capDuration(config.Timeout, 5*time.Second)
	config.ReadTimeout = capDuration(config.ReadTimeout, 30*time.Second)
	config.WriteTimeout = capDuration(config.WriteTimeout, 30*time.Second)
	return config, nil
}

func openDatabase(dsn string, timeout time.Duration) (schemaDatabase, error) {
	config, err := databaseConfig(dsn, timeout)
	if err != nil {
		return nil, errCredentials
	}
	connector, err := mysql.NewConnector(config)
	if err != nil {
		return nil, errCredentials
	}
	db := sql.OpenDB(connector)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	return sqlDatabase{db: db}, nil
}
