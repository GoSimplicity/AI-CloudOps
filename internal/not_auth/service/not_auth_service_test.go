package service

import (
	"context"
	"reflect"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	treeDAO "github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	pm "github.com/prometheus/common/model"
	"go.uber.org/zap"
)

type discoveryTree struct {
	treeDAO.TreeNodeDAO
	calls []int
}

func (s *discoveryTree) GetNode(_ context.Context, id int) (*model.TreeNode, error) {
	s.calls = append(s.calls, id)
	return &model.TreeNode{TreeLocalResources: []*model.TreeLocalResource{{IpAddr: "127.0.0.1"}, {IpAddr: "2001:db8::1"}, {IpAddr: "127.0.0.1"}}}, nil
}
func TestDiscoveryDeduplicatesQueriesAndAddressesInStableOrder(t *testing.T) {
	store := &discoveryTree{}
	svc := NewNotAuthService(zap.NewNop(), store)
	result, err := svc.BuildPrometheusServiceDiscovery(context.Background(), 9100, []int{7, 7, 8, 7, 8})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(store.calls, []int{7, 8}) {
		t.Fatal("duplicate IDs amplified database queries")
	}
	if len(result) != 1 || len(result[0].Targets) != 2 || result[0].Targets[1][pm.AddressLabel] != "[2001:db8::1]:9100" || result[0].Labels["tree_node_ids"] != "7,8" {
		t.Fatal("target order, IPv6 encoding or deduplicated labels changed")
	}
}
func TestDiscoveryLimitsBeforeDatabaseAccess(t *testing.T) {
	for _, tt := range []struct {
		name string
		ids  []int
	}{
		{"empty", nil}, {"negative", []int{-1}}, {"zero", []int{0}},
		{"too many unique", func() []int {
			ids := make([]int, 101)
			for i := range ids {
				ids[i] = i + 1
			}
			return ids
		}()},
		{"too many raw values", make([]int, MaxDiscoveryIDValues+1)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			store := &discoveryTree{}
			svc := NewNotAuthService(zap.NewNop(), store)
			if _, err := svc.BuildPrometheusServiceDiscovery(context.Background(), 9100, tt.ids); err == nil {
				t.Fatal("unbounded or invalid input accepted")
			}
			if len(store.calls) != 0 {
				t.Fatal("invalid input performed partial database reads")
			}
		})
	}
	ids := make([]int, MaxDiscoveryNodeIDs)
	for i := range ids {
		ids[i] = i + 1
	}
	store := &discoveryTree{}
	svc := NewNotAuthService(zap.NewNop(), store)
	if _, err := svc.BuildPrometheusServiceDiscovery(context.Background(), 9100, ids); err != nil {
		t.Fatal(err)
	}
	if len(store.calls) != MaxDiscoveryNodeIDs {
		t.Fatal("documented maximum rejected")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	store.calls = nil
	if _, err := svc.BuildPrometheusServiceDiscovery(ctx, 9100, []int{1}); err != context.Canceled {
		t.Fatal("request cancellation ignored")
	}
	if len(store.calls) != 0 {
		t.Fatal("cancelled request reached database")
	}
}
