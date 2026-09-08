package store

import (
	"context"
	"database/sql"
	"errors"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/objectstorage"
)

const AssetRetentionV2 = 30 * 24 * time.Hour

// This is deliberately separate from business transactions. The deployed backend
// holds one database runtime lease; the local mutex also serializes on-demand restores.
type AssetObjectStorageV2 interface {
	CopyVerified(context.Context, string, string, string) (time.Time, error)
	Delete(context.Context, string) error
	Exists(context.Context, string) (bool, error)
}

type AssetLifecycleCandidateV2 struct {
	AssetID   string `json:"assetId"`
	Bytes     int64  `json:"sizeBytes"`
	Eligible  bool   `json:"eligible"`
	Reason    string `json:"reason"`
	Lifecycle string `json:"lifecycle"`
}

type AssetLifecycleResultV2 struct {
	Checked  int `json:"checked"`
	Retired  int `json:"retired"`
	Purged   int `json:"purged"`
	Deferred int `json:"deferred"`
}

func (s *Store) assetCurrentUseTxV2(ctx context.Context, tx *sql.Tx, kind, ref string) (bool, error) {
	switch kind {
	case "MARKET_LISTING", "PRODUCT_DRAFT", "PRODUCT_PUBLISHED":
		var state string
		err := tx.QueryRowContext(ctx, "SELECT state FROM catalog_records_v2 WHERE resource_id=?", ref).Scan(&state)
		if errors.Is(err, sql.ErrNoRows) {
			return true, nil
		} // Unknown references are not a deletion authorization.
		if err != nil {
			return true, err
		}
		return state != "UNLISTED" && state != "ARCHIVED" && state != "INACTIVE", nil
	case "COMMISSION", "ORDER_COMPLETION", "COMMISSION_COMPLETION":
		return assetCommerceCurrentTxV2(ctx, tx, ref)
	case "ORDER_SNAPSHOT", "COMMISSION_SNAPSHOT":
		var resource string
		err := tx.QueryRowContext(ctx, "SELECT resource_id FROM commerce_snapshots_v2 WHERE snapshot_id=?", ref).Scan(&resource)
		if errors.Is(err, sql.ErrNoRows) {
			return true, nil
		}
		if err != nil {
			return true, err
		}
		return assetCommerceCurrentTxV2(ctx, tx, resource)
	case "REFUND":
		var state, resource string
		err := tx.QueryRowContext(ctx, "SELECT state,resource_id FROM commerce_refunds_v2 WHERE refund_id=?", ref).Scan(&state, &resource)
		if errors.Is(err, sql.ErrNoRows) {
			return true, nil
		}
		if err != nil {
			return true, err
		}
		if state == "REQUESTED" || state == "PROCESSING" {
			return true, nil
		}
		return assetCommerceCurrentTxV2(ctx, tx, resource)
	case "INTERVENTION_EVIDENCE":
		var state, resource string
		err := tx.QueryRowContext(ctx, "SELECT state,resource_id FROM commerce_interventions_v2 WHERE case_id=?", ref).Scan(&state, &resource)
		if errors.Is(err, sql.ErrNoRows) {
			return true, nil
		}
		if err != nil {
			return true, err
		}
		if state != "RESOLVED" && state != "WITHDRAWN" {
			return true, nil
		}
		return assetCommerceCurrentTxV2(ctx, tx, resource)
	default:
		// Current profile, shop identity, announcement and unrecognized bindings
		// must be explicitly detached by their business before they may expire.
		return true, nil
	}
}

func assetCommerceCurrentTxV2(ctx context.Context, tx *sql.Tx, id string) (bool, error) {
	var state, funds string
	var pending sql.NullString
	err := tx.QueryRowContext(ctx, "SELECT state,funds_state,pending_operation_id FROM commerce_resources_v2 WHERE resource_id=?", id).Scan(&state, &funds, &pending)
	if errors.Is(err, sql.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return true, err
	}
	if pending.Valid && pending.String != "" {
		return true, nil
	}
	terminal := state == "CONFIRMED" || state == "CLAIMED" || state == "REFUNDED" || state == "CANCELLED"
	settled := funds == "SETTLED" || funds == "REFUNDED" || (state == "CANCELLED" && funds == "UNPAID")
	if !terminal || !settled {
		return true, nil
	}
	var open int
	err = tx.QueryRowContext(ctx, `SELECT (SELECT COUNT(*) FROM commerce_interventions_v2 WHERE resource_id=? AND state NOT IN ('RESOLVED','WITHDRAWN')) + (SELECT COUNT(*) FROM commerce_refunds_v2 WHERE resource_id=? AND state IN ('REQUESTED','PROCESSING'))`, id, id).Scan(&open)
	return open > 0, err
}

