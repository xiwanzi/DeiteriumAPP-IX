package httpapi

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/supply"
)

func publicSupplyWeapon(r supply.Reward) map[string]any {
	return map[string]any{"name": r.Name, "gunId": r.GunID, "stars": r.Stars, "modelKey": r.ModelKey, "detailArt": r.Art.Detail, "fallbackArt": r.Art.Fallback}
}

func (s *Server) publicSupplyCatalog(ctx context.Context, pools []store.SupplyPool) (map[string]any, map[string]any, error) {
	weapons := map[string]any{}
	pages := map[string]any{}
	art := map[string]any{}
	order := []string{}
	for _, pool := range pools {
		d := pool.Definition
		assets, err := s.supplyArtwork(ctx, pool, true)
		if err != nil {
			return nil, nil, err
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
				return nil, nil, e
			}
			key := pool.ID + "_" + scheme
			order = append(order, key)
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
			label := d.Name
			if d.Kind == "LIMITED" {
				label += " · " + scheme
			}
			pages[key] = map[string]any{"title": title, "label": label, "badge": badge, "tag": fmt.Sprintf("出金概率 %s%%", strconv.FormatFloat(float64(rule.GoldPPM)/10000, 'f', -1, 64)), "time": when, "code": scheme, "featured": featured, "entries": entries, "poolId": pool.ID, "revision": pool.PublishedRevision, "scheme": scheme, "kind": d.Kind, "currency": currency, "scope": scope, "rule": rule, "art": d.Art, "description": d.Description, "startsAt": d.StartsAt, "endsAt": d.EndsAt, "open": d.Open(time.Now().UTC())}
		}
	}
	return map[string]any{"weapons": weapons, "pools": pages, "order": order}, art, nil
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
	pity := map[string]int{}
	for k, v := range account.Pity {
		pity[k] = v.Count
	}
	out["account"] = map[string]any{"version": account.Version, "standard": account.Standard, "limited": account.Limited, "pity": pity}
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
