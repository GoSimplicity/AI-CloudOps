package alert

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type eventNotificationTransport func(*http.Request) (*http.Response, error)

func (transport eventNotificationTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func TestGroupTextSenderDoesNotLogPrivateResponse(t *testing.T) {
	for _, mode := range []string{"transport", "HTTP failure", "HTTP success malformed JSON"} {
		t.Run(mode, func(t *testing.T) {
			const canary = "EVENT_PRIVATE"
			core, logs := observer.New(zap.DebugLevel)
			calls := 0
			sender := &alertManagerEventDAO{l: zap.New(core), httpClient: &http.Client{Transport: eventNotificationTransport(func(request *http.Request) (*http.Response, error) {
				calls++
				if mode == "transport" {
					return nil, errors.New(canary)
				}
				status := 200
				if mode == "HTTP failure" {
					status = 503
				}
				return &http.Response{StatusCode: status, Status: canary, Body: io.NopCloser(strings.NewReader(canary)), Header: make(http.Header), Request: request}, nil
			})}}
			err := sender.SendMessageToGroup(context.Background(), "https://"+canary+":"+canary+"@fixture.invalid/path?private="+canary, canary)
			if calls != 1 || (err != nil && strings.Contains(err.Error(), canary)) {
				t.Fatal("text sender retried or exposed private error")
			}
			for _, entry := range logs.All() {
				if strings.Contains(entry.Message+fmt.Sprint(entry.ContextMap()), canary) {
					t.Fatal("text sender success/failure log exposed URL, message or body")
				}
			}
		})
	}
}
