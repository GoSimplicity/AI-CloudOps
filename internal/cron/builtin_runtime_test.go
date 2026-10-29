package cron

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	crondao "github.com/GoSimplicity/AI-CloudOps/internal/cron/dao"
	crontask "github.com/GoSimplicity/AI-CloudOps/internal/cron/task"
	k8sdao "github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	k8smanager "github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	"github.com/hibiken/asynq"
	robcron "github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type builtinDAOStub struct {
	crondao.CronJobDAO
	mu   sync.Mutex
	jobs map[int]*model.CronJob
	err  error
}

func (d *builtinDAOStub) GetEnabledCronJobs(context.Context) ([]*model.CronJob, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.err != nil {
		return nil, d.err
	}
	var jobs []*model.CronJob
	for _, job := range d.jobs {
		if job.Status == model.CronJobStatusEnabled || job.Status == model.CronJobStatusRunning {
			copy := *job
			jobs = append(jobs, &copy)
		}
	}
	return jobs, nil
}
func (d *builtinDAOStub) GetCronJob(_ context.Context, id int) (*model.CronJob, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	job := d.jobs[id]
	if job == nil {
		return nil, errors.New("missing")
	}
	copy := *job
	return &copy, nil
}
func (d *builtinDAOStub) update(id int, fn func(*model.CronJob)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	fn(d.jobs[id])
}

type builtinRunnerStub struct {
	mu      sync.Mutex
	calls   []crontask.CronTaskPayload
	entered chan struct{}
	release chan struct{}
}

func (r *builtinRunnerStub) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload crontask.CronTaskPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return err
	}
	r.mu.Lock()
	r.calls = append(r.calls, payload)
	r.mu.Unlock()
	if r.entered != nil {
		select {
		case r.entered <- struct{}{}:
		default:
		}
		<-ctx.Done()
		<-r.release
	}
	return nil
}
func (r *builtinRunnerStub) IsTaskFailure(err error) bool { return err != nil }

