package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

func ValidateCommissionContentV2(content CatalogObjectV2) error {
	if !catalogFields(content, "title description location urgency reward workHours coverAssetId", "") || !catalogText(content["title"], 2, 60) || !catalogText(content["description"], 5, 2000) || !catalogText(content["location"], 2, 120) || !catalogEnum(content["urgency"], "NORMAL", "SOON", "URGENT") || !catalogRange(content["workHours"], 1, 8760) || !CatalogReferenceV2(catalogString(content, "coverAssetId")) {
		return catalogInvalid()
	}
	if _, ok := catalogMoney(content["reward"]); !ok {
		return catalogInvalid()
	}
	return nil
}
func CommerceActionInputV2(input CatalogObjectV2, action string) (string, int64, error) {
	required := "clientRequestId expectedVersion"
	switch action {
	case "complete", "complete-work":
		required += " description evidenceAssetIds"
	case "refund":
		required += " reasonCode description evidenceAssetIds"
	case "resolve-refund":
		required += " decision reason"
	case "delivery-retry":
		required += " reason"
	}
	if !catalogFields(input, required, "") || !CatalogReferenceV2(catalogString(input, "clientRequestId")) || !catalogRange(input["expectedVersion"], 1, 2147483647) {
		return "", 0, catalogInvalid()
	}
	if action == "complete" || action == "complete-work" || action == "refund" {
		if !catalogText(input["description"], 2, 500) {
			return "", 0, catalogInvalid()
		}
		if _, ok := catalogRefs(input["evidenceAssetIds"], 0, 5); !ok {
			return "", 0, catalogInvalid()
		}
	}
	if action == "refund" && !catalogEnum(input["reasonCode"], "NO_LONGER_NEEDED", "DELIVERY_DELAY", "NOT_AS_DESCRIBED", "OTHER") {
		return "", 0, catalogInvalid()
	}
	if action == "resolve-refund" {
		if !catalogEnum(input["decision"], "APPROVE", "REJECT") || !catalogText(input["reason"], 0, 500) || (input["decision"] == "REJECT" && !catalogText(input["reason"], 2, 500)) {
			return "", 0, catalogInvalid()
		}
	}
	if action == "delivery-retry" && !catalogText(input["reason"], 2, 500) {
		return "", 0, catalogInvalid()
	}
	return catalogString(input, "clientRequestId"), catalogNumber(input, "expectedVersion"), nil
}
func CommerceCreateOrderInputV2(input CatalogObjectV2) (string, string, int64, error) {
	if !catalogFields(input, "clientRequestId quoteId expectedQuoteVersion", "") || !CatalogReferenceV2(catalogString(input, "clientRequestId")) || !CatalogReferenceV2(catalogString(input, "quoteId")) || !catalogRange(input["expectedQuoteVersion"], 1, 2147483647) {
		return "", "", 0, catalogInvalid()
	}
	return catalogString(input, "clientRequestId"), catalogString(input, "quoteId"), catalogNumber(input, "expectedQuoteVersion"), nil
}
func CommerceStringV2(input CatalogObjectV2, key string) string { return catalogString(input, key) }
func CommerceContentV2(input CatalogObjectV2, key string) (CatalogObjectV2, bool) {
	return catalogObject(input[key])
}

