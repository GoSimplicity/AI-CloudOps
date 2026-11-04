package cache

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"time"

	"errors"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/redis/go-redis/v9"

	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
)

func (r *scrapeRuleProjection) publish(ctx context.Context, pool *model.MonitorScrapePool, generated map[string]string, targetHash string, sourceVersion uint64, fail func(string) error) error {
	decoded, err := hex.DecodeString(targetHash)
	if err != nil || len(decoded) != 32 {
		return fail("发布指纹无效")
	}
	var sourceHash [32]byte
	copy(sourceHash[:], decoded)
	previous, err := r.journal.LoadLastCommittedProjection(ctx, int8(r.scope), pool.ID)
	if err != nil {
		return fail("读取持久发布计划")
	}
	if previous != nil && (previous.Reservation.Family != int8(r.scope) || previous.Reservation.PoolID != pool.ID) {
		return fail("持久计划归属不一致")
	}
	superseded := []configdao.ScrapeRuleSupersededArtifact{}
	supersededKeys := map[string]bool{}
	supersededOwners := map[int]configdao.GeneratedConfigOwner{}
	if previous != nil {
		superseded, err = r.history.superseded(ctx, *previous)
		if err != nil {
			return fail("核对已完成替代投影")
		}
		applicable := make([]configdao.ScrapeRuleSupersededArtifact, 0, len(superseded))
		for _, proof := range superseded {
			if _, wanted := generated[proof.RetiredOwner.InstanceIP]; wanted {
				continue
			}
			applicable = append(applicable, proof)
			if proof.RetiredOwner.PoolID != pool.ID || proof.AcceptedOperationID != previous.Reservation.OperationID {
				return fail("替代退休证明超出归属")
			}
			supersededKeys[proof.RetiredOwner.InstanceIP] = true
			supersededOwners[proof.RetiredOwner.ID] = proof.RetiredOwner
		}
		superseded = applicable
	}
	oldIPs, err := r.redis.SMembers(ctx, r.scope.membersKey(pool.ID)).Result()
	if err != nil {
		return fail("读取实例成员")
	}
	// A committed plan remains a durable locator even if raw/membership writes
	// were lost. Completed retirements no longer claim a subsequently reused IP.
	if previous != nil {
		for _, artifact := range previous.Artifacts {
			if !supersededKeys[artifact.InstanceKey] && (artifact.Action == model.ConfigProjectionRender || previous.Phase != model.ConfigPublicationProjected) {
				oldIPs = append(oldIPs, artifact.InstanceKey)
			}
		}
	}
	historyIPs, resolvedIPs, legacyProofs, err := r.historicalLocators(ctx, pool.ID, generated, supersededOwners)
	if err != nil {
		return fail("发现历史配置")
	}
	superseded = append(superseded, legacyProofs...)
	for _, ip := range resolvedIPs {
		supersededKeys[ip] = true
	}
	filtered := make([]string, 0, len(oldIPs))
	for _, ip := range oldIPs {
		if !supersededKeys[ip] {
			filtered = append(filtered, ip)
		}
	}
	oldIPs = append(filtered, historyIPs...)
	plan, err := planScrapeRulePublication(ctx, r.scope, r.configDAO, pool.ID, generated, oldIPs)
	if err != nil {
		return fail("检查配置归属")
	}
	if previous != nil && previous.SourceHash == sourceHash && previous.ManifestVersion == 2 && previous.CompleteControls && len(plan.create) == 0 && len(plan.update) == 0 && scrapeRuleReceiptCoversPlan(*previous, generated, plan) {
		err := r.journal.WithPoolProjectionLease(ctx, *previous, func(lease configdao.ProjectionLease) error {
			healthy, err := r.healthy(ctx, *previous, generated)
			if err != nil {
				return err
			}
			if healthy {
				return nil
			}
			return r.projection.Project(ctx, lease)
		})
		if err == nil {
			return nil
		}
		if !errors.Is(err, configdao.ErrJournalConflict) {
			return fail("恢复持久投影")
		}
		// A current ordinary mutation can invalidate an old vector. Construct a new
		// operation from freshly validated owner snapshots, never reuse its authority.
	}
	// A signed current permit can be retried without minting new legacy
	// authority. Only a new plan needs fresh tombstone disambiguation below.
	if err := r.resolveRetirements(ctx, pool.ID, &plan, previous); err != nil {
		return fail("检查历史配置归属")
	}
	historyGuards, err := r.captureHistoryGuards(ctx, pool.ID, plan)
	if err != nil {
		return fail("读取退休历史前置条件")
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return fail("生成发布标识")
	}
	reservationRequest := configdao.PoolOperationRequest{Family: int8(r.scope), PoolID: pool.ID, Nonce: nonce}
	if previous != nil {
		operationID := previous.Reservation.OperationID
		reservationRequest.RecoveryOfOperationID = &operationID
	}
	reservation, err := r.journal.reserve(ctx, reservationRequest, sourceVersion)
	if err != nil {
		return fail("预留持久发布版本")
	}
	request := scrapeRuleGeneratedPlan(reservation, sourceHash, generated, plan)
	commitRequest := configdao.ScrapeRuleProjectionPlanRequest{
		GeneratedProjectionPlanRequest: request,
		History:                        historyGuards,
		Superseded:                     superseded,
	}
	guardedJournal := r.journal
	receipt, err := guardedJournal.commit(ctx, commitRequest)
	if err != nil {
		if !errors.Is(err, configdao.ErrJournalUncertain) {
			return fail("提交持久发布计划")
		}
		// A lost SQL COMMIT reply is not proof of rollback. Only the nonce's actual
		// committed payload can supply a receipt; reserved NULL is not completion.
		facts, loadErr := r.journal.LoadProjectionByNonce(ctx, nonce)
		if loadErr != nil || facts.Projection == nil || facts.Projection.ManifestVersion != 2 || !facts.Projection.CompleteControls ||
			facts.Projection.Reservation != reservation || facts.Projection.SourceHash != sourceHash {
			return fail("发布提交结果待核对")
		}
		receipt = *facts.Projection
	}
	if err := r.journal.WithPoolProjectionLease(ctx, receipt, func(lease configdao.ProjectionLease) error { return r.projection.Project(ctx, lease) }); err != nil {
		return fail("执行持久投影")
	}
	return nil
}

