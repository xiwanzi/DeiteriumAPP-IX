CREATE TABLE IF NOT EXISTS app_launcher_icon_settings (
  id TINYINT PRIMARY KEY,
  icon_id VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  version BIGINT NOT NULL,
  updated_at DATETIME(6) NOT NULL
) ENGINE=InnoDB;

INSERT IGNORE INTO app_launcher_icon_settings(id,icon_id,version,updated_at)
VALUES(1,'default',1,UTC_TIMESTAMP(6));
