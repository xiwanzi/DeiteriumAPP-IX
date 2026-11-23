CREATE TABLE IF NOT EXISTS supply_editor_ticket_catalogs (
 source_revision BIGINT PRIMARY KEY,
 tickets_json LONGTEXT NOT NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS supply_custom_ticket_balances (
 player_uuid CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 currency VARCHAR(80) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 balance BIGINT NOT NULL,
 PRIMARY KEY (player_uuid,currency)
) ENGINE=InnoDB;

ALTER TABLE supply_orders MODIFY COLUMN currency VARCHAR(80) CHARACTER SET ascii COLLATE ascii_bin NOT NULL;
ALTER TABLE supply_ledger MODIFY COLUMN currency VARCHAR(80) CHARACTER SET ascii COLLATE ascii_bin NOT NULL;
ALTER TABLE supply_exchanges MODIFY COLUMN currency VARCHAR(80) CHARACTER SET ascii COLLATE ascii_bin NOT NULL;

START TRANSACTION;
INSERT IGNORE INTO supply_editor_ticket_catalogs (source_revision,tickets_json)
SELECT source_revision,JSON_EXTRACT(catalog_json,'$.tickets')
FROM supply_editor_publications
WHERE JSON_TYPE(JSON_EXTRACT(catalog_json,'$.tickets'))='ARRAY'
ORDER BY source_revision DESC LIMIT 1;
UPDATE supply_settings SET version=version+1 WHERE id=1 AND ROW_COUNT()>0;
COMMIT;
