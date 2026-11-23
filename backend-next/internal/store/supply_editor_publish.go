package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/supply"
)

const supplyEditorPoolPrefix = "sp_editor_"

type SupplyEditorPublication struct {
	Revision int64           `json:"revision"`
	Catalog  json.RawMessage `json:"catalog"`
}

type editorCatalog struct {
	Tickets []editorTicket `json:"tickets"`
	Pools   []editorPool   `json:"pools"`
}

type editorTicket struct {
	ID      string      `json:"id"`
	Name    string      `json:"name"`
	Enabled bool        `json:"enabled"`
	Icon    string      `json:"icon"`
	Price   json.Number `json:"price,omitempty"`
}

type editorPool struct {
	ID               string               `json:"id"`
	Name             string               `json:"name"`
	OfficialTitle    string               `json:"officialTitle"`
	Subtitle         string               `json:"subtitle"`
	Description      string               `json:"description"`
	Cover            string               `json:"cover"`
	Thumbnail        string               `json:"thumbnail"`
	Currency         string               `json:"currency"`
	Enabled          bool                 `json:"enabled"`
	StartsAt         *time.Time           `json:"startsAt"`
	EndsAt           *time.Time           `json:"endsAt"`
	VisibleAt        *time.Time           `json:"visibleAt"`
	Prizes           []editorPrize        `json:"prizes"`
	FeaturedPrizeIDs []string             `json:"featuredPrizeIds"`
	Labels           *supply.PoolLabels   `json:"labels"`
	Sections         []supply.PoolSection `json:"sections"`
	Rules            editorRules          `json:"rules"`
}

type editorRules struct {
	Mode          string `json:"mode"`
	ActivityID    string `json:"activityId"`
	LegendaryPity int    `json:"legendaryPity"`
	GoldPPM       int    `json:"goldPpm"`
}

type editorPrize struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	ItemID      string             `json:"itemId"`
	GunID       string             `json:"gunId"`
	Count       int                `json:"count"`
	Stars       int                `json:"stars"`
	Probability float64            `json:"probability"`
	LibraryItem *editorLibraryLink `json:"libraryItem"`
}

type editorLibraryLink struct {
	ItemRef       string `json:"itemRef"`
	Revision      int64  `json:"revision"`
	PayloadSHA256 string `json:"payloadSha256"`
}

type editorPoolPublication struct {
	SourceID string `json:"sourceId"`
	PoolID   string `json:"poolId"`
	Revision int64  `json:"revision"`
}

func supplyEditorPoolID(id string) string {
	return supplyEditorPoolPrefix + Digest([]byte(id))[:32]
}