func (r *builtinRunnerStub) count() int { r.mu.Lock(); defer r.mu.Unlock(); return len(r.calls) }
func testBuiltinManager(store *builtinDAOStub, runner *builtinRunnerStub) *unifiedCronManager {
	return &unifiedCronManager{logger: zap.NewNop(), builtinTaskMgr: &BuiltinTaskManager{logger: zap.NewNop(), cronDAO: store}, builtinRunner: runner, builtinCron: robcron.New(robcron.WithLocation(time.UTC)), builtinEntries: make(map[int]builtinEntry)}
}
func TestBuiltinReconcileKeepsValidEntriesAndHonorsDisabledOrChangedState(t *testing.T) {
	job := &model.CronJob{Model: model.Model{ID: 1}, Name: "fixture", IsBuiltIn: 1, JobType: model.CronJobTypeSystem, Status: model.CronJobStatusEnabled, Command: "on_duty_history", Schedule: "@every 1h"}
	store := &builtinDAOStub{jobs: map[int]*model.CronJob{1: job}}
	runner := &builtinRunnerStub{}
	cm := testBuiltinManager(store, runner)
	if err := cm.reconcileBuiltinTasks(context.Background()); err != nil {
		t.Fatal(err)
	}
	first := cm.builtinEntries[1]
	old := cm.builtinCron.Entry(first.id).Job
	if err := cm.reconcileBuiltinTasks(context.Background()); err != nil {
		t.Fatal(err)
	}
	if cm.builtinEntries[1].id != first.id {
		t.Fatal("unchanged entry reset")
	}
	old.Run()
	if runner.count() != 1 {
		t.Fatal("enabled task not executed")
	}
	store.update(1, func(j *model.CronJob) { j.Status = model.CronJobStatusDisabled })
	old.Run()
	if runner.count() != 1 {
		t.Fatal("disabled task still executed")
	}
	if err := cm.reconcileBuiltinTasks(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(cm.builtinEntries) != 0 {
		t.Fatal("disabled entry retained")
	}
	store.update(1, func(j *model.CronJob) { j.Status = model.CronJobStatusEnabled })
	if err := cm.reconcileBuiltinTasks(context.Background()); err != nil {
		t.Fatal(err)
	}
	old = cm.builtinCron.Entry(cm.builtinEntries[1].id).Job
	store.update(1, func(j *model.CronJob) { j.Schedule = "@every 2h" })
	old.Run()
	if runner.count() != 1 {
		t.Fatal("obsolete schedule executed")
	}
	if err := cm.reconcileBuiltinTasks(context.Background()); err != nil {
		t.Fatal(err)
	}
	valid := cm.builtinEntries[1]
	store.update(1, func(j *model.CronJob) { j.Schedule = "invalid" })
	if err := cm.reconcileBuiltinTasks(context.Background()); err == nil {
		t.Fatal("invalid schedule accepted")
	}
	if cm.builtinEntries[1].id != valid.id || cm.builtinReady.Load() {
		t.Fatal("invalid update removed previous entry or marked ready")
	}
	store.mu.Lock()
	store.err = errors.New("offline")
	store.mu.Unlock()
	if err := cm.reconcileBuiltinTasks(context.Background()); err == nil || cm.builtinEntries[1].id != valid.id {
		t.Fatal("database failure lost valid registration")
	}
}
func TestBuiltinStopCancelsAndWaitsForRealCallback(t *testing.T) {
	job := &model.CronJob{Model: model.Model{ID: 1}, Name: "fixture", IsBuiltIn: 1, JobType: model.CronJobTypeSystem, Status: model.CronJobStatusEnabled, Command: "on_duty_history", Schedule: "@every 1h"}
	store := &builtinDAOStub{jobs: map[int]*model.CronJob{1: job}}
	runner := &builtinRunnerStub{entered: make(chan struct{}, 1), release: make(chan struct{})}
	cm := testBuiltinManager(store, runner)
	ctx, cancel := context.WithCancel(context.Background())
	cm.cancel = cancel
	if err := cm.StartSystemTasks(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case <-runner.entered:
	case <-time.After(3 * time.Second):
		t.Fatal("initial callback did not begin")
	}
	stopped := make(chan error, 1)
	go func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer stopCancel()
		stopped <- cm.Stop(stopCtx)
	}()
	select {
	case err := <-stopped:
		t.Fatalf("returned before callback finished: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	close(runner.release)
	if err := <-stopped; err != nil {
		t.Fatal(err)
	}
	if cm.builtinReady.Load() {
		t.Fatal("stopped scheduler remained ready")
	}
}

func TestConcurrentStopsShareCompletionEvenAfterTimeout(t *testing.T) {
	job := &model.CronJob{Model: model.Model{ID: 1}, IsBuiltIn: 1, JobType: model.CronJobTypeSystem, Status: model.CronJobStatusEnabled, Command: "on_duty_history", Schedule: "@every 1h"}
	store := &builtinDAOStub{jobs: map[int]*model.CronJob{1: job}}
	runner := &builtinRunnerStub{entered: make(chan struct{}, 1), release: make(chan struct{})}
	cm := testBuiltinManager(store, runner)
	ctx, cancel := context.WithCancel(context.Background())
	cm.cancel = cancel
	if err := cm.StartSystemTasks(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case <-runner.entered:
	case <-time.After(3 * time.Second):
		t.Fatal("callback did not begin")
	}
	stopCtx, stopCancel := context.WithCancel(context.Background())
	stopCancel()
	if err := cm.Stop(stopCtx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	if err := cm.Start(context.Background()); err == nil {
		t.Fatal("restart allowed before prior work finished")
	}
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() { results <- cm.Stop(context.Background()) }()
	}
	close(runner.release)
	for i := 0; i < 2; i++ {
		select {
		case err := <-results:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("stop did not complete")
		}
	}
	cm.lifecycleMu.Lock()
	defer cm.lifecycleMu.Unlock()
	if cm.cancel != nil || cm.stopping != nil || cm.builtinReady.Load() {
		t.Fatal("lifecycle not fully stopped")
	}
}

func TestOnDutySinglePassPersistsHistoryWithRealDAO(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "duty.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err = db.AutoMigrate(&model.User{}, &model.MonitorOnDutyGroup{}, &model.MonitorOnDutyChange{}); err != nil {
		t.Fatal(err)
	}
	// SQLite index names are global; these two MySQL tables use the same index name.
	if err = db.Exec("DROP INDEX idx_group_date_deleted_at").Error; err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&model.MonitorOnDutyHistory{}); err != nil {
		t.Fatal(err)
	}
	group := &model.MonitorOnDutyGroup{Name: "test", CreateUserName: "test", Enable: 1, Users: []*model.User{{Username: "fixture"}}}
	if err = db.Create(group).Error; err != nil {
		t.Fatal(err)
	}
	duty := alertdao.NewAlertManagerOnDutyDAO(db, zap.NewNop(), nil)
	for i := 0; i < 2; i++ {
		if _, err = ExecuteBuiltinOnce(context.Background(), "on_duty_history", zap.NewNop(), duty, nil, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	var histories []model.MonitorOnDutyHistory
	if err = db.Find(&histories).Error; err != nil {
		t.Fatal(err)
	}
	if len(histories) != 1 || histories[0].OnDutyGroupID != group.ID || histories[0].OnDutyUserID != group.Users[0].ID {
		t.Fatalf("unexpected histories: %+v", histories)
	}
}
func TestBuiltinInitializationPreservesExistingSettings(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "cron.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	defer pool.Close()
	if err = db.AutoMigrate(&model.CronJob{}); err != nil {
		t.Fatal(err)
	}
	store := crondao.NewCronJobDAO(zap.NewNop(), db)
	first := GetBuiltinTasks()[0]
	job := &model.CronJob{Name: "管理员重命名的值班任务", Command: first.TaskType, Schedule: "@every 3m", Status: model.CronJobStatusDisabled, IsBuiltIn: 1, JobType: model.CronJobTypeSystem, Timeout: 300, MaxRetry: 3}
	if err = store.CreateCronJob(context.Background(), job); err != nil {
		t.Fatal(err)
	}
	manager := NewBuiltinTaskManager(zap.NewNop(), store)
	for i := 0; i < 2; i++ {
		if err = manager.InitializeBuiltinTasks(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	got, err := store.GetCronJob(context.Background(), job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Schedule != "@every 3m" || got.Status != model.CronJobStatusDisabled {
		t.Fatal("existing operator settings overwritten")
	}
	jobs, err := store.GetEnabledCronJobs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 2 {
		t.Fatalf("unexpected defaults %d", len(jobs))
	}
}

func TestBuiltinReconcileRejectsMalformedHistoricalTimezone(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "historical.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err = db.AutoMigrate(&model.CronJob{}); err != nil {
		t.Fatal(err)
	}
	store := crondao.NewCronJobDAO(zap.NewNop(), db)
	job := &model.CronJob{Name: "historical", Command: "on_duty_history", Schedule: "@every 1h", Status: model.CronJobStatusEnabled, IsBuiltIn: 1, JobType: model.CronJobTypeSystem, Timeout: 300}
	if err = store.CreateCronJob(context.Background(), job); err != nil {
		t.Fatal(err)
	}
	cm := &unifiedCronManager{logger: zap.NewNop(), builtinTaskMgr: NewBuiltinTaskManager(zap.NewNop(), store), builtinRunner: &builtinRunnerStub{}, builtinCron: robcron.New(), builtinEntries: make(map[int]builtinEntry)}
	if err = cm.reconcileBuiltinTasks(context.Background()); err != nil {
		t.Fatal(err)
	}
	before := cm.builtinEntries[job.ID]
	for _, expression := range []string{"TZ=UTC", "CRON_TZ=UTC"} {
		// Simulate invalid persisted data from before write-side validation existed.
		if err = db.Model(&model.CronJob{}).Where("id = ?", job.ID).Update("schedule", expression).Error; err != nil {
			t.Fatal(err)
		}
		if err = cm.reconcileBuiltinTasks(context.Background()); err == nil {
			t.Fatalf("accepted %q", expression)
		}
		if cm.builtinReady.Load() || cm.builtinEntries[job.ID] != before || cm.builtinCron.Entry(before.id).Job == nil {
			t.Fatalf("lost registration or marked ready for %q", expression)
		}
	}
}

type dutyStub struct {
	alertdao.AlertManagerOnDutyDAO
	existsErr error
	calls     int
}

func (d *dutyStub) GetMonitorOnDutyList(_ context.Context, req *model.GetMonitorOnDutyGroupListReq) ([]*model.MonitorOnDutyGroup, int64, error) {
	if req.Page != 1 {
		return nil, 1, nil
	}
	return []*model.MonitorOnDutyGroup{{Model: model.Model{ID: 1}, Enable: 1, Users: []*model.User{{Model: model.Model{ID: 7}}}}}, 1, nil
}
func (d *dutyStub) ExistsMonitorOnDutyHistory(context.Context, int, string) (bool, error) {
	d.calls++
	return true, d.existsErr
}
func (d *dutyStub) GetMonitorOnDutyChangesByGroupAndTimeRange(context.Context, int, string, string) ([]*model.MonitorOnDutyChange, int64, error) {
	return nil, 0, nil
}
func TestOnDutyExecutesRealSinglePassAndReportsErrors(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			duty := &dutyStub{}
			if fail {
				duty.existsErr = errors.New("controlled failure")
			}
			_, err := ExecuteBuiltinOnce(context.Background(), "on_duty_history", zap.NewNop(), duty, nil, nil, nil)
			if (err != nil) != fail || duty.calls != 1 {
				t.Fatalf("calls %d err %v", duty.calls, err)
			}
		})
	}
}

type clusterDAOStub struct {
	k8sdao.ClusterDAO
	count      int
	failUpdate int
}

func (d *clusterDAOStub) GetClusterList(_ context.Context, req *model.ListClustersReq) ([]*model.K8sCluster, int64, error) {
	start := (req.Page - 1) * req.Size
	var rows []*model.K8sCluster
	for i := start; i < d.count && i < start+req.Size; i++ {
		rows = append(rows, &model.K8sCluster{Model: model.Model{ID: i + 1}, Status: model.StatusRunning})
	}
	return rows, int64(d.count), nil
}
func (d *clusterDAOStub) UpdateClusterStatus(_ context.Context, id int, _ model.ClusterStatus) error {
	if id == d.failUpdate {
		return errors.New("write failed")
	}
	return nil
}

type clusterManagerStub struct {
	k8smanager.ClusterManager
	mu     sync.Mutex
	count  int
	failID int
}

func (c *clusterManagerStub) CheckClusterStatus(_ context.Context, id int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count++
	if id == c.failID {
		return errors.New("offline")
	}
	return nil
}
func TestBuiltinK8sVisitsAllPagesAndReportsFailures(t *testing.T) {
	for _, failID := range []int{0, 1001} {
		t.Run(fmt.Sprint(failID), func(t *testing.T) {
			store := &clusterDAOStub{count: 1001}
			mgr := &clusterManagerStub{failID: failID}
			_, err := ExecuteBuiltinOnce(context.Background(), "k8s_status_check", zap.NewNop(), nil, store, mgr, nil)
			if (err != nil) != (failID != 0) || mgr.count != 1001 {
				t.Fatalf("checked=%d err=%v", mgr.count, err)
			}
		})
	}
}

type busyBuiltinRunner struct{ calls int }

func (r *busyBuiltinRunner) ProcessTask(context.Context, *asynq.Task) error {
	r.calls++
	return errors.New("lease busy")
}
func (*busyBuiltinRunner) IsTaskFailure(error) bool { return false }
func TestBuiltinLeaseContentionDoesNotResubmitExecution(t *testing.T) {
	job := &model.CronJob{Model: model.Model{ID: 1}, Name: "fixture", IsBuiltIn: 1, JobType: model.CronJobTypeSystem, Status: model.CronJobStatusEnabled, Command: "on_duty_history", Schedule: "@every 1h", MaxRetry: 3}
	store := &builtinDAOStub{jobs: map[int]*model.CronJob{1: job}}
	runner := &busyBuiltinRunner{}
	cm := &unifiedCronManager{logger: zap.NewNop(), builtinTaskMgr: &BuiltinTaskManager{cronDAO: store}, builtinRunner: runner}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { cm.runScheduledBuiltin(ctx, 1, job.Schedule, job.Command); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("lease contention incorrectly scheduled an additional attempt")
	}
	if runner.calls != 1 {
		t.Fatalf("lease contention calls=%d", runner.calls)
	}
}

type blockingDutyDAO struct {
	alertdao.AlertManagerOnDutyDAO
	entered, cancelled, release chan struct{}
}

func (d *blockingDutyDAO) GetMonitorOnDutyList(context.Context, *model.GetMonitorOnDutyGroupListReq) ([]*model.MonitorOnDutyGroup, int64, error) {
	return []*model.MonitorOnDutyGroup{{Model: model.Model{ID: 1}, Enable: 1, Users: []*model.User{{Model: model.Model{ID: 7}}}}}, 1, nil
}
func (d *blockingDutyDAO) ExistsMonitorOnDutyHistory(ctx context.Context, _ int, _ string) (bool, error) {
	close(d.entered)
	<-ctx.Done()
	close(d.cancelled)
	<-d.release
	return false, ctx.Err()
}
func TestBuiltinCancellationWaitsForStartedWorkers(t *testing.T) {
	duty := &blockingDutyDAO{entered: make(chan struct{}), cancelled: make(chan struct{}), release: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := ExecuteBuiltinOnce(ctx, "on_duty_history", zap.NewNop(), duty, nil, nil, nil)
		done <- err
	}()
	<-duty.entered
	cancel()
	<-duty.cancelled
	select {
	case err := <-done:
		t.Fatalf("returned before worker stopped: %v", err)
	default:
	}
	close(duty.release)
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel semantics lost: %v", err)
	}
}
