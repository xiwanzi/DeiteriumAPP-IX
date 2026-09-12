CREATE TABLE IF NOT EXISTS admission_guard (
  id INT PRIMARY KEY
) ENGINE=InnoDB;
INSERT IGNORE INTO admission_guard VALUES (1);

CREATE TABLE IF NOT EXISTS admission_applications (
  application_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
  receipt_hash CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL UNIQUE,
  fingerprint CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  server_uuid CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  pending_uuid CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NULL UNIQUE,
  game_id VARCHAR(16) NOT NULL,
  qq VARCHAR(11) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  interests_json TEXT NOT NULL,
  message VARCHAR(800) NOT NULL,
  covenant_version VARCHAR(40) CHARACTER SET ascii NOT NULL,
  status VARCHAR(16) CHARACTER SET ascii NOT NULL,
  version BIGINT NOT NULL DEFAULT 1,
  created_at DATETIME(6) NOT NULL,
  reviewed_at DATETIME(6) NULL,
  reviewer_id VARCHAR(40) CHARACTER SET ascii NULL,
  reason VARCHAR(500) NOT NULL DEFAULT '',
  INDEX ix_admission_application_status (status,created_at),
  INDEX ix_admission_application_uuid (server_uuid,created_at)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS admission_entries (
  server_uuid CHAR(36) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
  game_id VARCHAR(16) NOT NULL,
  qq VARCHAR(11) CHARACTER SET ascii COLLATE ascii_bin NOT NULL DEFAULT '',
  status VARCHAR(16) CHARACTER SET ascii NOT NULL,
  source VARCHAR(16) CHARACTER SET ascii NOT NULL,
  application_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NULL,
  reason VARCHAR(500) NOT NULL DEFAULT '',
  version BIGINT NOT NULL DEFAULT 1,
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  INDEX ix_admission_entry_status (status,updated_at),
  INDEX ix_admission_entry_name (game_id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS admission_events (
  sequence_id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  server_uuid CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  actor_id VARCHAR(64) CHARACTER SET ascii NOT NULL,
  action VARCHAR(32) CHARACTER SET ascii NOT NULL,
  application_id VARCHAR(40) CHARACTER SET ascii NULL,
  reason VARCHAR(500) NOT NULL DEFAULT '',
  created_at DATETIME(6) NOT NULL,
  INDEX ix_admission_events_uuid (server_uuid,sequence_id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS admission_kicks (
  command_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
  server_uuid CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  entry_version BIGINT NOT NULL,
  reason VARCHAR(500) NOT NULL,
  status VARCHAR(24) CHARACTER SET ascii NOT NULL,
  created_at DATETIME(6) NOT NULL,
  completed_at DATETIME(6) NULL,
  INDEX ix_admission_kick_status (status,created_at),
  INDEX ix_admission_kick_uuid (server_uuid,created_at)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS admission_gateway (
  id INT PRIMARY KEY,
  instance_id VARCHAR(64) CHARACTER SET ascii NOT NULL,
  plugin_version VARCHAR(32) CHARACTER SET ascii NOT NULL,
  online_players INT NOT NULL,
  last_seen DATETIME(6) NOT NULL
) ENGINE=InnoDB;
