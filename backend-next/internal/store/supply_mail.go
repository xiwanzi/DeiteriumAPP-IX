package store

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
)

// SupplyMail is an immutable attachment list once every roll's inventory outcome is known.
type SupplyMail struct {
	ID      string
	Items   []SupplyDelivery
	Servers []string
}

func (s *Store) SupplyOrderMail(ctx context.Context, order string) ([]SupplyMail, error) {
	var grouped bool
	var count int
	if err := s.DB.QueryRowContext(ctx, "SELECT grouped_mail,quantity FROM supply_orders WHERE id=?", order).Scan(&grouped, &count); err != nil {
		return nil, err
	}
	rows, err := s.supplyDeliveries(ctx, "order_id=?", []any{order}, 11)
	if err != nil {
		return nil, err
	}
	if len(rows) != count || count < 1 || count > 10 {
		return nil, ErrConflict
	}
	var out []SupplyMail
	for _, d := range rows {
		if !grouped {
			// Older workers may already have created this mail and lost the receipt.
			if d.State == "MAIL_PENDING" {
				out = append(out, SupplyMail{d.ID, []SupplyDelivery{d}, d.Reward.Servers})
			}
			continue
		}
		switch d.State {
		case "INVENTORY":
			continue
		case "MAIL_PENDING", "MAIL":
		default:
			return nil, nil
		}
		placed := false
		for i := range out {
			group := &out[i]
			if group.Items[0].Reward.InventoryDomain != d.Reward.InventoryDomain {
				continue
			}
			var common []string
			for _, node := range group.Servers {
				if slices.Contains(d.Reward.Servers, node) {
					common = append(common, node)
				}
			}
			if len(common) == 0 {
				continue
			}
			group.Servers = common
			group.Items = append(group.Items, d)
			placed = true
			break
		}
		if !placed {
			out = append(out, SupplyMail{fmt.Sprintf("batch_%s_%02d", order, d.Ordinal), []SupplyDelivery{d}, d.Reward.Servers})
		}
	}
	return slices.DeleteFunc(out, func(batch SupplyMail) bool { return batch.Items[0].State == "MAIL" }), nil
}

func (s *Store) CompleteSupplyOrderMail(ctx context.Context, batch SupplyMail, receipt any) error {
	raw, err := json.Marshal(receipt)
	if err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, item := range batch.Items {
		r, err := tx.ExecContext(ctx, "UPDATE supply_deliveries SET state='MAIL',receipt_json=?,mail_operation=?,last_error='',updated_at=UTC_TIMESTAMP(6) WHERE id=? AND order_id=? AND player_uuid=? AND state='MAIL_PENDING'", raw, batch.ID, item.ID, item.OrderID, item.PlayerUUID)
		if err != nil {
			return err
		}
		n, err := r.RowsAffected()
		if err != nil {
			return err
		}
		if n != 1 {
			var previous string
			if err = tx.QueryRowContext(ctx, "SELECT COALESCE(mail_operation,'') FROM supply_deliveries WHERE id=? AND state='MAIL' FOR UPDATE", item.ID).Scan(&previous); err != nil {
				return err
			}
			if previous != batch.ID {
				return ErrConflict
			}
		}
	}
	return tx.Commit()
}
