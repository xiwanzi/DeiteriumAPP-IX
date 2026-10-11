CREATE TABLE IF NOT EXISTS game_chat_avatars (
 player_uuid CHAR(36) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 png VARBINARY(4096) NOT NULL,
 refresh_after DATETIME(6) NOT NULL,
 updated_at DATETIME(6) NOT NULL
) ENGINE=InnoDB;
