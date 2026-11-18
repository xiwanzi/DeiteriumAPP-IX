package store

import "context"

func (s *Store) CatalogRecommendations(ctx context.Context) ([]CatalogRecordV2, error) {
	const query = `WITH sellable AS (
		SELECT resource_id FROM catalog_records_v2
		WHERE kind='product' AND state='ACTIVE' AND published_body IS NOT NULL
			AND (available_stock IS NULL OR available_stock > 0)
	), hot AS (
		SELECT h.product_id, SUM(h.quantity) AS units
		FROM commerce_resources_v2 r
		JOIN commerce_stock_holds_v2 h ON h.resource_id=r.resource_id
		JOIN sellable p ON p.resource_id=h.product_id
		WHERE r.resource_kind='ORDER' AND r.channel='OFFICIAL_STORE'
			AND r.created_at >= UTC_TIMESTAMP() - INTERVAL 30 DAY
			AND r.funds_state IN ('PAID','HELD','SETTLED')
			AND r.state NOT IN ('CANCELLED','REFUNDED') AND h.state <> 'RELEASED'
		GROUP BY h.product_id ORDER BY units DESC, h.product_id LIMIT 3
	), discovery AS (
		SELECT p.resource_id AS product_id, RAND() AS draw FROM sellable p
		WHERE NOT EXISTS (SELECT 1 FROM hot WHERE hot.product_id=p.resource_id)
		ORDER BY draw LIMIT 6
	), picks AS (
		SELECT product_id, 2 * ROW_NUMBER() OVER (ORDER BY units DESC, product_id) - 1 AS slot FROM hot
		UNION ALL
		SELECT product_id, 2 * ROW_NUMBER() OVER (ORDER BY draw, product_id) AS slot FROM discovery
	)
	` + catalogSelect + ` JOIN picks ON picks.product_id=catalog_records_v2.resource_id ORDER BY picks.slot LIMIT 6`
	rows, err := s.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []CatalogRecordV2
	for rows.Next() {
		product, err := catalogScan(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, product)
	}
	return records, rows.Err()
}
