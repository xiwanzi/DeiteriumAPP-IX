package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

func newSupplyToken() string {
	h := ID("")
	return h[:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
}

func (s *Store) ClaimSupplyDelivery(ctx context.Context, id, node, uuid, epoch string) (SupplyDelivery, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return SupplyDelivery{}, err
	}
	defer tx.Rollback()
	d, err := scanSupplyDelivery(tx.QueryRowContext(ctx, "SELECT "+supplyDeliveryColumns+" FROM supply_deliveries WHERE id=? AND player_uuid=? AND reveal_until IS NULL FOR UPDATE", id, uuid))
	if err != nil {
		return d, socialMissing(err)
	}
	allowed := false
	for _, server := range d.Reward.Servers {
		allowed = allowed || server == node
	}
	if !allowed {
		return d, ErrUnauthorized
	}
	if d.State == "PREPARING" && d.NodeID == node && d.SessionEpoch == epoch {
		_, err = tx.ExecContext(ctx, "UPDATE supply_deliveries SET lease_until=DATE_ADD(UTC_TIMESTAMP(6),INTERVAL 30 SECOND) WHERE id=?", id)
	} else if d.State == "PENDING" || d.State == "PREPARING" {
		// Only the HTTP handler's current authenticated session can take over preparation.
		// The previous holder cannot enter ISSUING with its obsolete token.
		d.State = "PREPARING"
		d.NodeID = node
		d.SessionEpoch = epoch
		d.Token = newSupplyToken()
		_, err = tx.ExecContext(ctx, "UPDATE supply_deliveries SET state='PREPARING',node_id=?,session_epoch=?,token=?,lease_until=DATE_ADD(UTC_TIMESTAMP(6),INTERVAL 30 SECOND),updated_at=UTC_TIMESTAMP(6) WHERE id=?", node, epoch, d.Token, id)
	}
	if err != nil {
		return d, err
	}
	return d, tx.Commit()
}

func (s *Store) BeginSupplyDelivery(ctx context.Context, id, node, epoch, token, inventoryEpoch string) error {
	if inventoryEpoch == "" || len(inventoryEpoch) > 160 {
		return ErrSocialInvalid
	}
	r, err := s.DB.ExecContext(ctx, "UPDATE supply_deliveries SET state='ISSUING',inventory_epoch=?,updated_at=UTC_TIMESTAMP(6) WHERE id=? AND state='PREPARING' AND node_id=? AND session_epoch=? AND token=? AND lease_until>=UTC_TIMESTAMP(6)", inventoryEpoch, id, node, epoch, token)
	if err != nil {
		return err
	}
	n, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrConflict
	}
	return nil
}

type SupplyDeliveryResult struct {
	ID              string `json:"id"`
	Token           string `json:"token"`
	SessionEpoch    string `json:"sessionEpoch"`
	State           string `json:"state"`
	Message         string `json:"message"`
	SaveReceipt     string `json:"saveReceipt"`
	InventoryDomain string `json:"inventoryDomain"`
	PlayerUUID      string `json:"playerUuid"`
	CommittedAt     int64  `json:"committedAt"`
	ProofNode       string `json:"proofNode"`
	InventoryEpoch  string `json:"inventoryEpoch"`
}

func (s *Store) FinishSupplyDelivery(ctx context.Context, node string, in SupplyDeliveryResult) (SupplyDelivery, error) {
	if !catalogEnum(in.State, "INVENTORY", "MAIL_PENDING", "REVIEW", "PREPARING", "PENDING") || len(in.Message) > 512 {
		return SupplyDelivery{}, ErrSocialInvalid
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return SupplyDelivery{}, err
	}
	defer tx.Rollback()
	d, err := scanSupplyDelivery(tx.QueryRowContext(ctx, "SELECT "+supplyDeliveryColumns+" FROM supply_deliveries WHERE id=? FOR UPDATE", in.ID))
	if err != nil {
		return d, err
	}
	proofFromCompatibleNode := false
	if in.State == "INVENTORY" && in.ProofNode == d.NodeID {
		for _, server := range d.Reward.Servers {
			proofFromCompatibleNode = proofFromCompatibleNode || server == node
		}
	}
	if d.Token != in.Token || (d.NodeID != node && !proofFromCompatibleNode) || d.SessionEpoch != in.SessionEpoch || d.PlayerUUID != in.PlayerUUID {
		return d, ErrUnauthorized
	}
	if d.State == "INVENTORY" || d.State == "MAIL_PENDING" || d.State == "MAIL" {
		if d.State != in.State && !(d.State == "MAIL" && in.State == "MAIL_PENDING") {
			return d, ErrConflict
		}
		return d, nil
	}
	if in.State == "INVENTORY" {
		if !catalogEnum(d.State, "ISSUING", "REVIEW") || in.SaveReceipt == "" || len(in.SaveReceipt) > 1024 || in.InventoryDomain != d.Reward.InventoryDomain || in.InventoryEpoch != d.InventoryEpoch || in.CommittedAt <= 0 {
			return d, ErrSocialInvalid
		}
	} else if in.State == "MAIL_PENDING" {
		if d.State != "ISSUING" || !catalogEnum(in.Message, "INVENTORY_FULL", "PLAYER_OFFLINE", "PLAYER_SESSION_CHANGED") {
			return d, ErrConflict
		}
	} else if in.State == "PENDING" {
		if d.State != "ISSUING" || in.Message != "INVENTORY_NOT_TOUCHED" {
			return d, ErrConflict
		}
	} else if in.State == "PREPARING" {
		if d.State != "PREPARING" {
			return d, ErrConflict
		}
	} else if !catalogEnum(d.State, "ISSUING", "REVIEW") {
		return d, ErrConflict
	}
	raw, err := json.Marshal(in)
	if err != nil {
		return d, err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE supply_deliveries SET state=?,last_error=?,receipt_json=?,updated_at=UTC_TIMESTAMP(6) WHERE id=?", in.State, in.Message, raw, in.ID); err != nil {
		return d, err
	}
	d.State = in.State
	d.LastError = in.Message
	d.Receipt = raw
	return d, tx.Commit()
}

