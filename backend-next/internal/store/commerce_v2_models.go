package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math/big"
	"time"
)

type CommerceRecordV2 struct {
	Sequence                                                                               int64
	ID, Kind, Channel, OwnerID, OwnerUUID, PayeeID, PayeeUUID, StoreID, QuoteID, EscrowRef string
	Amount, SettledAmount, RefundedAmount, State, FundsState                               string
	Body                                                                                   CatalogObjectV2
	SnapshotID, SnapshotSHA256                                                             string
	Version                                                                                int64
	PendingOperationID                                                                     string
	Deadline                                                                               *time.Time
	DeadlineKind                                                                           string
	PausedRemaining                                                                        *int64
	PausedDeadlineKind, RefundID                                                           string
	RefundAttempts                                                                         int64
	InterventionCaseID                                                                     string
	Automatic                                                                              bool
	NextAutoAttempt                                                                        *time.Time
	CreatedAt, UpdatedAt                                                                   time.Time
}
type CommerceRefundV2 struct {
	ID, ResourceID                                  string
	Version                                         int64
	State, ReasonCode, Description, RejectionReason string
	Evidence                                        []string
	Amount                                          string
	Immediate                                       bool
	RequestedAt                                     time.Time
	ResolvedAt                                      *time.Time
}
type CommerceStepV2 struct {
	Command         string         `json:"command"`
	Payload         map[string]any `json:"payload"`
	ClientKey       string         `json:"clientKey"`
	State           string         `json:"state"`
	CoreOperationID string         `json:"coreOperationId"`
	ErrorCode       string         `json:"errorCode"`
	Proof           map[string]any `json:"proof,omitempty"`
}
type CommerceOperationV2 struct {
	ID, ResourceID, ActorID, ClientRequestID, Kind, Action, Amount, State, ErrorCode string
	Steps                                                                            []CommerceStepV2
	StepIndex                                                                        int
	DispatchToken                                                                    string
	DispatchUntil                                                                    *time.Time
	PriorState, PriorFundsState                                                      string
	Automatic                                                                        bool
	CreatedAt, UpdatedAt                                                             time.Time
}
type CommerceMutationV2 struct {
	ResourceID, Kind, OperationID string
	Replayed                      bool
}
type CommerceCaseV2 struct {
	Sequence                                                      int64
	ID, ResourceID, ApplicantID, RespondentID, State              string
	Version                                                       int64
	Body, Snapshot                                                CatalogObjectV2
	AssignedAdminID                                               string
	FundsHeld                                                     bool
	DesiredRefundAmount, TargetRefundAmount, Decision, Resolution string
	EvidenceEntries                                               []CatalogObjectV2
	CreatedAt, UpdatedAt                                          time.Time
}

const commerceSelectV2 = `SELECT sequence_id,resource_id,resource_kind,channel,owner_id,owner_uuid,payee_id,payee_uuid,store_id,quote_id,escrow_ref,amount,settled_amount,refunded_amount,state,funds_state,body,snapshot_id,snapshot_sha256,version,pending_operation_id,deadline_at,deadline_kind,paused_remaining_seconds,paused_deadline_kind,refund_id,refund_attempts,intervention_case_id,automatic,next_auto_attempt,created_at,updated_at FROM commerce_resources_v2`
const commerceOperationSelectV2 = `SELECT operation_id,resource_id,actor_id,client_request_id,operation_kind,action,amount,state,error_code,steps,step_index,dispatch_token,dispatch_until,prior_state,prior_funds_state,automatic,created_at,updated_at FROM commerce_operations_v2`

