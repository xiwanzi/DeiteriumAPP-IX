package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"math"
	"math/big"
	"regexp"
	"sort"
	"strings"
	"time"
)

type CommerceStepResultV2 struct {
	CoreOperationID, Status, ErrorCode string
	Data                               map[string]any
	FromMailboxEvent                   bool
}
type commerceMoneyProofV2 struct{ Settled, Refunded, PoolUUID, PayeeUUID string }

var commerceUUIDPatternV2 = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func commerceUncertainCodeV2(code string) bool {
	switch code {
	case "RESULT_UNKNOWN", "STORAGE_UNAVAILABLE", "ECONOMY_UNAVAILABLE", "IDEMPOTENCY_CONFLICT", "ESCROW_CONFLICT", "ESCROW_ALREADY_RESERVED", "ESCROW_LEDGER_MISMATCH", "PAYEE_LOCKED", "PAYEE_NOT_BOUND", "ESCROW_AMOUNT_EXCEEDED":
		return true
	}
	return false
}
func commerceCheckMoneyProofV2(d CommerceRecordV2, step CommerceStepV2, r CommerceStepResultV2) (commerceMoneyProofV2, error) {
	invalid := catalogError(409, "PAYMENT_PROOF_MISMATCH", "资金结果尚未与原操作核对。")
	var proof commerceMoneyProofV2
	p := CatalogObjectV2(r.Data)
	if p == nil || !CatalogReferenceV2(r.CoreOperationID) || catalogString(p, "operationId") != r.CoreOperationID || catalogString(p, "businessRef") != d.ID || catalogString(p, "escrowRef") != d.EscrowRef || catalogString(p, "currency") != "CREDIT" || catalogString(p, "status") != "COMPLETED" || catalogString(p, "payerUuid") != d.OwnerUUID || catalogString(p, "escrowStatus") == "" || commerceBodyTimeV2(p["committedAt"]) == nil {
		return proof, invalid
	}
	businessType := "COMMISSION"
	if d.Kind == "ORDER" {
		businessType = "MARKET_ORDER"
		if d.Channel == "OFFICIAL_STORE" {
			businessType = "OFFICIAL_STORE"
		}
	}
	if catalogString(p, "businessType") != businessType {
		return proof, invalid
	}
	proof.PayeeUUID = catalogString(p, "payeeUuid")
	if d.PayeeUUID != "" && proof.PayeeUUID != d.PayeeUUID {
		return proof, invalid
	}
	if d.PayeeUUID == "" && d.Channel != "OFFICIAL_STORE" && proof.PayeeUUID != "" {
		return proof, invalid
	}
	if d.Channel == "OFFICIAL_STORE" && !commerceUUIDPatternV2.MatchString(proof.PayeeUUID) {
		return proof, invalid
	}
	amount, e := commerceAmountV2(catalogString(p, "amount"))
	if e != nil {
		return proof, invalid
	}
	expected := new(big.Int)
	if step.Command != "wallet.escrow.bind" {
		expected, e = commerceAmountV2(catalogString(CatalogObjectV2(step.Payload), "amount"))
		if e != nil || expected.Sign() <= 0 {
			return proof, invalid
		}
	}
	if amount.Cmp(expected) != 0 {
		return proof, invalid
	}
	reserved, e := commerceAmountV2(catalogString(p, "reservedAmount"))
	if e != nil {
		return proof, invalid
	}
	total, e := commerceAmountV2(d.Amount)
	if e != nil || reserved.Cmp(total) != 0 {
		return proof, invalid
	}
	settled, e := commerceAmountV2(catalogString(p, "settledAmount"))
	if e != nil {
		return proof, invalid
	}
	refunded, e := commerceAmountV2(catalogString(p, "refundedAmount"))
	if e != nil {
		return proof, invalid
	}
	held, e := commerceAmountV2(catalogString(p, "heldAmount"))
	if e != nil {
		return proof, invalid
	}
	sum := new(big.Int).Add(new(big.Int).Set(settled), refunded)
	sum.Add(sum, held)
	if sum.Cmp(reserved) != 0 {
		return proof, invalid
	}
	wantSettled, e := commerceAmountV2(d.SettledAmount)
	if e != nil {
		return proof, invalid
	}
	wantRefunded, e := commerceAmountV2(d.RefundedAmount)
	if e != nil {
		return proof, invalid
	}
	from, to := catalogString(p, "fromUuid"), catalogString(p, "toUuid")
	pool := catalogString(d.Body, "escrowPoolUuid")
	switch step.Command {
	case "wallet.escrow.reserve":
		if wantSettled.Sign() != 0 || wantRefunded.Sign() != 0 || from != d.OwnerUUID || !commerceUUIDPatternV2.MatchString(to) || to == d.OwnerUUID || to == proof.PayeeUUID {
			return proof, invalid
		}
		pool = to
	case "wallet.escrow.bind":
		if proof.PayeeUUID != catalogString(CatalogObjectV2(step.Payload), "payeeUuid") {
			return proof, invalid
		}
	case "wallet.escrow.settle":
		wantSettled.Add(wantSettled, expected)
		if pool == "" || from != pool || to != d.PayeeUUID {
			return proof, invalid
		}
	case "wallet.escrow.refund":
		wantRefunded.Add(wantRefunded, expected)
		if pool == "" || from != pool || to != d.OwnerUUID {
			return proof, invalid
		}
	default:
		return proof, invalid
	}
	if settled.Cmp(wantSettled) != 0 || refunded.Cmp(wantRefunded) != 0 {
		return proof, invalid
	}
	proof.Settled = catalogMoneyString(settled)
	proof.Refunded = catalogMoneyString(refunded)
	proof.PoolUUID = pool
	return proof, nil
}
func commerceVerifyMailReceiptV2(d CommerceRecordV2, receipt CatalogObjectV2) bool {
	plan, ok := catalogObject(d.Body["mailboxPlan"])
	if !ok || receipt == nil {
		return false
	}
	if catalogString(receipt, "deliveryId") != catalogString(plan, "deliveryId") || catalogString(receipt, "orderId") != d.ID || catalogString(receipt, "recipientUuid") != d.OwnerUUID || catalogString(receipt, "snapshotSha256") != catalogString(plan, "snapshotSha256") || catalogString(receipt, "inventoryDomain") != catalogString(plan, "inventoryDomain") || catalogString(receipt, "source") != "deuterium-commerce" || !catalogText(receipt["mailId"], 1, 128) || !catalogRange(receipt["revision"], 1, 2147483647) {
		return false
	}
	left, ok := catalogRefs(receipt["allowedServerIds"], 1, 32)
	if !ok {
		return false
	}
	right := catalogIDs(plan, "allowedServerIds")
	sort.Strings(left)
	sort.Strings(right)
	return strings.Join(left, "|") == strings.Join(right, "|") && catalogEnum(receipt["status"], "CREATED", "CLAIMING", "CLAIMED", "REVOKED", "FAILED", "UNKNOWN")
}
func commerceMailResultV2(d CommerceRecordV2, step CommerceStepV2, r CommerceStepResultV2) (string, string, CatalogObjectV2) {
	data := CatalogObjectV2(r.Data)
	code := catalogString(data, "code")
	receipt, has := catalogObject(data["value"])
	if has && !commerceVerifyMailReceiptV2(d, receipt) {
		return "UNKNOWN", "MAILBOX_PROOF_MISMATCH", nil
	}
	if code == "UNKNOWN" || code == "CLAIM_IN_PROGRESS" {
		return "UNKNOWN", code, nil
	}
	if !has {
		if code == "" {
			return "UNKNOWN", "MAILBOX_PROOF_MISSING", nil
		}
		return "FAILED", code, nil
	}
	if code == "OK" || code == "ALREADY_REVOKED" {
		status := catalogString(receipt, "status")
		if step.Command == "mailbox.revoke" {
			if status != "REVOKED" {
				return "UNKNOWN", "MAILBOX_NOT_REVOKED", receipt
			}
			return "COMPLETED", "", receipt
		}
		if status == "REVOKED" || status == "FAILED" {
			return "FAILED", "DELIVERY_REVOKED", receipt
		}
		if status != "CREATED" && status != "CLAIMING" && status != "CLAIMED" {
			return "UNKNOWN", "MAILBOX_STATUS_UNCONFIRMED", receipt
		}
		return "COMPLETED", "", receipt
	}
	if code == "ALREADY_CLAIMED" && receipt["status"] == "CLAIMED" {
		return "FAILED", code, receipt
	}
	return "FAILED", code, receipt
}

