ALTER TABLE email_verifications ADD COLUMN IF NOT EXISTS profile_checked BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS minecraft_profile_checks (
 game_id VARCHAR(16) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 expires_at DATETIME(6) NOT NULL
) ENGINE=InnoDB;
