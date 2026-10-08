package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// All money/mail evidence has passed the ordinary order validators before this
// function runs. A group coordinates existing per-order escrows, never invents
// a shared escrow balance or allows one store to refund another store's money.
func (s *Store) applyCheckoutStepV221(ctx context.Context, tx *sql.Tx, root, d CommerceRecordV2, op CommerceOperationV2, status string, money commerceMoneyProofV2, changed bool, now time.Time) (CommerceOperationV2, error) {
	step := op.Steps[op.StepIndex]
	if root.ID == d.ID {
		root = d
	}
	if status == "COMPLETED" {
		if step.Command == "wallet.escrow.reserve" || step.Command == "wallet.escrow.refund" {
			d.SettledAmount, d.RefundedAmount = money.Settled, money.Refunded
			if money.PoolUUID != "" {
				d.Body["escrowPoolUuid"] = money.PoolUUID
			}
			if d.PayeeUUID == "" {
				d.PayeeUUID = money.PayeeUUID
			}
		}
		switch step.Command {
		case "wallet.escrow.reserve":
			d.FundsState = "HELD"
			if _, e := tx.ExecContext(ctx, "UPDATE commerce_stock_holds_v2 SET state='PURCHASED' WHERE resource_id=? AND state='RESERVED'", d.ID); e != nil {
				return op, e
			}
		case "wallet.escrow.refund":
			d.State, d.FundsState = "REFUNDED", "REFUNDED"
			d.Body["mailboxState"] = "CANCELLED_BEFORE_CREATE"
			// Group completion clears PendingOperationID before returning coupons.
		case "mailbox.create":
			d.FundsState = "HELD"
			if catalogString(d.Body, "mailboxState") != "CLAIMED" {
				d.State = "AWAITING_CLAIM"
			}
		}
		op.StepIndex++
		op.ErrorCode = ""
	} else if status == "FAILED" {
		switch step.Command {
		case "wallet.escrow.reserve":
			root.Body["checkoutFailure"] = op.ErrorCode
			d.State, d.FundsState = "CANCELLED", "UNPAID"
			d.Body["mailboxState"] = "CANCELLED_BEFORE_CREATE"
			// Preserve the original steps/IDs for recovery and audit, but never
			// dispatch any other reserve or mail after a proven payment failure.
			for i := op.StepIndex + 1; i < len(op.Steps); i++ {
				op.Steps[i].State = "SKIPPED"
			}
			op.StepIndex = len(op.Steps)
		case "wallet.escrow.refund":
			root.Body["checkoutRefundFailure"] = op.ErrorCode
			d.State, d.FundsState = "PAYMENT_PROCESSING", "HELD"
			op.StepIndex++
		case "mailbox.create":
			// Payment already succeeded for the whole group. Preserve the held
			// order for the ordinary delivery retry/refund workflow and continue
			// delivering the other stores. Do not charge this order again.
			d.FundsState = "HELD"
			if catalogString(d.Body, "mailboxState") != "CLAIMED" {
				d.State = "PAYMENT_PROCESSING"
			}
			d.Body["checkoutDeliveryError"] = op.ErrorCode
			op.StepIndex++
		default:
			return op, catalogInvalid()
		}
	} else {
		op.State = status
		if status == "UNKNOWN" {
			d.FundsState = "UNKNOWN"
		}
	}
	if changed {
		commerceTouchV2(&d, now)
	}
	if root.ID == d.ID {
		// d and root share the decoded Body map, including the failure marker.
		root = d
	}
	if e := commerceSaveV2(ctx, tx, &d, false); e != nil {
		return op, e
	}
	if root.ID != d.ID {
		if e := commerceSaveV2(ctx, tx, &root, false); e != nil {
			return op, e
		}
	}
	if status == "FAILED" && step.Command == "wallet.escrow.reserve" {
		for _, id := range catalogIDs(root.Body, "checkoutOrderIds") {
			child, e := commerceRecordTxV2(ctx, tx, id, "ORDER", true)
			if e != nil {
				return op, e
			}
			if child.FundsState == "HELD" && child.Amount != "0.00" {
				refund := commerceMoneyStepV2(child, "wallet.escrow.refund", child.Amount)
				refund.ResourceID, refund.ClientKey, refund.State = id, fmt.Sprintf("commerce:%s:rollback:%s", op.ID, id), "PREPARED"
				op.Steps = append(op.Steps, refund)
				child.FundsState = "REFUNDING"
			} else {
				child.State, child.FundsState = "CANCELLED", "UNPAID"
				child.Body["mailboxState"] = "CANCELLED_BEFORE_CREATE"
				if e = commerceReleaseStockV2(ctx, tx, child); e != nil {
					return op, e
				}
			}
			commerceTouchV2(&child, now)
			if e = commerceSaveV2(ctx, tx, &child, false); e != nil {
				return op, e
			}
		}
	}
	if status == "COMPLETED" || status == "FAILED" {
		op.State = "PROCESSING"
		if op.StepIndex < len(op.Steps) && op.Steps[op.StepIndex].Command == "mailbox.create" && step.Command == "wallet.escrow.reserve" {
			// Consume the purchased cart quantities once all reserves are proven,
			// even when a later mail temporarily fails. This avoids rebuying them.
			for _, id := range catalogIDs(root.Body, "checkoutOrderIds") {
				child, e := commerceRecordTxV2(ctx, tx, id, "ORDER", true)
				if e != nil {
					return op, e
				}
				child.FundsState = "HELD"
				if e = promotionCompletePurchaseV209(ctx, tx, &child, now); e != nil {
					return op, e
				}
				if e = commerceSaveV2(ctx, tx, &child, false); e != nil {
					return op, e
				}
			}
		}
		if op.StepIndex == len(op.Steps) {
			failed := catalogString(root.Body, "checkoutFailure") != ""
			op.State, op.ErrorCode = "COMPLETED", ""
			if failed {
				op.State, op.ErrorCode = "FAILED", "CHECKOUT_PAYMENT_REVERSED"
			}
			if catalogString(root.Body, "checkoutRefundFailure") != "" {
				op.ErrorCode = "CHECKOUT_REFUND_REQUIRES_ATTENTION"
			}
			for _, id := range catalogIDs(root.Body, "checkoutOrderIds") {
				child, e := commerceRecordTxV2(ctx, tx, id, "ORDER", true)
				if e != nil {
					return op, e
				}
				child.PendingOperationID = ""
				if failed && (child.FundsState == "REFUNDED" || child.FundsState == "UNPAID") {
					if e = commerceReleaseStockV2(ctx, tx, child); e != nil {
						return op, e
					}
				}
				commerceTouchV2(&child, now)
				if e = commerceSaveV2(ctx, tx, &child, false); e != nil {
					return op, e
				}
				if e = commerceEventV2(ctx, tx, child, op.ActorID, "checkout."+op.State, "合并付款状态已更新，可查看各店订单。", map[string]string{"operationId": op.ID}); e != nil {
					return op, e
				}
			}
		}
	}
	if e := commerceSaveOperationV2(ctx, tx, op, false); e != nil {
		return op, e
	}
	if changed {
		if e := commerceEventV2(ctx, tx, d, op.ActorID, "operation."+status, "合并付款进度已更新。", map[string]any{"operationId": op.ID, "command": step.Command}); e != nil {
			return op, e
		}
	}
	if e := tx.Commit(); e != nil {
		return op, e
	}
	return op, nil
}
