//go:build integration

package cache

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	scrapedao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	"github.com/GoSimplicity/AI-CloudOps/internal/testinfra/ownedmysql"
	driver "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type combinedFixtureIdentity struct {
	Endpoint, ContainerID, OwnerLabel, OwnerToken, Address, Schema, User, DSNFile string
}

func combinedSQL(t *testing.T, pool *sql.DB) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: pool}), &gorm.Config{Logger: logger.Discard, IgnoreRelationshipsWhenMigrating: true})
	if err != nil {
		t.Fatal("owned combined SQL adapter unavailable")
	}
	return db
}

func combinedRecordPublisher(t *testing.T, db *gorm.DB, client *redis.Client, store *ProjectionStore) RecordRuleConfigCache {
	t.Helper()
	log := zap.NewNop()
	return NewRecordRuleConfigCache(log, scrapedao.NewScrapePoolDAO(db, log, nil), alertdao.NewAlertManagerRecordDAO(db, log, nil), configdao.NewMonitorConfigDAO(log, db), nil, client, configdao.NewRecordProjectionJournalDAO(log, db), configdao.NewGeneratedConfigHistoryReader(log, db), NewRecordProjectionRunner(store))
}

type combinedCrashHook struct{}

func (combinedCrashHook) DialHook(next redis.DialHook) redis.DialHook { return next }
func (combinedCrashHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
func (combinedCrashHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if cmd.Name() == "eval" {
			args := cmd.Args()
			if len(args) < 3 {
				return next(ctx, cmd)
			}
			keys, ok := args[2].(int)
			if ok && len(args) > keys+3 {
				frame, ok := args[keys+3].(string)
				var w struct {
					Mode string `json:"mode"`
				}
				if ok && json.Unmarshal([]byte(frame), &w) == nil && (w.Mode == "artifact" || w.Mode == "artifact_batch") {
					os.Exit(86)
				}
			}
		}
		return next(ctx, cmd)
	}
}

