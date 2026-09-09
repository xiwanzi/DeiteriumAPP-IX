package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math/big"
	"slices"
	"time"
)

const commerceCaseSelectV2 = `SELECT sequence_id,case_id,resource_id,applicant_id,respondent_id,state,version,body,snapshot,assigned_admin_id,funds_held,desired_refund_amount,target_refund_amount,decision,resolution,evidence_entries,created_at,updated_at FROM commerce_interventions_v2`

func CommerceInterventionInputV2(input CatalogObjectV2, action string) (string, int64, error) {
	required := "clientRequestId expectedVersion"
	optional := ""
	switch action {
	case "create":
		required += " reasonCode description desiredResolution evidenceAssetIds"
		optional = "requestedRefundAmount relatedMessageIds"
	case "evidence":
		required += " description evidenceAssetIds"
		optional = "relatedMessageIds"
	case "withdraw", "request-evidence":
		required += " reason"
	case "resolve":
		required += " decision reason"
		optional = "refundAmount"
	case "assign":
	default:
		return "", 0, catalogInvalid()
	}
	if !catalogFields(input, required, optional) || !CatalogReferenceV2(catalogString(input, "clientRequestId")) || !catalogRange(input["expectedVersion"], 1, 2147483647) {
		return "", 0, catalogInvalid()
	}
	if action == "create" {
		if !catalogEnum(input["reasonCode"], "NOT_DELIVERED", "NOT_AS_DESCRIBED", "REFUND_DISAGREEMENT", "OTHER") || !catalogText(input["description"], 10, 3000) || !catalogEnum(input["desiredResolution"], "FULL_REFUND", "PARTIAL_REFUND", "CONTINUE_FULFILLMENT", "OTHER") {
			return "", 0, catalogInvalid()
		}
		if input["desiredResolution"] == "PARTIAL_REFUND" {
			if _, ok := catalogMoney(input["requestedRefundAmount"]); !ok {
				return "", 0, catalogInvalid()
			}
		}
	}
	if action == "evidence" && !catalogText(input["description"], 2, 3000) {
		return "", 0, catalogInvalid()
	}
	if action == "create" || action == "evidence" {
		minimum := 0
		if action == "evidence" {
			minimum = 1
		}
		if _, ok := catalogRefs(input["evidenceAssetIds"], minimum, 10); !ok {
			return "", 0, catalogInvalid()
		}
		if ids, has := input["relatedMessageIds"]; has {
			if _, ok := catalogRefs(ids, 0, 30); !ok {
				return "", 0, catalogInvalid()
			}
		}
	}
	if action == "withdraw" || action == "request-evidence" {
		if !catalogText(input["reason"], 2, 500) {
			return "", 0, catalogInvalid()
		}
	}
	if action == "resolve" {
		if !catalogEnum(input["decision"], "FULL_REFUND", "PARTIAL_REFUND", "RELEASE_TO_PAYEE", "CONTINUE_FULFILLMENT", "REQUIRE_MANUAL_RECOVERY", "NO_ACTION") || !catalogText(input["reason"], 10, 3000) {
			return "", 0, catalogInvalid()
		}
		if input["decision"] == "PARTIAL_REFUND" {
			if _, ok := catalogMoney(input["refundAmount"]); !ok {
				return "", 0, catalogInvalid()
			}
		}
	}
	for _, k := range []string{"refundAmount", "requestedRefundAmount"} {
		if value, exists := input[k]; exists {
			if !catalogText(value, 1, 20) {
				return "", 0, catalogInvalid()
			}
			if _, e := commerceAmountV2(value.(string)); e != nil {
				return "", 0, catalogInvalid()
			}
		}
	}
	return catalogString(input, "clientRequestId"), catalogNumber(input, "expectedVersion"), nil
}
func commerceCaseScanV2(row catalogScanner) (c CommerceCaseV2, e error) {
	var respondent, assigned, target, decision sql.NullString
	var body, snapshot, entries string
	e = row.Scan(&c.Sequence, &c.ID, &c.ResourceID, &c.ApplicantID, &respondent, &c.State, &c.Version, &body, &snapshot, &assigned, &c.FundsHeld, &c.DesiredRefundAmount, &target, &decision, &c.Resolution, &entries, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(e, sql.ErrNoRows) {
		return c, catalogNotFound()
	}
	if e != nil {
		return
	}
	c.RespondentID = respondent.String
	c.AssignedAdminID = assigned.String
	c.TargetRefundAmount = target.String
	c.Decision = decision.String
	if e = json.Unmarshal([]byte(body), &c.Body); e != nil {
		return
	}
	if e = json.Unmarshal([]byte(snapshot), &c.Snapshot); e != nil {
		return
	}
	e = json.Unmarshal([]byte(entries), &c.EvidenceEntries)
	return
}
func commerceCaseTxV2(ctx context.Context, tx *sql.Tx, id string, lock bool) (CommerceCaseV2, error) {
	q := commerceCaseSelectV2 + " WHERE case_id=?"
	if lock {
		q += " FOR UPDATE"
	}
	return commerceCaseScanV2(tx.QueryRowContext(ctx, q, id))
}
func commerceSaveCaseV2(ctx context.Context, tx *sql.Tx, c *CommerceCaseV2, create bool) error {
	if create {
		r, e := tx.ExecContext(ctx, `INSERT INTO commerce_interventions_v2(case_id,resource_id,applicant_id,respondent_id,state,version,body,snapshot,assigned_admin_id,funds_held,desired_refund_amount,target_refund_amount,decision,resolution,evidence_entries,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, c.ID, c.ResourceID, c.ApplicantID, commerceOptionalString(c.RespondentID), c.State, c.Version, catalogJSON(c.Body), catalogJSON(c.Snapshot), commerceOptionalString(c.AssignedAdminID), c.FundsHeld, c.DesiredRefundAmount, commerceOptionalString(c.TargetRefundAmount), commerceOptionalString(c.Decision), c.Resolution, catalogJSON(c.EvidenceEntries), c.CreatedAt, c.UpdatedAt)
		if e != nil {
			return e
		}
		c.Sequence, e = r.LastInsertId()
		if e != nil {
			return e
		}
		return enqueueCaseEmailV204(ctx, tx, c.ID)
	}
	_, e := tx.ExecContext(ctx, "UPDATE commerce_interventions_v2 SET state=?,version=?,body=?,assigned_admin_id=?,funds_held=?,target_refund_amount=?,decision=?,resolution=?,evidence_entries=?,updated_at=? WHERE case_id=?", c.State, c.Version, catalogJSON(c.Body), commerceOptionalString(c.AssignedAdminID), c.FundsHeld, commerceOptionalString(c.TargetRefundAmount), commerceOptionalString(c.Decision), c.Resolution, catalogJSON(c.EvidenceEntries), c.UpdatedAt, c.ID)
	if e != nil {
		return e
	}
	return enqueueCaseEmailV204(ctx, tx, c.ID)
}
func commerceCaseAdminTxV2(ctx context.Context, tx *sql.Tx, user string) (bool, error) {
	var n int
	e := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM identity_permissions WHERE user_id=? AND permission IN ('intervention.manage','platform.admin')", user).Scan(&n)
	return n > 0, e
}
func commerceRelatedMessagesV2(ctx context.Context, tx *sql.Tx, user string, ids []string) ([]any, error) {
	out := []any{}
	for _, id := range ids {
		var content, sender string
		e := tx.QueryRowContext(ctx, "SELECT content,sender_ref FROM chat_messages_next WHERE message_id=?", id).Scan(&content, &sender)
		if e == sql.ErrNoRows {
			e = tx.QueryRowContext(ctx, `SELECT m.content,i.player_ref FROM social_messages_v2 m JOIN social_members_v2 member ON member.conversation_id=m.conversation_id AND member.user_id=? JOIN identities i ON i.id=m.sender_id WHERE m.message_id=?`, user, id).Scan(&content, &sender)
		}
		if e == sql.ErrNoRows {
			return nil, catalogError(403, "EVIDENCE_MESSAGE_FORBIDDEN", "引用消息不存在或不属于可访问的会话。")
		}
		if e != nil {
			return nil, e
		}
		out = append(out, map[string]any{"messageId": id, "senderPlayerRef": sender, "content": content})
	}
	return out, nil
}
func commerceStableAssetTxV2(ctx context.Context, tx *sql.Tx, businessType, ref, id string) (map[string]any, error) {
	var purpose, status, md5, alt string
	var size, width, height int64
	var hash sql.NullString
	var created time.Time
	e := tx.QueryRowContext(ctx, `SELECT a.purpose,a.status,a.size_bytes,a.width,a.height,a.sha256,a.alt_text,a.created_at,a.content_md5 FROM asset_uploads_v2 a JOIN asset_bindings_v2 b ON b.asset_id=a.asset_id WHERE a.asset_id=? AND b.business_type=? AND b.business_ref=? AND a.status='READY' AND a.removed_at IS NULL`, id, businessType, ref).Scan(&purpose, &status, &size, &width, &height, &hash, &alt, &created, &md5)
	if e != nil {
		return nil, ErrAssetUnavailable
	}
	return map[string]any{"assetId": id, "purpose": purpose, "status": status, "width": width, "height": height, "sizeBytes": size, "sha256": commerceOptionalString(hash.String), "altText": alt, "createdAt": created, "contentMd5": md5}, nil
}
func commerceStripAssetURLsV2(value any) any {
	switch v := value.(type) {
	case CatalogObjectV2:
		out := map[string]any{}
		for k, x := range v {
			if k != "url" && k != "urlExpiresAt" {
				out[k] = commerceStripAssetURLsV2(x)
			}
		}
		return out
	case map[string]any:
		out := map[string]any{}
		for k, x := range v {
			if k != "url" && k != "urlExpiresAt" {
				out[k] = commerceStripAssetURLsV2(x)
			}
		}
		return out
	case []any:
		out := []any{}
		for _, x := range v {
			out = append(out, commerceStripAssetURLsV2(x))
		}
		return out
	}
	return value
}
func commerceCaseSnapshotV2(ctx context.Context, tx *sql.Tx, d CommerceRecordV2, refund CommerceRefundV2, now time.Time) (CatalogObjectV2, error) {
	view := commerceBaseViewV2(d, d.OwnerID, &refund, false, now)
	if d.Kind == "COMMISSION" {
		content, _ := catalogObject(d.Body["content"])
		asset, e := commerceStableAssetTxV2(ctx, tx, "COMMISSION_SNAPSHOT", d.SnapshotID, catalogString(content, "coverAssetId"))
		if e != nil {
			return nil, e
		}
		view["cover"] = asset
	}
	rows, e := tx.QueryContext(ctx, `SELECT e.event_id,e.event_type,i.player_ref,e.created_at,e.summary FROM commerce_events_v2 e LEFT JOIN identities i ON i.id=e.actor_id WHERE e.resource_id=? ORDER BY e.sequence_id DESC LIMIT 500`, d.ID)
	if e != nil {
		return nil, e
	}
	events := []any{}
	for rows.Next() {
		var id, kind, summary string
		var player sql.NullString
		var at time.Time
		if e = rows.Scan(&id, &kind, &player, &at, &summary); e != nil {
			rows.Close()
			return nil, e
		}
		events = append(events, map[string]any{"eventId": id, "type": kind, "actorPlayerRef": commerceOptionalString(player.String), "occurredAt": at, "summary": summary})
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	slices.Reverse(events)
	snapshot := CatalogObjectV2{"snapshotId": ID("evidence_"), "transactionKind": d.Kind, "transactionId": d.ID, "transactionVersion": d.Version, "capturedAt": now, "order": nil, "commission": nil, "eventLog": events}
	if d.Kind == "ORDER" {
		snapshot["order"] = view
	} else {
		snapshot["commission"] = view
	}
	snapshot["sha256"] = Digest([]byte(catalogJSON(commerceStripAssetURLsV2(snapshot))))
	return snapshot, nil
}
func (s *Store) CreateCommerceInterventionV2(ctx context.Context, actor, id, kind, key string, expected int64, input CatalogObjectV2) (CommerceMutationV2, error) {
	if err := s.prepareInterventionImagesV2(ctx, actor, id, kind, expected); err != nil {
		return CommerceMutationV2{}, err
	}
	return s.commerceMutateV2(ctx, actor, key, "intervention.create:"+id, input, func(tx *sql.Tx) (CommerceMutationV2, error) {
		result := CommerceMutationV2{}
		d, e := commerceRecordTxV2(ctx, tx, id, kind, true)
		if e != nil {
			return result, e
		}
		if actor != d.OwnerID || d.Channel == "OFFICIAL_STORE" {
			return result, catalogDenied()
		}
		if e = commerceRequireVersionV2(d, expected); e != nil {
			return result, e
		}
		if d.InterventionCaseID != "" {
			return result, catalogError(409, "INTERVENTION_EXISTS", "这笔交易已有案件，请进入原案件查看或补充资料。")
		}
		if d.RefundID == "" {
			return result, catalogError(409, "REFUND_NOT_REJECTED", "请先通过原退款流程处理。")
		}
		refund, e := commerceRefundTxV2(ctx, tx, d.RefundID, true)
		if e != nil {
			return result, e
		}
		if refund.State != "REJECTED" {
			return result, catalogError(409, "REFUND_NOT_REJECTED", "仅被拒绝的退款可以申请平台介入。")
		}
		if d.FundsState != "HELD" && d.FundsState != "SETTLED" {
			return result, catalogError(409, "INVALID_FUNDS_STATE", "请先核实原交易资金结果。")
		}
		if err := lockInterventionImagesV2(ctx, tx, d); err != nil {
			return result, err
		}
		now := time.Now().UTC().Truncate(time.Microsecond)
		requested := "0.00"
		if input["desiredResolution"] == "FULL_REFUND" {
			requested = d.Amount
		}
		if value, has := input["requestedRefundAmount"]; has {
			requested = value.(string)
		}
		amount, e := commerceAmountV2(requested)
		if e != nil {
			return result, e
		}
		total, _ := commerceAmountV2(d.Amount)
		if amount.Cmp(total) > 0 || (input["desiredResolution"] == "PARTIAL_REFUND" && (amount.Sign() <= 0 || amount.Cmp(total) >= 0)) {
			return result, catalogError(422, "INVALID_REFUND_AMOUNT", "请求退款金额超出交易范围。")
		}
		requested = catalogMoneyString(amount)
		snapshot, e := commerceCaseSnapshotV2(ctx, tx, d, refund, now)
		if e != nil {
			return result, e
		}
		messages, e := commerceRelatedMessagesV2(ctx, tx, actor, catalogIDs(input, "relatedMessageIds"))
		if e != nil {
			return result, e
		}
		applicant, _ := commerceUserTxV2(ctx, tx, actor, false)
		party := d.Body["seller"]
		if d.Kind == "COMMISSION" {
			party = d.Body["worker"]
		}
		c := CommerceCaseV2{ID: ID("case_"), ResourceID: d.ID, ApplicantID: actor, RespondentID: d.PayeeID, State: "SUBMITTED", Version: 1, Body: CatalogObjectV2{"applicant": commercePartyV2(applicant), "respondent": party, "reasonCode": input["reasonCode"], "description": input["description"], "desiredResolution": input["desiredResolution"], "evidenceAssetIds": catalogIDs(input, "evidenceAssetIds")}, Snapshot: snapshot, FundsHeld: d.FundsState == "HELD", DesiredRefundAmount: requested, EvidenceEntries: []CatalogObjectV2{{"actorPlayerRef": applicant.PlayerRef, "description": input["description"], "evidenceAssetIds": catalogIDs(input, "evidenceAssetIds"), "relatedMessages": messages, "createdAt": now}}, CreatedAt: now, UpdatedAt: now}
		if e = s.commerceBindEvidenceV2(ctx, tx, actor, "INTERVENTION_EVIDENCE", c.ID, catalogIDs(input, "evidenceAssetIds")); e != nil {
			return result, e
		}
		if e = commerceSaveCaseV2(ctx, tx, &c, true); e != nil {
			return result, e
		}
		d.InterventionCaseID = c.ID
		if c.FundsHeld {
			commercePauseV2(&d, now)
			d.FundsState = "INTERVENTION_HOLD"
		}
		commerceTouchV2(&d, now)
		if e = commerceSaveV2(ctx, tx, &d, false); e != nil {
			return result, e
		}
		if e = commerceEventV2(ctx, tx, d, actor, "intervention.created", "平台介入已提交，原案件引用已同步到交易。", map[string]string{"caseId": c.ID}); e != nil {
			return result, e
		}
		return CommerceMutationV2{ResourceID: c.ID, Kind: "INTERVENTION"}, nil
	})
}

func (s *Store) CommerceCaseActionV2(ctx context.Context, actor, id, key, action string, expected int64, input CatalogObjectV2, available bool) (CommerceMutationV2, error) {
	return s.commerceMutateV2(ctx, actor, key, "intervention."+action+":"+id, input, func(tx *sql.Tx) (CommerceMutationV2, error) {
		result := CommerceMutationV2{}
		initial, e := commerceCaseTxV2(ctx, tx, id, false)
		if e != nil {
			return result, e
		}
		d, e := commerceRecordTxV2(ctx, tx, initial.ResourceID, "", true)
		if e != nil {
			return result, e
		}
		c, e := commerceCaseTxV2(ctx, tx, id, true)
		if e != nil {
			return result, e
		}
		if c.Version != expected || c.Version >= 2147483647 {
			return result, catalogVersion()
		}
		if d.PendingOperationID != "" {
			return result, catalogError(409, "OPERATION_IN_PROGRESS", "原资金步骤尚未确认，请先查询原结果。")
		}
		admin, e := commerceCaseAdminTxV2(ctx, tx, actor)
		if e != nil {
			return result, e
		}
		participant := actor == c.ApplicantID || actor == c.RespondentID
		adminAction := action == "assign" || action == "request-evidence" || action == "resolve"
		if adminAction && !admin {
			return result, catalogDenied()
		}
		if !adminAction && !participant && !(admin && actor == c.AssignedAdminID) {
			return result, catalogNotFound()
		}
		if c.State == "WITHDRAWN" || c.State == "RESOLVED" {
			return result, catalogError(409, "INVALID_STATE_TRANSITION", "案件已经结束。")
		}
		now := time.Now().UTC().Truncate(time.Microsecond)
		var opID string
		switch action {
		case "assign":
			if c.AssignedAdminID != "" && c.AssignedAdminID != actor {
				return result, catalogError(409, "CASE_ALREADY_ASSIGNED", "案件已由其他管理员处理。")
			}
			c.AssignedAdminID = actor
			c.State = "IN_REVIEW"
		case "request-evidence":
			if c.AssignedAdminID != actor {
				return result, catalogDenied()
			}
			if c.State == "RESOLVING" {
				return result, catalogError(409, "INVALID_STATE_TRANSITION", "已进入资金处理，不能再变更证据要求。")
			}
			c.State = "WAITING_EVIDENCE"
			c.Resolution = catalogString(input, "reason")
		case "evidence":
			if c.State == "RESOLVING" {
				return result, catalogError(409, "INVALID_STATE_TRANSITION", "资金处理期间不能修改证据。")
			}
			if len(c.EvidenceEntries) >= 100 {
				return result, catalogError(422, "EVIDENCE_LIMIT", "案件补充记录已达到上限。")
			}
			old := catalogIDs(c.Body, "evidenceAssetIds")
			ids := append([]string(nil), old...)
			seen := map[string]bool{}
			for _, x := range old {
				seen[x] = true
			}
			for _, x := range catalogIDs(input, "evidenceAssetIds") {
				if !seen[x] {
					ids = append(ids, x)
					seen[x] = true
				}
			}
			if len(ids) > 10 {
				return result, catalogError(422, "EVIDENCE_LIMIT", "同一案件最多保留十份图片证据。")
			}
			if e = s.commerceBindEvidenceV2(ctx, tx, actor, "INTERVENTION_EVIDENCE", id, ids); e != nil {
				return result, e
			}
			messages, e := commerceRelatedMessagesV2(ctx, tx, actor, catalogIDs(input, "relatedMessageIds"))
			if e != nil {
				return result, e
			}
			u, e := commerceUserTxV2(ctx, tx, actor, true)
			if e != nil {
				return result, e
			}
			c.Body["evidenceAssetIds"] = ids
			c.EvidenceEntries = append(c.EvidenceEntries, CatalogObjectV2{"actorPlayerRef": u.PlayerRef, "description": input["description"], "evidenceAssetIds": catalogIDs(input, "evidenceAssetIds"), "relatedMessages": messages, "createdAt": now})
			if c.State == "WAITING_EVIDENCE" {
				c.State = "IN_REVIEW"
			}
		case "withdraw":
			if actor != c.ApplicantID || c.State == "RESOLVING" {
				return result, catalogDenied()
			}
			c.State = "WITHDRAWN"
			c.Resolution = "申请人撤回：" + catalogString(input, "reason")
			if c.FundsHeld {
				d.FundsState = "HELD"
				commerceResumeV2(&d, now)
			}
			c.FundsHeld = false
		case "resolve":
			if c.AssignedAdminID != actor {
				return result, catalogDenied()
			}
			decision := catalogString(input, "decision")
			c.Decision = decision
			c.Resolution = catalogString(input, "reason")
			if decision == "CONTINUE_FULFILLMENT" || decision == "NO_ACTION" {
				if decision == "CONTINUE_FULFILLMENT" && !c.FundsHeld {
					return result, catalogError(409, "FUNDS_ALREADY_SETTLED", "已结算的交易不能恢复为托管履约。")
				}
				if c.FundsHeld {
					d.FundsState = "HELD"
					commerceResumeV2(&d, now)
				}
				c.FundsHeld = false
				c.State = "RESOLVED"
			} else if decision == "REQUIRE_MANUAL_RECOVERY" {
				if c.FundsHeld {
					return result, catalogError(409, "HELD_FUNDS_REQUIRE_DISPOSITION", "仍有平台在管冻结款，请选择退款、结算或继续履约。")
				}
				c.State = "RESOLVED"
			} else {
				if !catalogEnum(decision, "FULL_REFUND", "PARTIAL_REFUND", "RELEASE_TO_PAYEE") {
					return result, catalogInvalid()
				}
				if !c.FundsHeld || d.FundsState != "INTERVENTION_HOLD" {
					return result, catalogError(409, "FUNDS_ALREADY_SETTLED", "现有资金不能自动追回，请使用人工恢复处理。")
				}
				if e = commerceAvailableV2(available); e != nil {
					return result, e
				}
				total, _ := commerceAmountV2(d.Amount)
				target := new(big.Int)
				if decision == "FULL_REFUND" {
					target.Set(total)
				} else if decision == "PARTIAL_REFUND" {
					var e error
					target, e = commerceAmountV2(catalogString(input, "refundAmount"))
					if e != nil || target.Sign() <= 0 || target.Cmp(total) >= 0 {
						return result, catalogError(422, "INVALID_REFUND_AMOUNT", "部分退款必须大于零且小于原金额。")
					}
				}
				alreadyRefunded, _ := commerceAmountV2(d.RefundedAmount)
				alreadySettled, _ := commerceAmountV2(d.SettledAmount)
				targetSettled := new(big.Int).Sub(new(big.Int).Set(total), target)
				if target.Cmp(alreadyRefunded) < 0 || targetSettled.Cmp(alreadySettled) < 0 {
					return result, catalogError(409, "FUNDS_ALREADY_MOVED", "判决不能撤销已经证实转出的资金。")
				}
				refundDelta := new(big.Int).Sub(new(big.Int).Set(target), alreadyRefunded)
				settleDelta := new(big.Int).Sub(new(big.Int).Set(targetSettled), alreadySettled)
				steps := []CommerceStepV2{}
				if refundDelta.Sign() > 0 {
					steps = append(steps, commerceMoneyStepV2(d, "wallet.escrow.refund", catalogMoneyString(refundDelta)))
				}
				if settleDelta.Sign() > 0 {
					if d.PayeeUUID == "" {
						return result, catalogError(409, "BENEFICIARY_UNCONFIRMED", "原收款人尚未确认。")
					}
					steps = append(steps, commerceMoneyStepV2(d, "wallet.escrow.settle", catalogMoneyString(settleDelta)))
				}
				c.TargetRefundAmount = catalogMoneyString(target)
				if len(steps) == 0 {
					if e = s.commerceCompleteOperationV2(ctx, tx, &d, CommerceOperationV2{Action: "case-resolution"}, now); e != nil {
						return result, e
					}
					c.State = "RESOLVED"
					c.FundsHeld = false
				} else {
					op, e := commerceNewOperationV2(ctx, tx, &d, actor, key, "REFUND", "case-resolution", d.Amount, steps, false, now)
					if e != nil {
						return result, e
					}
					opID = op.ID
					c.State = "RESOLVING"
				}
			}
		default:
			return result, catalogInvalid()
		}
		c.Version++
		c.UpdatedAt = now
		commerceTouchV2(&d, now)
		if e = commerceSaveCaseV2(ctx, tx, &c, false); e != nil {
			return result, e
		}
		if e = commerceSaveV2(ctx, tx, &d, false); e != nil {
			return result, e
		}
		if e = commerceEventV2(ctx, tx, d, actor, "intervention."+action, "平台案件状态已更新，请查看原案件详情。", map[string]string{"caseId": c.ID, "action": action}); e != nil {
			return result, e
		}
		return CommerceMutationV2{ResourceID: c.ID, Kind: "INTERVENTION", OperationID: opID}, nil
	})
}