func commerceCancellationResultV2(d CommerceRecordV2, r CommerceStepResultV2) (string, string, CatalogObjectV2, bool) {
	data := CatalogObjectV2(r.Data)
	code := catalogString(data, "code")
	value, ok := catalogObject(data["value"])
	if !ok {
		return "UNKNOWN", "CANCELLATION_PROOF_MISSING", nil, false
	}
	if code != "OK" && code != "ALREADY_REVOKED" {
		if code == "UNKNOWN" || code == "CLAIM_IN_PROGRESS" {
			return "UNKNOWN", code, nil, false
		}
		return "FAILED", code, nil, false
	}
	plan, ok := catalogObject(d.Body["mailboxPlan"])
	if !ok {
		return "UNKNOWN", "DELIVERY_SNAPSHOT_MISSING", nil, false
	}
	at, validTime := commerceEpochSecondsV2(value["cancelledAt"])
	if !validTime || at <= 0 || r.CoreOperationID == "" || catalogString(value, "operationId") != r.CoreOperationID || catalogString(value, "source") != "deuterium-commerce" || catalogString(value, "deliveryId") != catalogString(plan, "deliveryId") || catalogString(value, "orderId") != d.ID || catalogString(value, "recipientUuid") != d.OwnerUUID || catalogString(value, "snapshotSha256") != catalogString(plan, "snapshotSha256") || catalogString(value, "inventoryDomain") != catalogString(plan, "inventoryDomain") {
		return "UNKNOWN", "CANCELLATION_PROOF_MISMATCH", nil, false
	}
	left, ok := catalogRefs(value["allowedServerIds"], 1, 32)
	right := catalogIDs(plan, "allowedServerIds")
	sort.Strings(left)
	sort.Strings(right)
	if !ok || strings.Join(left, "|") != strings.Join(right, "|") {
		return "UNKNOWN", "CANCELLATION_PROOF_MISMATCH", nil, false
	}
	switch catalogString(value, "proofKind") {
	case "REVOKED_MAIL":
		receipt, ok := catalogObject(value["mailReceipt"])
		if !ok || !commerceVerifyMailReceiptV2(d, receipt) || receipt["status"] != "REVOKED" {
			return "UNKNOWN", "MAILBOX_NOT_REVOKED", nil, false
		}
		return "COMPLETED", "", receipt, false
	case "CANCELLED_BEFORE_CREATE":
		_, present := value["mailReceipt"]
		if (present && value["mailReceipt"] != nil) || catalogString(d.Body, "mailId") != "" || catalogNumber(d.Body, "mailboxRevision") > 0 {
			return "UNKNOWN", "CANCELLATION_PROOF_MISMATCH", nil, false
		}
		return "COMPLETED", "", nil, true
	default:
		return "UNKNOWN", "CANCELLATION_PROOF_KIND_INVALID", nil, false
	}
}
func commerceEpochSecondsV2(value any) (int64, bool) {
	// Keep Unix seconds valid beyond 2038. JSON integers remain exact through
	// 2^53-1; fractional, negative and non-finite timestamps are not proof.
	if n, ok := value.(float64); ok {
		if math.IsNaN(n) || math.IsInf(n, 0) || n != math.Trunc(n) || n <= 0 || n > 9007199254740991 {
			return 0, false
		}
		return int64(n), true
	}
	return catalogInt(value)
}
func commerceApplyMailFactV2(d *CommerceRecordV2, receipt CatalogObjectV2) bool {
	if receipt == nil {
		return false
	}
	previous := catalogNumber(d.Body, "mailboxRevision")
	revision := catalogNumber(receipt, "revision")
	if revision < previous {
		return false
	}
	old := catalogString(d.Body, "mailboxState")
	status := catalogString(receipt, "status")
	if (old == "CLAIMED" || old == "REVOKED") && old != status {
		return false
	}
	if old == status && revision == previous {
		return false
	}
	d.Body["mailboxState"] = status
	d.Body["mailboxRevision"] = revision
	d.Body["mailId"] = receipt["mailId"]
	if status == "CLAIMED" && d.State != "REFUNDED" {
		d.State = "CLAIMED"
	}
	return true
}