func (s *Store) assetRetirementDecisionV2(ctx context.Context, tx *sql.Tx, u AssetUploadV2, now time.Time) (bool, string, error) {
	if u.Lifecycle != "ACTIVE" {
		return false, "ALREADY_MANAGED", nil
	}
	if u.KeepUntil.Valid && u.KeepUntil.Time.After(now) {
		return false, "RESTORE_GRACE", nil
	}
	if u.ExpiresAt.After(now) {
		return false, "UPLOAD_AUTHORIZATION_ACTIVE", nil
	}
	if u.Status == "VERIFYING" && u.VerificationStarted.Valid && u.VerificationStarted.Time.After(now.Add(-2*time.Minute)) {
		return false, "VERIFYING", nil
	}
	rows, err := tx.QueryContext(ctx, "SELECT business_type,business_ref FROM asset_bindings_v2 WHERE asset_id=? ORDER BY business_type,business_ref", u.AssetID)
	if err != nil {
		return false, "", err
	}
	type binding struct{ kind, ref string }
	bindings := []binding{}
	for rows.Next() {
		var b binding
		if err = rows.Scan(&b.kind, &b.ref); err != nil {
			rows.Close()
			return false, "", err
		}
		bindings = append(bindings, b)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return false, "", err
	}
	for _, b := range bindings {
		active, err := s.assetCurrentUseTxV2(ctx, tx, b.kind, b.ref)
		if err != nil {
			return false, "", err
		}
		if active {
			return false, "CURRENT_BUSINESS_REFERENCE", nil
		}
	}
	if len(bindings) == 0 {
		var wasBound bool
		if err = tx.QueryRowContext(ctx, "SELECT was_bound FROM asset_uploads_v2 WHERE asset_id=?", u.AssetID).Scan(&wasBound); err != nil {
			return false, "", err
		}
		if u.Status == "READY" && !wasBound && !u.RemovedAt.Valid && u.CreatedAt.After(now.Add(-72*time.Hour)) {
			return false, "UNSUBMITTED_UPLOAD_GRACE", nil
		}
		return true, "NO_CURRENT_REFERENCE", nil
	}
	return true, "HISTORICAL_REFERENCES_ONLY", nil
}

