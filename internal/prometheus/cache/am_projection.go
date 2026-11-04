package cache

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"sort"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/redis/go-redis/v9"

	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
)

type AMProjectionRunner interface {
	Project(context.Context, configdao.ProjectionLease) error
}
type amProjectionRunner struct{ store *ProjectionStore }

func NewAMProjectionRunner(store *ProjectionStore) AMProjectionRunner {
	return &amProjectionRunner{store: store}
}
func (r *amProjectionRunner) Project(ctx context.Context, lease configdao.ProjectionLease) error {
	if r == nil || r.store == nil || lease == nil {
		return errors.New("AM持久投影依赖不可用")
	}
	snapshot, err := lease.Snapshot()
	if err != nil {
		return err
	}
	if snapshot.Pool == nil || snapshot.Pool.AMControls == nil || (snapshot.Pool.Reservation.Family != 2 && snapshot.Pool.Reservation.Family != 5) {
		return errors.New("AM投影缺少双控制证明")
	}
	prepared, err := r.store.Prepare(ctx, lease)
	if err != nil {
		return err
	}
	for _, control := range snapshot.Pool.Controls {
		if control.Intents != nil {
			if err := prepared.AddIntents(ctx, control.ControlID); err != nil {
				return err
			}
		}
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
	for _, control := range snapshot.Pool.Controls {
		if control.Intents != nil {
			if err := prepared.RemoveIntents(ctx, control.ControlID); err != nil {
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
		return errors.New("AM投影回执已失效")
	}
	return nil
}

type amHistoryFacade struct {
	configdao.AMProjectionHistoryReader
}

func (h amHistoryFacade) superseded(ctx context.Context, receipt configdao.ProjectionReceipt) ([]configdao.RecordSupersededArtifact, error) {
	return h.GetSupersededAMArtifacts(ctx, receipt)
}
func (h amHistoryFacade) legacy(ctx context.Context, pool int, owners []configdao.GeneratedConfigOwner) ([]configdao.RecordSupersededArtifact, error) {
	return h.GetLegacySupersededAMArtifacts(ctx, pool, owners)
}
func (h amHistoryFacade) resolved(ctx context.Context, pool int, owners []configdao.GeneratedConfigOwner) ([]configdao.GeneratedConfigOwner, error) {
	return h.GetResolvedAMOwners(ctx, pool, owners)
}
func (a *alertManagerConfigCache) amJournal() (configdao.AMProjectionJournalDAO, error) {
	journal, ok := a.journal.(configdao.AMProjectionJournalDAO)
	if !ok || journal == nil || a.history == nil || a.projection == nil || a.alertConfigDAO() == nil || a.redis == nil {
		return nil, errors.New("AM持久发布依赖不可用")
	}
	return journal, nil
}
func (a *alertManagerConfigCache) amSourceVersions(ctx context.Context) (map[int]configdao.AMSourceVersion, error) {
	journal, err := a.amJournal()
	if err != nil {
		return nil, err
	}
	rows, err := journal.ReadAMSourceVersions(ctx)
	if err != nil {
		return nil, err
	}
	result := map[int]configdao.AMSourceVersion{}
	for _, row := range rows {
		if row.PoolID <= 0 {
			return nil, errors.New("AM源版本标识无效")
		}
		if _, duplicate := result[row.PoolID]; duplicate {
			return nil, errors.New("AM源版本标识重复")
		}
		result[row.PoolID] = row
	}
	return result, nil
}
func amSourceVersionFor(versions map[int]configdao.AMSourceVersion, poolID int) configdao.AMSourceVersion {
	version := versions[poolID]
	version.PoolID = poolID
	return version
}

type amNamespacePlan struct {
	kind      int8
	desired   map[string]ConfigData
	plan      scrapePublicationPlan
	namespace *scrapeProjection
}

func (a *alertManagerConfigCache) amLoadReceipts(ctx context.Context) (configdao.AMProjectionJournalDAO, error) {
	return a.amJournal()
}
func (a *alertManagerConfigCache) amProject(ctx context.Context, family int8, poolID int, desired map[int8]map[string]ConfigData, hash [32]byte, version configdao.AMSourceVersion) error {
	journal, err := a.amJournal()
	if err != nil {
		return err
	}
	receipts := []configdao.ProjectionReceipt{}
	for _, domain := range []int8{2, 5} {
		receipt, err := journal.LoadLastCommittedProjection(ctx, domain, poolID)
		if err != nil {
			return err
		}
		if receipt != nil {
			receipts = append(receipts, *receipt)
		}
	}
	var own *configdao.ProjectionReceipt
	for i := range receipts {
		if receipts[i].Reservation.Family == family {
			own = &receipts[i]
		}
	}
	issued := []configdao.ArtifactReceipt{}
	for _, receipt := range receipts {
		issued = append(issued, receipt.Artifacts...)
	}
	// Prefer the newest issued locator at a namespace; the exact immutable row
	// remains checked by the common namespace resolver and transaction guards.
	sort.Slice(issued, func(i, j int) bool { return issued[i].EventID < issued[j].EventID })
	namespaces := []amNamespacePlan{}
	superseded := []configdao.RecordSupersededArtifact{}
	kinds := []int8{5}
	if family == 2 {
		kinds = []int8{2, 5}
	}
	for _, kind := range kinds {
		rows := desired[kind]
		if rows == nil {
			rows = map[string]ConfigData{}
		}
		namespace := &scrapeProjection{scope: scrapePublicationScope(kind), redis: a.redis, configDAO: a.alertConfigDAO(), history: amHistoryFacade{a.history}}
		oldKeys := []string{}
		proofOwners := map[int]configdao.GeneratedConfigOwner{}
		if family == 2 {
			if kind == 2 {
				oldKeys, err = a.redis.SMembers(ctx, buildRedisSetKeyAlertManagerMainPoolIPs(poolID)).Result()
				if err != nil {
					return err
				}
			}
			for _, receipt := range receipts {
				proofs, err := a.history.GetSupersededAMArtifacts(ctx, receipt)
				if err != nil {
					return err
				}
				for _, proof := range proofs {
					if proof.RetiredOwner.ConfigType != kind {
						continue
					}
					if _, wanted := rows[proof.RetiredOwner.InstanceIP]; wanted {
						continue
					}
					superseded = append(superseded, proof)
					proofOwners[proof.RetiredOwner.ID] = proof.RetiredOwner
				}
				for _, artifact := range receipt.Artifacts {
					if artifact.ConfigType == kind && (artifact.Action == model.ConfigProjectionRender || receipt.Phase != model.ConfigPublicationProjected) {
						oldKeys = append(oldKeys, artifact.InstanceKey)
					}
				}
			}
			stringsDesired := map[string]string{}
			for key, row := range rows {
				stringsDesired[key] = row.Content
			}
			discovered, resolved, legacy, err := namespace.historicalLocators(ctx, poolID, stringsDesired, proofOwners)
			if err != nil {
				return err
			}
			superseded = append(superseded, legacy...)
			ignored := map[string]bool{}
			for _, key := range resolved {
				ignored[key] = true
			}
			for _, owner := range proofOwners {
				ignored[owner.InstanceIP] = true
			}
			filtered := []string{}
			for _, key := range oldKeys {
				if !ignored[key] {
					filtered = append(filtered, key)
				}
			}
			oldKeys = append(filtered, discovered...)
		}
		plan, err := planNamespaceProjection(ctx, kind, a.alertConfigDAO(), poolID, rows, oldKeys, family == 2)
		if err != nil {
			return err
		}
		namespaces = append(namespaces, amNamespacePlan{kind: kind, desired: rows, plan: plan, namespace: namespace})
	}
	if family == 2 {
		// A replay is still a writer. Validate the entire external retry index
		// before entering either the no-op check or a previously issued lease.
		for _, ns := range namespaces {
			if ns.kind != 5 {
				continue
			}
			retired := []retiredAlertWebhook{}
			for _, owner := range ns.plan.retired {
				row, err := a.alertConfigDAO().GetGeneratedMonitorConfigByID(ctx, owner.ID)
				if err != nil || row == nil || configdao.GeneratedConfigOwnerOf(row) != owner {
					return errors.New("AM退休webhook归属无效")
				}
				item, ok := ownedAlertWebhook(row, poolID)
				if !ok {
					return errors.New("AM退休webhook归属无效")
				}
				retired = append(retired, item)
			}
			if _, err := a.readAlertRetirementIntents(ctx, poolID, ns.desired, retired); err != nil {
				return err
			}
		}
	}
	// A successful projected cohort can cover the same output across full/subset
	// calls without manufacturing alternating generations or touching full state.
	if a.amPlansUnchanged(namespaces) {
		covered, err := a.amCovered(ctx, journal, family, poolID, hash, namespaces, receipts)
		if err == nil && covered {
			return nil
		}
		if err != nil && !errors.Is(err, configdao.ErrJournalConflict) {
			return err
		}
	}
	if own != nil && own.SourceHash == hash && own.AMControls != nil && a.amReceiptCovers(*own, namespaces) {
		err := journal.WithPoolProjectionLease(ctx, *own, func(lease configdao.ProjectionLease) error { return a.projection.Project(ctx, lease) })
		if err == nil {
			return nil
		}
		if !errors.Is(err, configdao.ErrJournalConflict) {
			return err
		}
	}
	guards := []configdao.RecordHistoryGuard{}
	for i := range namespaces {
		ns := &namespaces[i]
		if err := ns.namespace.resolveIssuedRetirements(ctx, poolID, &ns.plan, issued); err != nil {
			return err
		}
		history, err := ns.namespace.captureHistoryGuards(ctx, poolID, ns.plan)
		if err != nil {
			return err
		}
		guards = append(guards, history...)
	}
	// Decode and verify every legacy locator before reserving a durable attempt.
	// The real reservation is installed only after this read-only plan succeeds.
	request, err := a.amGeneratedPlan(ctx, configdao.PoolReservation{Family: family, PoolID: poolID}, hash, version, namespaces, guards, superseded)
	if err != nil {
		return err
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return errors.New("AM发布标识生成失败")
	}
	reservationRequest := configdao.PoolOperationRequest{Family: family, PoolID: poolID, Nonce: nonce}
	if own != nil {
		id := own.Reservation.OperationID
		reservationRequest.RecoveryOfOperationID = &id
	}
	ref, err := journal.ReserveAMPoolOperation(ctx, reservationRequest, version)
	if err != nil {
		return err
	}
	request.Reservation = ref
	receipt, err := journal.CommitAMProjectionPlan(ctx, request)
	if err != nil {
		if !errors.Is(err, configdao.ErrJournalUncertain) {
			return err
		}
		facts, lookupErr := journal.LoadProjectionByNonce(ctx, nonce)
		if lookupErr != nil || facts.Projection == nil || facts.Projection.Reservation != ref || facts.Projection.SourceHash != hash || facts.Projection.AMControls == nil {
			return errors.New("AM提交结果待核对")
		}
		receipt = *facts.Projection
	}
	return journal.WithPoolProjectionLease(ctx, receipt, func(lease configdao.ProjectionLease) error { return a.projection.Project(ctx, lease) })
}
func (a *alertManagerConfigCache) amPlansUnchanged(plans []amNamespacePlan) bool {
	for _, plan := range plans {
		if len(plan.plan.create) != 0 || len(plan.plan.update) != 0 || len(plan.plan.stale) != 0 {
			return false
		}
	}
	return true
}
func (a *alertManagerConfigCache) amReceiptCovers(receipt configdao.ProjectionReceipt, plans []amNamespacePlan) bool {
	for _, ns := range plans {
		desired := map[string]string{}
		for key, row := range ns.desired {
			desired[key] = row.Content
		}
		subset := receipt
		subset.Artifacts = []configdao.ArtifactReceipt{}
		for _, artifact := range receipt.Artifacts {
			if artifact.ConfigType == ns.kind {
				subset.Artifacts = append(subset.Artifacts, artifact)
			}
		}
		subset.Reservation.Family = ns.kind
		if len(ns.plan.create) > 0 || len(ns.plan.update) > 0 || !scrapeReceiptCoversPlan(subset, desired, ns.plan) {
			return false
		}
	}
	return true
}

func (a *alertManagerConfigCache) amCovered(ctx context.Context, journal configdao.AMProjectionJournalDAO, family int8, poolID int, hash [32]byte, plans []amNamespacePlan, receipts []configdao.ProjectionReceipt) (bool, error) {
	selected := map[string]configdao.ArtifactReceipt{}
	used := map[uint64]configdao.ProjectionReceipt{}
	var full *configdao.ProjectionReceipt
	for i := range receipts {
		receipt := &receipts[i]
		if receipt.Reservation.Family == 2 {
			full = receipt
		}
		if receipt.Phase != model.ConfigPublicationProjected || receipt.AMControls == nil {
			continue
		}
		for _, artifact := range receipt.Artifacts {
			if artifact.Action != model.ConfigProjectionRender {
				continue
			}
			key := fmt.Sprintf("%d/%s", artifact.ConfigType, artifact.InstanceKey)
			if selected[key].EventID < artifact.EventID {
				selected[key] = artifact
				used[receipt.Reservation.OperationID] = *receipt
			}
		}
	}
	if family == 2 && (full == nil || full.Phase != model.ConfigPublicationProjected || full.SourceHash != hash) {
		return false, nil
	}
	request := configdao.AMCoverageRequest{PoolID: poolID, Receipts: []configdao.ProjectionReceipt{}, Artifacts: []configdao.ArtifactReceipt{}}
	needed := map[uint64]bool{}
	for _, ns := range plans {
		for key := range ns.desired {
			artifact, ok := selected[fmt.Sprintf("%d/%s", ns.kind, key)]
			if !ok {
				return false, nil
			}
			request.Artifacts = append(request.Artifacts, artifact)
			needed[*artifact.OperationID] = true
		}
	}
	if family == 2 {
		needed[full.Reservation.OperationID] = true
	}
	if len(needed) == 0 {
		return false, nil
	}
	for _, receipt := range receipts {
		if needed[receipt.Reservation.OperationID] {
			request.Receipts = append(request.Receipts, receipt)
		}
	}
	_ = used
	healthy := false
	matched, err := journal.WithAMCoverage(ctx, request, func(snapshot configdao.AMCoverageSnapshot) error {
		for _, ns := range plans {
			for key, row := range ns.desired {
				rawKey, keyErr := amRawKey(ns.kind, key)
				if keyErr != nil {
					return keyErr
				}
				body, err := a.redis.Get(ctx, rawKey).Result()
				if err != nil {
					if errors.Is(err, redis.Nil) {
						return nil
					}
					return err
				}
				if body != row.Content {
					return nil
				}
			}
		}
		if family == 2 {
			var ips []string
			for _, ns := range plans {
				if ns.kind == 2 {
					for key := range ns.desired {
						ips = append(ips, key)
					}
				}
			}
			members, err := a.redis.SMembers(ctx, buildRedisSetKeyAlertManagerMainPoolIPs(poolID)).Result()
			if err != nil {
				return err
			}
			if len(members) != len(ips) {
				return nil
			}
			want := map[string]bool{}
			for _, ip := range ips {
				want[ip] = true
			}
			for _, ip := range members {
				if !want[ip] {
					return nil
				}
			}
			remaining, err := a.redis.SMembers(ctx, alertRetirementIntentKey(poolID)).Result()
			if err != nil {
				return err
			}
			if len(remaining) != 0 {
				return nil
			}
			sha, err := a.redis.Get(ctx, alertPublicationStateKey(poolID)).Result()
			if err != nil {
				if errors.Is(err, redis.Nil) {
					return nil
				}
				return err
			}
			if sha != hex.EncodeToString(hash[:]) {
				return nil
			}
		}
		// Redis's signed ACK receipt must match each current projected SQL cohort.
		for _, receipt := range snapshot.Receipts {
			raw, err := a.redis.Get(ctx, projectionPoolKey(receipt.Reservation.Family, poolID, "ack")).Result()
			if err != nil {
				if errors.Is(err, redis.Nil) {
					return nil
				}
				return err
			}
			expected := configdao.ProjectionAckProof{Reservation: receipt.Reservation, SourceHash: receipt.SourceHash, PlanDigest: receipt.PlanDigest}
			for _, control := range receipt.Controls {
				if control.Ack != nil {
					expected.ArtifactVector = control.Ack.ArtifactVector
					expected.RequiredControls = control.Ack.RequiredControls
				}
			}
			encoded, err := json.Marshal(expected)
			if err != nil {
				return err
			}
			if raw != string(encoded) {
				return nil
			}
		}
		healthy = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return matched && healthy, nil
}
func amRawKey(kind int8, key string) (string, error) {
	binding, err := configdao.ProjectionBindingForNamespace(kind, key)
	if err != nil {
		return "", err
	}
	return projectionRawKey(binding)
}

func (a *alertManagerConfigCache) amGeneratedPlan(ctx context.Context, ref configdao.PoolReservation, hash [32]byte, version configdao.AMSourceVersion, plans []amNamespacePlan, guards []configdao.RecordHistoryGuard, superseded []configdao.RecordSupersededArtifact) (configdao.AMProjectionPlanRequest, error) {
	items := []configdao.GeneratedPlanItem{}
	sources := []uint64{}
	mainIPs := []string{}
	byWebhook := map[string]uint64{}
	adds := []configdao.IntentOwnerChange{}
	removes := []configdao.IntentOwnerChange{}
	retired := []retiredAlertWebhook{}
	desiredHooks := map[string]ConfigData{}
	for _, ns := range plans {
		keys := []string{}
		for key := range ns.desired {
			keys = append(keys, key)
		}
		keys = append(keys, ns.plan.stale...)
		sort.Strings(keys)
		for _, key := range keys {
			binding, err := configdao.ProjectionBindingForNamespace(ns.kind, key)
			if err != nil {
				return configdao.AMProjectionPlanRequest{}, err
			}
			sourceID := uint64(len(items) + 1)
			item := configdao.GeneratedPlanItem{SourceID: sourceID, Binding: &binding}
			if desired, wanted := ns.desired[key]; wanted {
				item.Action = model.ConfigProjectionRender
				row := &model.MonitorConfig{PoolID: ref.PoolID, ConfigType: ns.kind, InstanceIP: key, Name: desired.Name, ConfigContent: desired.Content, ConfigHash: calculateConfigHash(desired.Content), Status: model.ConfigStatusActive, LastGeneratedTime: time.Now().Unix()}
				if current := ns.plan.owners[key]; current != nil {
					owner := configdao.GeneratedConfigOwnerOf(current)
					item.Expected = &owner
					row.Model = current.Model
					if current.ConfigContent == row.ConfigContent && current.ConfigHash == row.ConfigHash && current.Status == row.Status && current.Name == row.Name {
						row.LastGeneratedTime = current.LastGeneratedTime
					}
				}
				item.Desired = row
				if ns.kind == 2 {
					mainIPs = append(mainIPs, key)
				} else {
					desiredHooks[key] = desired
				}
			} else {
				item.Action = model.ConfigProjectionRetire
				owner := ns.plan.retired[key]
				item.Expected = &owner
				if ns.kind == 5 {
					row, err := a.alertConfigDAO().GetGeneratedMonitorConfigByID(ctx, owner.ID)
					if err != nil {
						return configdao.AMProjectionPlanRequest{}, err
					}
					parsed, ok := ownedAlertWebhook(row, ref.PoolID)
					if !ok {
						return configdao.AMProjectionPlanRequest{}, errors.New("AM退休webhook归属无效")
					}
					retired = append(retired, parsed)
				}
			}
			if ns.kind == 5 {
				byWebhook[key] = sourceID
			}
			items = append(items, item)
			sources = append(sources, sourceID)
		}
	}
	if ref.Family == 2 {
		intents, err := a.readAlertRetirementIntents(ctx, ref.PoolID, desiredHooks, retired)
		if err != nil {
			return configdao.AMProjectionPlanRequest{}, err
		}
		previous, err := a.journal.LoadLastCommittedProjection(ctx, 2, ref.PoolID)
		if err != nil {
			return configdao.AMProjectionPlanRequest{}, err
		}
		for _, intent := range intents {
			sourceID := byWebhook[intent.owner.InstanceIP]
			var attested *configdao.GeneratedConfigOwner
			if sourceID != 0 {
				item := items[sourceID-1]
				attested = item.Expected
				if item.Desired != nil && item.Desired.ID != 0 {
					owner := configdao.GeneratedConfigOwnerOf(item.Desired)
					attested = &owner
				}
			}
			change := configdao.IntentOwnerChange{Owner: intent.owner, Proof: configdao.ProjectionProofRef{SourceID: sourceID}}
			if attested == nil || attested.ID != intent.owner.ID {
				// A replacement row cannot attest an older row's retry locator.
				// Reuse only an actually issued proof for that exact historical owner.
				change.Proof = configdao.ProjectionProofRef{}
				if previous != nil {
					for _, control := range previous.Controls {
						if control.Intents == nil {
							continue
						}
						for _, issued := range control.Intents.Remove {
							if issued.Owner == intent.owner {
								change.Proof.EventID = issued.ProofEventID
								change.AliasCurrentOwner = issued.AliasCurrentOwner
							}
						}
					}
				}
				if change.Proof.EventID == 0 {
					return configdao.AMProjectionPlanRequest{}, errors.New("AM旧退休索引缺少已签归属证明")
				}
			} else if *attested != intent.owner {
				owner := *attested
				change.AliasCurrentOwner = &owner
			}
			if _, wanted := desiredHooks[intent.owner.InstanceIP]; !wanted && change.AliasCurrentOwner == nil && change.Proof.SourceID != 0 {
				adds = append(adds, change)
			}
			removes = append(removes, change)
		}
	}
	controls := []configdao.ProjectionControl{}
	if ref.Family == 2 {
		controls = append(controls,
			configdao.ProjectionControl{
				ControlID: 1, Kind: configdao.ProjectionMembersFull,
				Members: &configdao.FullMembersControl{DesiredIPs: mainIPs},
			},
			configdao.ProjectionControl{
				ControlID: 2, Kind: configdao.ProjectionWebhookIntents,
				Intents: &configdao.WebhookIntentsControl{Add: adds, Remove: removes, RequireEmpty: true},
			},
			configdao.ProjectionControl{
				ControlID: 3, Kind: configdao.ProjectionACKFull,
				Ack: &configdao.ProjectionAckControl{
					LegacySHA: &hash, ArtifactSourceIDs: sources, RequiredControlIDs: []uint64{1, 2},
				},
			},
		)
	} else {
		controls = append(controls, configdao.ProjectionControl{
			ControlID: 3, Kind: configdao.ProjectionACKWebhookSubset,
			Ack: &configdao.ProjectionAckControl{ArtifactSourceIDs: sources, RequiredControlIDs: []uint64{}},
		})
	}
	return configdao.AMProjectionPlanRequest{
		GeneratedProjectionPlanRequest: configdao.GeneratedProjectionPlanRequest{
			GeneratedPlanRequest: configdao.GeneratedPlanRequest{Reservation: ref, SourceHash: hash, Items: items},
			Controls:             controls,
		},
		SourceVersion: version, History: guards, Superseded: superseded,
	}, nil
}
