package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestUnsilenceConfirmationDoesNotPromiseDelete(t *testing.T) {
	handler, err := NewWebHookHandler(withActionKey(&config.WebhookConfig{ImFeishu: config.ImFeishuConfig{RequestTimeoutSeconds: 7}}), zap.NewNop(), &countingDAO{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	engine := gin.New()
	handler.RegisterRouters(engine)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, signActionRequest(t, http.MethodGet, "unsilence", "fixture", ""))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status %d", recorder.Code)
	}
	body := recorder.Body.String()
	for _, banned := range []string{"调用 Alertmanager", "DELETE", "清空"} {
		if strings.Contains(body, banned) {
			t.Fatalf("confirmation still promises %q", banned)
		}
	}
	if !strings.Contains(body, "未能取消，静默关联已保留") {
		t.Fatalf("confirmation omits the refusal: %s", body)
	}
}

func TestUnsilenceDocsDoNotPromiseDeleteOrClear(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..")
	checks := []struct {
		path    string
		banned  []string
		require string
	}{
		{
			path:    filepath.Join(root, "internal/prometheus/webhook/api/README.md"),
			banned:  []string{"取消静默 DELETE", "取消静默同样要求上游成功", "包括取消静默时的空字符串"},
			require: "未能取消，静默关联已保留",
		},
		{
			path:    filepath.Join(root, "docs/release/w1-webhook-env.md"),
			banned:  []string{"重复静默、取消静默或认领是幂等的"},
			require: "不会调用 Alertmanager",
		},
		{
			path:    filepath.Join(root, "docs/release/README.md"),
			banned:  []string{"未启用时 webhook 的静默/取消静默接口会返回失败"},
			require: "不会清空本地关联",
		},
	}
	for _, check := range checks {
		t.Run(filepath.Base(check.path), func(t *testing.T) {
			body, err := os.ReadFile(check.path)
			if err != nil {
				t.Fatal(err)
			}
			text := string(body)
			for _, banned := range check.banned {
				if strings.Contains(text, banned) {
					t.Fatalf("%s still says %q", check.path, banned)
				}
			}
			if !strings.Contains(text, check.require) {
				t.Fatalf("%s missing %q", check.path, check.require)
			}
		})
	}
}
