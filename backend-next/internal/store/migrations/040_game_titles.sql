CREATE TABLE game_title_catalog (
 id VARCHAR(64) PRIMARY KEY,
 internal_id VARCHAR(48) CHARACTER SET ascii COLLATE ascii_bin NOT NULL UNIQUE,
 display_name VARCHAR(512) NOT NULL,
 weight INT NOT NULL DEFAULT 0,
 status VARCHAR(16) NOT NULL,
 publish_sequence BIGINT NOT NULL DEFAULT 0,
 version BIGINT NOT NULL DEFAULT 1,
 description TEXT NOT NULL,
 is_public BOOLEAN NOT NULL DEFAULT TRUE
) ENGINE=InnoDB;
CREATE TABLE game_title_revision (id INT PRIMARY KEY, revision BIGINT NOT NULL) ENGINE=InnoDB;
INSERT INTO game_title_revision VALUES (1,1);
INSERT INTO game_title_catalog VALUES ('title_default','default','溯行者',0,'PUBLISHED',1,1,'每位玩家均可使用的默认头衔。',TRUE);
CREATE TABLE game_title_players (
 player_uuid CHAR(36) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 game_id VARCHAR(64) NOT NULL,
 title_id VARCHAR(64) NOT NULL DEFAULT 'title_default',
 history_cursor BIGINT NOT NULL DEFAULT 0,
 version BIGINT NOT NULL DEFAULT 1
) ENGINE=InnoDB;
CREATE TABLE game_title_history (
 sequence_id BIGINT AUTO_INCREMENT PRIMARY KEY,
 player_uuid CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 title_id VARCHAR(64) NOT NULL,
 INDEX player_history(player_uuid,sequence_id)
) ENGINE=InnoDB;
CREATE TABLE game_title_requests (
 player_uuid CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 request_id VARCHAR(100) NOT NULL,
 fingerprint CHAR(64) NOT NULL,
 PRIMARY KEY(player_uuid,request_id)
) ENGINE=InnoDB;
