CREATE TABLE IF NOT EXISTS mail_invalid_addresses (
 event_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 email VARCHAR(254) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 source VARCHAR(20) CHARACTER SET ascii NOT NULL,
 reported_at DATETIME(6) NOT NULL,
 blocked_until DATETIME(6) NOT NULL,
 INDEX ix_invalid_email (email,blocked_until)
) ENGINE=InnoDB;
