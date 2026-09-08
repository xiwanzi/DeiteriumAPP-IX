package store

import (
	"context"
	"database/sql"
	"encoding/json"
)

type MailReceipt struct {
	DeliveryID       string   `json:"deliveryId"`
	MailID           string   `json:"mailId"`
	OrderID          string   `json:"orderId"`
	Source           string   `json:"source"`
	SnapshotSHA256   string   `json:"snapshotSha256"`
	RecipientUUID    string   `json:"recipientUuid"`
	AllowedServerIDs []string `json:"allowedServerIds"`
	InventoryDomain  string   `json:"inventoryDomain"`
	Status           string   `json:"status"`
	Revision         int64    `json:"revision"`
	Replayed         bool     `json:"replayed"`
}
type MailEvent struct {
	EventID            string      `json:"eventId"`
	Type               string      `json:"type"`
	ClusterID          string      `json:"clusterId"`
	Receipt            MailReceipt `json:"receipt"`
	OccurredAt         int64       `json:"occurredAt"`
	ServerID           string      `json:"serverId"`
	ClaimOperationID   string      `json:"claimOperationId"`
	PlayerSessionEpoch string      `json:"playerSessionEpoch"`
	SaveReceipt        string      `json:"saveReceipt"`
}

func (s *Store) MailEvent(ctx context.Context, event MailEvent) (seq int64, replayed bool, err error) {
	payload, _ := json.Marshal(event)
	fingerprint := Digest(payload)
	source := "mail:" + event.ClusterID
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, "INSERT INTO core_events(source_id,event_id,event_type,fingerprint,payload,created_at) VALUES (?,?,?,?,?,UTC_TIMESTAMP(6))", source, event.EventID, event.Type, fingerprint, payload)
	if duplicate(err) {
		var old string
		err = tx.QueryRowContext(ctx, "SELECT sequence_id,fingerprint FROM core_events WHERE source_id=? AND event_id=?", source, event.EventID).Scan(&seq, &old)
		if err == nil && old != fingerprint {
			err = ErrConflict
		}
		return seq, true, err
	}
	if err != nil {
		return
	}
	seq, err = result.LastInsertId()
	if err != nil {
		return
	}
	if err = mergeReceipt(ctx, tx, event.ClusterID, event.Receipt); err != nil {
		return
	}
	err = tx.Commit()
	return
}
func mergeReceipt(ctx context.Context, tx *sql.Tx, cluster string, receipt MailReceipt) error {
	payload, _ := json.Marshal(receipt)
	result, err := tx.ExecContext(ctx, `INSERT IGNORE INTO core_mail_receipts (delivery_id,order_id,mail_cluster,snapshot_sha256,recipient_uuid,status,revision,receipt,updated_at) VALUES (?,?,?,?,?,?,?,?,UTC_TIMESTAMP(6))`, receipt.DeliveryID, receipt.OrderID, cluster, receipt.SnapshotSHA256, receipt.RecipientUUID, receipt.Status, receipt.Revision, payload)
	if err != nil {
		return err
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if inserted == 1 {
		return nil
	}
	var order, oldCluster, hash, recipient, status string
	var revision int64
	if err = tx.QueryRowContext(ctx, "SELECT order_id,mail_cluster,snapshot_sha256,recipient_uuid,status,revision FROM core_mail_receipts WHERE delivery_id=? FOR UPDATE", receipt.DeliveryID).Scan(&order, &oldCluster, &hash, &recipient, &status, &revision); err != nil {
		return err
	}
	if order != receipt.OrderID || oldCluster != cluster || hash != receipt.SnapshotSHA256 || recipient != receipt.RecipientUUID {
		return ErrConflict
	}
	if revision > receipt.Revision {
		return nil
	}
	if revision == receipt.Revision {
		if status != receipt.Status {
			return ErrConflict
		}
		return nil
	}
	if (status == "CLAIMED" || status == "REVOKED") && status != receipt.Status {
		return ErrConflict
	}
	_, err = tx.ExecContext(ctx, "UPDATE core_mail_receipts SET status=?,revision=?,receipt=?,updated_at=UTC_TIMESTAMP(6) WHERE delivery_id=?", receipt.Status, receipt.Revision, payload, receipt.DeliveryID)
	return err
}
func (s *Store) RememberMailReceipt(ctx context.Context, cluster string, receipt MailReceipt) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = mergeReceipt(ctx, tx, cluster, receipt); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Store) MailReceipt(ctx context.Context, id string) (r MailReceipt, err error) {
	var raw []byte
	err = s.DB.QueryRowContext(ctx, "SELECT receipt FROM core_mail_receipts WHERE delivery_id=?", id).Scan(&raw)
	if err == nil {
		err = json.Unmarshal(raw, &r)
	}
	return
}
