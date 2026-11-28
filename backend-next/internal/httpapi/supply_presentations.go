package httpapi

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/supply"
)

const supplyPresentationVersion = 2

func publicSupplyWeapon(r supply.Reward) map[string]any {
	model := r.ModelKey
	if model == "" {
		key := strings.Replace(r.GunID, ":", "-", 1)
		if supply.ValidID(key) {
			model = key
		}
	}
	out := map[string]any{"name": r.Name, "gunId": r.GunID, "stars": r.Stars, "modelKey": model, "detailArt": r.Art.Detail, "fallbackArt": r.Art.Fallback}
	if r.DisplayID != "" {
		out["displayId"] = r.DisplayID
	}
	return out
}

func (s *Server) publicSupplyCatalog(ctx context.Context, pools []store.SupplyPool) (map[string]any, map[string]any, error) {
	catalog, art, _, err := s.publicSupplyCatalogDelta(ctx, pools, nil, 0)
	return catalog, art, err
}

func (s *Server) publicSupplyCatalogDelta(ctx context.Context, pools []store.SupplyPool, knownPools map[string]int64, clientVersion int) (map[string]any, map[string]any, bool, error) {
	deltaMode := clientVersion == supplyPresentationVersion && len(knownPools) > 0
	weapons := map[string]any{}
	pages := map[string]any{}
	art := map[string]any{}
	order := []string{}
	openMap := map[string]bool{}
	now := time.Now().UTC()
	for _, pool := range pools {
		d := pool.Definition
		isOpen := d.Open(now)
		knownRev, hasKnown := knownPools[pool.ID]
		unchanged := deltaMode && hasKnown && knownRev == pool.PublishedRevision
		for _, scheme := range d.Schemes() {
			key := pool.ID + "_" + scheme
			order = append(order, key)
			openMap[key] = isOpen
		}
		if unchanged {
			continue
		}
		assets, err := s.supplyArtwork(ctx, pool, true)
		if err != nil {
			return nil, nil, false, err
		}
		for k, v := range assets {
			art[k] = v
		}
		featured := []string{}
		entries := []map[string]any{}
		for _, reward := range d.Rewards {
			key := pool.ID + "_" + reward.Key
			weapons[key] = publicSupplyWeapon(reward)
			entries = append(entries, map[string]any{"id": key, "weight": reward.Weight})
			if reward.UP || d.Kind == "STANDARD" && reward.Stars == 6 {
				featured = append(featured, key)
			}
		}
		for _, scheme := range d.Schemes() {
			rule, scope, currency, e := d.Rule(scheme)
			if e != nil {
				return nil, nil, false, e
			}
			key := pool.ID + "_" + scheme
			runes := []rune(d.Title)
			split := len(runes) / 2
			title := []string{string(runes[:split]), string(runes[split:])}
			badge := "限定"
			if d.Kind == "STANDARD" {
				badge = "常驻"
			}
			if d.Kind == "EVENT" {
				badge = "活动"
			}
			when := "长期开放"
			if d.EndsAt != nil {
				when = "结束于 " + d.EndsAt.In(time.FixedZone("UTC+8", 28800)).Format("01/02 15:04") + " (UTC+8)"
			}
			label := badge
			if d.Labels != nil && strings.TrimSpace(d.Labels.Rail) != "" {
				label = strings.TrimSpace(d.Labels.Rail)
			}
			if d.Kind == "LIMITED" {
				label += " · " + scheme
			}
			page := map[string]any{
				"title": title, "label": label, "badge": badge,
				"tag": fmt.Sprintf("6 星概率 %s%%", strconv.FormatFloat(float64(rule.GoldPPM)/10000, 'f', -1, 64)),
				"time": when, "code": scheme, "featured": featured, "entries": entries,
				"poolId": pool.ID, "revision": pool.PublishedRevision, "scheme": scheme,
				"kind": d.Kind, "currency": currency, "ticketIcon": d.TicketArt,
				"scope": scope, "rule": rule, "art": d.Art,
				"subtitle": d.Subtitle, "description": d.Description,
				"startsAt": d.StartsAt, "endsAt": d.EndsAt,
				"open": isOpen, "itemsBound": d.ItemsBound(),
			}
			if d.Labels != nil {
				if ft := strings.TrimSpace(d.Labels.FeaturedTitle); ft != "" {
					page["featuredTitle"] = ft
				}
				if fl := strings.TrimSpace(d.Labels.FeaturedLead); fl != "" {
					page["featuredLead"] = fl
				}
			}
			if len(d.Sections) > 0 {
				page["sections"] = d.Sections
			}
			pages[key] = page
		}
	}
	out := map[string]any{"weapons": weapons, "pools": pages, "order": order}
	if deltaMode {
		out["mode"] = "DELTA"
		out["open"] = openMap
		out["unchanged"] = len(pages) == 0
	} else {
		out["mode"] = "FULL"
	}
	return out, art, deltaMode, nil
}

func publicSupplyAccount(account store.SupplyAccount) map[string]any {
	pity := map[string]int{}
	for scope, progress := range account.Pity {
		pity[scope] = progress.Count
	}
	return map[string]any{"version": account.Version, "standard": account.Standard, "limited": account.Limited, "balances": account.Balances, "pity": pity}
}

func publicSupplyOrder(order store.SupplyOrder) map[string]any {
	rolls := []map[string]any{}
	weapons := map[string]any{}
	for _, roll := range order.Rolls {
		key := order.PoolID + "_" + roll.Reward.Key
		weapons[key] = publicSupplyWeapon(roll.Reward)
		rolls = append(rolls, map[string]any{"weapon": key, "before": roll.Before, "after": roll.After, "guaranteed": roll.Guaranteed, "cycle": roll.Cycle})
	}
	return map[string]any{"id": order.ID, "sequence": order.Sequence, "poolId": order.PoolID, "poolName": order.PoolName, "revision": order.Revision, "scheme": order.Scheme, "currency": order.Currency, "quantity": order.Quantity, "rolls": rolls, "weapons": weapons, "createdAt": order.CreatedAt}
}

func (s *Server) supplyPlayerOrder(ctx context.Context, id, uuid string) (map[string]any, error) {
	detail, err := s.Store.SupplyOrder(ctx, id, uuid)
	if err != nil {
		return nil, err
	}
	order := detail["order"].(store.SupplyOrder)
	out := publicSupplyOrder(order)
	results := []map[string]any{}
	for _, d := range detail["deliveries"].([]store.SupplyDelivery) {
		results = append(results, map[string]any{"ordinal": d.Ordinal, "state": d.State, "message": d.LastError})
	}
	out["deliveries"] = results
	account, err := s.Store.SupplyAccount(ctx, uuid)
	if err != nil {
		return nil, err
	}
	out["account"] = publicSupplyAccount(account)
	d, err := s.Store.SupplyPoolRevision(ctx, order.PoolID, order.Revision)
	if err != nil {
		return nil, err
	}
	art, err := s.supplyArtwork(ctx, store.SupplyPool{ID: order.PoolID, PublishedRevision: order.Revision, Definition: d}, true)
	if err != nil {
		return nil, err
	}
	out["artwork"] = art
	return out, nil
}
