package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/not_auth/service"
	"github.com/gin-gonic/gin"
	pm "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type discoveryService struct {
	ids   []int
	calls int
	port  int
}

func (s *discoveryService) BuildPrometheusServiceDiscovery(_ context.Context, port int, ids []int) ([]*targetgroup.Group, error) {
	s.calls++
	s.port = port
	s.ids = ids
	return []*targetgroup.Group{{Targets: []pm.LabelSet{{pm.AddressLabel: "127.0.0.1:9100"}}, Labels: pm.LabelSet{"instance": "ai-cloudops-tree"}}}, nil
}
func discoveryRequest(service *discoveryService, path, query string) *httptest.ResponseRecorder {
	return discoveryRequestFrom(&discoveryServiceRequest{
		service: service, path: path, query: query, remote: "127.0.0.1:12345",
	})
}

type discoveryServiceRequest struct {
	service *discoveryService
	cfg     *config.Config
	logger  *zap.Logger
	path    string
	query   string
	remote  string
	header  http.Header
}

func discoveryRequestFrom(input *discoveryServiceRequest) *httptest.ResponseRecorder {
	engine := gin.New()
	NewNotAuthHandler(input.service, input.cfg, input.logger).RegisterRouters(engine)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, input.path+"?"+input.query, nil)
	request.RemoteAddr = input.remote
	for key, values := range input.header {
		for _, value := range values {
			request.Header.Add(key, value)
		}
	}
	engine.ServeHTTP(response, request)
	return response
}
func TestHTTPDiscoveryAliasesKeepRawPrometheusProtocol(t *testing.T) {
	for _, path := range []string{"/api/not_auth/getBindIps", "/api/not_auth/getTreeNodeBindIps"} {
		t.Run(path, func(t *testing.T) {
			svc := &discoveryService{}
			response := discoveryRequest(svc, path, "port=9100&tree_node_ids="+url.QueryEscape(" 7,7,,8,7 "))
			if response.Code != http.StatusOK || svc.port != 9100 || !reflect.DeepEqual(svc.ids, []int{7, 8}) {
				t.Fatal("alias or normalized query changed")
			}
			var groups []*targetgroup.Group
			if err := json.Unmarshal(response.Body.Bytes(), &groups); err != nil || len(groups) != 1 || groups[0].Targets[0][pm.AddressLabel] != "127.0.0.1:9100" {
				t.Fatal("successful response no longer follows raw HTTP SD target-group format")
			}
		})
	}
}
func TestInvalidDiscoveryQueriesRejectedBeforeService(t *testing.T) {
	unique := make([]string, 101)
	for i := range unique {
		unique[i] = strconv.Itoa(i + 1)
	}
	for _, query := range []string{
		"port=9100", "port=9100&tree_node_ids=", "port=0&tree_node_ids=7", "port=65536&tree_node_ids=7", "port=no&tree_node_ids=7",
		"port=9100&tree_node_ids=7,bad,8", "port=9100&tree_node_ids=-1", "port=9100&tree_node_ids=0", "port=9100&tree_node_ids=,,,",
		"port=9100&tree_node_ids=" + strings.Join(unique, ","),
		"port=9100&tree_node_ids=" + strings.Repeat("1,", service.MaxDiscoveryIDValues+1),
		"port=9100&tree_node_ids=" + strings.Repeat("1", service.MaxDiscoveryIDsLength+1),
		"port=9100&tree_node_ids=7&extra=" + strings.Repeat("x", service.MaxDiscoveryQueryLength),
	} {
		svc := &discoveryService{}
		response := discoveryRequest(svc, "/api/not_auth/getBindIps", query)
		if response.Code != http.StatusBadRequest || svc.calls != 0 {
			t.Fatal("invalid query was accepted or caused partial service reads")
		}
	}
}

