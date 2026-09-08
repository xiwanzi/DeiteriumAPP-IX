package store

import (
	"context"
	"encoding/json"
)

type CoreCatalog struct {
	ItemRef        string `json:"itemRef"`
	CatalogVersion int64  `json:"catalogVersion"`
	LatestRevision int64  `json:"latestRevision"`
	Archived       bool   `json:"archived"`
}

func (s *Store) CoreCatalogEvent(ctx context.Context, node, event string, value CoreCatalog) (seq int64, replay bool, err error) {
	payload, _ := json.Marshal(value)
	fingerprint := Digest(append([]byte("item.catalog.updated:"), payload...))
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, "INSERT INTO core_events(source_id,event_id,event_type,fingerprint,payload,created_at) VALUES (?,?,'item.catalog.updated',?,?,UTC_TIMESTAMP(6))", node, event, fingerprint, payload)
	if duplicate(err) {
		var old string
		err = tx.QueryRowContext(ctx, "SELECT sequence_id,fingerprint FROM core_events WHERE source_id=? AND event_id=?", node, event).Scan(&seq, &old)
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
	_, err = tx.ExecContext(ctx, "INSERT IGNORE INTO core_catalog_heads VALUES (?,0,0,false,'')", value.ItemRef)
	if err != nil {
		return
	}
	var current int64
	var previous string
	if err = tx.QueryRowContext(ctx, "SELECT catalog_version,fingerprint FROM core_catalog_heads WHERE item_ref=? FOR UPDATE", value.ItemRef).Scan(&current, &previous); err != nil {
		return
	}
	if current == value.CatalogVersion && previous != fingerprint {
		return 0, false, ErrConflict
	}
	if current < value.CatalogVersion {
		_, err = tx.ExecContext(ctx, "UPDATE core_catalog_heads SET catalog_version=?,latest_revision=?,archived=?,fingerprint=? WHERE item_ref=?", value.CatalogVersion, value.LatestRevision, value.Archived, fingerprint, value.ItemRef)
		if err != nil {
			return
		}
	}
	err = tx.Commit()
	return
}
func (s *Store) CoreItem(ctx context.Context, ref string, revision int64) (item ItemVersion, archived bool, err error) {
	var metadata []byte
	err = s.DB.QueryRowContext(ctx, `SELECT v.metadata,COALESCE(h.archived,false) FROM item_versions v LEFT JOIN core_catalog_heads h ON h.item_ref=v.item_ref WHERE v.item_ref=? AND v.revision=?`, ref, revision).Scan(&metadata, &archived)
	if err == nil {
		err = json.Unmarshal(metadata, &item)
	}
	return
}
func (s *Store) ManualDelivery(ctx context.Context, id string) (order, sha string, err error) {
	var snapshot []byte
	err = s.DB.QueryRowContext(ctx, "SELECT snapshot FROM core_manual_deliveries WHERE delivery_id=?", id).Scan(&snapshot)
	if err != nil {
		return
	}
	var data struct {
		OrderID string `json:"orderId"`
	}
	if err = json.Unmarshal(snapshot, &data); err != nil {
		return
	}
	return data.OrderID, Digest(snapshot), nil
}
func (s *Store) SaveManualDelivery(ctx context.Context, id, actor, operation string, snapshot []byte) error {
	_, err := s.DB.ExecContext(ctx, "INSERT INTO core_manual_deliveries VALUES (?,?,?,?,UTC_TIMESTAMP(6))", id, actor, operation, snapshot)
	if duplicate(err) {
		var old []byte
		err = s.DB.QueryRowContext(ctx, "SELECT snapshot FROM core_manual_deliveries WHERE delivery_id=? AND operation_id=?", id, operation).Scan(&old)
		if err == nil && string(old) != string(snapshot) {
			return ErrConflict
		}
	}
	return err
}
