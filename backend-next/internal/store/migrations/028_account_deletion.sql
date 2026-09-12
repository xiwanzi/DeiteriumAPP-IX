CREATE TABLE account_lifecycle_guard (
  id INT PRIMARY KEY
) ENGINE=InnoDB;
INSERT INTO account_lifecycle_guard VALUES (1);

CREATE TABLE account_deletions (
  sequence_id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  user_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NOT NULL UNIQUE,
  player_ref VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL UNIQUE,
  original_uuid CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  deleted_at DATETIME(6) NOT NULL,
  INDEX ix_deleted_uuid (original_uuid,deleted_at),
  CONSTRAINT fk_deleted_account FOREIGN KEY (user_id) REFERENCES identities(id)
) ENGINE=InnoDB;

CREATE INDEX ix_identity_status ON identities(status);
