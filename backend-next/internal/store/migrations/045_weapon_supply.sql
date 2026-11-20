CREATE TABLE IF NOT EXISTS supply_pools (
 id VARCHAR(80) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 kind VARCHAR(16) CHARACTER SET ascii NOT NULL,
 series_id VARCHAR(80) CHARACTER SET ascii COLLATE ascii_bin NOT NULL DEFAULT '',
 draft_json LONGTEXT NOT NULL,
 version BIGINT NOT NULL,
 published_revision BIGINT NOT NULL DEFAULT 0,
 enabled BOOLEAN NOT NULL DEFAULT FALSE,
 starts_at DATETIME(6) NULL,
 ends_at DATETIME(6) NULL,
 visible_at DATETIME(6) NULL,
 created_at DATETIME(6) NOT NULL,
 updated_at DATETIME(6) NOT NULL,
 INDEX ix_supply_pool_kind(kind,enabled),
 INDEX ix_supply_pool_open(enabled,ends_at,visible_at)
) ENGINE=InnoDB;
CREATE TABLE IF NOT EXISTS supply_revisions (
 pool_id VARCHAR(80) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 revision BIGINT NOT NULL,
 definition_json LONGTEXT NOT NULL,
 actor_id VARCHAR(80) NOT NULL,
 created_at DATETIME(6) NOT NULL,
 PRIMARY KEY(pool_id,revision)
) ENGINE=InnoDB;
CREATE TABLE IF NOT EXISTS supply_event_rules (
 series_id VARCHAR(80) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 gold_ppm INT NOT NULL,
 hard_pity INT NOT NULL
) ENGINE=InnoDB;
CREATE TABLE IF NOT EXISTS supply_settings (
 id TINYINT PRIMARY KEY,
 version BIGINT NOT NULL,
 standard_price BIGINT NOT NULL DEFAULT 0,
 limited_price BIGINT NOT NULL DEFAULT 0
) ENGINE=InnoDB;
INSERT IGNORE INTO supply_settings(id,version) VALUES(1,1);
CREATE TABLE IF NOT EXISTS supply_accounts (
 player_uuid CHAR(36) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 game_id VARCHAR(32) NOT NULL,
 standard_tickets BIGINT NOT NULL DEFAULT 0,
 limited_tickets BIGINT NOT NULL DEFAULT 0,
 version BIGINT NOT NULL DEFAULT 1,
 updated_at DATETIME(6) NOT NULL
) ENGINE=InnoDB;
CREATE TABLE IF NOT EXISTS supply_pity (
 player_uuid CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 scope VARCHAR(120) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 misses INT NOT NULL,
 target INT NOT NULL,
 cycle BIGINT NOT NULL,
 PRIMARY KEY(player_uuid,scope)
) ENGINE=InnoDB;
CREATE TABLE IF NOT EXISTS supply_requests (
 actor_id VARCHAR(80) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 request_id VARCHAR(80) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 fingerprint CHAR(64) CHARACTER SET ascii NOT NULL,
 response_json LONGTEXT NULL,
 created_at DATETIME(6) NOT NULL,
 PRIMARY KEY(actor_id,request_id)
) ENGINE=InnoDB;
CREATE TABLE IF NOT EXISTS supply_orders (
 id VARCHAR(80) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 sequence_id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT UNIQUE,
 player_uuid CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 pool_id VARCHAR(80) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 pool_name VARCHAR(128) NOT NULL,
 revision BIGINT NOT NULL,
 scheme VARCHAR(16) CHARACTER SET ascii NOT NULL,
 currency VARCHAR(16) CHARACTER SET ascii NOT NULL,
 quantity INT NOT NULL,
 scope VARCHAR(120) CHARACTER SET ascii NOT NULL,
 rule_json LONGTEXT NOT NULL,
 definition_json LONGTEXT NOT NULL,
 result_json LONGTEXT NOT NULL,
 created_at DATETIME(6) NOT NULL,
 INDEX ix_supply_order_player(player_uuid,sequence_id),
 INDEX ix_supply_order_pool(pool_id,scheme,sequence_id),
 INDEX ix_supply_order_time(created_at,sequence_id)
) ENGINE=InnoDB;
CREATE TABLE IF NOT EXISTS supply_ledger (
 sequence_id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 player_uuid CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 currency VARCHAR(16) CHARACTER SET ascii NOT NULL,
 amount BIGINT NOT NULL,
 before_balance BIGINT NOT NULL,
 after_balance BIGINT NOT NULL,
 business_type VARCHAR(24) CHARACTER SET ascii NOT NULL,
 business_id VARCHAR(80) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 actor_id VARCHAR(80) NOT NULL,
 reason VARCHAR(512) NOT NULL,
 created_at DATETIME(6) NOT NULL,
 UNIQUE KEY uq_supply_ledger_business(player_uuid,currency,business_id),
 INDEX ix_supply_ledger_player(player_uuid,sequence_id),
 INDEX ix_supply_ledger_time(created_at,sequence_id)
) ENGINE=InnoDB;
CREATE TABLE IF NOT EXISTS supply_exchanges (
 id VARCHAR(80) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 player_uuid CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 currency VARCHAR(16) CHARACTER SET ascii NOT NULL,
 quantity BIGINT NOT NULL,
 unit_price BIGINT NOT NULL,
 total_price BIGINT NOT NULL,
 status VARCHAR(24) CHARACTER SET ascii NOT NULL,
 reserve_operation VARCHAR(80) CHARACTER SET ascii NULL,
 settle_operation VARCHAR(80) CHARACTER SET ascii NULL,
 error_message VARCHAR(512) NOT NULL DEFAULT '',
 created_at DATETIME(6) NOT NULL,
 updated_at DATETIME(6) NOT NULL,
 INDEX ix_supply_exchange_status(status,updated_at),
 INDEX ix_supply_exchange_player(player_uuid,created_at)
) ENGINE=InnoDB;
CREATE TABLE IF NOT EXISTS supply_deliveries (
 id VARCHAR(80) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 order_id VARCHAR(80) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 ordinal INT NOT NULL,
 player_uuid CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 reward_json LONGTEXT NOT NULL,
 state VARCHAR(24) CHARACTER SET ascii NOT NULL,
 node_id VARCHAR(64) CHARACTER SET ascii NOT NULL DEFAULT '',
 session_epoch VARCHAR(80) CHARACTER SET ascii NOT NULL DEFAULT '',
 inventory_epoch VARCHAR(160) CHARACTER SET ascii NOT NULL DEFAULT '',
 token CHAR(36) CHARACTER SET ascii NOT NULL DEFAULT '',
 lease_until DATETIME(6) NULL,
 receipt_json LONGTEXT NULL,
 last_error VARCHAR(512) NOT NULL DEFAULT '',
 mail_operation VARCHAR(80) CHARACTER SET ascii NULL,
 created_at DATETIME(6) NOT NULL,
 updated_at DATETIME(6) NOT NULL,
 UNIQUE KEY uq_supply_delivery_result(order_id,ordinal),
 INDEX ix_supply_delivery_player(player_uuid,state,updated_at),
 INDEX ix_supply_delivery_state(state,updated_at)
) ENGINE=InnoDB;
