package store

import (
	"context"
	"encoding/json"
)

func (s *Store) QueuedCoreOperations(ctx context.Context, node string) (out []CoreOperation, err error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT operation_id FROM core_operations WHERE node_id=? AND state='QUEUED' AND expires_at>UTC_TIMESTAMP(6) ORDER BY created_at LIMIT 16", node)
	if err != nil {
		return nil, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return
	}
	for _, id := range ids {
		var op CoreOperation
		op, err = s.CoreOperation(ctx, id)
		if err != nil {
			return
		}
		out = append(out, op)
	}
	return
}
func (s *Store) ExpireCoreOperations(ctx context.Context) error {
	// QUEUED has never crossed the send boundary. Expiry here is a definite
	// rejection; sent/processing expiry remains unknown and requires a receipt.
	if _, err := s.DB.ExecContext(ctx, `UPDATE core_operations SET state='FAILED',result=JSON_OBJECT('operationId',operation_id,'status','FAILED','error',JSON_OBJECT('code','COMMAND_EXPIRED','message','请求发送前已过期，未执行。')),updated_at=UTC_TIMESTAMP(6) WHERE state='QUEUED' AND expires_at<=UTC_TIMESTAMP(6) LIMIT 1000`); err != nil {
		return err
	}
	if _, err := s.DB.ExecContext(ctx, "UPDATE core_operations SET state='UNKNOWN',updated_at=UTC_TIMESTAMP(6) WHERE state IN ('SENT','PROCESSING') AND expires_at<=UTC_TIMESTAMP(6) LIMIT 1000"); err != nil {
		return err
	}
	if _, err := s.DB.ExecContext(ctx, "UPDATE core_operations SET payload='{}' WHERE command_type='verification.deliver' AND expires_at<=UTC_TIMESTAMP(6) AND payload<>'{}' LIMIT 1000"); err != nil {
		return err
	}
	_, err := s.DB.ExecContext(ctx, "DELETE FROM game_verifications WHERE expires_at<DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 1 DAY) LIMIT 1000")
	return err
}

// CoreItemListing adds mutable catalog state without changing immutable version
// metadata or the fingerprints of already published events.
type CoreItemListing struct {
	ItemVersion
	CatalogVersion int64 `json:"catalogVersion"`
	LatestRevision int64 `json:"latestRevision"`
	Archived       bool  `json:"archived"`
}

func (s *Store) CoreItems(ctx context.Context, ref string, rev int64, limit int) (out []CoreItemListing, err error) {
	return s.CoreItemsSearchV209(ctx, ref, rev, limit, "", "")
}
func (s *Store) CoreItemsSearchV209(ctx context.Context, ref string, rev int64, limit int, query, domain string) (out []CoreItemListing, err error) {
	q := `SELECT v.metadata,COALESCE(h.catalog_version,0),COALESCE(h.latest_revision,v.revision),COALESCE(h.archived,false) FROM item_versions v LEFT JOIN core_catalog_heads h ON h.item_ref=v.item_ref WHERE (v.item_ref>? OR(v.item_ref=? AND v.revision>?))`
	args := []any{ref, ref, rev}
	if query != "" {
		q += " AND (INSTR(LOWER(v.item_ref),LOWER(?))>0 OR INSTR(LOWER(JSON_UNQUOTE(JSON_EXTRACT(v.metadata,'$.displayName'))),LOWER(?))>0)"
		args = append(args, query, query)
	}
	if domain != "" {
		q += " AND JSON_UNQUOTE(JSON_EXTRACT(v.metadata,'$.inventoryDomain'))=?"
		args = append(args, domain)
	}
	args = append(args, limit)
	rows, err := s.DB.QueryContext(ctx, q+" ORDER BY v.item_ref,v.revision LIMIT ?", args...)
	if err != nil {
		return
	}
	defer rows.Close()
	out = []CoreItemListing{}
	for rows.Next() {
		var data []byte
		var v CoreItemListing
		if err = rows.Scan(&data, &v.CatalogVersion, &v.LatestRevision, &v.Archived); err != nil {
			return
		}
		if err = json.Unmarshal(data, &v.ItemVersion); err != nil {
			return
		}
		out = append(out, v)
	}
	err = rows.Err()
	return
}
