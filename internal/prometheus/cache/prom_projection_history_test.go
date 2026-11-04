package cache

import "testing"

func TestPromMainProjectionMemberOnlyLocatorCannotAuthorizeDeletion(t *testing.T) {
	runProjectionMemberOnlyLocatorCannotAuthorizeDeletion(t, newPromMainReuseFixture)
}

func TestPromMainProjectionHistoricalOwnersRequireExactNamespaceProof(t *testing.T) {
	runProjectionHistoricalOwnersRequireExactNamespaceProof(t, newPromMainReuseFixture)
}

func TestPromMainProjectionCommitRechecksHistoryAfterForeignABA(t *testing.T) {
	runProjectionCommitRechecksHistoryAfterForeignABA(t, newPromMainReuseFixture)
}
