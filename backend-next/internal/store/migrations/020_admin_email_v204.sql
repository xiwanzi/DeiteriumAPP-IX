CREATE TABLE IF NOT EXISTS admin_email_settings_v204 (
 id TINYINT PRIMARY KEY,
 settings_json TEXT NOT NULL,
 password_cipher TEXT NOT NULL,
 version BIGINT NOT NULL,
 updated_at DATETIME(6) NOT NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS admin_email_outbox_v204 (
 event_id VARCHAR(100) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 case_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
 case_version BIGINT NOT NULL,
 case_state VARCHAR(32) CHARACTER SET ascii NOT NULL,
 status VARCHAR(16) CHARACTER SET ascii NOT NULL DEFAULT 'PENDING',
 attempts INT NOT NULL DEFAULT 0,
 next_attempt_at DATETIME(6) NOT NULL,
 lease_until DATETIME(6) NULL,
 last_error VARCHAR(300) NOT NULL DEFAULT '',
 created_at DATETIME(6) NOT NULL,
 sent_at DATETIME(6) NULL,
 INDEX ix_email_pending(status,next_attempt_at),
 INDEX ix_email_created(created_at,event_id)
) ENGINE=InnoDB;
