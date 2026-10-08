package store

import (
	"context"
	"database/sql"
	"time"
)

func (s *Store) prepareStoreCheckoutV221(ctx context.Context, tx *sql.Tx, owner User, key, quoteID string, quote CatalogObjectV2, groups []any, nodes map[string]CatalogNodePolicyV2, now time.Time) (CommerceMutationV2, error) {
	if len(groups) < 2 || len(groups) > 100 {
		return CommerceMutationV2{}, catalogInvalid()
	}
	orders := make([]CommerceRecordV2, 0, len(groups))
	ids := []string{}
	amount := "0.00"
	seen := map[string]bool{}
	for _, value := range groups {
		group, ok := catalogObject(value)
		if !ok {
			return CommerceMutationV2{}, catalogInvalid()
		}
		store := catalogString(group, "storeId")
		if store == "" || seen[store] {
			return CommerceMutationV2{}, catalogInvalid()
		}
		seen[store] = true
		mutation, e := s.prepareOrderQuoteTxV2(ctx, tx, owner, key, "", group, "OFFICIAL_STORE", nodes, now, true)
		if e != nil {
			return CommerceMutationV2{}, e
		}
		d, e := commerceRecordTxV2(ctx, tx, mutation.ResourceID, "ORDER", true)
		if e != nil {
			return CommerceMutationV2{}, e
		}
		if d.StoreID != store {
			return CommerceMutationV2{}, catalogInvalid()
		}
		amount, e = commerceCreditSumV2(amount, d.Amount)
		if e != nil {
			return CommerceMutationV2{}, e
		}
		orders = append(orders, d)
		ids = append(ids, d.ID)
	}
	if amount != catalogString(quote, "totalAmount") {
		return CommerceMutationV2{}, catalogVersion()
	}
	if amount != "0.00" {
		if e := commerceWithinExecutionLimitV2(amount); e != nil {
			return CommerceMutationV2{}, e
		}
	}
	root := &orders[0]
	root.QuoteID = quoteID
	root.Body["checkoutOrderIds"] = ids
	steps := []CommerceStepV2{}
	for _, d := range orders {
		if d.Amount != "0.00" {
			step := commerceReserveStepV2(d)
			step.ResourceID = d.ID
			steps = append(steps, step)
		}
	}
	reserveCount := len(steps)
	for _, d := range orders {
		plan, _ := catalogObject(d.Body["mailboxPlan"])
		steps = append(steps, CommerceStepV2{ResourceID: d.ID, Command: "mailbox.create", Payload: map[string]any(plan)})
	}
	op, e := commerceNewOperationV2(ctx, tx, root, owner.ID, key, "STORE_PURCHASE", "checkout", amount, steps, false, now)
	if e != nil {
		return CommerceMutationV2{}, e
	}
	for i := range orders {
		d := &orders[i]
		d.PendingOperationID = op.ID
		d.Body["checkoutId"], d.Body["checkoutTotalAmount"] = root.ID, amount
		if reserveCount == 0 {
			d.FundsState = "HELD"
			if e = promotionCompletePurchaseV209(ctx, tx, d, now); e != nil {
				return CommerceMutationV2{}, e
			}
		}
		if e = commerceSaveV2(ctx, tx, d, false); e != nil {
			return CommerceMutationV2{}, e
		}
		if e = commerceEventV2(ctx, tx, *d, owner.ID, "order.payment.started", "合并付款已记录，正在核实各店订单。", map[string]string{"operationId": op.ID}); e != nil {
			return CommerceMutationV2{}, e
		}
	}
	return CommerceMutationV2{ResourceID: root.ID, Kind: "ORDER", OperationID: op.ID}, nil
}

// Only the buyer and platform administrators may see another store's order IDs.
// The normal merchant order view never includes these internal group fields.
func CheckoutOrderIDsV221(op CommerceOperationV2) []string {
	if op.Action != "checkout" {
		return []string{op.ResourceID}
	}
	seen := map[string]bool{}
	ids := []string{}
	for _, step := range op.Steps {
		if step.ResourceID != "" && !seen[step.ResourceID] {
			seen[step.ResourceID] = true
			ids = append(ids, step.ResourceID)
		}
	}
	return ids
}
