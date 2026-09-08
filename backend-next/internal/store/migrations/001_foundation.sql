CREATE TABLE IF NOT EXISTS identities (
  id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
  player_ref VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL UNIQUE,
  server_uuid CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL UNIQUE,
  game_id VARCHAR(32) NOT NULL,
  qq VARCHAR(20) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  password_hash VARCHAR(255) CHARACTER SET ascii NOT NULL,
  status VARCHAR(20) CHARACTER SET ascii NOT NULL,
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  legacy_fingerprint CHAR(64) CHARACTER SET ascii NOT NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS identity_aliases (
  alias_key VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
  user_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  CONSTRAINT fk_alias_user FOREIGN KEY (user_id) REFERENCES identities(id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS identity_sessions (
  token_hash CHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
  user_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  client_kind VARCHAR(16) CHARACTER SET ascii NOT NULL,
  csrf_token VARCHAR(64) CHARACTER SET ascii NOT NULL,
  created_at DATETIME(6) NOT NULL,
  expires_at DATETIME(6) NOT NULL,
  revoked_at DATETIME(6) NULL,
  INDEX ix_session_user (user_id, revoked_at),
  INDEX ix_session_expiry (expires_at),
  CONSTRAINT fk_session_user FOREIGN KEY (user_id) REFERENCES identities(id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS auth_budgets (
  budget_key CHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
  attempts INT NOT NULL,
  expires_at DATETIME(6) NOT NULL,
  INDEX ix_auth_expiry (expires_at)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS identity_permissions (
  user_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  permission VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  PRIMARY KEY (user_id, permission),
  CONSTRAINT fk_permission_user FOREIGN KEY (user_id) REFERENCES identities(id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS core_events (
  sequence_id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  source_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  event_id VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  event_type VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  fingerprint CHAR(64) CHARACTER SET ascii NOT NULL,
  payload MEDIUMTEXT NOT NULL,
  created_at DATETIME(6) NOT NULL,
  UNIQUE KEY uq_core_event (source_id, event_id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS chat_messages_next (
  sequence_id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  message_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL UNIQUE,
  source_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  client_message_id VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  fingerprint CHAR(64) CHARACTER SET ascii NOT NULL,
  sender_uuid CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  sender_name VARCHAR(32) NOT NULL,
  sender_ref VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  registered BOOLEAN NOT NULL,
  content VARCHAR(256) NOT NULL,
  kind VARCHAR(32) CHARACTER SET ascii NOT NULL,
  sent_at DATETIME(6) NOT NULL,
  UNIQUE KEY uq_chat_submission (source_id, client_message_id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS core_chat_deliveries (
  node_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  message_sequence BIGINT NOT NULL,
  acknowledged_at DATETIME(6) NULL,
  expires_at DATETIME(6) NOT NULL,
  PRIMARY KEY (node_id, message_sequence),
  INDEX ix_core_chat_pending (node_id, acknowledged_at, expires_at, message_sequence),
  INDEX ix_core_chat_expiry (expires_at),
  CONSTRAINT fk_chat_delivery FOREIGN KEY (message_sequence) REFERENCES chat_messages_next(sequence_id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS item_versions (
  item_ref VARCHAR(96) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  revision BIGINT NOT NULL,
  publisher_node VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  payload_sha256 CHAR(64) CHARACTER SET ascii NOT NULL,
  metadata TEXT NOT NULL,
  fingerprint CHAR(64) CHARACTER SET ascii NOT NULL,
  created_at DATETIME(6) NOT NULL,
  PRIMARY KEY (item_ref, revision)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS audit_events_next (
  sequence_id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  actor_id VARCHAR(64) CHARACTER SET ascii NOT NULL,
  action VARCHAR(64) CHARACTER SET ascii NOT NULL,
  resource_id VARCHAR(128) CHARACTER SET ascii NOT NULL,
  created_at DATETIME(6) NOT NULL
) ENGINE=InnoDB;
