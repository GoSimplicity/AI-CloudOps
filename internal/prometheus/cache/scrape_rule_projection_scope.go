package cache

import (
	"context"
	"errors"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	"github.com/redis/go-redis/v9"
)

type scrapeRuleScope int8

const (
	scrapeAlertScope  scrapeRuleScope = 3
	scrapeRecordScope scrapeRuleScope = 4
)

func (s scrapeRuleScope) valid() bool { return s == scrapeAlertScope || s == scrapeRecordScope }
func (s scrapeRuleScope) pattern() string {
	if s == scrapeAlertScope {
		return ConfigNameAlertRule
	}
	if s == scrapeRecordScope {
		return ConfigNameRecordRule
	}
	return ""
}
func (s scrapeRuleScope) rawKey(ip string) string {
	if s == scrapeAlertScope {
		return buildRedisKeyAlertRule(ip)
	}
	if s == scrapeRecordScope {
		return buildRedisKeyRecordRule(ip)
	}
	return ""
}
func (s scrapeRuleScope) membersKey(pool int) string {
	if s == scrapeAlertScope {
		return buildRedisSetKeyAlertRulePoolIPs(pool)
	}
	if s == scrapeRecordScope {
		return buildRedisSetKeyRecordRulePoolIPs(pool)
	}
	return ""
}
func (s scrapeRuleScope) stateKey(pool int) string {
	if s == scrapeAlertScope {
		return alertRulePublicationKey(pool)
	}
	if s == scrapeRecordScope {
		return recordPublicationKey(pool)
	}
	return ""
}
func ownedScrapeRuleArtifact(row *model.MonitorConfig, scope scrapeRuleScope, poolID int) bool {
	return scope.valid() && row != nil && row.ID > 0 && row.PoolID == poolID && row.ConfigType == int8(scope) && validateInstanceIPs([]string{row.InstanceIP}) == nil && row.Name == fmt.Sprintf(scope.pattern(), poolID, row.InstanceIP)
}

type scrapeRuleJournal interface {
	configdao.ProjectionPlanJournalDAO
	sourceVersions(context.Context) ([]configdao.RecordSourceVersion, error)
	reserve(context.Context, configdao.PoolOperationRequest, uint64) (configdao.PoolReservation, error)
	commit(context.Context, configdao.RecordProjectionPlanRequest) (configdao.ProjectionReceipt, error)
}
type recordJournalFacade struct {
	configdao.RecordProjectionJournalDAO
}

func (j recordJournalFacade) sourceVersions(ctx context.Context) ([]configdao.RecordSourceVersion, error) {
	return j.ReadRecordSourceVersions(ctx)
}
func (j recordJournalFacade) reserve(ctx context.Context, req configdao.PoolOperationRequest, version uint64) (configdao.PoolReservation, error) {
	return j.ReserveRecordPoolOperation(ctx, req, version)
}
func (j recordJournalFacade) commit(ctx context.Context, req configdao.RecordProjectionPlanRequest) (configdao.ProjectionReceipt, error) {
	return j.CommitRecordProjectionPlan(ctx, req)
}

type alertJournalFacade struct {
	configdao.AlertRuleProjectionJournalDAO
}

func (j alertJournalFacade) sourceVersions(ctx context.Context) ([]configdao.RecordSourceVersion, error) {
	return j.ReadAlertRuleSourceVersions(ctx)
}
func (j alertJournalFacade) reserve(ctx context.Context, req configdao.PoolOperationRequest, version uint64) (configdao.PoolReservation, error) {
	return j.ReserveAlertRulePoolOperation(ctx, req, version)
}
func (j alertJournalFacade) commit(ctx context.Context, req configdao.RecordProjectionPlanRequest) (configdao.ProjectionReceipt, error) {
	return j.CommitAlertRuleProjectionPlan(ctx, req)
}

