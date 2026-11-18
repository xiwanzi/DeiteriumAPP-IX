CREATE TABLE IF NOT EXISTS game_keybinding_profiles (
 player_uuid CHAR(36) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 game_id VARCHAR(64) NOT NULL,
 revision BIGINT NOT NULL DEFAULT 0,
 allow_remote BOOLEAN NOT NULL DEFAULT FALSE,
 synced_at DATETIME(6) NOT NULL,
 INDEX ix_keybinding_name(game_id,player_uuid)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS game_keybindings (
 player_uuid CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 action_id VARCHAR(192) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
 label VARCHAR(192) NOT NULL,
 category VARCHAR(192) NOT NULL,
 key_context VARCHAR(16) NOT NULL,
 desired VARCHAR(80) CHARACTER SET ascii NOT NULL,
 observed VARCHAR(80) CHARACTER SET ascii NOT NULL,
 conflict BOOLEAN NOT NULL DEFAULT FALSE,
 missing BOOLEAN NOT NULL DEFAULT FALSE,
 revision BIGINT NOT NULL,
 PRIMARY KEY(player_uuid,action_id),
 INDEX ix_keybinding_delta(player_uuid,revision),
 FOREIGN KEY(player_uuid) REFERENCES game_keybinding_profiles(player_uuid) ON DELETE CASCADE
) ENGINE=InnoDB;
