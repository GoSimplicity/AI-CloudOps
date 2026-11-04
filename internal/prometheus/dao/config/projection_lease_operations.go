package config

import (
	"errors"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
)

// These are only parsed membership facts, not a complete receipt or retirement
// permit. In particular, they cannot replace AM's full receipt/event/cardinality
// and current intent-owner validation. Nothing here grants namespace authority.
type leaseRootOperationFacts struct {
	family     int8
	byEventID  map[uint64]uint64
	bySourceID map[uint64]uint64
}

func (f *leaseRootOperationFacts) containsEvent(eventID uint64) bool {
	sourceID, ok := f.byEventID[eventID]
	return ok && f.bySourceID[sourceID] == eventID
}

// One instance belongs only to validateInitialCurrent and its original tx. It is
// never stored on the DAO/lease or reused by another callback or ordinary write.
type leaseInitialOperations struct {
	tx     *gorm.DB
	parsed map[uint64]*leaseRootOperationFacts
}

func newLeaseInitialOperations(tx *gorm.DB) *leaseInitialOperations {
	return &leaseInitialOperations{tx: tx, parsed: map[uint64]*leaseRootOperationFacts{}}
}
func parseLeaseRootOperation(op model.MonitorConfigPoolOperation) (*leaseRootOperationFacts, error) {
	if !operationStateValid(op) || op.CommittedAt == nil {
		return nil, ErrJournalConflict
	}
	// Decode the entire envelope, not just the matching artifact. Unknown fields,
	// malformed controls, duplicate IDs, and a bad later entry must still fail.
	envelope, err := decodeProjectionManifest(op.Manifest)
	if err != nil {
		return nil, err
	}
	facts := &leaseRootOperationFacts{family: op.Family, byEventID: map[uint64]uint64{}, bySourceID: map[uint64]uint64{}}
	for _, entry := range envelope.Artifacts {
		facts.byEventID[entry.EventID] = entry.SourceID
		facts.bySourceID[entry.SourceID] = entry.EventID
	}
	return facts, nil
}
func (r *leaseInitialOperations) operation(operationID uint64) (*leaseRootOperationFacts, error) {
	if facts, ok := r.parsed[operationID]; ok {
		return facts, nil
	}
	var op model.MonitorConfigPoolOperation
	// Historical payload is append-only; do not acquire its pool control or
	// unrelated namespace locks while traversing a namespace-only lineage.
	if err := r.tx.Where("operation_id = ?", operationID).First(&op).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	facts, err := parseLeaseRootOperation(op)
	if err != nil {
		return nil, nil
	} // Match trustedBindingRoot's invalid-root contract.
	r.parsed[operationID] = facts
	return facts, nil
}

// Called only with the receipt returned by the actual SQL full loader, after its
// equality/current-control checks. A caller-supplied receipt is never a seed.
// Retain only its lower-level manifest membership, not an AM full-proof cache.
func (r *leaseInitialOperations) seedVerifiedSQLReceipt(op model.MonitorConfigPoolOperation, receipt ProjectionReceipt) error {
	if !operationMatches(op, receipt.Reservation) || !sameHash(receipt.SourceHash, op.SourceHash) {
		return ErrJournalConflict
	}
	facts, err := parseLeaseRootOperation(op)
	if err != nil {
		return err
	}
	if len(facts.byEventID) != len(receipt.Artifacts) {
		return ErrJournalConflict
	}
	for _, artifact := range receipt.Artifacts {
		if artifact.OperationID == nil || *artifact.OperationID != op.OperationID ||
			!familyAllowsKind(op.Family, artifact.ConfigType) || facts.bySourceID[artifact.SourceID] != artifact.EventID ||
			facts.byEventID[artifact.EventID] != artifact.SourceID {
			return ErrJournalConflict
		}
	}
	r.parsed[op.OperationID] = facts
	return nil
}