func scrapeRuleGeneratedPlan(ref configdao.PoolReservation, hash [32]byte, generated map[string]string, plan scrapeRulePublicationPlan) configdao.GeneratedProjectionPlanRequest {
	keys := make([]string, 0, len(generated)+len(plan.stale))
	for ip := range generated {
		keys = append(keys, ip)
	}
	keys = append(keys, plan.stale...)
	sort.Strings(keys)
	items := make([]configdao.GeneratedPlanItem, 0, len(keys))
	sourceIDs := make([]uint64, 0, len(keys))
	desiredIPs := make([]string, 0, len(generated))
	for _, ip := range keys {
		sourceID := uint64(len(items) + 1)
		item := configdao.GeneratedPlanItem{SourceID: sourceID, Binding: &configdao.ProjectionBinding{Version: 1, Role: "artifact", ConfigType: ref.Family, InstanceIP: ip}}
		if body, wanted := generated[ip]; wanted {
			item.Action = model.ConfigProjectionRender
			row := &model.MonitorConfig{
				Name: fmt.Sprintf(scrapeRuleScope(ref.Family).pattern(), ref.PoolID, ip), PoolID: ref.PoolID,
				ConfigType: ref.Family, InstanceIP: ip, Status: model.ConfigStatusActive,
				ConfigContent: body, ConfigHash: calculateConfigHash(body), LastGeneratedTime: time.Now().Unix(),
			}
			if current := plan.owners[ip]; current != nil {
				owner := configdao.GeneratedConfigOwnerOf(current)
				item.Expected = &owner
				row.Model = current.Model
				if current.ConfigContent == body && current.ConfigHash == row.ConfigHash && current.Status == row.Status {
					row.LastGeneratedTime = current.LastGeneratedTime
				}
			}
			item.Desired = row
			desiredIPs = append(desiredIPs, ip)
		} else {
			item.Action = model.ConfigProjectionRetire
			owner := plan.retired[ip]
			item.Expected = &owner
		}
		items = append(items, item)
		sourceIDs = append(sourceIDs, sourceID)
	}
	return configdao.GeneratedProjectionPlanRequest{GeneratedPlanRequest: configdao.GeneratedPlanRequest{Reservation: ref, SourceHash: hash, Items: items}, Controls: []configdao.ProjectionControl{
		{ControlID: 1, Kind: configdao.ProjectionMembersFull, Members: &configdao.FullMembersControl{DesiredIPs: desiredIPs}},
		{ControlID: 2, Kind: configdao.ProjectionACKFull, Ack: &configdao.ProjectionAckControl{LegacySHA: &hash, ArtifactSourceIDs: sourceIDs, RequiredControlIDs: []uint64{1}}},
	}}
}