func (s *Store) PreviewAssetLifecycleV2(ctx context.Context, limit int) ([]AssetLifecycleCandidateV2, error) {
	if limit < 1 || limit > 1000 {
		limit = 100
	}
	rows, err := s.DB.QueryContext(ctx, assetSelectV2+" ORDER BY created_at,asset_id LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	assets := []AssetUploadV2{}
	for rows.Next() {
		u, e := scanAssetV2(rows)
		if e != nil {
			rows.Close()
			return nil, e
		}
		assets = append(assets, u)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	out := []AssetLifecycleCandidateV2{}
	for _, u := range assets {
		eligible, reason, e := s.assetRetirementDecisionV2(ctx, tx, u, time.Now().UTC())
		if e != nil {
			return nil, e
		}
		out = append(out, AssetLifecycleCandidateV2{u.AssetID, u.SizeBytes, eligible, reason, u.Lifecycle})
	}
	return out, nil
}

func validAssetPrefixV2(prefix string) bool {
	return prefix != "" && strings.HasSuffix(prefix, "/") && !strings.HasPrefix(prefix, "/") && !strings.Contains(prefix, "..")
}

func (s *Store) RunAssetLifecycleV2(ctx context.Context, objects AssetObjectStorageV2, prefix string, limit int) (AssetLifecycleResultV2, error) {
	var result AssetLifecycleResultV2
	if !validAssetPrefixV2(prefix) {
		return result, ErrAssetUnavailable
	}
	if !s.assetLifecycle.TryLock() {
		return result, nil
	}
	defer s.assetLifecycle.Unlock()
	if limit < 1 || limit > 64 {
		limit = 32
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT asset_id FROM asset_uploads_v2 WHERE lifecycle_state<>'PURGED' AND (lifecycle_retry_at IS NULL OR lifecycle_retry_at<=UTC_TIMESTAMP(6)) AND (lifecycle_state<>'EXPIRED' OR lifecycle_checked_at<DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 1 DAY)) ORDER BY CASE WHEN lifecycle_state IN ('MOVING','RESTORING') OR source_object_key IS NOT NULL THEN 0 ELSE 1 END,COALESCE(lifecycle_checked_at,created_at),asset_id LIMIT ?`, limit)
	if err != nil {
		return result, err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return result, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	for _, id := range ids {
		if ctx.Err() != nil {
			return result, ctx.Err()
		}
		workCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
		phase, e := s.advanceAssetLifecycleV2(workCtx, objects, prefix, id)
		cancel()
		result.Checked++
		if e != nil {
			result.Deferred++
			_, _ = s.DB.ExecContext(ctx, "UPDATE asset_uploads_v2 SET lifecycle_retry_at=DATE_ADD(UTC_TIMESTAMP(6),INTERVAL 1 MINUTE),lifecycle_error='STORAGE_RETRY',lifecycle_checked_at=UTC_TIMESTAMP(6) WHERE asset_id=?", id)
			continue
		}
		if phase == "RETIRED" {
			result.Retired++
		}
		if phase == "PURGED" {
			result.Purged++
		}
	}
	return result, nil
}

func (s *Store) advanceAssetLifecycleV2(ctx context.Context, objects AssetObjectStorageV2, prefix, id string) (string, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	u, err := scanAssetV2(tx.QueryRowContext(ctx, assetSelectV2+" WHERE asset_id=? FOR UPDATE", id))
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(u.ObjectKey, prefix) {
		return "", ErrAssetUnavailable
	}
	if u.Lifecycle == "ACTIVE" && u.SourceKey == "" {
		eligible, _, e := s.assetRetirementDecisionV2(ctx, tx, u, time.Now().UTC())
		if e != nil {
			return "", e
		}
		if eligible {
			u.SourceKey = u.ObjectKey
			u.PendingKey = prefix + "gc/" + time.Now().UTC().Format("2006-01-02") + "/" + ID("retired_") + path.Ext(u.ObjectKey)
			u.Lifecycle = "MOVING"
			_, err = tx.ExecContext(ctx, "UPDATE asset_uploads_v2 SET lifecycle_state='MOVING',source_object_key=?,pending_object_key=? WHERE asset_id=?", u.SourceKey, u.PendingKey, id)
			if err != nil {
				return "", err
			}
		}
	}
	if _, err = tx.ExecContext(ctx, "UPDATE asset_uploads_v2 SET lifecycle_checked_at=UTC_TIMESTAMP(6),lifecycle_retry_at=NULL,lifecycle_error=NULL WHERE asset_id=?", id); err != nil {
		return "", err
	}
	if err = tx.Commit(); err != nil {
		return "", err
	}
	if u.Lifecycle == "MOVING" || u.Lifecycle == "RESTORING" {
		return s.finishAssetMoveV2(ctx, objects, u)
	}
	if u.SourceKey != "" {
		if err = s.deleteOldAssetObjectV2(ctx, objects, u); err != nil {
			return "", err
		}
	}
	if (u.Lifecycle == "RETIRED" && u.RetainUntil.Valid && !u.RetainUntil.Time.After(time.Now())) || u.Lifecycle == "EXPIRED" {
		exists, err := objects.Exists(ctx, u.ObjectKey)
		if err != nil {
			return "", err
		}
		phase := "EXPIRED"
		if !exists {
			phase = "PURGED"
		}
		_, err = s.DB.ExecContext(ctx, "UPDATE asset_uploads_v2 SET lifecycle_state=? WHERE asset_id=? AND lifecycle_state IN ('RETIRED','EXPIRED')", phase, id)
		return phase, err
	}
	return "", nil
}

func (s *Store) finishAssetMoveV2(ctx context.Context, objects AssetObjectStorageV2, u AssetUploadV2) (string, error) {
	if u.SourceKey == "" || u.PendingKey == "" || u.SourceKey == u.PendingKey {
		return "", ErrAssetUnavailable
	}
	exists, err := objects.Exists(ctx, u.SourceKey)
	if err != nil {
		return "", err
	}
	if !exists {
		targetExists, e := objects.Exists(ctx, u.PendingKey)
		if e != nil {
			return "", e
		}
		if !targetExists {
			_, err = s.DB.ExecContext(ctx, "UPDATE asset_uploads_v2 SET lifecycle_state='PURGED',source_object_key=NULL,pending_object_key=NULL,retain_until=COALESCE(retain_until,UTC_TIMESTAMP(6)) WHERE asset_id=? AND lifecycle_state=? AND pending_object_key=?", u.AssetID, u.Lifecycle, u.PendingKey)
			return "PURGED", err
		}
	}
	created, err := objects.CopyVerified(ctx, u.SourceKey, u.PendingKey, u.SHA256.String)
	if err != nil {
		return "", err
	}
	if created.IsZero() || created.After(time.Now().Add(5*time.Minute)) {
		return "", ErrAssetUnavailable
	}
	phase := "RETIRED"
	var retired, until, keep any = created, created.Add(AssetRetentionV2), nil
	if u.Lifecycle == "RESTORING" {
		phase = "ACTIVE"
		retired = nil
		until = nil
		keep = time.Now().UTC().Add(10 * time.Minute)
	}
	result, err := s.DB.ExecContext(ctx, `UPDATE asset_uploads_v2 SET object_key=pending_object_key,pending_object_key=NULL,lifecycle_state=?,retired_at=?,retain_until=?,lifecycle_keep_until=?,lifecycle_retry_at=NULL,lifecycle_error=NULL WHERE asset_id=? AND lifecycle_state=? AND pending_object_key=?`, phase, retired, until, keep, u.AssetID, u.Lifecycle, u.PendingKey)
	if err != nil {
		return "", err
	}
	changed, _ := result.RowsAffected()
	if changed != 1 {
		return "", ErrConflict
	}
	u.ObjectKey = u.PendingKey
	u.Lifecycle = phase
	if err = s.deleteOldAssetObjectV2(ctx, objects, u); err != nil {
		return "", err
	}
	return phase, nil
}

func (s *Store) deleteOldAssetObjectV2(ctx context.Context, objects AssetObjectStorageV2, u AssetUploadV2) error {
	if u.SourceKey == "" {
		return nil
	}
	if u.SourceKey == u.ObjectKey {
		return ErrAssetUnavailable
	}
	if err := objects.Delete(ctx, u.SourceKey); err != nil {
		return err
	}
	_, err := s.DB.ExecContext(ctx, "UPDATE asset_uploads_v2 SET source_object_key=NULL,lifecycle_retry_at=NULL,lifecycle_error=NULL WHERE asset_id=? AND object_key=? AND source_object_key=?", u.AssetID, u.ObjectKey, u.SourceKey)
	return err
}

// The containing catalog/announcement permission must be checked before passing
// a non-empty binding here. New unrelated assets must still belong to the actor.
func (s *Store) EnsureAssetsActiveV2(ctx context.Context, actor, binding, ref string, ids []string) error {
	ordered := append([]string(nil), ids...)
	sort.Strings(ordered)
	for _, id := range ordered {
		var phase string
		if err := s.DB.QueryRowContext(ctx, "SELECT lifecycle_state FROM asset_uploads_v2 WHERE asset_id=?", id).Scan(&phase); err != nil {
			return ErrAssetUnavailable
		}
		if phase == "ACTIVE" {
			continue
		} // Existing business transactions validate owner/purpose again under their row locks.
		objects, err := objectstorage.FromEnvironment()
		if err != nil {
			return err
		}
		if err = s.RestoreAssetLifecycleV2(ctx, objects, objects.Config.Prefix, actor, binding, ref, id); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) RestoreAssetLifecycleV2(ctx context.Context, objects AssetObjectStorageV2, prefix, actor, binding, ref, id string) error {
	if !validAssetPrefixV2(prefix) {
		return ErrAssetUnavailable
	}
	if !s.assetLifecycle.TryLock() {
		return ErrAssetBusy
	}
	defer s.assetLifecycle.Unlock()
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	u, err := scanAssetV2(tx.QueryRowContext(ctx, assetSelectV2+" WHERE asset_id=? FOR UPDATE", id))
	if err != nil {
		return ErrAssetUnavailable
	}
	if u.Status != "READY" || u.RemovedAt.Valid || !strings.HasPrefix(u.ObjectKey, prefix) {
		return ErrAssetUnavailable
	}
	if u.UserID != actor {
		var existing int
		if binding == "" || ref == "" {
			return ErrAssetForbidden
		}
		if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM asset_bindings_v2 WHERE asset_id=? AND business_type=? AND business_ref=?", id, binding, ref).Scan(&existing); err != nil {
			return err
		}
		if existing != 1 {
			return ErrAssetForbidden
		}
	}
	if u.Lifecycle == "ACTIVE" {
		return nil
	}
	if u.Lifecycle != "RETIRED" {
		return ErrAssetUnavailable
	}
	if u.SourceKey != "" {
		return ErrAssetBusy
	}
	if !u.RetainUntil.Valid || !u.RetainUntil.Time.After(time.Now()) {
		return ErrAssetUnavailable
	}
	u.SourceKey = u.ObjectKey
	u.PendingKey = prefix + "uploads/" + ID("restored_") + path.Ext(u.ObjectKey)
	u.Lifecycle = "RESTORING"
	_, err = tx.ExecContext(ctx, "UPDATE asset_uploads_v2 SET lifecycle_state='RESTORING',source_object_key=object_key,pending_object_key=?,lifecycle_retry_at=NULL WHERE asset_id=?", u.PendingKey, id)
	if err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	phase, err := s.finishAssetMoveV2(ctx, objects, u)
	if err != nil {
		return err
	}
	if phase != "ACTIVE" {
		return ErrAssetUnavailable
	}
	return nil
}

type transactionImageBindingV2 struct {
	id, kind, ref, phase string
	until                sql.NullTime
}

func transactionImageBindingsV2(ctx context.Context, tx *sql.Tx, d CommerceRecordV2) ([]transactionImageBindingV2, error) {
	rows, err := tx.QueryContext(ctx, `SELECT b.asset_id,b.business_type,b.business_ref,a.lifecycle_state,a.retain_until FROM asset_bindings_v2 b JOIN asset_uploads_v2 a ON a.asset_id=b.asset_id WHERE (b.business_type=? AND b.business_ref=?) OR (b.business_type=? AND b.business_ref=?) OR (b.business_type='REFUND' AND b.business_ref=?) ORDER BY b.asset_id,b.business_type`, d.Kind+"_SNAPSHOT", d.SnapshotID, d.Kind+"_COMPLETION", d.ID, d.RefundID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []transactionImageBindingV2{}
	for rows.Next() {
		var b transactionImageBindingV2
		if err = rows.Scan(&b.id, &b.kind, &b.ref, &b.phase, &b.until); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// Opening a case on an already-settled order is the one supported transition
// from historical use back to current use. Restore available photos beforehand;
// images that already expired remain optional evidence, never a blocker to a case.
func (s *Store) prepareInterventionImagesV2(ctx context.Context, actor, id, kind string, expected int64) error {
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	d, err := commerceRecordTxV2(ctx, tx, id, kind, false)
	if err != nil {
		return err
	}
	if d.OwnerID != actor || d.Channel == "OFFICIAL_STORE" || d.Version != expected || d.InterventionCaseID != "" || d.FundsState != "SETTLED" {
		return nil
	}
	var rejected int
	if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM commerce_refunds_v2 WHERE refund_id=? AND resource_id=? AND state='REJECTED'", d.RefundID, id).Scan(&rejected); err != nil {
		return err
	}
	if rejected != 1 {
		return nil
	}
	bindings, err := transactionImageBindingsV2(ctx, tx, d)
	if err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	for _, b := range bindings {
		if b.phase == "ACTIVE" || b.phase == "EXPIRED" || b.phase == "PURGED" || (b.until.Valid && !b.until.Time.After(time.Now())) {
			continue
		}
		if err = s.EnsureAssetsActiveV2(ctx, actor, b.kind, b.ref, []string{b.id}); err != nil {
			return err
		}
	}
	return nil
}

func lockInterventionImagesV2(ctx context.Context, tx *sql.Tx, d CommerceRecordV2) error {
	bindings, err := transactionImageBindingsV2(ctx, tx, d)
	if err != nil {
		return err
	}
	for _, b := range bindings {
		var phase string
		var until sql.NullTime
		if err = tx.QueryRowContext(ctx, "SELECT lifecycle_state,retain_until FROM asset_uploads_v2 WHERE asset_id=? FOR UPDATE", b.id).Scan(&phase, &until); err != nil {
			return err
		}
		if phase == "ACTIVE" || phase == "EXPIRED" || phase == "PURGED" || (until.Valid && !until.Time.After(time.Now())) {
			continue
		}
		return ErrAssetBusy
	}
	return nil
}
