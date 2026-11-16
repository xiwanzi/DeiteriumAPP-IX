package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/supply"
)

func builtinSupplyArt() map[string]any {
	out := map[string]any{}
	for id, file := range supply.BuiltinArtwork() {
		out[id] = map[string]any{"assetId": id, "builtin": "deuterium_pve:textures/gui/supply/" + file, "url": "/supply-art/" + file, "altText": id}
	}
	return out
}

func (s *Server) supplyArtwork(ctx context.Context, pool store.SupplyPool, published bool) (map[string]any, error) {
	out := map[string]any{}
	builtin := builtinSupplyArt()
	for _, id := range pool.Definition.AllArtwork() {
		if strings.HasPrefix(id, "builtin-") {
			if builtin[id] == nil {
				return nil, store.ErrAssetUnavailable
			}
			out[id] = builtin[id]
			continue
		}
		kind, ref := "SUPPLY_DRAFT", pool.ID
		if published {
			kind = "SUPPLY_REVISION"
			ref = pool.ID + "_" + strconv.FormatInt(pool.PublishedRevision, 10)
		}
		asset, err := s.Store.AssetForBindingV2(ctx, kind, ref, id)
		if err != nil {
			return nil, err
		}
		out[id] = asset
	}
	return out, nil
}

func (s *Server) registerSupplyArtwork(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/admin/supply/assets", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.admin(r, "platform.admin"); err != nil {
			failError(w, r, err)
			return
		}
		rows, err := s.Store.DB.QueryContext(r.Context(), "SELECT asset_id,user_id FROM asset_uploads_v2 WHERE purpose='SUPPLY_ART' AND status='READY' AND removed_at IS NULL AND lifecycle_state='ACTIVE' ORDER BY created_at DESC LIMIT 200")
		if err != nil {
			assetErrorV2(w, r, err)
			return
		}
		type owned struct{ id, owner string }
		ids := []owned{}
		for rows.Next() {
			var a owned
			if err = rows.Scan(&a.id, &a.owner); err != nil {
				rows.Close()
				assetErrorV2(w, r, err)
				return
			}
			ids = append(ids, a)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			assetErrorV2(w, r, err)
			return
		}
		items := []map[string]any{}
		for _, a := range ids {
			asset, e := s.Store.AssetV2(r.Context(), a.owner, a.id)
			if e != nil {
				assetErrorV2(w, r, e)
				return
			}
			items = append(items, asset)
		}
		v2Success(w, r, map[string]any{"items": items, "builtins": builtinSupplyArt(), "models": supply.ModelKeys})
	})
	mux.HandleFunc("GET /api/v1/admin/supply/pools/{poolId}/preview", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.admin(r, "platform.admin"); err != nil {
			failError(w, r, err)
			return
		}
		pools, err := s.Store.SupplyPools(r.Context(), true)
		if err != nil {
			catalogFailV2(w, r, err)
			return
		}
		for _, pool := range pools {
			if pool.ID != r.PathValue("poolId") {
				continue
			}
			art, e := s.supplyArtwork(r.Context(), pool, false)
			if e != nil {
				assetErrorV2(w, r, e)
				return
			}
			v2Success(w, r, map[string]any{"pool": pool, "artwork": art})
			return
		}
		failure(w, r, 404, "NOT_FOUND", "卡池不存在")
	})
}