func commerceOptionalString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
func commerceTime(t sql.NullTime) *time.Time {
	if t.Valid {
		return &t.Time
	}
	return nil
}
func commerceScanV2(row catalogScanner) (d CommerceRecordV2, e error) {
	var payee, payeeUUID, shop, quote, pending, deadlineKind, pausedKind, refund, intervention sql.NullString
	var deadline, next sql.NullTime
	var paused sql.NullInt64
	var body string
	e = row.Scan(&d.Sequence, &d.ID, &d.Kind, &d.Channel, &d.OwnerID, &d.OwnerUUID, &payee, &payeeUUID, &shop, &quote, &d.EscrowRef, &d.Amount, &d.SettledAmount, &d.RefundedAmount, &d.State, &d.FundsState, &body, &d.SnapshotID, &d.SnapshotSHA256, &d.Version, &pending, &deadline, &deadlineKind, &paused, &pausedKind, &refund, &d.RefundAttempts, &intervention, &d.Automatic, &next, &d.CreatedAt, &d.UpdatedAt)
	if errors.Is(e, sql.ErrNoRows) {
		return d, catalogNotFound()
	}
	if e != nil {
		return
	}
	d.PayeeID = payee.String
	d.PayeeUUID = payeeUUID.String
	d.StoreID = shop.String
	d.QuoteID = quote.String
	d.PendingOperationID = pending.String
	d.Deadline = commerceTime(deadline)
	d.DeadlineKind = deadlineKind.String
	d.PausedDeadlineKind = pausedKind.String
	d.RefundID = refund.String
	d.InterventionCaseID = intervention.String
	d.NextAutoAttempt = commerceTime(next)
	if paused.Valid {
		d.PausedRemaining = &paused.Int64
	}
	e = json.Unmarshal([]byte(body), &d.Body)
	return
}
func commerceOperationScanV2(row catalogScanner) (o CommerceOperationV2, e error) {
	var errorCode, token sql.NullString
	var until sql.NullTime
	var steps string
	e = row.Scan(&o.ID, &o.ResourceID, &o.ActorID, &o.ClientRequestID, &o.Kind, &o.Action, &o.Amount, &o.State, &errorCode, &steps, &o.StepIndex, &token, &until, &o.PriorState, &o.PriorFundsState, &o.Automatic, &o.CreatedAt, &o.UpdatedAt)
	if errors.Is(e, sql.ErrNoRows) {
		return o, catalogNotFound()
	}
	if e != nil {
		return
	}
	o.ErrorCode = errorCode.String
	o.DispatchToken = token.String
	o.DispatchUntil = commerceTime(until)
	e = json.Unmarshal([]byte(steps), &o.Steps)
	return
}
func commerceRecordTxV2(ctx context.Context, tx *sql.Tx, id, kind string, lock bool) (CommerceRecordV2, error) {
	q := commerceSelectV2 + " WHERE resource_id=?"
	args := []any{id}
	if kind != "" {
		q += " AND resource_kind=?"
		args = append(args, kind)
	}
	if lock {
		q += " FOR UPDATE"
	}
	return commerceScanV2(tx.QueryRowContext(ctx, q, args...))
}
func (s *Store) CommerceRecordV2(ctx context.Context, id string) (CommerceRecordV2, error) {
	return commerceScanV2(s.DB.QueryRowContext(ctx, commerceSelectV2+" WHERE resource_id=?", id))
}
func (s *Store) CommerceOperationV2(ctx context.Context, id string) (CommerceOperationV2, error) {
	return commerceOperationScanV2(s.DB.QueryRowContext(ctx, commerceOperationSelectV2+" WHERE operation_id=?", id))
}
func commerceSaveV2(ctx context.Context, tx *sql.Tx, d *CommerceRecordV2, create bool) error {
	args := []any{d.Kind, d.Channel, d.OwnerID, d.OwnerUUID, commerceOptionalString(d.PayeeID), commerceOptionalString(d.PayeeUUID), commerceOptionalString(d.StoreID), commerceOptionalString(d.QuoteID), d.EscrowRef, d.Amount, d.SettledAmount, d.RefundedAmount, d.State, d.FundsState, catalogJSON(d.Body), d.SnapshotID, d.SnapshotSHA256, d.Version, commerceOptionalString(d.PendingOperationID), d.Deadline, commerceOptionalString(d.DeadlineKind), d.PausedRemaining, commerceOptionalString(d.PausedDeadlineKind), commerceOptionalString(d.RefundID), d.RefundAttempts, commerceOptionalString(d.InterventionCaseID), d.Automatic, d.NextAutoAttempt, d.CreatedAt, d.UpdatedAt}
	if create {
		args = append([]any{d.ID}, args...)
		r, e := tx.ExecContext(ctx, `INSERT INTO commerce_resources_v2(resource_id,resource_kind,channel,owner_id,owner_uuid,payee_id,payee_uuid,store_id,quote_id,escrow_ref,amount,settled_amount,refunded_amount,state,funds_state,body,snapshot_id,snapshot_sha256,version,pending_operation_id,deadline_at,deadline_kind,paused_remaining_seconds,paused_deadline_kind,refund_id,refund_attempts,intervention_case_id,automatic,next_auto_attempt,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, args...)
		if e != nil {
			return e
		}
		d.Sequence, e = r.LastInsertId()
		return e
	}
	args = append(args, d.ID)
	_, e := tx.ExecContext(ctx, `UPDATE commerce_resources_v2 SET resource_kind=?,channel=?,owner_id=?,owner_uuid=?,payee_id=?,payee_uuid=?,store_id=?,quote_id=?,escrow_ref=?,amount=?,settled_amount=?,refunded_amount=?,state=?,funds_state=?,body=?,snapshot_id=?,snapshot_sha256=?,version=?,pending_operation_id=?,deadline_at=?,deadline_kind=?,paused_remaining_seconds=?,paused_deadline_kind=?,refund_id=?,refund_attempts=?,intervention_case_id=?,automatic=?,next_auto_attempt=?,created_at=?,updated_at=? WHERE resource_id=?`, args...)
	return e
}
func commerceSaveOperationV2(ctx context.Context, tx *sql.Tx, o CommerceOperationV2, create bool) error {
	if create {
		_, e := tx.ExecContext(ctx, `INSERT INTO commerce_operations_v2(operation_id,resource_id,actor_id,client_request_id,operation_kind,action,amount,state,error_code,steps,step_index,dispatch_token,dispatch_until,prior_state,prior_funds_state,automatic,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, o.ID, o.ResourceID, o.ActorID, o.ClientRequestID, o.Kind, o.Action, o.Amount, o.State, commerceOptionalString(o.ErrorCode), catalogJSON(o.Steps), o.StepIndex, commerceOptionalString(o.DispatchToken), o.DispatchUntil, o.PriorState, o.PriorFundsState, o.Automatic, o.CreatedAt, o.UpdatedAt)
		return e
	}
	_, e := tx.ExecContext(ctx, "UPDATE commerce_operations_v2 SET state=?,error_code=?,steps=?,step_index=?,dispatch_token=?,dispatch_until=?,updated_at=? WHERE operation_id=?", o.State, commerceOptionalString(o.ErrorCode), catalogJSON(o.Steps), o.StepIndex, commerceOptionalString(o.DispatchToken), o.DispatchUntil, o.UpdatedAt, o.ID)
	return e
}
func commerceRefundTxV2(ctx context.Context, tx *sql.Tx, id string, lock bool) (r CommerceRefundV2, e error) {
	q := "SELECT refund_id,resource_id,version,state,reason_code,description,rejection_reason,evidence_json,amount,immediate,requested_at,resolved_at FROM commerce_refunds_v2 WHERE refund_id=?"
	if lock {
		q += " FOR UPDATE"
	}
	var evidence string
	var resolved sql.NullTime
	e = tx.QueryRowContext(ctx, q, id).Scan(&r.ID, &r.ResourceID, &r.Version, &r.State, &r.ReasonCode, &r.Description, &r.RejectionReason, &evidence, &r.Amount, &r.Immediate, &r.RequestedAt, &resolved)
	if errors.Is(e, sql.ErrNoRows) {
		return r, catalogNotFound()
	}
	if e != nil {
		return
	}
	r.ResolvedAt = commerceTime(resolved)
	e = json.Unmarshal([]byte(evidence), &r.Evidence)
	return
}
func commerceSaveRefundV2(ctx context.Context, tx *sql.Tx, r CommerceRefundV2, create bool) error {
	if create {
		_, e := tx.ExecContext(ctx, "INSERT INTO commerce_refunds_v2 VALUES(?,?,?,?,?,?,?,?,?,?,?,?)", r.ID, r.ResourceID, r.Version, r.State, r.ReasonCode, r.Description, r.RejectionReason, catalogJSON(r.Evidence), r.Amount, r.Immediate, r.RequestedAt, r.ResolvedAt)
		return e
	}
	_, e := tx.ExecContext(ctx, "UPDATE commerce_refunds_v2 SET version=?,state=?,rejection_reason=?,amount=?,resolved_at=? WHERE refund_id=?", r.Version, r.State, r.RejectionReason, r.Amount, r.ResolvedAt, r.ID)
	return e
}
func commerceAmountV2(s string) (*big.Int, error) {
	if s == "0" || s == "0.0" || s == "0.00" {
		return new(big.Int), nil
	}
	n, ok := catalogMoney(s)
	if !ok {
		return nil, catalogInvalid()
	}
	return n, nil
}
func commerceRemainingV2(d CommerceRecordV2) (*big.Int, error) {
	amount, e := commerceAmountV2(d.Amount)
	if e != nil {
		return nil, e
	}
	settled, e := commerceAmountV2(d.SettledAmount)
	if e != nil {
		return nil, e
	}
	refunded, e := commerceAmountV2(d.RefundedAmount)
	if e != nil {
		return nil, e
	}
	amount.Sub(amount, settled).Sub(amount, refunded)
	if amount.Sign() < 0 {
		return nil, catalogError(503, "LEDGER_INCONSISTENT", "交易资金需要核对。")
	}
	return amount, nil
}
func commerceRequireVersionV2(d CommerceRecordV2, expected int64) error {
	if expected != d.Version || expected < 1 || expected >= 2147483647 {
		return catalogVersion()
	}
	if d.PendingOperationID != "" {
		return catalogError(409, "OPERATION_IN_PROGRESS", "上一笔操作仍在处理中，请稍后查看。")
	}
	return nil
}
func commercePauseV2(d *CommerceRecordV2, now time.Time) {
	if d.Deadline != nil {
		seconds := int64(d.Deadline.Sub(now) / time.Second)
		if seconds < 0 {
			seconds = 0
		}
		d.PausedRemaining = &seconds
		d.PausedDeadlineKind = d.DeadlineKind
		if d.Kind == "COMMISSION" {
			if d.DeadlineKind == "WORK" {
				d.Body["workDueAt"] = nil
			} else if d.DeadlineKind == "ACCEPTANCE" {
				d.Body["acceptanceDueAt"] = nil
			}
		}
		d.Deadline = nil
		d.DeadlineKind = ""
	}
}
func commerceResumeV2(d *CommerceRecordV2, now time.Time) {
	if d.PausedRemaining != nil {
		deadline := now.Add(time.Duration(*d.PausedRemaining) * time.Second)
		d.Deadline = &deadline
		d.DeadlineKind = d.PausedDeadlineKind
		if d.Kind == "COMMISSION" {
			if d.DeadlineKind == "WORK" {
				d.Body["workDueAt"] = deadline
			} else if d.DeadlineKind == "ACCEPTANCE" {
				d.Body["acceptanceDueAt"] = deadline
			}
		}
		d.PausedRemaining = nil
		d.PausedDeadlineKind = ""
	}
}
