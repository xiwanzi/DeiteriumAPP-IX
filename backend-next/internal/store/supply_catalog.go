package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/supply"
)

type SupplyPool struct {
	ID                string            `json:"id"`
	Version           int64             `json:"version"`
	PublishedRevision int64             `json:"publishedRevision"`
	Enabled           bool              `json:"enabled"`
	Definition        supply.Definition `json:"definition"`
	UpdatedAt         time.Time         `json:"updatedAt"`
}

type SupplyPoolChange struct {
	ClientRequestID string            `json:"clientRequestId"`
	ExpectedVersion int64             `json:"expectedVersion"`
	Action          string            `json:"action"`
	SourcePoolID    string            `json:"sourcePoolId"`
	Definition      supply.Definition `json:"definition"`
}

func scanSupplyPool(row interface{ Scan(...any) error }) (SupplyPool, error) {
	var p SupplyPool
	var raw string
	err := row.Scan(&p.ID, &p.Version, &p.PublishedRevision, &p.Enabled, &raw, &p.UpdatedAt)
	if err == nil {
		err = json.Unmarshal([]byte(raw), &p.Definition)
	}
	return p, err
}

func (s *Store) SupplyPools(ctx context.Context, admin bool) ([]SupplyPool, error) {
	return supplyPools(ctx, s.DB, admin)
}

func supplyPools(ctx context.Context, reader interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, admin bool) ([]SupplyPool, error) {
	query := `SELECT p.id,p.version,p.published_revision,p.enabled,p.draft_json,p.updated_at FROM supply_pools p ORDER BY p.updated_at DESC,p.id LIMIT 500`
	if !admin {
		query = `SELECT p.id,p.version,p.published_revision,p.enabled,r.definition_json,p.updated_at FROM supply_pools p JOIN supply_revisions r ON r.pool_id=p.id AND r.revision=p.published_revision WHERE p.enabled=TRUE AND (p.ends_at IS NULL OR p.ends_at>UTC_TIMESTAMP(6)) AND (COALESCE(p.visible_at,p.starts_at) IS NULL OR COALESCE(p.visible_at,p.starts_at)<=UTC_TIMESTAMP(6)) ORDER BY p.updated_at DESC,p.id`
	}
	rows, err := reader.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []SupplyPool{}
	now := time.Now().UTC()
	for rows.Next() {
		p, e := scanSupplyPool(rows)
		if e != nil {
			return nil, e
		}
		if admin || p.Definition.Visible(now) {
			items = append(items, p)
		}
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].Definition.Sort > items[j].Definition.Sort })
	return items, rows.Err()
}

func (s *Store) SupplyPoolRevision(ctx context.Context, id string, revision int64) (supply.Definition, error) {
	var raw string
	var d supply.Definition
	err := s.DB.QueryRowContext(ctx, "SELECT definition_json FROM supply_revisions WHERE pool_id=? AND revision=?", id, revision).Scan(&raw)
	if err == nil {
		err = json.Unmarshal([]byte(raw), &d)
	}
	return d, socialMissing(err)
}

func (s *Store) SupplyPoolHistory(ctx context.Context, id string) ([]map[string]any, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT revision,actor_id,created_at FROM supply_revisions WHERE pool_id=? ORDER BY revision DESC", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var rev int64
		var actor string
		var at time.Time
		if err = rows.Scan(&rev, &actor, &at); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{"revision": rev, "actor": actor, "createdAt": at})
	}
	return items, rows.Err()
}

