CREATE TABLE IF NOT EXISTS game_verifications (
  verification_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
  token_hash CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL UNIQUE,
  code_hash CHAR(64) CHARACTER SET ascii NOT NULL,
  purpose VARCHAR(24) CHARACTER SET ascii NOT NULL,
  player_uuid CHAR(36) CHARACTER SET ascii NOT NULL,
  game_id VARCHAR(32) NOT NULL,
  qq VARCHAR(20) CHARACTER SET ascii NOT NULL,
  user_id VARCHAR(40) CHARACTER SET ascii NULL,
  node_id VARCHAR(32) CHARACTER SET ascii NOT NULL,
  state VARCHAR(20) CHARACTER SET ascii NOT NULL,
  attempts INT NOT NULL DEFAULT 0,
  expires_at DATETIME(6) NOT NULL,
  created_at DATETIME(6) NOT NULL,
  consumed_at DATETIME(6) NULL,
  INDEX ix_game_verification_player (player_uuid,purpose,created_at),
  INDEX ix_game_verification_expiry (expires_at)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS game_verification_cooldowns (
  cooldown_key CHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
  expires_at DATETIME(6) NOT NULL
) ENGINE=InnoDB;
