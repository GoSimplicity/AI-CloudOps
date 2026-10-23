package utils

import "testing"

func TestPaginateDoesNotOverflowOnUntrustedBounds(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	for _, test := range []struct{ page, size, length int }{
		{maxInt, maxInt, 0}, {maxInt, 2, 0}, {2, maxInt, 0},
		{1, maxInt, 3}, {2, 2, 1}, {0, 0, 3},
	} {
		items, total := Paginate([]int{1, 2, 3}, test.page, test.size)
		if total != 3 || len(items) != test.length {
			t.Fatalf("page=%d size=%d: total=%d items=%v", test.page, test.size, total, items)
		}
	}
}
