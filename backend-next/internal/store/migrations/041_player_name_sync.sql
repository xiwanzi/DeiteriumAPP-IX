CREATE TABLE IF NOT EXISTS player_names_state (
 player_uuid CHAR(36) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 player_ref VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 game_id VARCHAR(32) NOT NULL,
 version BIGINT NOT NULL DEFAULT 0,
 observed_at DATETIME(6) NOT NULL DEFAULT '1970-01-01 00:00:00',
 updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
 INDEX player_name_reference(player_ref)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS player_name_changes (
 sequence_id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
 player_uuid CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 player_ref VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 previous_game_id VARCHAR(32) NOT NULL,
 game_id VARCHAR(32) NOT NULL,
 changed_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
 INDEX player_name_latest(player_uuid,sequence_id)
) ENGINE=InnoDB;
