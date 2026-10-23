package notification

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestFeishuDoesNotExposePayloadURLOrProviderBody(t *testing.T) {
	for _, mode := range []string{"success", "business_error", "http_error", "token_error"} {
		t.Run(mode, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "/token") {
					if mode == "token_error" {
						_, _ = io.WriteString(w, `{"code":999,"msg":"private-provider-marker"}`)
						return
					}
					_, _ = io.WriteString(w, `{"code":0,"tenant_access_token":"test-token","expire":7200}`)
					return
				}
				if mode == "http_error" {
					w.WriteHeader(503)
				}
				if mode == "business_error" || mode == "http_error" {
					_, _ = io.WriteString(w, `{"code":999,"msg":"private-provider-marker","error":"private-provider-marker"}`)
					return
				}
				_, _ = io.WriteString(w, `{"code":0,"data":{"message_id":"test-message"}}`)
			}))
			defer server.Close()
			core, observed := observer.New(zap.DebugLevel)
			channel := NewFeishuChannel(localFeishuConfig{base: server.URL + "/private-url-marker"}, zap.New(core))
			response, err := channel.Send(context.Background(), &SendRequest{MessageID: "test-message", RecipientAddr: "oc_fixture", Subject: "private-message-marker", Content: "private-message-marker"})
			if (err == nil) != (mode == "success") {
				t.Fatal("delivery success semantics changed")
			}
			var diagnostic strings.Builder
			for _, entry := range observed.All() {
				diagnostic.WriteString(entry.Message)
				encoded, _ := json.Marshal(entry.ContextMap())
				diagnostic.Write(encoded)
			}
			if err != nil {
				diagnostic.WriteString(err.Error())
			}
			if response != nil {
				diagnostic.WriteString(response.ErrorMessage)
			}
			for _, marker := range []string{"private-url-marker", "private-message-marker", "private-provider-marker", "oc_fixture"} {
				if strings.Contains(diagnostic.String(), marker) {
					t.Error("notification diagnostics exposed a private URL, payload or upstream body")
				}
			}
		})
	}
}