func (r *scrapeRuleProjection) resolveRetirements(ctx context.Context, poolID int, plan *scrapeRulePublicationPlan, previous *configdao.ProjectionReceipt) error {
	known := map[string]int{}
	if previous != nil {
		for _, artifact := range previous.Artifacts {
			if artifact.ConfigRowID != nil {
				known[artifact.InstanceKey] = *artifact.ConfigRowID
			}
		}
	}
	missing := []string{}
	knownRows := map[string]*model.MonitorConfig{}
	for _, ip := range plan.stale {
		if _, ok := plan.retired[ip]; ok {
			continue
		}
		if id, ok := known[ip]; ok {
			row, err := r.configDAO.GetGeneratedMonitorConfigByID(ctx, id)
			if err != nil || !ownedScrapeRuleArtifact(row, r.scope, poolID) || row.InstanceIP != ip {
				return errors.New("历史发布归属已变化")
			}
			knownRows[ip] = row
		}
		missing = append(missing, ip)
	}
	seenHistoryIDs := map[int]bool{}
	for start := 0; start < len(missing); start += configdao.GeneratedConfigQueryLimit {
		end := start + configdao.GeneratedConfigQueryLimit
		if end > len(missing) {
			end = len(missing)
		}
		batch := append([]string(nil), missing[start:end]...)
		wanted := map[string]bool{}
		for _, ip := range batch {
			wanted[ip] = true
		}
		rows, err := readConfigMetadataPages(ctx, func(page, size int) ([]*model.MonitorConfig, int64, error) {
			return r.history.GetGeneratedConfigHistoryPage(ctx, configdao.GeneratedConfigKeysQuery{ConfigType: int8(r.scope), InstanceKeys: batch, Page: page, Size: size})
		})
		if err != nil {
			return err
		}
		byKey := map[string][]*model.MonitorConfig{}
		for _, row := range rows {
			if seenHistoryIDs[row.ID] {
				return errors.New("历史配置跨批次标识重复")
			}
			seenHistoryIDs[row.ID] = true
			if row.ConfigType != int8(r.scope) || !wanted[row.InstanceIP] {
				return errors.New("历史配置超出查询范围")
			}
			byKey[row.InstanceIP] = append(byKey[row.InstanceIP], row)
		}
		for _, ip := range batch {
			matches := byKey[ip]
			for _, row := range matches {
				if !ownedScrapeRuleArtifact(row, r.scope, poolID) {
					return errors.New("历史命名空间含其他归属，保留原始缓存")
				}
			}
			var selected *model.MonitorConfig
			if known := knownRows[ip]; known != nil {
				for _, row := range matches {
					if row.ID == known.ID && configdao.GeneratedConfigOwnerOf(row) == configdao.GeneratedConfigOwnerOf(known) {
						selected = row
					}
				}
			} else if len(matches) == 1 {
				selected = matches[0]
			}
			if selected == nil {
				return errors.New("旧实例缺少唯一可验证配置归属，保留原始缓存")
			}
			plan.retired[ip] = configdao.GeneratedConfigOwnerOf(selected)
		}
	}
	return nil
}