func (s *Store) ClaimCommerceStepV2(ctx context.Context, id string, now time.Time) (CommerceOperationV2, CommerceRecordV2, bool, error) {
	op, e := s.CommerceOperationV2(ctx, id)
	if e != nil {
		return op, CommerceRecordV2{}, false, e
	}
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return op, CommerceRecordV2{}, false, e
	}
	defer tx.Rollback()
	d, e := commerceRecordTxV2(ctx, tx, op.ResourceID, "", true)
	if e != nil {
		return op, d, false, e
	}
	op, e = commerceOperationScanV2(tx.QueryRowContext(ctx, commerceOperationSelectV2+" WHERE operation_id=? FOR UPDATE", id))
	if e != nil {
		return op, d, false, e
	}
	if op.State == "COMPLETED" || op.State == "FAILED" || d.PendingOperationID != id || op.StepIndex >= len(op.Steps) {
		return op, d, false, nil
	}
	if op.DispatchUntil != nil && op.DispatchUntil.After(now) {
		return op, d, false, nil
	}
	token := ID("dispatch_")
	until := now.Add(30 * time.Second)
	op.DispatchToken = token
	op.DispatchUntil = &until
	op.UpdatedAt = now
	if op.Steps[op.StepIndex].State == "PREPARED" {
		op.Steps[op.StepIndex].State = "DISPATCHING"
	}
	if e = commerceSaveOperationV2(ctx, tx, op, false); e != nil {
		return op, d, false, e
	}
	if e = tx.Commit(); e != nil {
		return op, d, false, e
	}
	return op, d, true, nil
}
func (s *Store) ApplyCommerceStepV2(ctx context.Context, id string, index int, token string, result CommerceStepResultV2, now time.Time) (CommerceOperationV2, error) {
	original, e := s.CommerceOperationV2(ctx, id)
	if e != nil {
		return original, e
	}
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return original, e
	}
	defer tx.Rollback()
	d, e := commerceRecordTxV2(ctx, tx, original.ResourceID, "", true)
	if e != nil {
		return original, e
	}
	op, e := commerceOperationScanV2(tx.QueryRowContext(ctx, commerceOperationSelectV2+" WHERE operation_id=? FOR UPDATE", id))
	if e != nil {
		return op, e
	}
	if op.State == "COMPLETED" || op.State == "FAILED" || op.StepIndex != index || index < 0 || index >= len(op.Steps) || d.PendingOperationID != id {
		return op, nil
	}
	step := &op.Steps[index]
	if !result.FromMailboxEvent && op.DispatchToken != token {
		return op, nil
	}
	if result.FromMailboxEvent && step.Command != "mailbox.create" && step.Command != "mailbox.revoke" {
		return op, catalogInvalid()
	}
	priorFunds, priorState, priorStepState, priorError := d.FundsState, d.State, step.State, step.ErrorCode
	status := result.Status
	code := commerceErrorCodeV2(result.ErrorCode)
	if !catalogEnum(status, "COMPLETED", "FAILED", "PROCESSING", "UNKNOWN") {
		status = "UNKNOWN"
		code = "CORE_RESULT_INVALID"
	}
	if status == "FAILED" && commerceUncertainCodeV2(code) {
		status = "UNKNOWN"
	}
	if status == "FAILED" && !result.FromMailboxEvent && !CatalogReferenceV2(result.CoreOperationID) {
		status, code = "UNKNOWN", "CORE_OPERATION_MISSING"
	}
	identityMismatch := false
	if result.CoreOperationID != "" {
		if !CatalogReferenceV2(result.CoreOperationID) || (step.CoreOperationID != "" && step.CoreOperationID != result.CoreOperationID) {
			status = "UNKNOWN"
			code = "CORE_OPERATION_MISMATCH"
			identityMismatch = true
		} else {
			step.CoreOperationID = result.CoreOperationID
		}
	}
	var money commerceMoneyProofV2
	var receipt CatalogObjectV2
	cancelledBeforeCreate := false
	mail := step.Command == "mailbox.create" || step.Command == "mailbox.revoke"
	if mail && result.Data != nil && !identityMismatch && (status == "COMPLETED" || status == "FAILED" || result.FromMailboxEvent) {
		value, _ := catalogObject(result.Data["value"])
		if step.Command == "mailbox.revoke" && value["proofKind"] != nil {
			status, code, receipt, cancelledBeforeCreate = commerceCancellationResultV2(d, result)
		} else {
			status, code, receipt = commerceMailResultV2(d, *step, result)
		}
	} else if status == "COMPLETED" {
		if money, e = commerceCheckMoneyProofV2(d, *step, result); e != nil {
			status = "UNKNOWN"
			code = "PAYMENT_PROOF_MISMATCH"
		}
	}
	changedMail := commerceApplyMailFactV2(&d, receipt)
	if cancelledBeforeCreate {
		d.Body["mailboxState"] = "CANCELLED_BEFORE_CREATE"
		d.Body["mailboxCancellation"] = result.Data["value"]
		changedMail = true
	}
	step.State = status
	step.ErrorCode = code
	op.ErrorCode = code
	op.DispatchToken = ""
	op.DispatchUntil = nil
	op.UpdatedAt = now
	switch status {
	case "COMPLETED":
		step.Proof = result.Data
		op.ErrorCode = ""
		if !mail {
			d.SettledAmount = money.Settled
			d.RefundedAmount = money.Refunded
			if money.PoolUUID != "" {
				d.Body["escrowPoolUuid"] = money.PoolUUID
			}
			if d.Channel == "OFFICIAL_STORE" && d.PayeeUUID == "" {
				d.PayeeUUID = money.PayeeUUID
			}
		}
		if step.Command == "wallet.escrow.reserve" {
			d.FundsState = "HELD"
			if d.Kind == "COMMISSION" {
				d.State = "OPEN"
			} else if d.Channel == "PLAYER_MARKET" {
				d.State = "AWAITING_SHIPMENT"
			}
			if _, e = tx.ExecContext(ctx, "UPDATE commerce_stock_holds_v2 SET state='PURCHASED' WHERE resource_id=? AND state='RESERVED'", d.ID); e != nil {
				return op, e
			}
		}
		if step.Command == "wallet.escrow.bind" {
			d.State = "ACTIVE"
			d.FundsState = "HELD"
			d.Body["acceptedAt"] = now
			content, _ := catalogObject(d.Body["content"])
			commerceSetDeadlineV2(&d, "WORK", now.Add(time.Duration(catalogNumber(content, "workHours"))*time.Hour))
			d.SnapshotID = ID("snapshot_")
			snapshot := CatalogObjectV2{"snapshotId": d.SnapshotID, "commissionId": d.ID, "owner": d.Body["owner"], "worker": d.Body["worker"], "content": content, "acceptanceHours": 72, "capturedAt": now}
			d.SnapshotSHA256 = commerceSnapshotHashV2(snapshot)
			snapshot["sha256"] = d.SnapshotSHA256
			record := d
			record.Version++
			if e = commerceInsertSnapshotV2(ctx, tx, record, snapshot, now); e != nil {
				return op, e
			}
			if e = s.CopyAssetBindingsV2(ctx, tx, "COMMISSION", d.ID, "COMMISSION_SNAPSHOT", d.SnapshotID, []string{catalogString(content, "coverAssetId")}); e != nil {
				return op, e
			}
		}
		if step.Command == "mailbox.create" {
			d.FundsState = "HELD"
			if catalogString(d.Body, "mailboxState") != "CLAIMED" {
				d.State = "AWAITING_CLAIM"
			}
		}
		op.StepIndex++
		if op.StepIndex == len(op.Steps) {
			op.State = "COMPLETED"
			d.PendingOperationID = ""
			if e = s.commerceCompleteOperationV2(ctx, tx, &d, op, now); e != nil {
				return op, e
			}
		} else {
			op.State = "PROCESSING"
			if op.Action == "case-resolution" {
				d.FundsState = "INTERVENTION_HOLD"
			}
		}
	case "FAILED":
		op.State = "FAILED"
		d.PendingOperationID = ""
		if e = s.commerceFailOperationV2(ctx, tx, &d, op, now); e != nil {
			return op, e
		}
	default:
		op.State = status
		if status == "UNKNOWN" {
			d.FundsState = "UNKNOWN"
		}
	}
	changed := status == "COMPLETED" || status == "FAILED" || changedMail || priorFunds != d.FundsState || priorState != d.State || priorStepState != status || priorError != code
	if changed {
		commerceTouchV2(&d, now)
	}
	if e = commerceSaveOperationV2(ctx, tx, op, false); e != nil {
		return op, e
	}
	if changed {
		if e = commerceSaveV2(ctx, tx, &d, false); e != nil {
			return op, e
		}
		summary := "原操作结果仍待核实，不会换标识重复执行。"
		if status == "COMPLETED" {
			summary = "原资金或交付步骤已由服务端证明确认。"
		}
		if status == "FAILED" {
			summary = "原操作明确未完成，请查看交易当前状态后处理。"
		}
		if e = commerceEventV2(ctx, tx, d, op.ActorID, "operation."+status, summary, map[string]any{"operationId": op.ID, "step": index, "command": step.Command, "errorCode": code}); e != nil {
			return op, e
		}
	}
	if e = tx.Commit(); e != nil {
		return op, e
	}
	return op, nil
}

