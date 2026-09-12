package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

func (s *Store) commerceMutateV2(ctx context.Context, actor, key, scope string, input any, fn func(*sql.Tx) (CommerceMutationV2, error)) (CommerceMutationV2, error) {
	// All callbacks persist intents only. A deadlock/restart error rolls the
	// whole transaction back before another attempt; no Core RPC is retried here.
	for attempt := 0; ; attempt++ {
		result, err := s.commerceMutateAttemptV2(ctx, actor, key, scope, input, fn)
		var conflict *mysql.MySQLError
		if attempt >= 3 || !errors.As(err, &conflict) || (conflict.Number != 1020 && conflict.Number != 1213 && conflict.Number != 1205) {
			return result, err
		}
		if ctx.Err() != nil {
			return result, ctx.Err()
		}
	}
}
func (s *Store) commerceMutateAttemptV2(ctx context.Context, actor, key, scope string, input any, fn func(*sql.Tx) (CommerceMutationV2, error)) (CommerceMutationV2, error) {
	var result CommerceMutationV2
	if !CatalogReferenceV2(key) {
		return result, catalogInvalid()
	}
	tx, e := s.beginAccountTx(ctx, actor)
	if e != nil {
		return result, e
	}
	defer tx.Rollback()
	fingerprint := Digest([]byte(scope + ":" + catalogJSON(input)))
	_, e = tx.ExecContext(ctx, `INSERT INTO commerce_requests_v2(actor_id,client_request_id,scope,fingerprint,result,created_at) VALUES(?,?,?,?,'',UTC_TIMESTAMP(6)) ON DUPLICATE KEY UPDATE client_request_id=VALUES(client_request_id)`, actor, key, scope, fingerprint)
	if e != nil {
		return result, e
	}
	var priorScope, priorFingerprint, raw string
	if e = tx.QueryRowContext(ctx, "SELECT scope,fingerprint,result FROM commerce_requests_v2 WHERE actor_id=? AND client_request_id=? FOR UPDATE", actor, key).Scan(&priorScope, &priorFingerprint, &raw); e != nil {
		return result, e
	}
	if priorScope != scope || priorFingerprint != fingerprint {
		return result, catalogError(409, "IDEMPOTENCY_CONFLICT", "这笔请求的内容已变化，请先查看之前的处理结果。")
	}
	if raw != "" {
		if e = json.Unmarshal([]byte(raw), &result); e == nil {
			result.Replayed = true
		}
		return result, e
	}
	if _, e = commerceUserTxV2(ctx, tx, actor, true); e != nil {
		return result, e
	}
	result, e = fn(tx)
	if e != nil {
		return result, e
	}
	if _, e = tx.ExecContext(ctx, "UPDATE commerce_requests_v2 SET result=? WHERE actor_id=? AND client_request_id=?", catalogJSON(result), actor, key); e != nil {
		return result, e
	}
	if e = tx.Commit(); e != nil {
		return result, e
	}
	return result, nil
}
func commerceUserTxV2(ctx context.Context, tx *sql.Tx, id string, active bool) (u User, e error) {
	e = tx.QueryRowContext(ctx, "SELECT id,player_ref,server_uuid,game_id,qq,status FROM identities WHERE id=?", id).Scan(&u.ID, &u.PlayerRef, &u.ServerUUID, &u.GameID, &u.QQ, &u.Status)
	if errors.Is(e, sql.ErrNoRows) {
		return u, catalogNotFound()
	}
	if e != nil {
		return
	}
	if active && u.Status != "active" {
		return u, ErrUnauthorized
	}
	return
}
func commercePartyV2(u User) CatalogObjectV2 {
	var qq any
	if catalogQQPattern.MatchString(u.QQ) {
		qq = u.QQ
	}
	return CatalogObjectV2{"kind": "PLAYER", "playerRef": u.PlayerRef, "storeId": nil, "displayName": u.GameID, "contactQq": qq}
}
func commercePermissionTxV2(ctx context.Context, tx *sql.Tx, user string, d CommerceRecordV2, public bool) error {
	if d.OwnerID == user || d.PayeeID == user {
		return nil
	}
	if public && d.Kind == "COMMISSION" && d.State == "OPEN" && d.FundsState == "HELD" && d.PendingOperationID == "" {
		return nil
	}
	var n int
	if e := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM identity_permissions WHERE user_id=? AND permission='platform.admin'", user).Scan(&n); e != nil {
		return e
	}
	if n > 0 {
		return nil
	}
	if d.Channel == "OFFICIAL_STORE" && d.StoreID != "" {
		if e := catalogPermissionTx(ctx, tx, user, d.StoreID, "ORDER_MANAGE"); e == nil {
			return nil
		}
	}
	return catalogNotFound()
}
func (s *Store) CanAccessCommerceV2(ctx context.Context, user, businessType, reference string) (bool, error) {
	tx, e := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return false, e
	}
	defer tx.Rollback()
	if businessType == "INTERVENTION" {
		var resource string
		e = tx.QueryRowContext(ctx, "SELECT resource_id FROM commerce_interventions_v2 WHERE case_id=?", reference).Scan(&resource)
		if errors.Is(e, sql.ErrNoRows) {
			return false, nil
		}
		if e != nil {
			return false, e
		}
		reference = resource
		businessType = ""
	}
	d, e := commerceRecordTxV2(ctx, tx, reference, businessType, false)
	if e != nil {
		var p *CatalogErrorV2
		if errors.As(e, &p) && p.Code == "NOT_FOUND" {
			return false, nil
		}
		return false, e
	}
	if e = commercePermissionTxV2(ctx, tx, user, d, false); e != nil {
		var p *CatalogErrorV2
		if errors.As(e, &p) && p.Code == "NOT_FOUND" {
			return false, nil
		}
		return false, e
	}
	return true, nil
}