// Only the parent selects this entry in a second OS process. Without its
// private identity file this entry performs no database or network operation.
func TestProjectionCombinedRecoveryChild(t *testing.T) {
	phase := os.Getenv("CLOUDOPS_COMBINED_CHILD_PHASE")
	if phase == "" {
		return
	}
	if phase != "publish_crash" && phase != "recover" {
		t.Fatal("invalid child phase")
	}
	path := os.Getenv("CLOUDOPS_COMBINED_IDENTITY_FILE")
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > 16384 {
		t.Fatal("private parent identity unavailable")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal("private parent identity unreadable")
	}
	var identity combinedFixtureIdentity
	if json.Unmarshal(raw, &identity) != nil {
		t.Fatal("private parent identity invalid")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	endpoint, err := ownedmysql.LocalEndpoint(ctx)
	if err != nil || endpoint != identity.Endpoint || identity.OwnerLabel != ownedmysql.OwnerLabel || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(identity.ContainerID) {
		t.Fatal("unverified parent Docker identity")
	}
	inspect, err := exec.CommandContext(ctx, "docker", "--host", endpoint, "inspect", "--format", `{{.Id}} {{index .Config.Labels "com.cloudops.projection-recovery.owner"}}`, identity.ContainerID).Output()
	if err != nil || strings.TrimSpace(string(inspect)) != identity.ContainerID+" "+identity.OwnerToken {
		t.Fatal("parent MySQL ownership changed")
	}
	port, err := exec.CommandContext(ctx, "docker", "--host", endpoint, "port", identity.ContainerID, "3306/tcp").Output()
	if err != nil || strings.TrimSpace(string(port)) != identity.Address {
		t.Fatal("parent MySQL binding changed")
	}
	info, err = os.Lstat(identity.DSNFile)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > 16384 {
		t.Fatal("private child DSN unavailable")
	}
	raw, err = os.ReadFile(identity.DSNFile)
	if err != nil {
		t.Fatal("private child DSN unreadable")
	}
	cfg, err := driver.ParseDSN(string(raw))
	if err != nil || cfg.Net != "tcp" || cfg.Addr != identity.Address || cfg.User != identity.User || cfg.DBName != identity.Schema || !regexp.MustCompile(`^cloudops_recovery_[0-9a-f]+$`).MatchString(cfg.DBName) {
		t.Fatal("child DSN is outside owned fixture")
	}
	cfg.Logger = &driver.NopLogger{}
	connector, err := driver.NewConnector(cfg)
	if err != nil {
		t.Fatal("child connector unavailable")
	}
	pool := sql.OpenDB(connector)
	defer pool.Close()
	pool.SetMaxOpenConns(1)
	db := combinedSQL(t, pool)
	client := projectionRedisNative(t)
	store, err := NewProjectionStore(client)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if phase == "publish_crash" {
		p := &model.MonitorScrapePool{Model: model.Model{ID: 25001}, Name: "combined_process", SupportRecord: 1, PrometheusInstances: model.StringList{"127.0.250.1", "127.0.250.2"}}
		if db.Create(p).Error != nil || db.Create(&model.MonitorRecordRule{PoolID: p.ID, Name: "combined_total", Enable: 1, Expr: "sum(up)"}).Error != nil {
			t.Fatal("child source fixture unavailable")
		}
		store.client.AddHook(combinedCrashHook{})
		if err := combinedRecordPublisher(t, db, client, store).GenerateMainConfig(ctx); err != nil {
			t.Fatal("publisher failed before the intended crash boundary")
		}
		t.Fatal("child publisher did not reach its actual artifact command")
	}
	pub := combinedRecordPublisher(t, db, client, store)
	worker := rootRecoveryWorker(t, configdao.NewProjectionRecoveryJournalDAO(zap.NewNop(), db), map[int8]PoolProjectionRecoverer{4: pub.(PoolProjectionRecoverer)})
	if err := worker.RunOnce(ctx); err != nil {
		t.Fatal("new process worker did not recover")
	}
}

func TestProjectionCombinedOwnedRecovery(t *testing.T) {
	if os.Getenv("CLOUDOPS_TEST_PROJECTION_COMBINED") != "true" {
		t.Skip("requires explicit combined owned fixture flag")
	}
	f := ownedmysql.Start(t)
	client := projectionOwnedRedis(t)
	db := combinedSQL(t, f.A)
	if err := db.AutoMigrate(&model.MonitorConfig{}, &model.MonitorConfigNamespaceGuard{}, &model.MonitorConfigPoolControl{}, &model.MonitorConfigPoolOperation{}, &model.MonitorConfigArtifactEvent{}, &model.MonitorScrapePool{}, &model.MonitorRecordRule{}); err != nil {
		t.Fatal("combined private models unavailable")
	}
	identity := combinedFixtureIdentity{f.Endpoint, f.ContainerID, f.OwnerLabel, f.OwnerToken, f.Address, f.Schema, f.User, f.RuntimeDSNFile}
	private := t.TempDir()
	if os.Chmod(private, 0700) != nil {
		t.Fatal("cannot secure combined process evidence")
	}
	encoded, err := json.Marshal(identity)
	if err != nil {
		t.Fatal(err)
	}
	identityFile := filepath.Join(private, "identity.json")
	if os.WriteFile(identityFile, encoded, 0600) != nil {
		t.Fatal("cannot save private fixture identity")
	}
	var processIDs []int
	run := func(t *testing.T, phase string) error {
		t.Helper()
		binary, err := os.Executable()
		if err != nil {
			t.Fatal("test executable unavailable")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, "-test.run=^TestProjectionCombinedRecoveryChild$", "-test.timeout=55s", "-test.v")
		cmd.Env = append(os.Environ(), "CLOUDOPS_COMBINED_CHILD_PHASE="+phase, "CLOUDOPS_COMBINED_IDENTITY_FILE="+identityFile)
		log, err := os.OpenFile(filepath.Join(private, phase+".log"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			t.Fatal("private process log unavailable")
		}
		defer log.Close()
		cmd.Stdout, cmd.Stderr = log, log
		if err := cmd.Start(); err != nil {
			t.Fatal("child process could not start")
		}
		processIDs = append(processIDs, cmd.Process.Pid)
		return cmd.Wait()
	}
	t.Run("crashed publisher then a new process recovers", func(t *testing.T) {
		err := run(t, "publish_crash")
		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 86 {
			t.Fatal("publisher process did not exit at the exact crash boundary; private log withheld")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		journal := configdao.NewProjectionPlanJournalDAO(zap.NewNop(), db)
		pending, err := journal.LoadLastCommittedProjection(ctx, 4, 25001)
		if err != nil || pending == nil || pending.Phase != model.ConfigPublicationCommitted {
			t.Fatal("crash did not leave durable committed state")
		}
		if n, err := client.Exists(ctx, projectionPoolKey(4, 25001, "ack")).Result(); err != nil || n != 0 {
			t.Fatal("crashed process announced false completion")
		}
		if err := journal.WithPoolProjectionLease(ctx, *pending, func(lease configdao.ProjectionLease) error {
			snapshot, err := lease.Snapshot()
			if err != nil {
				return err
			}
			handle, err := compileProjection(nil, lease, snapshot)
			if err != nil {
				return err
			}
			projectionTrack(t, client, handle)
			return nil
		}); err != nil {
			t.Fatal("crash did not release its original SQL locks")
		}
		if err := run(t, "recover"); err != nil {
			t.Fatal("new OS process could not recover pending operation; private log withheld")
		}
		if len(processIDs) != 2 || processIDs[0] == processIDs[1] || processIDs[0] == os.Getpid() || processIDs[1] == os.Getpid() {
			t.Fatal("recovery did not execute in distinct processes")
		}
		projected, err := journal.LoadLastProjectedProjection(ctx, 4, 25001)
		if err != nil || projected == nil || projected.Reservation != pending.Reservation {
			t.Fatal("new process did not recover the original valid receipt")
		}
		rows, err := configdao.NewMonitorConfigDAO(zap.NewNop(), db).GetMonitorConfigsByInstances(ctx, []string{"127.0.250.1", "127.0.250.2"}, 4)
		if err != nil || len(rows) != 2 {
			t.Fatal("combined accepted rows missing")
		}
		for _, row := range rows {
			body, err := client.Get(ctx, buildRedisKeyRecordRule(row.InstanceIP)).Result()
			if err != nil || body != row.ConfigContent {
				t.Fatal("combined raw bytes differ from accepted MySQL rows")
			}
		}
		var expected configdao.ProjectionAckProof
		expected.Reservation, expected.SourceHash, expected.PlanDigest = projected.Reservation, projected.SourceHash, projected.PlanDigest
		for _, control := range projected.Controls {
			if control.Ack != nil {
				expected.ArtifactVector = control.Ack.ArtifactVector
				expected.RequiredControls = control.Ack.RequiredControls
			}
		}
		encoded, err := json.Marshal(expected)
		if err != nil {
			t.Fatal(err)
		}
		if ack, err := client.Get(ctx, projectionPoolKey(4, 25001, "ack")).Result(); err != nil || ack != string(encoded) {
			t.Fatal("combined recovery did not persist the exact issued Redis ACK")
		}
		if sha, err := client.Get(ctx, recordPublicationKey(25001)).Result(); err != nil || sha != hex.EncodeToString(projected.SourceHash[:]) {
			t.Fatal("combined recovery legacy SHA differs")
		}
		members, err := client.SMembers(ctx, buildRedisSetKeyRecordRulePoolIPs(25001)).Result()
		if err != nil || len(members) != 2 {
			t.Fatal("combined recovery members incomplete")
		}
		set := map[string]bool{}
		for _, ip := range members {
			set[ip] = true
		}
		if !set["127.0.250.1"] || !set["127.0.250.2"] {
			t.Fatal("combined recovery members contain unexpected target")
		}
		t.Logf("publisher_pid=%d recovery_pid=%d parent_pid=%d: actual MySQL+Redis cross-process recovery", processIDs[0], processIDs[1], os.Getpid())
	})
	for index, oldSubset := range []bool{false, true} {
		t.Run(map[bool]string{false: "MySQL AM full pending then subset", true: "MySQL AM subset pending then full"}[oldSubset], func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			if err := db.AutoMigrate(&model.MonitorAlertManagerPool{}, &model.MonitorSendGroup{}); err != nil {
				t.Fatal("combined AM source models unavailable")
			}
			log := zap.NewNop()
			pools := alertdao.NewAlertManagerPoolDAO(db, log)
			groups := alertdao.NewAlertManagerSendDAO(db, log)
			pool := &model.MonitorAlertManagerPool{Model: model.Model{ID: 26001 + index}, Name: "combined_am", AlertManagerInstances: model.StringList{fmt.Sprintf("127.0.252.%d", index+1)}, ResolveTimeout: "5m", GroupWait: "30s", GroupInterval: "5m", RepeatInterval: "1h", GroupBy: model.StringList{"alertname"}, Receiver: "combined_group"}
			group := &model.MonitorSendGroup{Model: model.Model{ID: 26001 + index}, PoolID: pool.ID, Name: "combined_group", NameZh: "combined", Enable: 1, RepeatInterval: "1h", SendResolved: 1}
			if pools.CreateMonitorAlertManagerPool(ctx, pool) != nil || groups.CreateMonitorSendGroup(ctx, group) != nil {
				t.Fatal("combined AM authoritative source unavailable")
			}
			store, err := NewProjectionStore(client)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			fixture := &rootAMNativeFixture{t: t, ctx: ctx, client: client, db: db, metadata: configdao.NewMonitorConfigDAO(log, db), journal: configdao.NewAMProjectionJournalDAO(log, db), pools: pools, groups: groups, pool: pool, group: group, store: store}
			rootRecoveryAMPeerCase(t, fixture, oldSubset)
		})
	}
	// This destructive case deliberately leaves an invalidated committed locator.
	// Keep it after the scoped AM cases, whose one-item worker fixture is not a
	// separate poison-queue fairness test (that has its own native regression).
	t.Run("KILL original MySQL lease prevents a new Redis batch", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		journal := configdao.NewProjectionPlanJournalDAO(zap.NewNop(), db)
		ref := projectionReserve(t, journal, 4, 25002, 2)
		items := []configdao.GeneratedPlanItem{projectionRecordItem(25002, 1, "127.0.251.1", "groups: []\n"), projectionRecordItem(25002, 2, "127.0.251.2", "groups: []\n")}
		receipt, err := journal.CommitGeneratedProjectionPlan(ctx, projectionFullRequest(ref, items...))
		if err != nil {
			t.Fatal(err)
		}
		var original interface {
			QueryRowContext(context.Context, string, ...interface{}) *sql.Row
		}
		if db.Callback().Query().After("gorm:query").Register("combined:original", func(tx *gorm.DB) {
			if c, ok := tx.Statement.ConnPool.(interface {
				QueryRowContext(context.Context, string, ...interface{}) *sql.Row
			}); ok {
				original = c
			}
		}) != nil {
			t.Fatal("original transaction observer unavailable")
		}
		store, err := NewProjectionStore(client)
		if err != nil {
			t.Fatal(err)
		}
		defer store.Close()
		err = journal.WithPoolProjectionLease(ctx, receipt, func(lease configdao.ProjectionLease) error {
			p, err := store.Prepare(ctx, lease)
			if err != nil {
				return err
			}
			projectionTrack(t, client, p)
			var id uint64
			if original == nil || original.QueryRowContext(ctx, "SELECT CONNECTION_ID()").Scan(&id) != nil || id == 0 {
				t.Fatal("original connection unavailable")
			}
			var database, user string
			if f.Admin.QueryRowContext(ctx, "SELECT DB,USER FROM information_schema.PROCESSLIST WHERE ID=?", id).Scan(&database, &user) != nil || database != f.Schema || user != f.User {
				t.Fatal("refusing KILL outside owned fixture")
			}
			if _, err := f.Admin.ExecContext(ctx, fmt.Sprintf("KILL CONNECTION %d", id)); err != nil {
				t.Fatal("owned original connection KILL failed")
			}
			other := combinedSQL(t, f.B)
			ordinary := configdao.NewMonitorConfigDAO(zap.NewNop(), other)
			row, err := ordinary.GetMonitorConfigByID(ctx, items[0].Desired.ID)
			if err != nil {
				return err
			}
			row.ConfigContent = "groups: [] # after actual MySQL KILL\n"
			if err := ordinary.UpdateMonitorConfig(ctx, row); err != nil {
				return err
			}
			calls, _ := rootProjectionEvalStats(t, ctx, client)
			before := rootProjectionValues(t, client, p.keys)
			err = p.ApplyArtifacts(ctx, []uint64{receipt.Artifacts[0].EventID, receipt.Artifacts[1].EventID})
			if err == nil {
				t.Fatal("dead MySQL lease accepted a Redis batch")
			}
			after, _ := rootProjectionEvalStats(t, ctx, client)
			if after != calls || !reflect.DeepEqual(before, rootProjectionValues(t, client, p.keys)) {
				t.Fatal("dead MySQL lease sent EVAL or changed Redis")
			}
			return err
		})
		if err == nil {
			t.Fatal("dead original lease falsely completed")
		}
	})
}
