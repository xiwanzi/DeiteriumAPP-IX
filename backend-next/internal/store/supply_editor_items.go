package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
)

type SupplyEditorItem struct {
	ItemVersion
	CatalogVersion int64  `json:"catalogVersion"`
	Archived       bool   `json:"archived"`
	StackData      string `json:"stackData"`
}

const (
	SupplyEditorActor         = "gacha-editor"
	SupplyEditorImportCommand = "items.editor_import"
)

func SupplyEditorOperationKey(node, command, ref string, catalogVersion int64) string {
	return "editor_" + Digest([]byte(node + ":" + command + ":" + ref + ":" + strconv.FormatInt(catalogVersion, 10)))[:48]
}

// The initial import identifies the editor as the source and retains its ItemStack for asset previews.
// Its client key is SupplyEditorOperationKey with the initial catalog version, zero.
const supplyEditorItemQuery = `SELECT v.metadata,h.catalog_version,h.archived,
	JSON_UNQUOTE(JSON_EXTRACT(origin.payload,'$.stackData'))
	FROM core_catalog_heads h JOIN item_versions v ON v.item_ref=h.item_ref AND v.revision=h.latest_revision
	JOIN core_operations origin ON origin.actor_id=CONVERT(? USING ascii) COLLATE ascii_bin
		AND origin.command_type=CONVERT(? USING ascii) COLLATE ascii_bin AND origin.node_id=v.publisher_node
		AND origin.client_request_id=CONVERT(CONCAT('editor_',LEFT(SHA2(CONCAT(v.publisher_node,':',origin.command_type,':',h.item_ref,':0'),256),48)) USING ascii) COLLATE ascii_bin
	WHERE v.publisher_node=CONVERT(? USING ascii) COLLATE ascii_bin
		AND LEFT(h.item_ref,?)=CONVERT(? USING ascii) COLLATE ascii_bin AND CHAR_LENGTH(h.item_ref)=?`

func scanSupplyEditorItem(row interface{ Scan(...any) error }) (SupplyEditorItem, error) {
	var item SupplyEditorItem
	var raw []byte
	err := row.Scan(&raw, &item.CatalogVersion, &item.Archived, &item.StackData)
	if err == nil {
		err = json.Unmarshal(raw, &item.ItemVersion)
	}
	return item, err
}

func (s *Store) SupplyEditorItem(ctx context.Context, node, prefix, ref string) (SupplyEditorItem, error) {
	return scanSupplyEditorItem(s.DB.QueryRowContext(ctx, supplyEditorItemQuery+" AND h.item_ref=CONVERT(? USING ascii) COLLATE ascii_bin",
		SupplyEditorActor, SupplyEditorImportCommand, node, len(prefix), prefix, len(prefix)+32, ref))
}

func (s *Store) SupplyEditorItems(ctx context.Context, node, prefix, query string, page int) ([]SupplyEditorItem, bool, error) {
	const pageSize = 30
	rows, err := s.DB.QueryContext(ctx, supplyEditorItemQuery+` AND h.archived=FALSE
		AND (?='' OR LOCATE(LOWER(CONVERT(? USING utf8mb4)) COLLATE utf8mb4_bin,
			LOWER(CONVERT(JSON_UNQUOTE(JSON_EXTRACT(v.metadata,'$.displayName')) USING utf8mb4)) COLLATE utf8mb4_bin)>0
		 OR LOCATE(LOWER(CONVERT(? USING utf8mb4)) COLLATE utf8mb4_bin,
			LOWER(CONVERT(JSON_UNQUOTE(JSON_EXTRACT(v.metadata,'$.itemId')) USING utf8mb4)) COLLATE utf8mb4_bin)>0
		 OR LOCATE(LOWER(CONVERT(? USING utf8mb4)) COLLATE utf8mb4_bin,
			CONVERT(v.item_ref USING utf8mb4) COLLATE utf8mb4_bin)>0)
		ORDER BY v.created_at DESC,h.item_ref LIMIT ? OFFSET ?`, SupplyEditorActor, SupplyEditorImportCommand,
		node, len(prefix), prefix, len(prefix)+32,
		query, query, query, query, pageSize+1, page*pageSize)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	items := make([]SupplyEditorItem, 0, pageSize)
	for rows.Next() {
		item, err := scanSupplyEditorItem(rows)
		if err != nil {
			return nil, false, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	more := len(items) > pageSize
	if more {
		items = items[:pageSize]
	}
	return items, more, nil
}

func (s *Store) SupplyItemReferenced(ctx context.Context, ref string) (bool, error) {
	var one int
	err := s.DB.QueryRowContext(ctx, `SELECT 1 FROM supply_pools p
		LEFT JOIN supply_revisions r ON p.enabled AND r.pool_id=p.id AND r.revision=p.published_revision
		WHERE JSON_CONTAINS(JSON_EXTRACT(p.draft_json,'$.rewards[*].itemRef'),JSON_QUOTE(?))
		   OR JSON_CONTAINS(JSON_EXTRACT(r.definition_json,'$.rewards[*].itemRef'),JSON_QUOTE(?))
		LIMIT 1`, ref, ref).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}
