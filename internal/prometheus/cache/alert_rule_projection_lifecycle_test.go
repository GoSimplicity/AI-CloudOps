package cache

import "testing"

func newAlertRuleReuseFixture(t *testing.T) *recordReuseFixture {
	return newRuleFamilyReuseFixture(t, 3)
}

func TestAlertRuleProjectionCompletedRetirementDoesNotReclaimReusedIP(t *testing.T) {
	runRecordProjectionCompletedRetirementDoesNotReclaimReusedIP(t, newAlertRuleReuseFixture)
}

func TestAlertRuleProjectionPendingRetirementRequiresProjectedReplacement(t *testing.T) {
	runRecordProjectionPendingRetirementRequiresProjectedReplacement(t, newAlertRuleReuseFixture)
}

func TestAlertRuleProjectionSupersededRecoveryRetainsProofBeforeACK(t *testing.T) {
	runRecordProjectionSupersededRecoveryRetainsProofBeforeACK(t, newAlertRuleReuseFixture)
}

func TestAlertRuleProjectionAcceptedRenderCanCloseOnlyAfterTargetRemoved(t *testing.T) {
	runRecordProjectionAcceptedRenderCanCloseOnlyAfterTargetRemoved(t, newAlertRuleReuseFixture)
}

func TestAlertRuleProjectionLegacyOwnerRequiresCompletedReplacement(t *testing.T) {
	runRecordProjectionLegacyOwnerRequiresCompletedReplacement(t, newAlertRuleReuseFixture)
}

func TestAlertRuleProjectionCurrentOwnerCanRetireWithDeletedForeignHistory(t *testing.T) {
	runRecordProjectionCurrentOwnerCanRetireWithDeletedForeignHistory(t, newAlertRuleReuseFixture)
}

func TestAlertRuleProjectionIssuedRetirementRetriesWithDeletedForeignHistory(t *testing.T) {
	runRecordProjectionIssuedRetirementRetriesWithDeletedForeignHistory(t, newAlertRuleReuseFixture)
}

func TestAlertRuleProjectionRetirementReplayDoesNotHideNewDBOnlyStale(t *testing.T) {
	runRecordProjectionRetirementReplayDoesNotHideNewDBOnlyStale(t, newAlertRuleReuseFixture)
}
