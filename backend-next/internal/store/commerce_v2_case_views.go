package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

type CommerceCaseFilterV2 struct {
	State, TransactionID string
	AssignedToMe         bool
	Before               int64
	Limit                int
}

func (s *Store) CommerceCaseViewV2(ctx context.Context, viewer, caseID string, admin bool) (map[string]any, error) {
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	c, err := commerceCaseTxV2(ctx, tx, caseID, false)
	if err != nil {
		return nil, err
	}
	manager, err := commerceCaseAdminTxV2(ctx, tx, viewer)
	if err != nil {
		return nil, err
	}
	if admin && !manager {
		return nil, catalogDenied()
	}
	if viewer != c.ApplicantID && viewer != c.RespondentID && !manager {
		return nil, catalogNotFound()
	}
	transaction, err := commerceRecordTxV2(ctx, tx, c.ResourceID, "", false)
	if err != nil {
		return nil, err
	}
	if catalogString(c.Snapshot, "transactionKind") != transaction.Kind || catalogString(c.Snapshot, "transactionId") != transaction.ID {
		return nil, catalogError(503, "EVIDENCE_SNAPSHOT_INVALID", "交易证据快照需要核对。")
	}
	unsigned := CatalogObjectV2{}
	for key, value := range c.Snapshot {
		if key != "sha256" {
			unsigned[key] = value
		}
	}
	if Digest([]byte(catalogJSON(commerceStripAssetURLsV2(unsigned)))) != catalogString(c.Snapshot, "sha256") {
		return nil, catalogError(503, "EVIDENCE_SNAPSHOT_INVALID", "交易证据快照摘要不一致。")
	}
	var assigned any
	if c.AssignedAdminID != "" {
		var ref string
		err = tx.QueryRowContext(ctx, "SELECT player_ref FROM identities WHERE id=?", c.AssignedAdminID).Scan(&ref)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		if err == nil {
			assigned = ref
		}
	}
	snapshotBindings, err := transactionImageBindingsV2(ctx, tx, transaction)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	snapshotAssets := []map[string]any{}
	for _, binding := range snapshotBindings {
		if binding.kind != transaction.Kind+"_SNAPSHOT" {
			continue
		}
		image, err := s.AssetForBindingV2(ctx, binding.kind, binding.ref, binding.id)
		if err != nil {
			return nil, err
		}
		snapshotAssets = append(snapshotAssets, image)
	}
	if transaction.Kind == "COMMISSION" {
		commission, ok := catalogObject(c.Snapshot["commission"])
		if !ok {
			return nil, catalogError(503, "EVIDENCE_SNAPSHOT_INVALID", "委托证据快照需要核对。")
		}
		cover, ok := catalogObject(commission["cover"])
		if !ok {
			return nil, catalogError(503, "EVIDENCE_SNAPSHOT_INVALID", "委托封面快照需要核对。")
		}
		asset, err := s.AssetForBindingV2(ctx, "COMMISSION_SNAPSHOT", catalogString(commission, "snapshotId"), catalogString(cover, "assetId"))
		if err != nil {
			return nil, err
		}
		// Refresh only temporary access URLs; the frozen evidence hash omits them.
		cover["url"], cover["urlExpiresAt"] = asset["url"], asset["urlExpiresAt"]
		commission["cover"] = cover
		c.Snapshot["commission"] = commission
	}
	assets := []map[string]any{}
	for _, id := range catalogIDs(c.Body, "evidenceAssetIds") {
		asset, err := s.AssetForBindingV2(ctx, "INTERVENTION_EVIDENCE", c.ID, id)
		if err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}
	completionAssets := []map[string]any{}
	if frozen, ok := catalogObject(c.Snapshot[strings.ToLower(transaction.Kind)]); ok {
		for _, id := range catalogIDs(frozen, "completionAssetIds") {
			asset, err := s.AssetForBindingV2(ctx, transaction.Kind+"_COMPLETION", transaction.ID, id)
			if err != nil {
				return nil, err
			}
			completionAssets = append(completionAssets, asset)
		}
	}
	entries := c.EvidenceEntries
	if entries == nil {
		entries = []CatalogObjectV2{}
	}
	return map[string]any{
		"caseId": c.ID, "transactionKind": transaction.Kind, "transactionId": c.ResourceID,
		"applicant": c.Body["applicant"], "respondent": c.Body["respondent"], "status": c.State,
		"description": c.Body["description"], "desiredResolution": c.Body["desiredResolution"], "requestedRefundAmount": c.DesiredRefundAmount,
		"evidenceAssetIds": catalogIDs(c.Body, "evidenceAssetIds"), "evidenceAssets": assets, "evidenceEntries": entries, "completionAssets": completionAssets,
		"snapshot": c.Snapshot, "snapshotAssets": snapshotAssets, "fundsHeldForReview": c.FundsHeld, "assignedAdminRef": assigned,
		"resolution": c.Resolution, "decision": commerceOptionalString(c.Decision), "pendingOperationId": commerceOptionalString(transaction.PendingOperationID),
		"createdAt": c.CreatedAt, "updatedAt": c.UpdatedAt, "version": c.Version,
	}, nil
}

func (s *Store) CommerceCaseListV2(ctx context.Context, viewer string, filter CommerceCaseFilterV2) ([]CommerceCaseV2, error) {
	if filter.Limit < 1 || filter.Limit > 101 || filter.Before < 0 || (filter.State != "" && !catalogEnum(filter.State, "SUBMITTED", "IN_REVIEW", "WAITING_EVIDENCE", "RESOLVING", "RESOLVED", "WITHDRAWN")) || (filter.TransactionID != "" && !CatalogReferenceV2(filter.TransactionID)) {
		return nil, catalogInvalid()
	}
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	allowed, err := commerceCaseAdminTxV2(ctx, tx, viewer)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, catalogDenied()
	}
	query := commerceCaseSelectV2 + " WHERE 1=1"
	args := []any{}
	if filter.State != "" {
		query += " AND state=?"
		args = append(args, filter.State)
	}
	if filter.TransactionID != "" {
		query += " AND resource_id=?"
		args = append(args, filter.TransactionID)
	}
	if filter.AssignedToMe {
		query += " AND assigned_admin_id=?"
		args = append(args, viewer)
	}
	if filter.Before > 0 {
		query += " AND sequence_id<?"
		args = append(args, filter.Before)
	}
	query += " ORDER BY sequence_id DESC LIMIT ?"
	args = append(args, filter.Limit)
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []CommerceCaseV2{}
	for rows.Next() {
		record, err := commerceCaseScanV2(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, record)
	}
	return result, rows.Err()
}