type scrapeRuleHistory interface {
	GetGeneratedConfigHistoryCandidates(context.Context, configdao.GeneratedConfigPoolQuery) ([]*model.MonitorConfig, int64, error)
	GetGeneratedConfigHistoryPage(context.Context, configdao.GeneratedConfigKeysQuery) ([]*model.MonitorConfig, int64, error)
	superseded(context.Context, configdao.ProjectionReceipt) ([]configdao.RecordSupersededArtifact, error)
	legacy(context.Context, int, []configdao.GeneratedConfigOwner) ([]configdao.RecordSupersededArtifact, error)
	resolved(context.Context, int, []configdao.GeneratedConfigOwner) ([]configdao.GeneratedConfigOwner, error)
}
type recordHistoryFacade struct {
	configdao.GeneratedConfigHistoryReader
}

func (h recordHistoryFacade) superseded(ctx context.Context, receipt configdao.ProjectionReceipt) ([]configdao.RecordSupersededArtifact, error) {
	return h.GetSupersededRecordArtifacts(ctx, receipt)
}
func (h recordHistoryFacade) legacy(ctx context.Context, pool int, owners []configdao.GeneratedConfigOwner) ([]configdao.RecordSupersededArtifact, error) {
	return h.GetLegacySupersededRecordArtifacts(ctx, pool, owners)
}
func (h recordHistoryFacade) resolved(ctx context.Context, pool int, owners []configdao.GeneratedConfigOwner) ([]configdao.GeneratedConfigOwner, error) {
	return h.GetProjectedRecordRetirements(ctx, pool, owners)
}

type alertHistoryFacade struct {
	configdao.AlertRuleHistoryReader
}

func (h alertHistoryFacade) superseded(ctx context.Context, receipt configdao.ProjectionReceipt) ([]configdao.RecordSupersededArtifact, error) {
	return h.GetSupersededAlertRuleArtifacts(ctx, receipt)
}
func (h alertHistoryFacade) legacy(ctx context.Context, pool int, owners []configdao.GeneratedConfigOwner) ([]configdao.RecordSupersededArtifact, error) {
	return h.GetLegacySupersededAlertRuleArtifacts(ctx, pool, owners)
}
func (h alertHistoryFacade) resolved(ctx context.Context, pool int, owners []configdao.GeneratedConfigOwner) ([]configdao.GeneratedConfigOwner, error) {
	return h.GetProjectedAlertRuleRetirements(ctx, pool, owners)
}

type scrapeRuleProjection struct {
	scope      scrapeRuleScope
	redis      redis.Cmdable
	configDAO  configdao.MonitorConfigDAO
	journal    scrapeRuleJournal
	history    scrapeRuleHistory
	projection interface {
		Project(context.Context, configdao.ProjectionLease) error
	}
}

func (r *scrapeRuleProjection) ready() error {
	if r == nil || !r.scope.valid() || r.redis == nil || r.configDAO == nil || r.journal == nil || r.history == nil || r.projection == nil {
		return errors.New("规则持久发布依赖不可用")
	}
	return nil
}

type scrapeRuleProjectionRunner struct {
	store *ProjectionStore
	scope scrapeRuleScope
}

func (r *scrapeRuleProjectionRunner) Project(ctx context.Context, lease configdao.ProjectionLease) error {
	if r == nil || !r.scope.valid() || r.store == nil || lease == nil {
		return errors.New("规则投影存储不可用")
	}
	snapshot, err := lease.Snapshot()
	if err != nil {
		return err
	}
	if snapshot.Pool == nil || snapshot.Pool.Reservation.Family != int8(r.scope) {
		return errors.New("规则投影缺少匹配的持久计划")
	}
	prepared, err := r.store.Prepare(ctx, lease)
	if err != nil {
		return err
	}
	for _, artifact := range snapshot.Pool.Artifacts {
		if err := prepared.ApplyArtifact(ctx, artifact.EventID); err != nil {
			return err
		}
	}
	for _, control := range snapshot.Pool.Controls {
		if control.Members != nil {
			if err := prepared.ApplyMembers(ctx, control.ControlID); err != nil {
				return err
			}
		}
	}
	proof, err := prepared.Acknowledge(ctx)
	if err != nil {
		return err
	}
	marked, err := lease.MarkComplete(proof)
	if err != nil {
		return err
	}
	if !marked {
		return errors.New("规则投影回执已失效")
	}
	return nil
}