// PublishSupplyEditor installs one immutable catalog with the same pool rules used by the admin.
// Only pools with editor-owned IDs are candidates for replacement or pausing.
func (s *Store) PublishSupplyEditor(ctx context.Context, node, itemPrefix string, deliveryNodes map[string]bool, input SupplyEditorPublication) (json.RawMessage, error) {
	var catalog editorCatalog
	if input.Revision < 1 || json.Unmarshal(input.Catalog, &catalog) != nil ||
		catalog.Pools == nil || catalog.Tickets == nil || len(catalog.Pools) > 64 || len(catalog.Tickets) > 64 {
		return nil, ErrSocialInvalid
	}
	tickets := map[string]editorTicket{}
	for _, ticket := range catalog.Tickets {
		if _, exists := tickets[ticket.ID]; exists || !supply.ValidCurrency(ticket.ID) || ticket.Name != "" && !ValidSocialText(ticket.Name, 1, 80) {
			return nil, ErrSocialInvalid
		}
		if _, err := supplyTicketPrice(ticket.Price, 0); err != nil {
			return nil, err
		}
		tickets[ticket.ID] = ticket
	}
	active := map[string]bool{}
	seen := map[string]bool{}
	for _, p := range catalog.Pools {
		if seen[p.ID] || !supply.ValidID(p.ID) {
			return nil, ErrSocialInvalid
		}
		ticket, exists := tickets[p.Currency]
		if !exists {
			return nil, ErrSocialInvalid
		}
		seen[p.ID] = true
		active[supplyEditorPoolID(p.ID)] = p.Enabled && ticket.Enabled
	}
	return s.supplyMutate(ctx, SupplyEditorActor, "publish_"+strconv.FormatInt(input.Revision, 10), input, "", "", func(tx *sql.Tx) (any, error) {
		var settingsVersion, latest int64
		if err := tx.QueryRowContext(ctx, "SELECT version FROM supply_settings WHERE id=1 FOR UPDATE").Scan(&settingsVersion); err != nil {
			return nil, err
		}
		if err := tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(source_revision),0) FROM supply_editor_publications").Scan(&latest); err != nil {
			return nil, err
		}
		if input.Revision <= latest {
			return nil, ErrConflict
		}
		existing, err := editorSupplyPools(ctx, tx)
		if err != nil {
			return nil, err
		}
		byID := map[string]SupplyPool{}
		for _, p := range existing {
			byID[p.ID] = p
			if p.Enabled && !active[p.ID] {
				if _, err := s.changeSupplyPoolTx(ctx, tx, SupplyEditorActor, p.ID, "", SupplyPoolChange{Action: "pause", ExpectedVersion: p.Version}); err != nil {
					return nil, err
				}
			}
		}
		released := make([]editorPoolPublication, 0, len(catalog.Pools))
		for sort, p := range catalog.Pools {
			id := supplyEditorPoolID(p.ID)
			if !active[id] {
				continue
			}
			definition, err := editorSupplyDefinition(ctx, tx, node, itemPrefix, deliveryNodes, p, tickets[p.Currency], len(catalog.Pools)-sort)
			if err != nil {
				return nil, err
			}
			current := byID[id]
			saved, err := s.changeSupplyPoolTx(ctx, tx, SupplyEditorActor, current.ID, id,
				SupplyPoolChange{Action: "save", ExpectedVersion: current.Version, Definition: definition})
			if err != nil {
				return nil, err
			}
			published, err := s.changeSupplyPoolTx(ctx, tx, SupplyEditorActor, saved.ID, "",
				SupplyPoolChange{Action: "publish", ExpectedVersion: saved.Version})
			if err != nil {
				return nil, err
			}
			released = append(released, editorPoolPublication{p.ID, published.ID, published.PublishedRevision})
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO supply_editor_publications VALUES(?,?,UTC_TIMESTAMP(6))", input.Revision, input.Catalog); err != nil {
			return nil, err
		}
		ticketJSON, err := json.Marshal(catalog.Tickets)
		if err != nil {
			return nil, err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO supply_editor_ticket_catalogs(source_revision,tickets_json) VALUES(?,?)", input.Revision, ticketJSON); err != nil {
			return nil, err
		}
		if _, err = tx.ExecContext(ctx, "UPDATE supply_settings SET version=version+1 WHERE id=1"); err != nil {
			return nil, err
		}
		return map[string]any{"revision": input.Revision, "pools": released}, nil
	})
}

