CREATE TABLE IF NOT EXISTS core_catalog_heads (
  item_ref VARCHAR(96) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
  catalog_version BIGINT NOT NULL,
  latest_revision BIGINT NOT NULL,
  archived BOOLEAN NOT NULL,
  fingerprint CHAR(64) CHARACTER SET ascii NOT NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS core_operations (
  operation_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
  actor_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  client_request_id VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  node_id VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  command_type VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  fingerprint CHAR(64) CHARACTER SET ascii NOT NULL,
  payload MEDIUMTEXT NOT NULL,
  state VARCHAR(24) CHARACTER SET ascii NOT NULL,
  result MEDIUMTEXT NULL,
  expires_at DATETIME(6) NOT NULL,
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  UNIQUE KEY uq_core_operation_request (actor_id, client_request_id),
  INDEX ix_core_operation_node (node_id,state,created_at)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS core_player_directory (
  player_ref VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
  player_uuid CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL UNIQUE,
  game_id VARCHAR(32) NOT NULL,
  node_id VARCHAR(32) CHARACTER SET ascii NOT NULL,
  last_seen DATETIME(6) NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS core_mail_receipts (
  delivery_id VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
  order_id VARCHAR(128) CHARACTER SET ascii NOT NULL,
  mail_cluster VARCHAR(64) CHARACTER SET ascii NOT NULL,
  snapshot_sha256 CHAR(64) CHARACTER SET ascii NOT NULL,
  recipient_uuid CHAR(36) CHARACTER SET ascii NOT NULL,
  status VARCHAR(24) CHARACTER SET ascii NOT NULL,
  revision BIGINT NOT NULL,
  receipt MEDIUMTEXT NOT NULL,
  updated_at DATETIME(6) NOT NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS core_manual_deliveries (
  delivery_id VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
  actor_id VARCHAR(40) CHARACTER SET ascii NOT NULL,
  operation_id VARCHAR(64) CHARACTER SET ascii NOT NULL UNIQUE,
  snapshot MEDIUMTEXT NOT NULL,
  created_at DATETIME(6) NOT NULL
) ENGINE=InnoDB;
