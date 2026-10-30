package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type privateReceiveReader struct{}

func (privateReceiveReader) Read([]byte) (int, error) { return 0, errors.New("RECEIVE_PRIVATE") }

func TestReceiveReadErrorDoesNotLeak(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	handler, err := NewWebHookHandler(&config.WebhookConfig{ImFeishu: config.ImFeishuConfig{RequestTimeoutSeconds: 7}}, zap.New(core), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	engine := gin.New()
	handler.RegisterRouters(engine)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("POST", "/api/v1/alerts/receive", privateReceiveReader{})
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)
	var response struct{ Code int }
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code == 0 || strings.Contains(recorder.Body.String(), "RECEIVE_PRIVATE") {
		t.Fatal("receive failure accepted or exposed reader error")
	}
	for _, entry := range logs.All() {
		if strings.Contains(entry.Message+fmt.Sprint(entry.ContextMap()), "RECEIVE_PRIVATE") {
			t.Fatal("receive reader error leaked")
		}
	}
}