func (s *Store) ChangeSupplyPool(ctx context.Context, actor, id string, in SupplyPoolChange) (json.RawMessage, error) {
	if !catalogEnum(in.Action, "save", "publish", "pause", "resume") || id != "" && !supply.ValidID(id) || id == "" && (in.Action != "save" || in.ExpectedVersion != 0) {
		return nil, ErrSocialInvalid
	}
	return s.supplyMutate(ctx, actor, in.ClientRequestID, struct {
		ID    string
		Input SupplyPoolChange
	}{id, in}, "", "", func(tx *sql.Tx) (any, error) {
		if err := requirePlatformAdminTxV206(ctx, tx, actor); err != nil {
			return nil, err
		}
		// This short catalog lock makes publication checks and simultaneous operator edits deterministic.
		var settingsVersion int64
		if err := tx.QueryRowContext(ctx, "SELECT version FROM supply_settings WHERE id=1 FOR UPDATE").Scan(&settingsVersion); err != nil {
			return nil, err
		}
		return s.changeSupplyPoolTx(ctx, tx, actor, id, "", in)
	})
}

// Both admin edits and editor publications hold supply_settings while changing the active catalog.
func (s *Store) changeSupplyPoolTx(ctx context.Context, tx *sql.Tx, actor, id, editorPoolID string, in SupplyPoolChange) (SupplyPool, error) {
	p := SupplyPool{ID: id}
	creating := id == ""
	if !creating {
		var err error
		p, err = scanSupplyPool(tx.QueryRowContext(ctx, `SELECT id,version,published_revision,enabled,draft_json,updated_at FROM supply_pools WHERE id=? FOR UPDATE`, id))
		if err != nil {
			return SupplyPool{}, socialMissing(err)
		}
		if p.Version != in.ExpectedVersion {
			return SupplyPool{}, ErrConflict
		}
	} else {
		p.ID = editorPoolID
		if p.ID == "" {
			p.ID = ID("sp_")
		}
	}
	if in.Action == "save" {
		d := in.Definition
		d.Rules = nil
		if creating && d.Kind == "EVENT" {
			if in.SourcePoolID != "" {
				var sourceRaw string
				if err := tx.QueryRowContext(ctx, `SELECT r.definition_json FROM supply_pools p JOIN supply_revisions r ON r.pool_id=p.id AND r.revision=p.published_revision WHERE p.id=? AND p.kind='EVENT'`, in.SourcePoolID).Scan(&sourceRaw); err != nil {
					return SupplyPool{}, socialMissing(err)
				}
				var source supply.Definition
				if err := json.Unmarshal([]byte(sourceRaw), &source); err != nil {
					return SupplyPool{}, err
				}
				d.SeriesID = source.SeriesID
				if d.EventGoldPPM != source.EventGoldPPM || d.EventPity != source.EventPity {
					return SupplyPool{}, supplyFailure("EVENT_RULES_IMMUTABLE", "复刻必须沿用原活动的概率和保底")
				}
			} else if editorPoolID == "" {
				d.SeriesID = p.ID
			}
		}
		if !creating && (d.Kind != p.Definition.Kind || d.SeriesID != p.Definition.SeriesID) {
			return SupplyPool{}, supplyFailure("POOL_KIND_IMMUTABLE", "卡池类型和活动身份创建后不能改变")
		}
		if err := d.Validate(false); err != nil {
			return SupplyPool{}, supplyFailure("SUPPLY_INVALID", err.Error())
		}
		if err := s.bindSupplyAssets(ctx, tx, actor, p.ID, d); err != nil {
			return SupplyPool{}, err
		}
		p.Definition = d
	}
	if in.Action == "publish" {
		if err := p.Definition.Validate(true); err != nil {
			return SupplyPool{}, supplyFailure("SUPPLY_INVALID", err.Error())
		}
		if err := validateSupplyRewards(ctx, tx, p.Definition.Rewards); err != nil {
			return SupplyPool{}, err
		}
		if p.Definition.Kind == "EVENT" {
			if _, err := tx.ExecContext(ctx, "INSERT IGNORE INTO supply_event_rules(series_id,gold_ppm,hard_pity) VALUES(?,?,?)", p.Definition.SeriesID, p.Definition.EventGoldPPM, p.Definition.EventPity); err != nil {
				return SupplyPool{}, err
			}
			var ppm, pity int
			if err := tx.QueryRowContext(ctx, "SELECT gold_ppm,hard_pity FROM supply_event_rules WHERE series_id=?", p.Definition.SeriesID).Scan(&ppm, &pity); err != nil {
				return SupplyPool{}, err
			}
			if ppm != p.Definition.EventGoldPPM || pity != p.Definition.EventPity {
				return SupplyPool{}, supplyFailure("EVENT_RULES_IMMUTABLE", "已发布活动的概率和保底必须保持不变")
			}
		}
		if p.Definition.Kind == "STANDARD" {
			var count int
			if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM supply_pools WHERE kind='STANDARD' AND enabled=TRUE AND id<>?", p.ID).Scan(&count); err != nil {
				return SupplyPool{}, err
			}
			if count > 0 {
				return SupplyPool{}, supplyFailure("STANDARD_ALREADY_OPEN", "已有开放的常驻池，请编辑该池或先暂停旧池")
			}
		}
		p.PublishedRevision = p.Version + 1
		p.Enabled = true
		p.Definition.FreezeRules()
		raw, err := json.Marshal(p.Definition)
		if err != nil {
			return SupplyPool{}, err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO supply_revisions(pool_id,revision,definition_json,actor_id,created_at) VALUES(?,?,?,?,UTC_TIMESTAMP(6))", p.ID, p.PublishedRevision, raw, actor); err != nil {
			return SupplyPool{}, err
		}
		if err = s.CopyAssetBindingsV2(ctx, tx, "SUPPLY_DRAFT", p.ID, "SUPPLY_REVISION", fmt.Sprintf("%s_%d", p.ID, p.PublishedRevision), SupplyAssetIDs(p.Definition)); err != nil {
			return SupplyPool{}, err
		}
		if err = s.publishSupplyEditorArtwork(ctx, tx, p.Definition); err != nil {
			return SupplyPool{}, err
		}
	}
	if in.Action == "pause" {
		p.Enabled = false
	}
	if in.Action == "resume" {
		if p.PublishedRevision == 0 {
			return SupplyPool{}, supplyFailure("SUPPLY_UNPUBLISHED", "请先发布卡池")
		}
		p.Enabled = true
		if p.Definition.Kind == "STANDARD" {
			var count int
			if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM supply_pools WHERE kind='STANDARD' AND enabled=TRUE AND id<>?", p.ID).Scan(&count); err != nil {
				return SupplyPool{}, err
			}
			if count > 0 {
				return SupplyPool{}, supplyFailure("STANDARD_ALREADY_OPEN", "已有开放的常驻池，请先暂停旧池")
			}
		}
	}
	p.Version++
	p.UpdatedAt = time.Now().UTC()
	raw, err := json.Marshal(p.Definition)
	if err != nil {
		return SupplyPool{}, err
	}
	if creating {
		_, err = tx.ExecContext(ctx, `INSERT INTO supply_pools(id,kind,series_id,draft_json,version,published_revision,enabled,created_at,updated_at) VALUES(?,?,?,?,?,?,?,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))`, p.ID, p.Definition.Kind, p.Definition.SeriesID, raw, p.Version, p.PublishedRevision, p.Enabled)
	} else {
		_, err = tx.ExecContext(ctx, "UPDATE supply_pools SET draft_json=?,version=?,published_revision=?,enabled=?,updated_at=UTC_TIMESTAMP(6) WHERE id=?", raw, p.Version, p.PublishedRevision, p.Enabled, p.ID)
	}
	if err != nil {
		return SupplyPool{}, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO audit_events_next(actor_id,action,resource_id,created_at) VALUES(?,?,?,UTC_TIMESTAMP(6))", actor, "supply.pool."+in.Action, p.ID); err != nil {
		return SupplyPool{}, err
	}
	if in.Action == "publish" {
		if _, err = tx.ExecContext(ctx, "UPDATE supply_pools SET starts_at=?,ends_at=?,visible_at=? WHERE id=?", p.Definition.StartsAt, p.Definition.EndsAt, p.Definition.VisibleAt, p.ID); err != nil {
			return SupplyPool{}, err
		}
	}
	return p, nil
}

