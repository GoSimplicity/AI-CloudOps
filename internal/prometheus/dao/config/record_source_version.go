package config

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

type RecordSourceVersion struct {
	PoolID  int
	Counter uint64
}

// Capture durable attempt versions before reading any pool/rule source. An
// absent pool is version zero; a later writer's reservation makes its CAS fail.
func (d *monitorConfigDAO) ReadRecordSourceVersions(ctx context.Context) ([]RecordSourceVersion, error) {
	return d.readScrapeRuleSourceVersions(ctx, scrapeRecordPublication)
}
func (d *monitorConfigDAO) ReserveRecordPoolOperation(ctx context.Context, req PoolOperationRequest, expectedCounter uint64) (PoolReservation, error) {
	if req.Family != model.ConfigPublicationFamilyRecordRule {
		return PoolReservation{}, ErrJournalInvalid
	}
	return d.reservePoolOperation(ctx, req, &expectedCounter)
}
