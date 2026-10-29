package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type httpRoundTripFunc func(*http.Request) (*http.Response, error)

func (f httpRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestHTTPExecutorHonorsTaskDeadline(t *testing.T) {
	for _, tc := range []struct {
		name              string
		seconds, expected int
	}{
		{"默认超时", 0, 30}, {"允许超过旧客户端上限", 120, 120},
	} {
		t.Run(tc.name, func(t *testing.T) {
			executor := NewHTTPExecutor(zap.NewNop())
			if executor.client.Timeout != 0 {
				t.Fatal("客户端总超时不应覆盖任务期限")
			}
			executor.client.Transport = httpRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				deadline, ok := req.Context().Deadline()
				if !ok || time.Until(deadline) < time.Duration(tc.expected-1)*time.Second {
					t.Fatal("任务期限未正确传递")
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("ok"))}, nil
			})
			output, err := executor.Execute(context.Background(), &HTTPTask{Method: "GET", URL: "http://example.test", Timeout: tc.seconds})
			if err != nil || output != "ok" {
				t.Fatalf("请求失败: %v", err)
			}
		})
	}
}

func TestHTTPExecutorCancellationAndPrivateURL(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	executor := NewHTTPExecutor(zap.New(core))
	executor.client.Transport = httpRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		<-req.Context().Done()
		return nil, req.Context().Err()
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := executor.Execute(ctx, &HTTPTask{Method: "GET", URL: "http://example.test/?token=private-canary", Timeout: 120})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("取消语义丢失: %v", err)
	}
	if strings.Contains(err.Error(), "private-canary") {
		t.Fatal("错误泄露URL")
	}
	for _, entry := range logs.All() {
		if strings.Contains(entry.Message, "private-canary") {
			t.Fatal("日志泄露URL")
		}
		for _, field := range entry.ContextMap() {
			if value, ok := field.(string); ok && strings.Contains(value, "private-canary") {
				t.Fatal("日志泄露URL")
			}
		}
	}
}

func TestHTTPExecutorHidesTransportDetails(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	executor := NewHTTPExecutor(zap.New(core))
	executor.client.Transport = httpRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("private-canary")
	})
	_, err := executor.Execute(context.Background(), &HTTPTask{Method: "GET", URL: "http://example.test/?token=private-canary"})
	if err == nil || strings.Contains(err.Error(), "private-canary") {
		t.Fatal("传输错误包含私有信息或未返回错误")
	}
	for _, entry := range logs.All() {
		for _, field := range entry.ContextMap() {
			if value, ok := field.(string); ok && strings.Contains(value, "private-canary") {
				t.Fatal("日志泄露私有信息")
			}
		}
	}
}
