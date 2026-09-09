package store

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

type auditQuerierV204 interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func requireAuditV204(ctx context.Context, q auditQuerierV204, actor string) error {
	var allowed bool
	err := q.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM identity_permissions p JOIN identities i ON i.id=p.user_id WHERE p.user_id=? AND i.status='active' AND p.permission IN ('audit.read','platform.admin'))`, actor).Scan(&allowed)
	if err != nil {
		return err
	}
	if !allowed {
		return catalogDenied()
	}
	return nil
}

type AdminPlayerV204 struct {
	PlayerRef  string `json:"playerRef"`
	UUID       string `json:"uuid"`
	GameID     string `json:"gameId"`
	QQ         string `json:"qq"`
	Registered bool   `json:"registered"`
	Status     string `json:"status"`
}

func (s *Store) AdminPlayersV204(ctx context.Context, actor, query, after string, limit int) ([]AdminPlayerV204, error) {
	if err := requireAuditV204(ctx, s.DB, actor); err != nil {
		return nil, err
	}
	if len(query) > 100 || len(after) > 36 || limit < 1 || limit > 100 {
		return nil, ErrSocialInvalid
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT player_ref,uuid,game_id,qq,registered,status FROM (
 SELECT player_ref,server_uuid AS uuid,game_id,qq,TRUE AS registered,status FROM identities
 UNION ALL SELECT p.player_ref,p.player_uuid,p.game_id,'',FALSE,'game_only' FROM core_player_directory p WHERE NOT EXISTS(SELECT 1 FROM identities i WHERE i.server_uuid=p.player_uuid)
 ) players WHERE uuid>? AND (?='' OR LOCATE(?,LOWER(CONCAT(game_id,' ',qq,' ',uuid,' ',player_ref)))>0) ORDER BY uuid LIMIT ?`, after, query, strings.ToLower(query), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AdminPlayerV204{}
	for rows.Next() {
		var p AdminPlayerV204
		if err = rows.Scan(&p.PlayerRef, &p.UUID, &p.GameID, &p.QQ, &p.Registered, &p.Status); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

type AdminCommerceFilterV204 struct {
	PlayerRef      string    `json:"playerRef"`
	Kind           string    `json:"kind"`
	Channel        string    `json:"channel"`
	Status         string    `json:"status"`
	Query          string    `json:"q"`
	BeforeTime     time.Time `json:"beforeTime"`
	BeforeSequence int64     `json:"beforeSequence"`
}

func (f AdminCommerceFilterV204) Valid() bool {
	return len(f.PlayerRef) <= 64 && len(f.Kind) <= 20 && len(f.Channel) <= 32 && len(f.Status) <= 40 && len(f.Query) <= 100 && f.BeforeSequence >= 0 && (f.BeforeSequence == 0 || !f.BeforeTime.IsZero())
}
func adminTimePageV204(query string, args []any, f AdminCommerceFilterV204, limit int) (string, []any) {
	if f.BeforeSequence > 0 {
		query += ` AND (created_at<? OR (created_at=? AND sequence_id<?))`
		args = append(args, f.BeforeTime.UTC(), f.BeforeTime.UTC(), f.BeforeSequence)
	}
	query += ` ORDER BY created_at DESC,sequence_id DESC LIMIT ?`
	return query, append(args, limit)
}
func (s *Store) AdminOrdersV204(ctx context.Context, actor string, f AdminCommerceFilterV204, limit int) ([]CommerceRecordV2, error) {
	if err := requireAuditV204(ctx, s.DB, actor); err != nil {
		return nil, err
	}
	if !f.Valid() || limit < 1 || limit > 100 {
		return nil, ErrSocialInvalid
	}
	query := commerceSelectV2 + ` WHERE resource_kind='ORDER'`
	args := []any{}
	if f.PlayerRef != "" {
		query += ` AND (owner_uuid=(SELECT server_uuid FROM identities WHERE player_ref=?) OR payee_uuid=(SELECT server_uuid FROM identities WHERE player_ref=?) OR store_id IN (SELECT resource_id FROM catalog_records_v2 WHERE kind='store' AND owner_id=(SELECT id FROM identities WHERE player_ref=?)) OR store_id IN (SELECT store_id FROM catalog_members_v2 WHERE user_id=(SELECT id FROM identities WHERE player_ref=?) AND active=TRUE))`
		args = append(args, f.PlayerRef, f.PlayerRef, f.PlayerRef, f.PlayerRef)
	}
	if f.Channel != "" {
		query += ` AND channel=?`
		args = append(args, f.Channel)
	}
	if f.Status != "" {
		query += ` AND state=?`
		args = append(args, f.Status)
	}
	if f.Query != "" {
		query += ` AND (LOCATE(?,resource_id)>0 OR LOCATE(?,LOWER(body))>0)`
		args = append(args, f.Query, strings.ToLower(f.Query))
	}
	query, args = adminTimePageV204(query, args, f, limit)
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CommerceRecordV2{}
	for rows.Next() {
		d, err := commerceScanV2(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
func (s *Store) AdminProductsV204(ctx context.Context, actor string, f AdminCommerceFilterV204, limit int) ([]CatalogRecordV2, error) {
	if err := requireAuditV204(ctx, s.DB, actor); err != nil {
		return nil, err
	}
	if !f.Valid() || limit < 1 || limit > 100 {
		return nil, ErrSocialInvalid
	}
	query := catalogSelect + ` WHERE kind IN ('listing','product')`
	args := []any{}
	if f.Kind != "" {
		query += ` AND kind=?`
		args = append(args, f.Kind)
	}
	if f.PlayerRef != "" {
		query += ` AND (owner_id=(SELECT id FROM identities WHERE player_ref=?) OR store_id IN (SELECT resource_id FROM catalog_records_v2 stores WHERE stores.kind='store' AND stores.owner_id=(SELECT id FROM identities WHERE player_ref=?)) OR store_id IN (SELECT store_id FROM catalog_members_v2 WHERE user_id=(SELECT id FROM identities WHERE player_ref=?) AND active=TRUE))`
		args = append(args, f.PlayerRef, f.PlayerRef, f.PlayerRef)
	}
	if f.Status != "" {
		query += ` AND state=?`
		args = append(args, f.Status)
	}
	if f.Query != "" {
		query += ` AND (LOCATE(?,resource_id)>0 OR LOCATE(?,LOWER(title))>0)`
		args = append(args, f.Query, strings.ToLower(f.Query))
	}
	query, args = adminTimePageV204(query, args, f, limit)
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CatalogRecordV2{}
	for rows.Next() {
		d, err := catalogScan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
func (s *Store) AdminProductV204(ctx context.Context, actor, id string) (map[string]any, error) {
	if err := requireAuditV204(ctx, s.DB, actor); err != nil {
		return nil, err
	}
	d, err := catalogScan(s.DB.QueryRowContext(ctx, catalogSelect+` WHERE resource_id=? AND kind IN ('listing','product')`, id))
	if err != nil {
		return nil, err
	}
	return s.AdminProductViewV204(ctx, actor, d)
}
func (s *Store) AdminProductViewV204(ctx context.Context, actor string, d CatalogRecordV2) (map[string]any, error) {
	if err := requireAuditV204(ctx, s.DB, actor); err != nil {
		return nil, err
	}
	view, err := s.CatalogViewV2(ctx, actor, d, true)
	if err != nil {
		return nil, err
	}
	view["kind"] = d.Kind
	view["createdAt"] = d.CreatedAt
	view["state"] = d.State
	view["stock"] = d.Stock
	view["readOnly"] = true
	view["canHideRecord"] = false
	view["hiddenFromHistory"] = false
	return view, nil
}
