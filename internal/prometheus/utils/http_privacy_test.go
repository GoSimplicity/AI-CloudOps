package utils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type privacyTransport func(*http.Request) (*http.Response, error)

func (transport privacyTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

type privateResponseReader struct{ sent bool }

func (body *privateResponseReader) Read(buffer []byte) (int, error) {
	if !body.sent {
		body.sent = true
		return copy(buffer, "NOTIFY_PRIVATE"), nil
	}
	return 0, errors.New("NOTIFY_PRIVATE read error")
}
func (*privateResponseReader) Close() error { return nil }

func TestNotificationHTTPFailuresArePrivate(t *testing.T) {
	const canary = "NOTIFY_PRIVATE"
	for _, mode := range []string{"build", "transport", "read", "status", "success-201"} {
		t.Run(mode, func(t *testing.T) {
			core, logs := observer.New(zap.DebugLevel)
			calls := 0
			client := &http.Client{Timeout: 7 * time.Second, Transport: privacyTransport(func(request *http.Request) (*http.Response, error) {
				calls++
				if request.Method != "POST" || request.Header.Get("Authorization") != "Bearer "+canary || request.URL.Query().Get("existing") != canary || request.URL.Query().Get("extra") != canary {
					t.Fatal("HTTP method, headers or query contract changed")
				}
				deadline, ok := request.Context().Deadline()
				if !ok || time.Until(deadline) > 7*time.Second {
					t.Fatal("client timeout/context deadline was lost")
				}
				if mode == "transport" {
					return nil, errors.New(canary)
				}
				var body io.ReadCloser = io.NopCloser(strings.NewReader(canary))
				if mode == "read" {
					body = &privateResponseReader{}
				}
				status := 201
				if mode == "status" {
					status = 503
				}
				return &http.Response{StatusCode: status, Status: canary, Body: body, Header: make(http.Header), Request: request}, nil
			})}
			endpoint := "https://" + canary + ":" + canary + "@fixture.invalid/path?existing=" + canary
			if mode == "build" {
				endpoint = "https://fixture.invalid/%zz"
			}
			body, err := PostWithJson(context.Background(), client, zap.New(core), endpoint, canary, map[string]string{"extra": canary}, map[string]string{"Authorization": "Bearer " + canary})
			expected := 1
			if mode == "build" {
				expected = 0
			}
			if calls != expected {
				t.Fatal("HTTP failure retried")
			}
			if mode == "success-201" {
				if err != nil || string(body) != canary {
					t.Fatal("2xx success body contract changed")
				}
			} else if err == nil || strings.Contains(err.Error(), canary) || len(body) != 0 {
				t.Fatal("private failure body or error escaped")
			}
			for _, entry := range logs.All() {
				if strings.Contains(entry.Message+fmt.Sprint(entry.ContextMap()), canary) {
					t.Fatal("HTTP log leaked private data")
				}
			}
		})
	}
}
