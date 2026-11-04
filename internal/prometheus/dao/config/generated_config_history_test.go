package config

import (
	"context"
	"errors"
	"testing"
)

func TestGeneratedHistoryPreservesAllOwnersDeletedRowsAndByteKeys(t *testing.T) {
	dao, db := generatedQueryFixture(t)
	seedGeneratedQueryRows(t, db)
	ctx := context.Background()
	for page, want := range [][]int{{1, 2}, {3, 6}, {}} {
		rows, total, err := dao.GetGeneratedConfigHistoryPage(ctx, GeneratedConfigKeysQuery{ConfigType: 4, InstanceKeys: []string{"192.0.2.1"}, Page: page + 1, Size: 2})
		assertConfigQueryIDs(t, rows, total, err, 4, want...)
	}
	for _, test := range []struct {
		key string
		id  int
	}{{"2001:db8::1", 7}, {"2001:DB8::1", 8}, {"192.0.2.3", 9}, {"192.0.2.3 ", 10}, {" 192.0.2.3", 11}} {
		rows, total, err := dao.GetGeneratedConfigHistoryPage(ctx, GeneratedConfigKeysQuery{ConfigType: 4, InstanceKeys: []string{test.key}, Page: 1, Size: 100})
		assertConfigQueryIDs(t, rows, total, err, 1, test.id)
	}
	rows, total, err := dao.GetGeneratedConfigHistoryPage(ctx, GeneratedConfigKeysQuery{ConfigType: 4, Page: 1, Size: 100})
	assertConfigQueryIDs(t, rows, total, err, 0)
}
func TestGeneratedHistoryRejectsInvalidQueryAndPreservesCancellation(t *testing.T) {
	dao, _ := generatedQueryFixture(t)
	for _, req := range []GeneratedConfigKeysQuery{{ConfigType: 4, Page: 0, Size: 100}, {ConfigType: 0, Page: 1, Size: 100}, {ConfigType: 4, Page: 1, Size: 101}, {ConfigType: 4, Page: 1, Size: 100, InstanceKeys: []string{""}}, {ConfigType: 4, Page: 1, Size: 100, InstanceKeys: make([]string, 101)}} {
		if _, _, err := dao.GetGeneratedConfigHistoryPage(context.Background(), req); err == nil {
			t.Fatal("invalid history query accepted")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := dao.GetGeneratedConfigHistoryPage(ctx, GeneratedConfigKeysQuery{ConfigType: 4, Page: 1, Size: 100}); !errors.Is(err, context.Canceled) {
		t.Fatal("history cancellation lost", err)
	}
}

func TestGeneratedHistoryCandidatesLocateUnindexedTombstonesWithoutGrantingOwnership(t *testing.T) {
	dao, db := generatedQueryFixture(t)
	seedGeneratedQueryRows(t, db)
	rows, total, err := dao.GetGeneratedConfigHistoryCandidates(context.Background(), GeneratedConfigPoolQuery{PoolID: 7, ConfigType: 4, Page: 1, Size: 100})
	assertConfigQueryIDs(t, rows, total, err, 8, 1, 2, 4, 6, 7, 8, 9, 10)
	// A locator scope must not replace the full namespace history check.
	rows, total, err = dao.GetGeneratedConfigHistoryPage(context.Background(), GeneratedConfigKeysQuery{ConfigType: 4, InstanceKeys: []string{"192.0.2.1"}, Page: 1, Size: 100})
	assertConfigQueryIDs(t, rows, total, err, 4, 1, 2, 3, 6)
	if _, _, err := dao.GetGeneratedConfigHistoryCandidates(context.Background(), GeneratedConfigPoolQuery{PoolID: 0, ConfigType: 4, Page: 1, Size: 100}); err == nil {
		t.Fatal("unscoped invalid locator accepted")
	}
}
