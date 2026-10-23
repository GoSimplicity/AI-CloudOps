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

	"github.com/GoSimplicity/AI-CloudOps/internal/not_auth/service"
	"github.com/gin-gonic/gin"
	pm "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery/targetgroup"
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
	engine := gin.New()
	NewNotAuthHandler(service).RegisterRouters(engine)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path+"?"+query, nil))
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