func validateSupplyRewards(ctx context.Context, tx *sql.Tx, rewards []supply.Reward) error {
	for _, r := range rewards {
		var raw string
		var archived bool
		if err := tx.QueryRowContext(ctx, `SELECT v.metadata,COALESCE(h.archived,FALSE) FROM item_versions v LEFT JOIN core_catalog_heads h ON h.item_ref=v.item_ref WHERE v.item_ref=? AND v.revision=?`, r.ItemRef, r.ItemRevision).Scan(&raw, &archived); err != nil {
			return socialMissing(err)
		}
		var item ItemVersion
		if err := json.Unmarshal([]byte(raw), &item); err != nil {
			return err
		}
		if archived || item.PayloadSHA256 != r.PayloadSHA256 || item.InventoryDomain != r.InventoryDomain || item.MaxQuantity < 1 || item.Codec != "bukkit-bytes-v1" {
			return supplyFailure("SUPPLY_ITEM_CHANGED", "奖品版本、摘要或交付范围无效，请重新选择物品")
		}
		allowed := map[string]bool{}
		for _, server := range item.CompatibleServerIDs {
			allowed[server] = true
		}
		for _, server := range r.Servers {
			if !allowed[server] {
				return supplyFailure("SUPPLY_ITEM_SERVER", "奖品不支持所选领取服务器")
			}
		}
	}
	return nil
}

