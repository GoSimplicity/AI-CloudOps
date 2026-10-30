package base

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type helperTransport func(*http.Request) (*http.Response, error)

func (transport helperTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

type privateReadFailure struct{ sent bool }

func (body *privateReadFailure) Read(buffer []byte) (int, error) {
	if !body.sent {
		body.sent = true
		return copy(buffer, "PRIVATE_CANARY"), nil
	}
	return 0, errors.New("PRIVATE_CANARY read error")
}
func (*privateReadFailure) Close() error { return nil }

func TestHTTPHelpersExposeOnlySafeErrors(t *testing.T) {
	const canary = "PRIVATE_CANARY"
	for _, method := range []string{"POST", "DELETE"} {
		t.Run(method, func(t *testing.T) {
			for _, test := range []struct {
				name                                        string
				status                                      int
				buildFailure, transportFailure, readFailure bool
				want                                        error
			}{
				{name: "request construction", buildFailure: true, want: ErrHTTPRequest},
				{name: "transport failure", transportFailure: true, want: ErrHTTPTransport},
				{name: "response read", status: 200, readFailure: true, want: ErrHTTPRead},
				{name: "HTTP 201 remains failure", status: 201},
				{name: "HTTP 404 remains failure", status: 404},
				{name: "HTTP 503", status: 503},
				{name: "HTTP 200", status: 200},
			} {
				t.Run(test.name, func(t *testing.T) {
					core, logs := observer.New(zap.DebugLevel)
					logger := zap.New(core)
					calls := 0
					original := http.DefaultTransport
					t.Cleanup(func() { http.DefaultTransport = original })
					http.DefaultTransport = helperTransport(func(request *http.Request) (*http.Response, error) {
						calls++
						if request.Method != method {
							t.Fatal("HTTP method changed")
						}
						if request.URL.Query().Get("private") != canary || request.Header.Get("X-Private") != canary {
							t.Fatal("request parameter/header contract changed")
						}
						if test.transportFailure {
							return nil, errors.New(canary + " transport error")
						}
						var body io.ReadCloser = io.NopCloser(strings.NewReader(canary))
						if test.readFailure {
							body = &privateReadFailure{}
						}
						return &http.Response{StatusCode: test.status, Status: canary, Body: body, Header: make(http.Header), Request: request}, nil
					})
					endpoint := "https://" + canary + ":" + canary + "@target.invalid/private"
					if test.buildFailure {
						endpoint += "/%zz"
					}
					var output []byte
					var err error
					if method == "POST" {
						output, err = PostWithJsonString(logger, canary, 7, endpoint, canary, map[string]string{"private": canary}, map[string]string{"X-Private": canary})
					} else {
						output, err = DeleteWithId(logger, canary, 7, endpoint, map[string]string{"private": canary}, map[string]string{"X-Private": canary})
					}
					expectedCalls := 1
					if test.buildFailure {
						expectedCalls = 0
					}
					if calls != expectedCalls {
						t.Fatalf("unexpected attempt count %d", calls)
					}
					if test.want != nil {
						if !errors.Is(err, test.want) {
							t.Fatalf("wrong static category %v", err)
						}
					} else if test.status != 200 {
						var statusError *HTTPStatusError
						if !errors.As(err, &statusError) || statusError.StatusCode != test.status {
							t.Fatal("numeric status classification lost")
						}
					} else if err != nil || string(output) != canary {
						t.Fatal("successful response contract changed")
					}
					if err != nil && (strings.Contains(err.Error(), canary) || len(output) != 0) {
						t.Fatal("private failure data escaped helper")
					}
					for _, entry := range logs.All() {
						if strings.Contains(entry.Message+fmt.Sprint(entry.ContextMap()), canary) {
							t.Fatal("private URL/body/request/error leaked into log")
						}
					}
				})
			}
		})
	}
}