func (s *Store) RouteOfflineSupplyDelivery(ctx context.Context, id string) error {
	_, err := s.DB.ExecContext(ctx, "UPDATE supply_deliveries SET state='MAIL_PENDING',last_error='PLAYER_OFFLINE',updated_at=UTC_TIMESTAMP(6) WHERE id=? AND reveal_until IS NULL AND (state='PENDING' OR (state='PREPARING' AND lease_until<UTC_TIMESTAMP(6)))", id)
	return err
}

// Acknowledging the rendered summary releases only the owner's existing undelivered rewards.
func (s *Store) ReleaseSupplyReveal(ctx context.Context, orderID, uuid string) error {
	var exists int
	if err := s.DB.QueryRowContext(ctx, "SELECT 1 FROM supply_orders WHERE id=? AND player_uuid=?", orderID, uuid).Scan(&exists); err != nil {
		return socialMissing(err)
	}
	_, err := s.DB.ExecContext(ctx, "UPDATE supply_deliveries SET reveal_until=NULL,updated_at=UTC_TIMESTAMP(6) WHERE order_id=? AND player_uuid=? AND state IN ('PENDING','MAIL_PENDING') AND reveal_until IS NOT NULL", orderID, uuid)
	return err
}

// If the display was interrupted, move still-held rewards straight to mail.
// Eligibility and its destination change atomically, so an inventory claim cannot win after the fallback.
func (s *Store) MailExpiredSupplyReveals(ctx context.Context) error {
	_, err := s.DB.ExecContext(ctx, `
		UPDATE supply_deliveries SET state='MAIL_PENDING',reveal_until=NULL,last_error=CASE WHEN state='PENDING' THEN 'REVEAL_EXPIRED' ELSE last_error END,updated_at=UTC_TIMESTAMP(6)
		WHERE state IN ('PENDING','MAIL_PENDING') AND reveal_until<=UTC_TIMESTAMP(6) ORDER BY reveal_until,id LIMIT 32`)
	return err
}

func (s *Store) CompleteSupplyMail(ctx context.Context, id string, receipt any) error {
	raw, err := json.Marshal(receipt)
	if err != nil {
		return err
	}
	_, err = s.DB.ExecContext(ctx, "UPDATE supply_deliveries SET state='MAIL',receipt_json=?,last_error='',updated_at=UTC_TIMESTAMP(6) WHERE id=? AND state='MAIL_PENDING'", raw, id)
	return err
}

func (s *Store) SupplyDeliveryError(ctx context.Context, id, message string) error {
	if len(message) > 500 {
		message = "DELIVERY_PENDING"
	}
	_, err := s.DB.ExecContext(ctx, "UPDATE supply_deliveries SET last_error=?,updated_at=UTC_TIMESTAMP(6) WHERE id=? AND state NOT IN ('INVENTORY','MAIL')", message, id)
	return err
}

func (s *Store) RetrySupplyDelivery(ctx context.Context, actor, key, id, reason string) (json.RawMessage, error) {
	if !ValidSocialText(reason, 1, 200) {
		return nil, ErrSocialInvalid
	}
	return s.supplyMutate(ctx, actor, key, struct{ ID, Reason string }{id, reason}, "", "", func(tx *sql.Tx) (any, error) {
		if err := requirePlatformAdminTxV206(ctx, tx, actor); err != nil {
			return nil, err
		}
		d, err := scanSupplyDelivery(tx.QueryRowContext(ctx, "SELECT "+supplyDeliveryColumns+" FROM supply_deliveries WHERE id=? FOR UPDATE", id))
		if err != nil {
			return nil, err
		}
		if d.State == "ISSUING" || d.State == "REVIEW" {
			return nil, supplyFailure("DELIVERY_REQUIRES_PROOF", "背包写入结果未确认；只能由原保存回执核对，不能直接补寄")
		}
		if d.State == "INVENTORY" || d.State == "MAIL" {
			return nil, ErrConflict
		}
		if _, err = tx.ExecContext(ctx, "UPDATE supply_deliveries SET last_error='',updated_at=? WHERE id=?", time.Unix(0, 0), id); err != nil {
			return nil, err
		}
		_, err = tx.ExecContext(ctx, "INSERT INTO audit_events_next(actor_id,action,resource_id,created_at) VALUES(?,?,?,UTC_TIMESTAMP(6))", actor, "supply.delivery.retry", id)
		return d, err
	})
}