func SupplyAssetIDs(d supply.Definition) []string {
	seen := map[string]bool{}
	arts := []supply.Artwork{d.Art}
	for _, r := range d.Rewards {
		arts = append(arts, r.Art)
	}
	for _, a := range arts {
		for _, id := range []string{a.Hero, a.Rail, a.Detail, a.Fallback} {
			if id != "" && !strings.HasPrefix(id, "builtin-") && supply.EditorArtworkHash(id) == "" {
				seen[id] = true
			}
		}
	}
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (s *Store) bindSupplyAssets(ctx context.Context, tx *sql.Tx, actor, pool string, d supply.Definition) error {
	for _, id := range d.AllArtwork() {
		if hash := supply.EditorArtworkHash(id); hash != "" {
			var published bool
			if err := tx.QueryRowContext(ctx, "SELECT published FROM supply_editor_artwork WHERE sha256=?", hash).Scan(&published); err != nil {
				return socialMissing(err)
			}
			if !published && actor != SupplyEditorActor {
				return ErrAssetUnavailable
			}
		}
	}
	ids := SupplyAssetIDs(d)
	for _, id := range ids {
		var owner, purpose, contentType string
		var size int64
		var width, height sql.NullInt64
		if err := tx.QueryRowContext(ctx, "SELECT user_id,purpose,content_type,size_bytes,width,height FROM asset_uploads_v2 WHERE asset_id=?", id).Scan(&owner, &purpose, &contentType, &size, &width, &height); err != nil {
			return socialMissing(err)
		}
		if purpose != "SUPPLY_ART" {
			return ErrAssetForbidden
		}
		if !catalogEnum(contentType, "image/png", "image/jpeg") || size < 1 || size > 8<<20 || !width.Valid || !height.Valid || width.Int64 < 1 || height.Int64 < 1 || width.Int64 > 4096 || height.Int64 > 4096 || width.Int64*height.Int64 > 8388608 {
			return ErrAssetUnavailable
		}
		if owner != actor {
			var sourceType, sourceRef string
			if err := tx.QueryRowContext(ctx, "SELECT business_type,business_ref FROM asset_bindings_v2 WHERE asset_id=? AND business_type IN ('SUPPLY_DRAFT','SUPPLY_REVISION') LIMIT 1", id).Scan(&sourceType, &sourceRef); err != nil {
				return ErrAssetForbidden
			}
			if err := s.AppendAssetBindingsV2(ctx, tx, sourceType, sourceRef, "SUPPLY_DRAFT", pool, []string{id}); err != nil {
				return err
			}
		}
	}
	return s.SetAssetBindingsV2(ctx, tx, actor, "SUPPLY_DRAFT", pool, ids)
}

type SupplySettings struct {
	Version       int64 `json:"version"`
	StandardPrice int64 `json:"standardPrice"`
	LimitedPrice  int64 `json:"limitedPrice"`
}

func (s *Store) SupplySettings(ctx context.Context) (SupplySettings, error) {
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return SupplySettings{}, err
	}
	defer tx.Rollback()
	settings, err := scanSupplySettings(tx.QueryRowContext(ctx, supplySettingsQuery))
	if err != nil {
		return settings, err
	}
	tickets, err := supplyTickets(ctx, tx, settings)
	if err != nil {
		return settings, err
	}
	settings.StandardPrice = tickets["STANDARD"].UnitPrice
	settings.LimitedPrice = tickets["LIMITED"].UnitPrice
	return settings, tx.Commit()
}

