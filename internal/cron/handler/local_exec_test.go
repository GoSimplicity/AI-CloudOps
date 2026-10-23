package handler

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/cron/task"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// localJobStore 返回指定任务，并记录本次运行写入的结果
type localJobStore struct {
	*executionStore
	job          *model.CronJob
	runStatus    int8
	runOutput    string
	runError     string
	completeCall int
}

func (s *localJobStore) GetCronJob(context.Context, int) (*model.CronJob, error) {
	copy := *s.job
	return &copy, nil
}
func (s *localJobStore) CompleteCronJobLease(ctx context.Context, id int, owner string, started *time.Time, status int8, duration int, output, errorMsg string) (bool, error) {
	s.completeCall++
	s.runStatus, s.runOutput, s.runError = status, output, errorMsg
	return s.executionStore.CompleteCronJobLease(ctx, id, owner, started, status, duration, output, errorMsg)
}

func TestScheduledLocalExecJobsRespectSwitch(t *testing.T) {
	const secretMarker = "fake-command-content-marker"
	for _, allowed := range []bool{false, true} {
		for _, jobType := range []model.CronJobType{model.CronJobTypeCommand, model.CronJobTypeScript} {
			marker := filepath.Join(t.TempDir(), "ran")
			job := &model.CronJob{Model: model.Model{ID: 42}, Status: model.CronJobStatusEnabled, JobType: jobType, Timeout: 10}
			if jobType == model.CronJobTypeCommand {
				job.Command = "/bin/sh"
				job.Args = model.StringList{"-c", "touch '" + marker + "' # " + secretMarker}
			} else {
				job.ScriptType = "bash"
				job.ScriptContent = "touch '" + marker + "' # " + secretMarker
			}
			store := &localJobStore{executionStore: &executionStore{status: model.CronJobStatusEnabled}, job: job}
			core, logs := observer.New(zap.DebugLevel)
			h := &CronHandlers{logger: zap.New(core), cronDAO: store, commandExecutor: NewCommandExecutor(zap.NewNop()), scriptExecutor: NewScriptExecutor(zap.NewNop()), localExecAllowed: func() bool { return allowed }}

			err := h.ProcessTask(context.Background(), asynq.NewTask("cron:task", []byte(`{"job_id":42}`)))
			_, statErr := os.Stat(marker)
			ran := statErr == nil

			if store.completeCall != 1 || store.status != model.CronJobStatusEnabled {
				t.Fatalf("allowed=%v type=%d: run must be recorded once and lease released (calls=%d status=%d)", allowed, jobType, store.completeCall, store.status)
			}
			if allowed {
				if err != nil || !ran || store.runStatus != 1 {
					t.Fatalf("allowed=%v type=%d: behaviour changed err=%v ran=%v status=%d", allowed, jobType, err, ran, store.runStatus)
				}
				continue
			}
			if ran {
				t.Fatalf("type=%d executed while cron.allow_local_exec=false", jobType)
			}
			if store.runStatus != 2 || store.runError != task.ErrLocalExecDisabled.Error() {
				t.Fatalf("type=%d: want recorded failure with stable message, status=%d error=%q", jobType, store.runStatus, store.runError)
			}
			if !errors.Is(err, task.ErrLocalExecDisabled) || !errors.Is(err, asynq.SkipRetry) {
				t.Fatalf("type=%d: want non-retryable disabled error, got %v", jobType, err)
			}
			warned := logs.FilterMessage("本地命令/脚本执行已停用，跳过任务").All()
			if len(warned) != 1 || warned[0].Level != zap.WarnLevel {
				t.Fatalf("type=%d: want exactly one warning, got %d", jobType, len(warned))
			}
			fields := warned[0].ContextMap()
			if fmt.Sprint(fields["jobID"]) != "42" || fmt.Sprint(fields["jobType"]) != fmt.Sprint(int8(jobType)) || len(fields) != 2 {
				t.Fatalf("type=%d: warning must carry only job id and type, got %v", jobType, fields)
			}
			for _, entry := range logs.All() {
				for _, value := range entry.ContextMap() {
					if s, ok := value.(string); ok && strings.Contains(s, secretMarker) {
						t.Fatalf("command content leaked into log %q", entry.Message)
					}
				}
			}
		}
	}
}