func (r *scrapeRuleProjection) healthy(ctx context.Context, receipt configdao.ProjectionReceipt, generated map[string]string) (bool, error) {
	if receipt.Phase != model.ConfigPublicationProjected {
		return false, nil
	}
	expected := configdao.ProjectionAckProof{Reservation: receipt.Reservation, SourceHash: receipt.SourceHash, PlanDigest: receipt.PlanDigest}
	for _, control := range receipt.Controls {
		if control.Ack != nil {
			expected.ArtifactVector = append([]configdao.ProjectionVectorEntry{}, control.Ack.ArtifactVector...)
			expected.RequiredControls = append([]configdao.ProjectionControlDigest{}, control.Ack.RequiredControls...)
		}
	}
	raw, err := r.redis.Get(ctx, projectionPoolKey(int8(r.scope), receipt.Reservation.PoolID, "ack")).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var actual configdao.ProjectionAckProof
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&actual) != nil || decoder.Decode(new(interface{})) != io.EOF || !reflect.DeepEqual(actual, expected) {
		return false, nil
	}
	sha, err := r.redis.Get(ctx, r.scope.stateKey(receipt.Reservation.PoolID)).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if sha != hex.EncodeToString(receipt.SourceHash[:]) {
		return false, nil
	}
	members, err := r.redis.SMembers(ctx, r.scope.membersKey(receipt.Reservation.PoolID)).Result()
	if err != nil {
		return false, err
	}
	if len(members) != len(generated) {
		return false, nil
	}
	seen := map[string]bool{}
	for _, ip := range members {
		if _, wanted := generated[ip]; !wanted || seen[ip] {
			return false, nil
		}
		seen[ip] = true
	}
	for _, artifact := range receipt.Artifacts {
		body, err := r.redis.Get(ctx, r.scope.rawKey(artifact.InstanceKey)).Result()
		if artifact.Action == model.ConfigProjectionRender {
			if errors.Is(err, redis.Nil) {
				return false, nil
			}
			if err != nil {
				return false, err
			}
			if body != generated[artifact.InstanceKey] {
				return false, nil
			}
		} else {
			if err == nil {
				return false, nil
			}
			if !errors.Is(err, redis.Nil) {
				return false, err
			}
		}
	}
	return true, nil
}

