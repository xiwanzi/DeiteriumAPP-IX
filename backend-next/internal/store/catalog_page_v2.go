package store

import (
	"context"
	"strings"
)

// CatalogPageViewsV2 is called only after CatalogListV2 has authorized and
// filtered the records. Public product media uses two bounded queries per page,
// preserving the exact (product, asset) binding instead of checking each image.
func (s *Store) CatalogPageViewsV2(ctx context.Context, viewer string, records []CatalogRecordV2, management bool) ([]any, error) {
	views := make([]any, 0, len(records))
	if len(records) == 0 {
		return views, nil
	}
	if management || records[0].Kind != "product" {
		for _, d := range records {
			view, err := s.CatalogViewV2(ctx, viewer, d, management)
			if err != nil {
				return nil, err
			}
			views = append(views, view)
		}
		return views, nil
	}
	refs := make([]any, 0, len(records))
	wanted := map[string]bool{}
	for _, d := range records {
		if d.Kind != "product" || d.Published == nil {
			return nil, catalogInvalid()
		}
		refs = append(refs, d.ID)
		for _, id := range catalogIDs(d.Published, "galleryAssetIds") {
			wanted[id] = true
		}
	}
	bindings := map[string]map[string]bool{}
	assets := map[string]map[string]any{}
	if len(wanted) > 0 {
		marks := strings.TrimSuffix(strings.Repeat("?,", len(refs)), ",")
		rows, err := s.DB.QueryContext(ctx, `SELECT business_ref,asset_id FROM asset_bindings_v2 WHERE business_type='PRODUCT_PUBLISHED' AND business_ref IN (`+marks+`)`, refs...)
		if err != nil {
			return nil, err
		}
		ids := map[string]bool{}
		for rows.Next() {
			var ref, id string
			if err = rows.Scan(&ref, &id); err != nil {
				rows.Close()
				return nil, err
			}
			if !wanted[id] {
				continue
			}
			if bindings[ref] == nil {
				bindings[ref] = map[string]bool{}
			}
			bindings[ref][id] = true
			ids[id] = true
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
		if len(ids) > 0 {
			args := make([]any, 0, len(ids))
			for id := range ids {
				args = append(args, id)
			}
			marks = strings.TrimSuffix(strings.Repeat("?,", len(args)), ",")
			rows, err = s.DB.QueryContext(ctx, assetSelectV2+` WHERE asset_id IN (`+marks+`) AND status='READY' AND removed_at IS NULL`, args...)
			if err != nil {
				return nil, err
			}
			defer rows.Close()
			for rows.Next() {
				asset, scanErr := scanAssetV2(rows)
				if scanErr != nil {
					return nil, scanErr
				}
				view, viewErr := assetViewV2(ctx, asset)
				if viewErr != nil {
					return nil, viewErr
				}
				assets[asset.AssetID] = view
			}
			if err = rows.Err(); err != nil {
				return nil, err
			}
		}
	}
	for _, d := range records {
		images := []any{}
		for _, id := range catalogIDs(d.Published, "galleryAssetIds") {
			if view := assets[id]; bindings[d.ID][id] && view != nil {
				images = append(images, view)
			} else {
				// An unavailable cover must not hide the rest of the catalog.
				images = append(images, map[string]any{"assetId": id, "status": "UNAVAILABLE", "url": ""})
			}
		}
		view, err := catalogPublishedViewV2(d, images)
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	return views, nil
}

func catalogPublishedViewV2(d CatalogRecordV2, images []any) (map[string]any, error) {
	state := d.State
	if state == "DRAFT" {
		state = "UNLISTED"
	}
	effective, err := promotionPriceV209(d.Published)
	if err != nil {
		return nil, err
	}
	return map[string]any{"productId": d.ID, "storeId": d.StoreID, "version": d.PublishedVersion, "content": d.Published, "effectivePrice": catalogMoneyString(effective), "images": images, "visibility": state, "availableStock": d.Stock, "publishedAt": d.PublishedAt, "updatedAt": d.UpdatedAt}, nil
}