func commerceRefundViewV2(r CommerceRefundV2) map[string]any {
	return map[string]any{"refundId": r.ID, "status": r.State, "attempt": 1, "reason": r.Description, "rejectionReason": r.RejectionReason, "requestedAt": r.RequestedAt, "resolvedAt": r.ResolvedAt, "amount": r.Amount, "version": r.Version}
}
func commerceCaseActiveTxV2(ctx context.Context, tx *sql.Tx, d CommerceRecordV2) (bool, error) {
	if d.InterventionCaseID == "" {
		return false, nil
	}
	var state string
	e := tx.QueryRowContext(ctx, "SELECT state FROM commerce_interventions_v2 WHERE case_id=? AND resource_id=?", d.InterventionCaseID, d.ID).Scan(&state)
	if e != nil {
		return true, e
	}
	return state != "RESOLVED" && state != "WITHDRAWN", nil
}
func commerceActionsV2(d CommerceRecordV2, viewer string, refund *CommerceRefundV2, caseActive bool) []string {
	out := []string{}
	owner, payee := viewer == d.OwnerID, viewer == d.PayeeID
	if d.Channel == "OFFICIAL_STORE" && owner && d.State != "CANCELLED" {
		out = append(out, "VIEW_MAILBOX")
	}
	if d.PendingOperationID != "" || d.FundsState == "UNKNOWN" || caseActive {
		return out
	}
	if refund != nil && (refund.State == "REQUESTED" || refund.State == "PROCESSING") {
		if owner && refund.State == "REQUESTED" {
			out = append(out, "WITHDRAW_REFUND")
		}
		if payee && refund.State == "REQUESTED" {
			out = append(out, "RESOLVE_REFUND")
		}
		if owner && refund.Immediate && refund.State == "PROCESSING" && d.FundsState == "HELD" {
			out = append(out, "REQUEST_REFUND")
		}
		return out
	}
	if owner && refund != nil && refund.State == "REJECTED" && d.Channel != "OFFICIAL_STORE" && d.InterventionCaseID == "" {
		out = append(out, "REQUEST_INTERVENTION")
	}
	if d.FundsState != "HELD" {
		return out
	}
	if d.Kind == "COMMISSION" {
		switch d.State {
		case "OPEN":
			if owner {
				out = append(out, "CANCEL")
			} else if !payee {
				out = append(out, "ACCEPT")
			}
		case "ACTIVE":
			if payee {
				out = append(out, "COMPLETE")
			}
			if owner && d.RefundAttempts == 0 {
				out = append(out, "REQUEST_REFUND")
			}
		case "COMPLETED":
			if owner {
				out = append(out, "CONFIRM")
				if d.RefundAttempts == 0 {
					out = append(out, "REQUEST_REFUND")
				}
			}
		}
		return out
	}
	construction, _ := d.Body["construction"].(bool)
	switch d.State {
	case "AWAITING_SHIPMENT":
		if payee {
			if construction {
				out = append(out, "START_WORK")
			} else {
				out = append(out, "SHIP")
			}
		}
	case "SHIPPED":
		if construction {
			if payee {
				out = append(out, "COMPLETE_WORK")
			}
		} else if owner {
			out = append(out, "CONFIRM_RECEIPT")
		}
	case "WORK_COMPLETED":
		if owner {
			out = append(out, "CONFIRM_ACCEPTANCE")
		}
	}
	if owner && d.RefundAttempts == 0 && (d.State == "AWAITING_SHIPMENT" || d.State == "SHIPPED" || d.State == "WORK_COMPLETED" || d.State == "AWAITING_CLAIM" || (d.Channel == "OFFICIAL_STORE" && d.State == "PAYMENT_PROCESSING")) {
		out = append(out, "REQUEST_REFUND")
	}
	return out
}
func commerceBaseViewV2(d CommerceRecordV2, viewer string, refund *CommerceRefundV2, caseActive bool, now time.Time) map[string]any {
	out := map[string]any{}
	fields := "orderNo construction buyer seller items delivery confirmationHours shippedAt workCompletedAt confirmedAt completionDescription completionAssetIds"
	if d.Kind == "COMMISSION" {
		fields = "owner worker content workDueAt acceptanceDueAt completionDescription completionAssetIds acceptedAt completedAt confirmedAt"
	}
	for _, k := range strings.Fields(fields) {
		out[k] = d.Body[k]
	}
	out["completionDescription"] = catalogString(d.Body, "completionDescription")
	completionIDs := catalogIDs(d.Body, "completionAssetIds")
	if completionIDs == nil {
		completionIDs = []string{}
	}
	out["completionAssetIds"] = completionIDs
	out["status"] = d.State
	out["fundsStatus"] = d.FundsState
	out["snapshotId"] = d.SnapshotID
	out["serverNow"] = now
	out["pausedRemainingSeconds"] = d.PausedRemaining
	out["refund"] = nil
	if refund != nil {
		out["refund"] = commerceRefundViewV2(*refund)
	}
	out["refundAttemptsUsed"] = d.RefundAttempts
	out["availableActions"] = commerceActionsV2(d, viewer, refund, caseActive)
	out["canHideRecord"] = commerceCanHideV203(d, viewer, refund, caseActive)
	out["createdAt"] = d.CreatedAt
	out["automatic"] = d.Automatic
	out["version"] = d.Version
	out["interventionCaseId"] = commerceOptionalString(d.InterventionCaseID)
	out["pendingOperationId"] = commerceOptionalString(d.PendingOperationID)
	if d.Kind == "ORDER" {
		out["orderId"] = d.ID
		out["channel"] = d.Channel
		out["amount"] = d.Amount
		out["currency"] = "CREDIT"
		out["snapshotSha256"] = d.SnapshotSHA256
		out["autoConfirmAt"] = d.Deadline
	} else {
		out["commissionId"] = d.ID
	}
	return out
}
func (s *Store) CommerceViewV2(ctx context.Context, viewer, id string, public bool) (map[string]any, error) {
	tx, e := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	d, e := commerceRecordTxV2(ctx, tx, id, "", false)
	if e != nil {
		return nil, e
	}
	if e = commercePermissionTxV2(ctx, tx, viewer, d, public); e != nil {
		return nil, e
	}
	var refund *CommerceRefundV2
	if d.RefundID != "" {
		r, e := commerceRefundTxV2(ctx, tx, d.RefundID, false)
		if e != nil {
			return nil, e
		}
		refund = &r
	}
	active, e := commerceCaseActiveTxV2(ctx, tx, d)
	if e != nil {
		return nil, e
	}
	out := commerceBaseViewV2(d, viewer, refund, active, time.Now().UTC())
	hidden, e := hiddenRecordV203(ctx, tx, viewer, d.Kind, d.ID)
	if e != nil {
		return nil, e
	}
	out["hiddenFromHistory"] = hidden && commerceCanHideV203(d, viewer, refund, active)
	if e = tx.Commit(); e != nil {
		return nil, e
	}
	if d.Kind == "COMMISSION" {
		content, _ := catalogObject(d.Body["content"])
		cover, e := s.AssetForBindingV2(ctx, "COMMISSION", d.ID, catalogString(content, "coverAssetId"))
		if e != nil {
			return nil, e
		}
		out["cover"] = cover
	}
	if d.Kind == "ORDER" {
		// Materialize authorized URLs beside the immutable item snapshot. This also
		// lets older orders load images after the listing is no longer public.
		rows, err := s.DB.QueryContext(ctx, "SELECT asset_id FROM asset_bindings_v2 WHERE business_type='ORDER_SNAPSHOT' AND business_ref=? ORDER BY asset_id", d.SnapshotID)
		if err != nil {
			return nil, err
		}
		ids := []string{}
		for rows.Next() {
			var id string
			if err = rows.Scan(&id); err != nil {
				rows.Close()
				return nil, err
			}
			ids = append(ids, id)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
		images := []map[string]any{}
		for _, id := range ids {
			image, err := s.AssetForBindingV2(ctx, "ORDER_SNAPSHOT", d.SnapshotID, id)
			if err != nil {
				return nil, err
			}
			images = append(images, image)
		}
		out["images"] = images
	}
	completionAssets := []map[string]any{}
	for _, assetID := range catalogIDs(d.Body, "completionAssetIds") {
		asset, err := s.AssetForBindingV2(ctx, d.Kind+"_COMPLETION", d.ID, assetID)
		if err != nil {
			return nil, err
		}
		completionAssets = append(completionAssets, asset)
	}
	out["completionAssets"] = completionAssets
	return out, nil
}
func (s *Store) CommerceSnapshotV2(ctx context.Context, viewer, id string, public bool) (CatalogObjectV2, error) {
	tx, e := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	d, e := commerceRecordTxV2(ctx, tx, id, "", false)
	if e != nil {
		return nil, e
	}
	if e = commercePermissionTxV2(ctx, tx, viewer, d, public); e != nil {
		return nil, e
	}
	var raw string
	e = tx.QueryRowContext(ctx, "SELECT body FROM commerce_snapshots_v2 WHERE snapshot_id=? AND resource_id=?", d.SnapshotID, d.ID).Scan(&raw)
	if e != nil {
		return nil, e
	}
	var out CatalogObjectV2
	e = json.Unmarshal([]byte(raw), &out)
	return out, e
}
func (s *Store) CommerceRefundViewV2(ctx context.Context, viewer, resource, id string) (map[string]any, error) {
	tx, e := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	d, e := commerceRecordTxV2(ctx, tx, resource, "", false)
	if e != nil {
		return nil, e
	}
	if e = commercePermissionTxV2(ctx, tx, viewer, d, false); e != nil {
		return nil, e
	}
	r, e := commerceRefundTxV2(ctx, tx, id, false)
	if e != nil {
		return nil, e
	}
	if r.ResourceID != d.ID {
		return nil, catalogNotFound()
	}
	return commerceRefundViewV2(r), nil
}
func CommerceOperationViewV2(o CommerceOperationV2, kind string) map[string]any {
	return map[string]any{"operationId": o.ID, "clientRequestId": o.ClientRequestID, "kind": o.Kind, "status": o.State, "resourceType": kind, "resourceId": o.ResourceID, "amount": o.Amount, "createdAt": o.CreatedAt, "updatedAt": o.UpdatedAt, "errorCode": commerceOptionalString(o.ErrorCode), "retryAfterSeconds": 3}
}
func (s *Store) CommerceVisibleOperationV2(ctx context.Context, viewer, id, key, kind string) (CommerceOperationV2, CommerceRecordV2, error) {
	var o CommerceOperationV2
	var e error
	if id != "" {
		o, e = s.CommerceOperationV2(ctx, id)
	} else {
		o, e = commerceOperationScanV2(s.DB.QueryRowContext(ctx, commerceOperationSelectV2+" WHERE actor_id=? AND client_request_id=? AND operation_kind=?", viewer, key, kind))
	}
	if e != nil {
		return o, CommerceRecordV2{}, e
	}
	d, e := s.CommerceRecordV2(ctx, o.ResourceID)
	if e != nil {
		return o, d, e
	}
	if o.ActorID != viewer && d.OwnerID != viewer && d.PayeeID != viewer {
		allowed, e := s.HasPermission(ctx, viewer, "platform.admin")
		if e != nil {
			return o, d, e
		}
		if !allowed {
			return o, d, catalogNotFound()
		}
	}
	return o, d, nil
}

type CommerceFilterV2 struct {
	Kind, Channel, Role, State, Query string
	Urgency, Sort, BeforeAmount       string
	HasRefund                         *bool
	Public                            bool
	Before                            int64
	Limit                             int
	StoreID                           string
}

func (s *Store) CommerceListV2(ctx context.Context, viewer string, f CommerceFilterV2) ([]CommerceRecordV2, error) {
	if f.Limit < 1 || f.Limit > 101 {
		return nil, catalogInvalid()
	}
	if f.Sort == "" {
		f.Sort = "NEWEST"
	}
	if f.Sort != "NEWEST" && f.Sort != "REWARD_DESC" && f.Sort != "REWARD_ASC" {
		return nil, catalogInvalid()
	}
	if f.Sort != "NEWEST" && f.Kind != "COMMISSION" {
		return nil, catalogInvalid()
	}
	q := commerceSelectV2 + " WHERE resource_kind=?"
	args := []any{f.Kind}
	if !f.Public && f.StoreID == "" {
		q += " AND NOT ((" + commerceFinalVisibilityV203 + ") AND EXISTS (SELECT 1 FROM personal_record_visibility_v203 vh WHERE vh.user_id=? AND vh.resource_kind=commerce_resources_v2.resource_kind AND vh.resource_id=commerce_resources_v2.resource_id))"
		args = append(args, viewer)
	}
	if f.Public {
		if f.Kind != "COMMISSION" {
			return nil, catalogInvalid()
		}
		q += " AND state='OPEN' AND funds_state='HELD' AND pending_operation_id IS NULL"
	} else if f.StoreID != "" {
		if e := s.CatalogCanManageV2(ctx, viewer, f.StoreID, "ORDER_MANAGE"); e != nil {
			return nil, e
		}
		q += " AND store_id=?"
		args = append(args, f.StoreID)
	} else {
		switch f.Role {
		case "BUYER", "OWNER", "PUBLISHED":
			q += " AND owner_id=?"
			args = append(args, viewer)
		case "SELLER", "WORKER", "ACCEPTED":
			q += " AND payee_id=?"
			args = append(args, viewer)
		default:
			q += " AND (owner_id=? OR payee_id=?)"
			args = append(args, viewer, viewer)
		}
	}
	if f.Before > 0 && f.Sort == "NEWEST" {
		q += " AND sequence_id<?"
		args = append(args, f.Before)
	}
	if f.Before > 0 && f.Sort != "NEWEST" {
		n, e := commerceAmountV2(f.BeforeAmount)
		if e != nil || n.Sign() <= 0 {
			return nil, catalogInvalid()
		}
		comparison := "<"
		if f.Sort == "REWARD_ASC" {
			comparison = ">"
		}
		q += " AND (CAST(amount AS DECIMAL(20,2))" + comparison + "CAST(? AS DECIMAL(20,2)) OR (CAST(amount AS DECIMAL(20,2))=CAST(? AS DECIMAL(20,2)) AND sequence_id<?))"
		args = append(args, f.BeforeAmount, f.BeforeAmount, f.Before)
	}
	if f.Channel != "" {
		q += " AND channel=?"
		args = append(args, f.Channel)
	}
	if f.State != "" {
		q += " AND state=?"
		args = append(args, f.State)
	}
	if f.HasRefund != nil {
		if *f.HasRefund {
			q += " AND refund_id IS NOT NULL"
		} else {
			q += " AND refund_id IS NULL"
		}
	}
	if f.Urgency != "" {
		if !catalogEnum(f.Urgency, "NORMAL", "SOON", "URGENT") {
			return nil, catalogInvalid()
		}
		q += " AND JSON_UNQUOTE(JSON_EXTRACT(body,'$.content.urgency'))=?"
		args = append(args, f.Urgency)
	}
	if f.Query != "" {
		q += " AND LOCATE(?,LOWER(JSON_UNQUOTE(JSON_EXTRACT(body,'$.content.title'))))>0"
		args = append(args, strings.ToLower(f.Query))
	}
	if f.Sort == "REWARD_DESC" {
		q += " ORDER BY CAST(amount AS DECIMAL(20,2)) DESC,sequence_id DESC LIMIT ?"
	} else if f.Sort == "REWARD_ASC" {
		q += " ORDER BY CAST(amount AS DECIMAL(20,2)) ASC,sequence_id DESC LIMIT ?"
	} else {
		q += " ORDER BY sequence_id DESC LIMIT ?"
	}
	args = append(args, f.Limit)
	rows, e := s.DB.QueryContext(ctx, q, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []CommerceRecordV2{}
	for rows.Next() {
		d, e := commerceScanV2(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
func commerceIsNotFoundV2(e error) bool {
	var p *CatalogErrorV2
	return errors.As(e, &p) && p.Code == "NOT_FOUND"
}
