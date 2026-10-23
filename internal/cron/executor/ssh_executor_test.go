package executor

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	treeUtils "github.com/GoSimplicity/AI-CloudOps/internal/tree/utils"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type resourceStub struct {
	dao.TreeLocalDAO
	resource *model.TreeLocalResource
	called   bool
}

func (s *resourceStub) GetByID(ctx context.Context, _ int) (*model.TreeLocalResource, error) {
	s.called = true
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return s.resource, nil
}

type connectionStub struct {
	password, key, command, output string
	mode                           int8
	closed, ran                    bool
	runError                       error
	deadline                       time.Time
}

func (s *connectionStub) ConnectContext(ctx context.Context, _ string, _ int, _ string, password, key string, mode int8, _ int) error {
	s.password = password
	s.key = key
	s.mode = mode
	s.deadline, _ = ctx.Deadline()
	return ctx.Err()
}
func (s *connectionStub) RunContext(ctx context.Context, command string) (string, error) {
	s.ran = true
	s.command = command
	return s.output, s.runError
}
func (s *connectionStub) Close() error { s.closed = true; return nil }

func testSSHJob() *model.CronJob {
	id := 7
	return &model.CronJob{SSHResourceID: &id, JobType: model.CronJobTypeSSH, SSHCommand: "printf private-command-marker", Timeout: 0}
}
func configureEncryption(t *testing.T) {
	t.Helper()
	viper.Set("tree.password_encryption_key", strings.Repeat("a", 32))
	t.Cleanup(viper.Reset)
}

func TestSSHExecutorDecryptsWithoutMutatingDAO(t *testing.T) {
	configureEncryption(t)
	encrypted, err := treeUtils.EncryptPassword("private-fixture-password")
	if err != nil {
		t.Fatal(err)
	}
	resource := &model.TreeLocalResource{Status: model.RUNNING, AuthMode: model.AuthModePassword, Password: encrypted}
	data := &resourceStub{resource: resource}
	connection := &connectionStub{output: "executed"}
	executor := NewSSHExecutor(zap.NewNop(), data)
	executor.newSSH = func() sshConnection { return connection }
	output, err := executor.ExecuteSSHJob(context.Background(), testSSHJob())
	if err != nil || output != "executed" {
		t.Fatalf("execution failed: %v", err)
	}
	if connection.password != "private-fixture-password" || resource.Password != encrypted {
		t.Fatal("credential not decrypted locally or DAO mutated")
	}
	if remaining := time.Until(connection.deadline); remaining < 4*time.Minute || remaining > 5*time.Minute {
		t.Fatal("zero timeout did not use finite default")
	}
	if !connection.closed {
		t.Fatal("connection not closed")
	}
}

func TestSSHExecutorValidationAndFailureOutput(t *testing.T) {
	for _, tt := range []struct {
		name        string
		auth        model.AuthMode
		password    string
		environment model.KeyValueList
		wantRun     bool
	}{
		{name: "key ignores unused password", auth: model.AuthModeKey, password: "not ciphertext", wantRun: true},
		{name: "encrypted password failure", auth: model.AuthModePassword, password: "invalid"},
		{name: "unknown mode", auth: 99},
		{name: "environment injection", auth: model.AuthModeKey, environment: model.KeyValueList{{Key: "BAD; printf injected", Value: "value"}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			configureEncryption(t)
			resource := &model.TreeLocalResource{Status: model.RUNNING, AuthMode: tt.auth, Password: tt.password, Key: "private-key-marker"}
			connection := &connectionStub{output: "partial-output-marker", runError: errors.New("execution failed")}
			core, logs := observer.New(zap.DebugLevel)
			executor := NewSSHExecutor(zap.New(core), &resourceStub{resource: resource})
			executor.newSSH = func() sshConnection { return connection }
			job := testSSHJob()
			job.SSHEnvironment = tt.environment
			output, err := executor.ExecuteSSHJob(context.Background(), job)
			if err == nil {
				t.Fatal("expected execution/validation error")
			}
			if connection.ran != tt.wantRun {
				t.Fatal("unexpected remote execution")
			}
			if tt.wantRun && output != "partial-output-marker" {
				t.Fatal("partial error output was discarded")
			}
			for _, entry := range logs.All() {
				text := entry.Message
				for _, field := range entry.Context {
					text += field.String
				}
				for _, secret := range []string{"private-command-marker", "private-key-marker", "partial-output-marker"} {
					if strings.Contains(text, secret) {
						t.Fatal("sensitive execution content logged")
					}
				}
			}
		})
	}
}

func TestSSHCommandRejectsInvalidStructuredInput(t *testing.T) {
	for _, tt := range []struct{ name, workDir, command, key, value string }{
		{name: "invalid name", key: "bad-name"},
		{name: "name substitution", key: "$(command)"},
		{name: "empty name", key: ""},
		{name: "null directory", workDir: "/tmp\x00bad", key: "VALID"},
		{name: "null value", key: "VALID", value: "bad\x00value"},
		{name: "null command", key: "VALID", command: "bad\x00command"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			job := testSSHJob()
			job.SSHWorkDir = tt.workDir
			if tt.command != "" {
				job.SSHCommand = tt.command
			}
			job.SSHEnvironment = model.KeyValueList{{Key: tt.key, Value: tt.value}}
			if _, err := NewSSHExecutor(nil, nil).buildSSHCommand(job); err == nil {
				t.Fatal("invalid structured input accepted")
			}
		})
	}
}

func TestSSHExecutorCancellationDuringResourceLookup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	data := &resourceStub{}
	executor := NewSSHExecutor(zap.NewNop(), data)
	executor.newSSH = func() sshConnection { t.Fatal("cancelled request established connection"); return nil }
	if _, err := executor.ExecuteSSHJob(ctx, testSSHJob()); !errors.Is(err, context.Canceled) {
		t.Fatalf("resource lookup cancellation lost: %v", err)
	}
}
