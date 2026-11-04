package cache

import "testing"

// Main maps the shared off intent to an empty target list; no support flag is
// added. Jobs remain validated unless the pool is explicitly deleted.

func newPromMainReuseFixture(t *testing.T) *recordReuseFixture {
	return newRuleFamilyReuseFixture(t, 1)
}

func TestPromMainProjectionCompletedRetirementDoesNotReclaimReusedIP(t *testing.T) {
	runRecordProjectionCompletedRetirementDoesNotReclaimReusedIP(t, newPromMainReuseFixture)
}

func TestPromMainProjectionPendingRetirementRequiresProjectedReplacement(t *testing.T) {
	runRecordProjectionPendingRetirementRequiresProjectedReplacement(t, newPromMainReuseFixture)
}

func TestPromMainProjectionSupersededRecoveryRetainsProofBeforeACK(t *testing.T) {
	runRecordProjectionSupersededRecoveryRetainsProofBeforeACK(t, newPromMainReuseFixture)
}

func TestPromMainProjectionAcceptedRenderCanCloseOnlyAfterTargetRemoved(t *testing.T) {
	runRecordProjectionAcceptedRenderCanCloseOnlyAfterTargetRemoved(t, newPromMainReuseFixture)
}

func TestPromMainProjectionLegacyOwnerRequiresCompletedReplacement(t *testing.T) {
	runRecordProjectionLegacyOwnerRequiresCompletedReplacement(t, newPromMainReuseFixture)
}

func TestPromMainProjectionCurrentOwnerCanRetireWithDeletedForeignHistory(t *testing.T) {
	runRecordProjectionCurrentOwnerCanRetireWithDeletedForeignHistory(t, newPromMainReuseFixture)
}

func TestPromMainProjectionIssuedRetirementRetriesWithDeletedForeignHistory(t *testing.T) {
	runRecordProjectionIssuedRetirementRetriesWithDeletedForeignHistory(t, newPromMainReuseFixture)
}

func TestPromMainProjectionRetirementReplayDoesNotHideNewDBOnlyStale(t *testing.T) {
	runRecordProjectionRetirementReplayDoesNotHideNewDBOnlyStale(t, newPromMainReuseFixture)
}