func (r *scrapeRuleProjection) historicalLocators(ctx context.Context, poolID int, desired map[string]string, superseded map[int]configdao.GeneratedConfigOwner) ([]string, []string, []configdao.ScrapeRuleSupersededArtifact, error) {
	rows, err := readConfigMetadataPages(ctx, func(page, size int) ([]*model.MonitorConfig, int64, error) {
		return r.history.GetGeneratedConfigHistoryCandidates(ctx, configdao.GeneratedConfigPoolQuery{PoolID: poolID, ConfigType: int8(r.scope), Page: page, Size: size})
	})
	if err != nil {
		return nil, nil, nil, err
	}
	candidates := []configdao.GeneratedConfigOwner{}
	for _, row := range rows {
		if row.PoolID != poolID || row.ConfigType != int8(r.scope) {
			return nil, nil, nil, errors.New("历史候选超出查询范围")
		}
		if row.DeletedAt > 0 && ownedScrapeRuleArtifact(row, r.scope, poolID) {
			candidates = append(candidates, configdao.GeneratedConfigOwnerOf(row))
		}
	}
	completed := map[int]bool{}
	for _, owner := range candidates {
		if superseded[owner.ID] == owner {
			completed[owner.ID] = true
		}
	}
	for start := 0; start < len(candidates); start += configdao.GeneratedConfigQueryLimit {
		end := start + configdao.GeneratedConfigQueryLimit
		if end > len(candidates) {
			end = len(candidates)
		}
		proof, err := r.history.resolved(ctx, poolID, append([]configdao.GeneratedConfigOwner(nil), candidates[start:end]...))
		if err != nil {
			return nil, nil, nil, err
		}
		expected := map[int]configdao.GeneratedConfigOwner{}
		for _, owner := range candidates[start:end] {
			expected[owner.ID] = owner
		}
		for _, owner := range proof {
			if expected[owner.ID] != owner {
				return nil, nil, nil, errors.New("退休完成证明超出查询范围")
			}
			completed[owner.ID] = true
		}
	}
	legacyProofs := []configdao.ScrapeRuleSupersededArtifact{}
	resolvedKey := map[string]bool{}
	for _, owner := range candidates {
		if completed[owner.ID] {
			resolvedKey[owner.InstanceIP] = true
		}
	}
	pendingLegacy := []configdao.GeneratedConfigOwner{}
	for _, owner := range candidates {
		if _, wanted := desired[owner.InstanceIP]; !wanted && !completed[owner.ID] {
			pendingLegacy = append(pendingLegacy, owner)
		}
	}
	for start := 0; start < len(pendingLegacy); start += configdao.GeneratedConfigQueryLimit {
		end := start + configdao.GeneratedConfigQueryLimit
		if end > len(pendingLegacy) {
			end = len(pendingLegacy)
		}
		proofs, err := r.history.legacy(ctx, poolID, append([]configdao.GeneratedConfigOwner(nil), pendingLegacy[start:end]...))
		if err != nil {
			return nil, nil, nil, err
		}
		expected := map[int]configdao.GeneratedConfigOwner{}
		for _, owner := range pendingLegacy[start:end] {
			expected[owner.ID] = owner
		}
		for _, proof := range proofs {
			owner := proof.RetiredOwner
			if proof.Origin != configdao.RecordSupersededLegacyOwner || expected[owner.ID] != owner {
				return nil, nil, nil, errors.New("旧版本替代证明超出归属")
			}
			if resolvedKey[owner.InstanceIP] {
				continue
			}
			legacyProofs = append(legacyProofs, proof)
			resolvedKey[owner.InstanceIP] = true
		}
	}
	resolved := []string{}
	for _, owner := range candidates {
		if resolvedKey[owner.InstanceIP] {
			resolved = append(resolved, owner.InstanceIP)
		}
	}
	keys := map[string]bool{}
	for _, row := range rows {
		if row.PoolID != poolID || row.ConfigType != int8(r.scope) {
			return nil, nil, nil, errors.New("历史候选超出查询范围")
		}
		if row.DeletedAt == 0 || !ownedScrapeRuleArtifact(row, r.scope, poolID) || resolvedKey[row.InstanceIP] {
			continue
		}
		if _, wanted := desired[row.InstanceIP]; wanted {
			continue
		}
		keys[row.InstanceIP] = true
	}
	ordered := make([]string, 0, len(keys))
	for key := range keys {
		ordered = append(ordered, key)
	}
	sort.Strings(ordered)
	discovered := []string{}
	for _, key := range ordered {
		// A fully retired historical row does not perpetually reclaim a vacant raw
		// slot. Presence only locates work; full cross-pool history grants permission.
		_, err := r.redis.Get(ctx, r.scope.rawKey(key)).Result()
		if errors.Is(err, redis.Nil) {
			continue
		}
		if err != nil {
			return nil, nil, nil, err
		}
		discovered = append(discovered, key)
	}
	return discovered, resolved, legacyProofs, nil
}

