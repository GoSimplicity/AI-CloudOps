//go:build smoke

package service

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/utils"
	"go.uber.org/zap"
)

// Local OS commands only: no Redis, database or remote host is involved.
func TestNativeCriticalResourceParsers(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native probe platform not supported")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	s := &systemService{l: zap.NewNop()}
	snapshot := &model.System{CPUCores: runtime.NumCPU()}
	if err := s.collectCriticalResources(ctx, snapshot); err != nil {
		t.Fatal(err)
	}
	if !utils.HasValidResourceMetrics(snapshot) {
		t.Fatal("native critical probe did not provide complete finite measurements")
	}
	t.Logf("native=%s cpu=%.2f memory=%d/%dMiB disk=%d/%dGiB", runtime.GOOS, snapshot.CPUUsage, snapshot.MemoryUsed, snapshot.MemoryTotal, snapshot.DiskUsed, snapshot.DiskTotal)
}