func (s *Store) SaveSupplySettings(ctx context.Context, actor, key string, settings SupplySettings) (json.RawMessage, error) {
	if settings.StandardPrice < 0 || settings.LimitedPrice < 0 || settings.StandardPrice > 100000000 || settings.LimitedPrice > 100000000 {
		return nil, ErrSocialInvalid
	}
	return s.supplyMutate(ctx, actor, key, struct {
		Action   string
		Settings SupplySettings
	}{"supply.settings", settings}, "", "", func(tx *sql.Tx) (any, error) {
		if err := requirePlatformAdminTxV206(ctx, tx, actor); err != nil {
			return nil, err
		}
		current, err := scanSupplySettings(tx.QueryRowContext(ctx, supplySettingsQuery+" FOR UPDATE"))
		if err != nil {
			return nil, err
		}
		if current.Version != settings.Version {
			return nil, ErrConflict
		}
		tickets, err := supplyTickets(ctx, tx, current)
		if err != nil {
			return nil, err
		}
		standardPrice, limitedPrice := settings.StandardPrice, settings.LimitedPrice
		if ticket := tickets["STANDARD"]; ticket.managedPrice {
			if standardPrice != ticket.UnitPrice {
				return nil, supplyFailure("SUPPLY_PRICE_OWNED_BY_EDITOR", "请在抽卡编辑器修改并发布常驻券价格")
			}
			standardPrice = current.StandardPrice
		}
		if ticket := tickets["LIMITED"]; ticket.managedPrice {
			if limitedPrice != ticket.UnitPrice {
				return nil, supplyFailure("SUPPLY_PRICE_OWNED_BY_EDITOR", "请在抽卡编辑器修改并发布限定券价格")
			}
			limitedPrice = current.LimitedPrice
		}
		result, err := tx.ExecContext(ctx, "UPDATE supply_settings SET standard_price=?,limited_price=?,version=version+1 WHERE id=1 AND version=?", standardPrice, limitedPrice, settings.Version)
		if err != nil {
			return nil, err
		}
		n, err := result.RowsAffected()
		if err != nil {
			return nil, err
		}
		if n != 1 {
			return nil, ErrConflict
		}
		settings.Version++
		_, err = tx.ExecContext(ctx, "INSERT INTO audit_events_next(actor_id,action,resource_id,created_at) VALUES(?,?,?,UTC_TIMESTAMP(6))", actor, "supply.settings", fmt.Sprint(settings.Version))
		return settings, err
	})
}

// IsSupplyMissing supports transport-specific 404 responses without leaking SQL diagnostics.
func IsSupplyMissing(err error) bool {
	return errors.Is(err, sql.ErrNoRows) || errors.Is(err, ErrSocialNotFound)
}