func (r *scrapeRuleProjection) captureHistoryGuards(ctx context.Context, poolID int, plan scrapeRulePublicationPlan) ([]configdao.ScrapeRuleHistoryGuard, error) {
	guards := make([]configdao.ScrapeRuleHistoryGuard, 0, len(plan.stale))
	seen := map[int]bool{}
	for start := 0; start < len(plan.stale); start += configdao.GeneratedConfigQueryLimit {
		end := start + configdao.GeneratedConfigQueryLimit
		if end > len(plan.stale) {
			end = len(plan.stale)
		}
		batch := append([]string(nil), plan.stale[start:end]...)
		wanted := map[string]bool{}
		for _, ip := range batch {
			wanted[ip] = true
		}
		rows, err := readConfigMetadataPages(ctx, func(page, size int) ([]*model.MonitorConfig, int64, error) {
			return r.history.GetGeneratedConfigHistoryPage(ctx, configdao.GeneratedConfigKeysQuery{ConfigType: int8(r.scope), InstanceKeys: batch, Page: page, Size: size})
		})
		if err != nil {
			return nil, err
		}
		byKey := map[string][]*model.MonitorConfig{}
		for _, row := range rows {
			if !wanted[row.InstanceIP] || row.ConfigType != int8(r.scope) || seen[row.ID] {
				return nil, errors.New("历史退休前置归属不完整")
			}
			seen[row.ID] = true
			byKey[row.InstanceIP] = append(byKey[row.InstanceIP], row)
		}
		for _, ip := range batch {
			found := false
			for _, row := range byKey[ip] {
				if configdao.GeneratedConfigOwnerOf(row) == plan.retired[ip] {
					found = true
				}
			}
			if !found {
				return nil, errors.New("退休配置历史归属已变化")
			}
			guard, err := configdao.ScrapeRuleHistoryGuardOf(configdao.ScrapeRuleScope(r.scope), ip, byKey[ip])
			if err != nil {
				return nil, err
			}
			if plan.retired[ip].PoolID != poolID {
				return nil, errors.New("退休配置归属不一致")
			}
			if err := configdao.ValidateScrapeRuleRetirementHistory(configdao.ScrapeRuleScope(r.scope), plan.retired[ip], guard); err != nil {
				return nil, err
			}
			guards = append(guards, guard)
		}
	}
	return guards, nil
}

func (r *scrapeRuleProjection) sourceVersions(ctx context.Context) (map[int]uint64, error) {
	if err := r.ready(); err != nil {
		return nil, err
	}
	snapshot, err := r.journal.sourceVersions(ctx)
	if err != nil {
		return nil, err
	}
	versions := map[int]uint64{}
	for _, version := range snapshot {
		if version.PoolID <= 0 {
			return nil, errors.New("规则源版本池标识无效")
		}
		if _, exists := versions[version.PoolID]; exists {
			return nil, errors.New("规则源版本池标识重复")
		}
		versions[version.PoolID] = version.Counter
	}
	return versions, nil
}

func scrapeRuleReceiptCoversPlan(receipt configdao.ProjectionReceipt, desired map[string]string, plan scrapeRulePublicationPlan) bool {
	issued := map[string]configdao.ArtifactReceipt{}
	for _, artifact := range receipt.Artifacts {
		if artifact.ConfigType != receipt.Reservation.Family {
			return false
		}
		if _, duplicate := issued[artifact.InstanceKey]; duplicate {
			return false
		}
		issued[artifact.InstanceKey] = artifact
	}
	for ip := range desired {
		artifact, ok := issued[ip]
		owner := plan.owners[ip]
		if !ok || artifact.Action != model.ConfigProjectionRender || artifact.ConfigRowID == nil || owner == nil || owner.ID != *artifact.ConfigRowID {
			return false
		}
	}
	for _, ip := range plan.stale {
		artifact, ok := issued[ip]
		if !ok || artifact.Action != model.ConfigProjectionRetire || artifact.ConfigRowID == nil {
			return false
		}
		if owner, known := plan.retired[ip]; known && owner.ID != *artifact.ConfigRowID {
			return false
		}
	}
	return true
}