func TestLocalTaskEnvDoesNotInheritServiceSecrets(t *testing.T) {
	fakeSecrets := map[string]string{
		"JWT_KEY1":      "fake-jwt-key-for-test-only",
		"JWT_KEY2":      "fake-jwt-key2-for-test-only",
		"AIOPS_API_KEY": "fake-aiops-key-for-test-only",
		"MYSQL_ADDR":    "fake:fake@tcp(127.0.0.1:1)/fake",
	}
	for name, value := range fakeSecrets {
		t.Setenv(name, value)
	}
	t.Setenv("LANG", "C.UTF-8")
	t.Setenv("LC_ALL", "C.UTF-8")
	t.Setenv("TZ", "Asia/Shanghai")

	check := func(t *testing.T, output string, extra ...string) {
		t.Helper()
		vars := map[string]string{}
		for _, line := range strings.Split(output, "\n") {
			if name, value, ok := strings.Cut(strings.TrimSpace(line), "="); ok {
				vars[name] = value
			}
		}
		for name := range fakeSecrets {
			if _, ok := vars[name]; ok {
				t.Fatalf("%s was inherited by the task process", name)
			}
		}
		for _, value := range fakeSecrets {
			if strings.Contains(output, value) {
				t.Fatal("fake secret value visible to the task process")
			}
		}
		for _, name := range append([]string{"PATH", "LANG", "LC_ALL", "TZ"}, extra...) {
			if _, ok := vars[name]; !ok {
				t.Fatalf("allow-listed variable %s missing", name)
			}
		}
	}

	t.Run("command", func(t *testing.T) {
		output, err := NewCommandExecutor(zap.NewNop()).Execute(context.Background(), &CommandTask{
			Command:     "env",
			Environment: model.KeyValueList{{Key: "TASK_VAR", Value: "task-value"}, {Key: "TZ", Value: "UTC"}},
			Timeout:     10,
		})
		if err != nil {
			t.Fatal(err)
		}
		check(t, output, "TASK_VAR")
		if !strings.Contains(output, "TASK_VAR=task-value") || !strings.Contains(output, "TZ=UTC") {
			t.Fatal("task-defined variables must be passed and override inherited ones")
		}
	})
	t.Run("script", func(t *testing.T) {
		output, err := NewScriptExecutor(zap.NewNop()).Execute(context.Background(), &ScriptTask{Type: "bash", Content: "env", Timeout: 10})
		if err != nil {
			t.Fatal(err)
		}
		check(t, output)
	})
}

// P3-3：执行前再次校验，数据库中已有的危险环境变量不会进入子进程
func TestCommandExecutorRejectsDangerousTaskEnv(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "ran")
	for _, name := range []string{"LD_PRELOAD", "Ld_Library_Path", "DYLD_INSERT_LIBRARIES", "bash_env", "ENV", "ifs", "LD_PRELOAD=/tmp/x.so"} {
		output, err := NewCommandExecutor(zap.NewNop()).Execute(context.Background(), &CommandTask{
			Command:     "touch",
			Args:        model.StringList{marker},
			Environment: model.KeyValueList{{Key: name, Value: "fake-env-value-canary"}},
			Timeout:     5,
		})
		if !errors.Is(err, task.ErrForbiddenTaskEnv) || output != "" {
			t.Fatalf("%q: want ErrForbiddenTaskEnv, got %v", name, err)
		}
		if strings.Contains(err.Error(), "fake-env-value-canary") {
			t.Fatal("错误信息不应包含变量值")
		}
		if _, statErr := os.Stat(marker); statErr == nil {
			t.Fatalf("%q: 命令不应被执行", name)
		}
	}
}

// P2-2：执行路径上，数据库中已有的 {key:”,value:”} 空行被跳过，任务照常执行且不产生 "=" 条目
func TestCommandExecutorSkipsBlankEnvRows(t *testing.T) {
	output, err := NewCommandExecutor(zap.NewNop()).Execute(context.Background(), &CommandTask{
		Command:     "env",
		Environment: model.KeyValueList{{Key: "", Value: ""}, {Key: "TASK_VAR", Value: "task-value"}, {Key: " ", Value: ""}},
		Timeout:     10,
	})
	if err != nil {
		t.Fatalf("空行不应导致执行失败: %v", err)
	}
	if !strings.Contains(output, "TASK_VAR=task-value") {
		t.Fatal("非空变量应传入子进程")
	}
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "=") || strings.HasPrefix(line, " =") {
			t.Fatalf("不应生成空变量名条目: %q", line)
		}
	}
	for _, kv := range buildTaskEnv(model.KeyValueList{{Key: "", Value: ""}}) {
		if strings.HasPrefix(kv, "=") {
			t.Fatalf("buildTaskEnv 不应生成空变量名条目: %q", kv)
		}
	}
}