func (s *Store) commerceCompleteOperationV2(ctx context.Context, tx *sql.Tx, d *CommerceRecordV2, op CommerceOperationV2, now time.Time) error {
	switch op.Action {
	case "reserve", "bind", "deliver":
		return nil
	case "settle":
		d.FundsState = "SETTLED"
		d.State = "CONFIRMED"
		if d.Channel == "OFFICIAL_STORE" {
			d.State = "CLAIMED"
		}
		d.Body["confirmedAt"] = now
		d.Automatic = op.Automatic
		d.Deadline = nil
		d.DeadlineKind = ""
		d.PausedRemaining = nil
		d.PausedDeadlineKind = ""
		return commerceFinalizeStockV2(ctx, tx, d.ID)
	case "refund", "cancel":
		d.FundsState = "REFUNDED"
		d.State = "REFUNDED"
		if d.Kind == "COMMISSION" {
			d.State = "CANCELLED"
		}
		d.Deadline = nil
		d.DeadlineKind = ""
		d.PausedRemaining = nil
		d.PausedDeadlineKind = ""
		if d.RefundID != "" {
			r, e := commerceRefundTxV2(ctx, tx, d.RefundID, true)
			if e != nil {
				return e
			}
			r.State = "APPROVED"
			r.Version++
			r.ResolvedAt = &now
			if e = commerceSaveRefundV2(ctx, tx, r, false); e != nil {
				return e
			}
		}
		return commerceReleaseStockV2(ctx, tx, *d)
	case "case-resolution":
		remaining, e := commerceRemainingV2(*d)
		if e != nil {
			return e
		}
		if remaining.Sign() != 0 {
			return catalogError(409, "CASE_FUNDS_INCOMPLETE", "案件资金尚未全部处理。")
		}
		refund, e := commerceAmountV2(d.RefundedAmount)
		if e != nil {
			return e
		}
		total, e := commerceAmountV2(d.Amount)
		if e != nil {
			return e
		}
		d.Deadline = nil
		d.DeadlineKind = ""
		d.PausedRemaining = nil
		d.PausedDeadlineKind = ""
		if refund.Cmp(total) == 0 {
			d.FundsState = "REFUNDED"
			d.State = "REFUNDED"
			if d.Kind == "COMMISSION" {
				d.State = "CANCELLED"
			}
			if e = commerceReleaseStockV2(ctx, tx, *d); e != nil {
				return e
			}
		} else {
			d.FundsState = "SETTLED"
			d.State = "CONFIRMED"
			d.Body["confirmedAt"] = now
			if e = commerceFinalizeStockV2(ctx, tx, d.ID); e != nil {
				return e
			}
		}
		if d.RefundID != "" && refund.Sign() > 0 {
			r, e := commerceRefundTxV2(ctx, tx, d.RefundID, true)
			if e != nil {
				return e
			}
			r.State = "APPROVED"
			r.Version++
			r.Amount = d.RefundedAmount
			r.ResolvedAt = &now
			if e = commerceSaveRefundV2(ctx, tx, r, false); e != nil {
				return e
			}
		}
		_, e = tx.ExecContext(ctx, "UPDATE commerce_interventions_v2 SET state='RESOLVED',version=version+1,funds_held=FALSE,updated_at=? WHERE case_id=? AND resource_id=?", now, d.InterventionCaseID, d.ID)
		if e != nil {
			return e
		}
		return enqueueCaseEmailV204(ctx, tx, d.InterventionCaseID)
	}
	return catalogInvalid()
}
func (s *Store) commerceFailOperationV2(ctx context.Context, tx *sql.Tx, d *CommerceRecordV2, op CommerceOperationV2, now time.Time) error {
	switch op.Action {
	case "reserve":
		if op.StepIndex == 0 {
			d.State = "CANCELLED"
			d.FundsState = "UNPAID"
			return commerceReleaseStockV2(ctx, tx, *d)
		}
		d.FundsState = "HELD"
		if catalogString(d.Body, "mailboxState") != "CLAIMED" {
			d.State = "PAYMENT_PROCESSING"
		}
	case "bind":
		d.PayeeID = ""
		d.PayeeUUID = ""
		d.Body["worker"] = nil
		d.Body["acceptedAt"] = nil
		d.Body["workDueAt"] = nil
		d.State = "OPEN"
		d.FundsState = "HELD"
		d.Deadline = nil
		d.DeadlineKind = ""
	case "settle":
		d.State = op.PriorState
		d.FundsState = op.PriorFundsState
		if op.Automatic {
			next := now.Add(5 * time.Minute)
			d.NextAutoAttempt = &next
		}
	case "refund":
		d.FundsState = "HELD"
		if d.RefundID != "" {
			r, e := commerceRefundTxV2(ctx, tx, d.RefundID, true)
			if e != nil {
				return e
			}
			r.Version++
			if catalogString(d.Body, "mailboxState") == "CLAIMED" {
				r.State = "REJECTED"
				r.RejectionReason = "游戏邮箱已领取，无法按未领取订单退款。"
				r.ResolvedAt = &now
				d.State = "CLAIMED"
			} else if r.Immediate {
				r.State = "PROCESSING"
			} else {
				r.State = "REQUESTED"
			}
			if e = commerceSaveRefundV2(ctx, tx, r, false); e != nil {
				return e
			}
		}
	case "cancel":
		d.FundsState = "HELD"
		d.State = "OPEN"
	case "deliver":
		d.FundsState = "HELD"
		if catalogString(d.Body, "mailboxState") != "CLAIMED" {
			d.State = "PAYMENT_PROCESSING"
		}
	case "case-resolution":
		d.FundsState = "INTERVENTION_HOLD"
		_, e := tx.ExecContext(ctx, "UPDATE commerce_interventions_v2 SET state='RESOLVING',version=version+1,updated_at=? WHERE case_id=?", now, d.InterventionCaseID)
		if e != nil {
			return e
		}
		return enqueueCaseEmailV204(ctx, tx, d.InterventionCaseID)
	}
	return nil
}