func (s *Store) CommerceCanUploadEvidenceV2(ctx context.Context, user, businessType, reference string) (bool, error) {
	if businessType != "ORDER" && businessType != "COMMISSION" && businessType != "INTERVENTION" {
		return false, nil
	}
	allowed, e := s.CanAccessCommerceV2(ctx, user, businessType, reference)
	if e != nil || allowed {
		return allowed, e
	}
	moderator, e := s.HasPermission(ctx, user, "intervention.manage")
	if e != nil || !moderator {
		return false, e
	}
	var n int
	q := "SELECT COUNT(*) FROM commerce_interventions_v2 WHERE resource_id=? AND assigned_admin_id=?"
	if businessType == "INTERVENTION" {
		q = "SELECT COUNT(*) FROM commerce_interventions_v2 WHERE case_id=? AND assigned_admin_id=?"
	}
	e = s.DB.QueryRowContext(ctx, q, reference, user).Scan(&n)
	return n > 0, e
}
func commerceEventV2(ctx context.Context, tx *sql.Tx, d CommerceRecordV2, actor, kind, summary string, metadata any) error {
	id := ID("event_")
	_, e := tx.ExecContext(ctx, "INSERT INTO commerce_events_v2(event_id,resource_id,event_type,actor_id,summary,metadata,created_at) VALUES(?,?,?,?,?,?,UTC_TIMESTAMP(6))", id, d.ID, kind, commerceOptionalString(actor), customerCommerceTextV206(summary), catalogJSON(metadata))
	if e != nil {
		return e
	}
	topic := "MARKET_ORDERS"
	if d.Channel == "OFFICIAL_STORE" {
		topic = "STORE_ORDERS"
	}
	if d.Kind == "COMMISSION" {
		topic = "COMMISSIONS"
	}
	target := SocialTarget{Kind: d.Kind, ReferenceID: d.ID, StateVersion: d.Version}
	recipients := []string{d.OwnerID}
	if d.PayeeID != "" && d.PayeeID != d.OwnerID {
		recipients = append(recipients, d.PayeeID)
	}
	for _, user := range recipients {
		title, body, milestone, push := commerceNoticeV206(d, kind, actor, user)
		if title == "" {
			continue
		}
		if e = AddNotificationWithPushV206(ctx, tx, user, "commerce:"+d.ID+":"+milestone, topic, title, body, target, push); e != nil {
			return e
		}
	}
	return nil
}
func commerceSnapshotHashV2(body CatalogObjectV2) string {
	b := CatalogObjectV2{}
	for k, v := range body {
		if k != "sha256" {
			b[k] = v
		}
	}
	return Digest([]byte(catalogJSON(b)))
}
func commerceInsertSnapshotV2(ctx context.Context, tx *sql.Tx, d CommerceRecordV2, body CatalogObjectV2, now time.Time) error {
	_, e := tx.ExecContext(ctx, "INSERT INTO commerce_snapshots_v2 VALUES(?,?,?,?,?,?,?)", d.SnapshotID, d.ID, d.Kind, d.Version, catalogJSON(body), d.SnapshotSHA256, now)
	return e
}
func commerceNewOperationV2(ctx context.Context, tx *sql.Tx, d *CommerceRecordV2, actor, key, kind, action, amount string, steps []CommerceStepV2, automatic bool, now time.Time) (CommerceOperationV2, error) {
	if d.PendingOperationID != "" {
		return CommerceOperationV2{}, catalogError(409, "OPERATION_IN_PROGRESS", "原资金操作尚未确认。")
	}
	op := CommerceOperationV2{ID: ID("op_"), ResourceID: d.ID, ActorID: actor, ClientRequestID: key, Kind: kind, Action: action, Amount: amount, State: "PROCESSING", Steps: steps, PriorState: d.State, PriorFundsState: d.FundsState, Automatic: automatic, CreatedAt: now, UpdatedAt: now}
	for i := range op.Steps {
		op.Steps[i].ClientKey = fmt.Sprintf("commerce:%s:%d", op.ID, i)
		op.Steps[i].State = "PREPARED"
	}
	if e := commerceSaveOperationV2(ctx, tx, op, true); e != nil {
		return op, e
	}
	d.PendingOperationID = op.ID
	switch action {
	case "settle", "case-resolution":
		d.FundsState = "SETTLING"
	case "refund", "cancel":
		d.FundsState = "REFUNDING"
	case "reserve":
		d.FundsState = "PROCESSING"
	}
	return op, nil
}
func commerceMoneyPayloadV2(d CommerceRecordV2, amount string) map[string]any {
	return map[string]any{"escrowRef": d.EscrowRef, "businessRef": d.ID, "amount": amount, "currency": "CREDIT"}
}
func commerceReserveStepV2(d CommerceRecordV2) CommerceStepV2 {
	p := commerceMoneyPayloadV2(d, d.Amount)
	kind := "COMMISSION"
	if d.Kind == "ORDER" {
		kind = "MARKET_ORDER"
		if d.Channel == "OFFICIAL_STORE" {
			kind = "OFFICIAL_STORE"
		}
	}
	p["businessType"] = kind
	p["payerUuid"] = d.OwnerUUID
	if d.PayeeUUID != "" {
		p["payeeUuid"] = d.PayeeUUID
	}
	return CommerceStepV2{Command: "wallet.escrow.reserve", Payload: p}
}
func commerceMoneyStepV2(d CommerceRecordV2, command, amount string) CommerceStepV2 {
	return CommerceStepV2{Command: command, Payload: commerceMoneyPayloadV2(d, amount)}
}
func commerceAvailableV2(available bool) error {
	if !available {
		return catalogError(503, "CAPABILITY_UNAVAILABLE", "交易服务暂不可用，本次未扣款，请稍后重试。")
	}
	return nil
}
func commerceGuardHeldV2(d CommerceRecordV2) error {
	if d.FundsState != "HELD" {
		return catalogError(409, "INVALID_FUNDS_STATE", "当前资金状态不允许此操作。")
	}
	return nil
}
func commerceRequireFulfillmentV2(ctx context.Context, tx *sql.Tx, d CommerceRecordV2) error {
	if e := commerceGuardHeldV2(d); e != nil {
		return e
	}
	active, e := commerceCaseActiveTxV2(ctx, tx, d)
	if e != nil {
		return e
	}
	if active {
		return catalogError(409, "INTERVENTION_ACTIVE", "交易正在平台处理中。")
	}
	return commerceRequireNoRefundV2(ctx, tx, d)
}
func commerceRefundBlocksV2(ctx context.Context, tx *sql.Tx, d CommerceRecordV2) (bool, error) {
	if d.RefundID == "" {
		return false, nil
	}
	r, e := commerceRefundTxV2(ctx, tx, d.RefundID, false)
	if e != nil {
		return true, e
	}
	return r.State == "REQUESTED" || r.State == "PROCESSING", nil
}
func commerceRequireNoRefundV2(ctx context.Context, tx *sql.Tx, d CommerceRecordV2) error {
	blocked, e := commerceRefundBlocksV2(ctx, tx, d)
	if e != nil {
		return e
	}
	if blocked {
		return catalogError(409, "REFUND_IN_PROGRESS", "请先处理当前退款申请。")
	}
	return nil
}
func commerceBodyTimeV2(v any) *time.Time {
	switch t := v.(type) {
	case time.Time:
		return &t
	case *time.Time:
		return t
	case string:
		if p, e := time.Parse(time.RFC3339Nano, t); e == nil {
			return &p
		}
	}
	return nil
}
func commerceTouchV2(d *CommerceRecordV2, now time.Time) { d.Version++; d.UpdatedAt = now.UTC() }
func commerceSetDeadlineV2(d *CommerceRecordV2, kind string, deadline time.Time) {
	d.Deadline = &deadline
	d.DeadlineKind = kind
	if d.Kind == "COMMISSION" {
		if kind == "WORK" {
			d.Body["workDueAt"] = deadline
		} else if kind == "ACCEPTANCE" {
			d.Body["acceptanceDueAt"] = deadline
		}
	}
}
func commerceReleaseStockV2(ctx context.Context, tx *sql.Tx, d CommerceRecordV2) error {
	if e := promotionReleaseV209(ctx, tx, d); e != nil {
		return e
	}
	rows, e := tx.QueryContext(ctx, "SELECT product_id,quantity FROM commerce_stock_holds_v2 WHERE resource_id=? AND state IN ('RESERVED','PURCHASED') ORDER BY product_id", d.ID)
	if e != nil {
		return e
	}
	type hold struct {
		id       string
		quantity int64
	}
	holds := []hold{}
	for rows.Next() {
		var h hold
		if e = rows.Scan(&h.id, &h.quantity); e != nil {
			rows.Close()
			return e
		}
		holds = append(holds, h)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, h := range holds {
		p, e := catalogScan(tx.QueryRowContext(ctx, catalogSelect+" WHERE resource_id=? FOR UPDATE", h.id))
		if e != nil {
			return e
		}
		if p.Stock != nil {
			limit := int64(999999)
			if p.Kind == "listing" {
				limit = 999
			}
			if *p.Stock+h.quantity > limit {
				return catalogError(503, "STOCK_RECONCILIATION_REQUIRED", "退款库存恢复需要核对，不会重复退款。")
			}
			n := *p.Stock + h.quantity
			p.Stock = &n
			p.Version++
			p.UpdatedAt = time.Now().UTC()
			if e = catalogSave(ctx, tx, &p, false); e != nil {
				return e
			}
		}
		if _, e = tx.ExecContext(ctx, "UPDATE commerce_stock_holds_v2 SET state='RELEASED' WHERE resource_id=? AND product_id=? AND state IN ('RESERVED','PURCHASED')", d.ID, h.id); e != nil {
			return e
		}
	}
	return nil
}
func commerceFinalizeStockV2(ctx context.Context, tx *sql.Tx, id string) error {
	_, e := tx.ExecContext(ctx, "UPDATE commerce_stock_holds_v2 SET state='FINALIZED' WHERE resource_id=? AND state IN ('RESERVED','PURCHASED')", id)
	return e
}
func catalogHeldStockV2(ctx context.Context, tx *sql.Tx, product string) (int64, error) {
	var held int64
	e := tx.QueryRowContext(ctx, "SELECT COALESCE(SUM(quantity),0) FROM commerce_stock_holds_v2 WHERE product_id=? AND state IN ('RESERVED','PURCHASED')", product).Scan(&held)
	return held, e
}
func commerceCreditSumV2(values ...string) (string, error) {
	n := new(big.Int)
	for _, v := range values {
		q, e := commerceAmountV2(v)
		if e != nil {
			return "", e
		}
		n.Add(n, q)
	}
	if len(catalogMoneyString(n)) > 20 {
		return "", catalogInvalid()
	}
	return catalogMoneyString(n), nil
}
func commerceSortedKeysV2(m map[string]int64) []string {
	out := []string{}
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
func commerceErrorCodeV2(code string) string {
	if len(code) > 80 || strings.ContainsAny(code, "\r\n\x00") {
		return "CORE_RESULT_INVALID"
	}
	return code
}