func TestHTTPServiceDiscoveryAuthentication(t *testing.T) {
	const token = "fixture-http-sd-token"
	const wrong = "fixture-http-sd-wrong"
	query := "port=9100&tree_node_ids=7"
	configured := &config.Config{Prometheus: config.PrometheusConfig{HttpSdToken: token}}
	invalid := &config.Config{Prometheus: config.PrometheusConfig{HttpSdToken: "fixture token"}}
	for _, path := range []string{"/api/not_auth/getBindIps", "/api/not_auth/getTreeNodeBindIps"} {
		t.Run(path, func(t *testing.T) {
			for _, tt := range []struct {
				name, remote, authorization string
				cfg                         *config.Config
				extraQuery                  string
				forwarded                   string
				status                      int
				called                      bool
			}{
				{name: "configured missing token", remote: "203.0.113.10:1234", cfg: configured, status: http.StatusUnauthorized},
				{name: "configured wrong token", remote: "203.0.113.10:1234", cfg: configured, authorization: "Bearer " + wrong, status: http.StatusUnauthorized},
				{name: "configured token prefix", remote: "203.0.113.10:1234", cfg: configured, authorization: "Bearer " + token + "-extra", status: http.StatusUnauthorized},
				{name: "configured correct token", remote: "203.0.113.10:1234", cfg: configured, authorization: "Bearer " + token, status: http.StatusOK, called: true},
				{name: "configured bearer case", remote: "198.51.100.10:1234", cfg: configured, authorization: "bearer " + token, status: http.StatusOK, called: true},
				{name: "configured loopback still needs token", remote: "127.0.0.1:1234", cfg: configured, status: http.StatusUnauthorized},
				{name: "configured query token ignored", remote: "203.0.113.10:1234", cfg: configured, extraQuery: "&token=" + token, status: http.StatusUnauthorized},
				{name: "unconfigured remote", remote: "203.0.113.10:1234", status: http.StatusUnauthorized},
				{name: "unconfigured private network", remote: "172.18.0.5:1234", status: http.StatusUnauthorized},
				{name: "unconfigured forwarded loopback", remote: "203.0.113.10:1234", forwarded: "127.0.0.1", status: http.StatusUnauthorized},
				{name: "unconfigured loopback", remote: "127.0.0.1:1234", status: http.StatusOK, called: true},
				{name: "unconfigured ipv6 loopback", remote: "[::1]:1234", status: http.StatusOK, called: true},
				{name: "invalid token uses loopback default", remote: "203.0.113.10:1234", cfg: invalid, authorization: "Bearer fixture token", status: http.StatusUnauthorized},
				{name: "invalid token allows loopback", remote: "127.0.0.1:1234", cfg: invalid, status: http.StatusOK, called: true},
			} {
				t.Run(tt.name, func(t *testing.T) {
					core, recorded := observer.New(zap.DebugLevel)
					svc := &discoveryService{}
					header := make(http.Header)
					if tt.authorization != "" {
						header.Set("Authorization", tt.authorization)
					}
					if tt.forwarded != "" {
						header.Set("X-Forwarded-For", tt.forwarded)
						header.Set("X-Real-IP", tt.forwarded)
					}
					response := discoveryRequestFrom(&discoveryServiceRequest{
						service: svc, cfg: tt.cfg, logger: zap.New(core), path: path,
						query: query + tt.extraQuery, remote: tt.remote, header: header,
					})
					if response.Code != tt.status || (svc.calls > 0) != tt.called {
						t.Fatalf("status=%d calls=%d", response.Code, svc.calls)
					}
					body := response.Body.String()
					for _, secret := range []string{token, wrong, "fixture token"} {
						if strings.Contains(body, secret) {
							t.Fatal("response revealed a token")
						}
					}
					if !tt.called && strings.Contains(body, "127.0.0.1:9100") {
						t.Fatal("rejected request still returned discovery targets")
					}
					for _, entry := range recorded.All() {
						encoded, err := json.Marshal(entry)
						if err != nil {
							t.Fatal(err)
						}
						text := string(encoded)
						if strings.Contains(text, token) || strings.Contains(text, wrong) || strings.Contains(text, "Authorization") || strings.Contains(text, "fixture token") {
							t.Fatal("log revealed the HTTP SD credential")
						}
					}
				})
			}
		})
	}
}

func TestConfiguredHTTPDiscoveryRejectsBeforeQueryValidation(t *testing.T) {
	svc := &discoveryService{}
	response := discoveryRequestFrom(&discoveryServiceRequest{
		service: svc,
		cfg:     &config.Config{Prometheus: config.PrometheusConfig{HttpSdToken: "fixture-http-sd-token"}},
		path:    "/api/not_auth/getBindIps",
		query:   "port=no&tree_node_ids=7",
		remote:  "203.0.113.10:1234",
	})
	if response.Code != http.StatusUnauthorized || svc.calls != 0 || strings.Contains(response.Body.String(), "port") {
		t.Fatal("unauthenticated request reached query validation")
	}
}