func (s *Store) PendingCommerceOperationsV2(ctx context.Context, limit int) ([]CommerceOperationV2, error) {
	if limit < 1 || limit > 100 {
		return nil, catalogInvalid()
	}
	rows, e := s.DB.QueryContext(ctx, commerceOperationSelectV2+" WHERE state IN ('PROCESSING','UNKNOWN') AND (dispatch_until IS NULL OR dispatch_until<=UTC_TIMESTAMP(6)) ORDER BY updated_at,operation_id LIMIT ?", limit)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []CommerceOperationV2{}
	for rows.Next() {
		o, e := commerceOperationScanV2(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, o)
	}
	return out, rows.Err()
}
func (s *Store) DueCommerceResourcesV2(ctx context.Context, now time.Time, limit int) ([]CommerceRecordV2, error) {
	if limit < 1 || limit > 100 {
		return nil, catalogInvalid()
	}
	rows, e := s.DB.QueryContext(ctx, commerceSelectV2+" WHERE funds_state='HELD' AND pending_operation_id IS NULL AND (next_auto_attempt IS NULL OR next_auto_attempt<=?) AND ((deadline_at<=? AND ((resource_kind='COMMISSION' AND state='COMPLETED') OR (resource_kind='ORDER' AND (state='WORK_COMPLETED' OR (state='SHIPPED' AND COALESCE(JSON_UNQUOTE(JSON_EXTRACT(body,'$.construction')),'false')='false'))))) OR (channel='OFFICIAL_STORE' AND state='CLAIMED')) ORDER BY sequence_id LIMIT ?", now, now, limit)
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
		construction, _ := d.Body["construction"].(bool)
		if d.Kind == "ORDER" && construction && d.State != "WORK_COMPLETED" && d.Channel != "OFFICIAL_STORE" {
			continue
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
func (s *Store) OfficialCommerceResourcesV2(ctx context.Context, limit int) ([]CommerceRecordV2, error) {
	if limit < 1 || limit > 100 {
		return nil, catalogInvalid()
	}
	rows, e := s.DB.QueryContext(ctx, commerceSelectV2+" WHERE channel='OFFICIAL_STORE' AND funds_state IN ('HELD','UNKNOWN','REFUNDING','PROCESSING') AND EXISTS (SELECT 1 FROM core_mail_receipts m WHERE m.order_id=commerce_resources_v2.resource_id AND m.delivery_id=JSON_UNQUOTE(JSON_EXTRACT(body,'$.mailboxPlan.deliveryId')) AND m.recipient_uuid=owner_uuid AND m.snapshot_sha256=JSON_UNQUOTE(JSON_EXTRACT(body,'$.mailboxPlan.snapshotSha256')) AND m.revision>COALESCE(CAST(JSON_UNQUOTE(JSON_EXTRACT(body,'$.mailboxRevision')) AS UNSIGNED),0)) ORDER BY updated_at,sequence_id LIMIT ?", limit)
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
func (s *Store) ObserveCommerceMailboxV2(ctx context.Context, resource string, receipt MailReceipt, now time.Time) (string, error) {
	bytes, _ := json.Marshal(receipt)
	var value CatalogObjectV2
	if e := json.Unmarshal(bytes, &value); e != nil {
		return "", e
	}
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return "", e
	}
	defer tx.Rollback()
	d, e := commerceRecordTxV2(ctx, tx, resource, "ORDER", true)
	if e != nil {
		return "", e
	}
	if d.Channel != "OFFICIAL_STORE" || !commerceVerifyMailReceiptV2(d, value) {
		return "", catalogError(409, "MAILBOX_PROOF_MISMATCH", "邮件回执与原订单不一致。")
	}
	pending := d.PendingOperationID
	changed := commerceApplyMailFactV2(&d, value)
	// A definite cancellation rejection may precede the durable claimed event.
	// An in-flight refund remains owned by its original operation and runner.
	if d.PendingOperationID == "" && d.RefundID != "" && catalogString(d.Body, "mailboxState") == "CLAIMED" {
		refund, err := commerceRefundTxV2(ctx, tx, d.RefundID, true)
		if err != nil {
			return "", err
		}
		if refund.Immediate && refund.State == "PROCESSING" {
			refund.State = "REJECTED"
			refund.RejectionReason = "游戏邮箱已领取，无法按未领取订单退款。"
			refund.Version++
			refund.ResolvedAt = &now
			if err = commerceSaveRefundV2(ctx, tx, refund, false); err != nil {
				return "", err
			}
			changed = true
		}
	}
	if changed {
		commerceTouchV2(&d, now)
		if e = commerceSaveV2(ctx, tx, &d, false); e != nil {
			return "", e
		}
		if e = commerceEventV2(ctx, tx, d, "", "mailbox.observed", "游戏邮箱状态已按原订单回执同步。", map[string]any{"status": receipt.Status, "revision": receipt.Revision}); e != nil {
			return "", e
		}
	}
	if e = tx.Commit(); e != nil {
		return "", e
	}
	return pending, nil
}

func (s *Store) DeferCommerceOperationV2(ctx context.Context, id string, until time.Time) error {
	_, e := s.DB.ExecContext(ctx, "UPDATE commerce_operations_v2 SET dispatch_until=?,updated_at=UTC_TIMESTAMP(6) WHERE operation_id=? AND state IN ('PROCESSING','UNKNOWN') AND (dispatch_until IS NULL OR dispatch_until<=UTC_TIMESTAMP(6))", until, id)
	return e
}