func editorSupplyPools(ctx context.Context, tx *sql.Tx) ([]SupplyPool, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id,version,published_revision,enabled,draft_json,updated_at
		FROM supply_pools WHERE LEFT(id,?)=CONVERT(? USING ascii) COLLATE ascii_bin ORDER BY id FOR UPDATE`,
		len(supplyEditorPoolPrefix), supplyEditorPoolPrefix)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SupplyPool
	for rows.Next() {
		p, err := scanSupplyPool(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func editorSupplyDefinition(ctx context.Context, tx *sql.Tx, node, itemPrefix string, deliveryNodes map[string]bool, p editorPool, ticket editorTicket, sort int) (supply.Definition, error) {
	title := p.OfficialTitle
	if title == "" {
		title = p.Name
	}
	d := supply.Definition{Name: p.Name, Title: title, Subtitle: p.Subtitle, Description: p.Description,
		Kind: p.Rules.Mode, StartsAt: p.StartsAt, EndsAt: p.EndsAt, VisibleAt: p.VisibleAt, Sort: sort,
		Labels: p.Labels, Sections: p.Sections, TicketID: ticket.ID, TicketArt: editorArtworkID(ticket.Icon),
		Art: supply.Artwork{Hero: editorArtworkID(p.Cover), Rail: editorArtworkID(p.Thumbnail)}}
	switch p.Rules.Mode {
	case "STANDARD":
		if len(p.FeaturedPrizeIDs) > 0 {
			return d, editorPoolError(p, "常驻池的六星使用组内权重，请取消 UP 标记")
		}
	case "LIMITED":
	case "EVENT":
		if !supply.ValidID(p.Rules.ActivityID) {
			return d, editorPoolError(p, "请填写有效的活动编号")
		}
		d.SeriesID = "gacha_event_" + Digest([]byte(p.Rules.ActivityID))[:32]
		d.EventGoldPPM, d.EventPity = p.Rules.GoldPPM, p.Rules.LegendaryPity
	default:
		return d, editorPoolError(p, "正式补给支持常驻池、限定池和活动池，请选择正式规则")
	}
	if ticket.Icon != "" && d.TicketArt == "" {
		return d, editorPoolError(p, "正式票券图标格式不受支持")
	}
	featured := map[string]bool{}
	for _, id := range p.FeaturedPrizeIDs {
		featured[id] = true
	}
	for _, prize := range p.Prizes {
		weight := math.Round(prize.Probability * 100)
		if prize.Count != 1 || weight <= 0 || weight > 10000 {
			return d, editorPoolError(p, fmt.Sprintf("奖品「%s」须发放单件，且组内权重大于零", prize.Name))
		}
		item, err := verifyEditorPrize(ctx, tx, node, itemPrefix, prize)
		if err != nil {
			if IsSupplyMissing(err) || errors.Is(err, ErrAssetUnavailable) {
				return d, editorPoolError(p, fmt.Sprintf("奖品「%s」缺少有效的物品库版本，请重新选取", prize.Name))
			}
			return d, err
		}
		servers := make([]string, 0, len(item.CompatibleServerIDs))
		for _, server := range item.CompatibleServerIDs {
			if deliveryNodes[server] {
				servers = append(servers, server)
			}
		}
		if len(servers) == 0 {
			return d, editorPoolError(p, fmt.Sprintf("奖品「%s」在正式服没有可用的交付节点", prize.Name))
		}
		name := strings.TrimSpace(prize.Name)
		if name == "" {
			name = item.DisplayName
		}
		d.Rewards = append(d.Rewards, supply.Reward{
			Key: prize.ID, Name: name, GunID: prize.GunID, Stars: prize.Stars, Weight: int(weight),
			UP: featured[prize.ID], ItemRef: item.ItemRef, ItemRevision: item.Revision, PayloadSHA256: item.PayloadSHA256,
			InventoryDomain: item.InventoryDomain, Servers: servers,
		})
	}
	if err := d.Validate(true); err != nil {
		return d, editorPoolError(p, err.Error())
	}
	return d, nil
}

func verifyEditorPrize(ctx context.Context, tx *sql.Tx, node, itemPrefix string, prize editorPrize) (ItemVersion, error) {
	link := prize.LibraryItem
	prefix := itemPrefix + "gacha_"
	if link == nil || link.Revision < 1 || !strings.HasPrefix(link.ItemRef, prefix) {
		return ItemVersion{}, ErrAssetUnavailable
	}
	owned, err := scanSupplyEditorItem(tx.QueryRowContext(ctx, supplyEditorItemQuery+" AND h.item_ref=CONVERT(? USING ascii) COLLATE ascii_bin",
		SupplyEditorActor, SupplyEditorImportCommand, node, len(prefix), prefix, len(prefix)+32, link.ItemRef))
	if err != nil {
		return ItemVersion{}, err
	}
	var metadata []byte
	if err := tx.QueryRowContext(ctx, "SELECT metadata FROM item_versions WHERE item_ref=? AND revision=?",
		link.ItemRef, link.Revision).Scan(&metadata); err != nil {
		return ItemVersion{}, err
	}
	var frozen ItemVersion
	if err := json.Unmarshal(metadata, &frozen); err != nil {
		return ItemVersion{}, err
	}
	if owned.Archived || frozen.PayloadSHA256 != link.PayloadSHA256 || frozen.ItemID != prize.ItemID || frozen.Codec != "bukkit-bytes-v1" {
		return ItemVersion{}, ErrAssetUnavailable
	}
	return frozen, nil
}

func editorArtworkID(ref string) string {
	switch ref {
	case "builtin:ticket":
		return "builtin-ticket"
	case "builtin:pool-up":
		return "builtin-hero-up"
	case "builtin:pool-up-vector":
		return "builtin-hero-vector"
	case "builtin:pool-standard-1":
		return "builtin-hero-standard"
	case "builtin:rail-up":
		return "builtin-rail-up"
	case "builtin:rail-up-vector":
		return "builtin-rail-vector"
	case "builtin:rail-standard-1":
		return "builtin-rail-standard"
	}
	if hash, ok := strings.CutPrefix(ref, "sha256:"); ok {
		id := supply.EditorArtworkPrefix + hash
		if supply.EditorArtworkHash(id) != "" {
			return id
		}
	}
	return ""
}

func editorPoolError(pool editorPool, message string) error {
	return supplyFailure("SUPPLY_EDITOR_POOL_INVALID", "「"+pool.Name+"」 "+message+"。")
}
