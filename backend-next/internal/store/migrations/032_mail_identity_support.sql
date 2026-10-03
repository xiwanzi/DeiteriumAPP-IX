ALTER TABLE identities
 ADD COLUMN IF NOT EXISTS email VARCHAR(254) CHARACTER SET ascii COLLATE ascii_bin NULL UNIQUE,
 ADD COLUMN IF NOT EXISTS game_verified BOOLEAN NOT NULL DEFAULT TRUE,
 ADD COLUMN IF NOT EXISTS display_name VARCHAR(40) NULL,
 MODIFY COLUMN server_uuid CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NULL;
ALTER TABLE identity_aliases MODIFY COLUMN alias_key VARCHAR(254) CHARACTER SET ascii COLLATE ascii_bin NOT NULL;

CREATE TABLE IF NOT EXISTS mail_channels (
 channel_id VARCHAR(20) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 settings_json TEXT NOT NULL, password_cipher TEXT NOT NULL,
 version BIGINT NOT NULL, updated_at DATETIME(6) NOT NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS email_verifications (
 verification_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 token_hash CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL UNIQUE,
 code_hash CHAR(64) CHARACTER SET ascii NOT NULL,
 email VARCHAR(254) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 game_id VARCHAR(32) NOT NULL DEFAULT '',
 purpose VARCHAR(24) CHARACTER SET ascii NOT NULL,
 user_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NULL,
 state VARCHAR(20) CHARACTER SET ascii NOT NULL,
 attempts INT NOT NULL DEFAULT 0,
 expires_at DATETIME(6) NOT NULL, created_at DATETIME(6) NOT NULL,
 INDEX ix_email_verification (email,purpose,created_at)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS mail_outbox (
 event_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 channel_id VARCHAR(20) CHARACTER SET ascii NOT NULL,
 kind VARCHAR(32) CHARACTER SET ascii NOT NULL,
 recipient VARCHAR(254) CHARACTER SET ascii NOT NULL,
 subject VARCHAR(200) NOT NULL, payload_cipher MEDIUMTEXT NOT NULL,
 reference_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
 actor_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NULL,
 status VARCHAR(16) CHARACTER SET ascii NOT NULL DEFAULT 'PENDING',
 attempts INT NOT NULL DEFAULT 0,
 next_attempt_at DATETIME(6) NOT NULL, lease_until DATETIME(6) NULL,
 last_error VARCHAR(300) NOT NULL DEFAULT '', created_at DATETIME(6) NOT NULL, sent_at DATETIME(6) NULL,
 INDEX ix_mail_pending (status,next_attempt_at), INDEX ix_mail_reference (reference_id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS support_tickets (
 ticket_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 requester_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NULL,
 requester_email VARCHAR(254) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 requester_name VARCHAR(100) NOT NULL DEFAULT '',
 source VARCHAR(16) CHARACTER SET ascii NOT NULL DEFAULT 'EMAIL',
 subject VARCHAR(120) NOT NULL,
 category VARCHAR(24) CHARACTER SET ascii NOT NULL,
 status VARCHAR(24) CHARACTER SET ascii NOT NULL DEFAULT 'OPEN',
 priority VARCHAR(16) CHARACTER SET ascii NOT NULL DEFAULT 'NORMAL',
 assignee_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NULL,
 version BIGINT NOT NULL DEFAULT 1,
 created_at DATETIME(6) NOT NULL, updated_at DATETIME(6) NOT NULL,
 INDEX ix_support_requester (requester_id,updated_at), INDEX ix_support_queue (status,updated_at)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS support_messages (
 message_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 ticket_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 actor_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NULL,
 author_name VARCHAR(100) NOT NULL DEFAULT '',
 author_email VARCHAR(254) CHARACTER SET ascii NOT NULL DEFAULT '',
 direction VARCHAR(16) CHARACTER SET ascii NOT NULL DEFAULT 'INBOUND',
 mail_message_id VARCHAR(998) CHARACTER SET ascii NOT NULL DEFAULT '',
 visibility VARCHAR(16) CHARACTER SET ascii NOT NULL,
 body TEXT NOT NULL, created_at DATETIME(6) NOT NULL,
 INDEX ix_support_messages (ticket_id,created_at,message_id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS support_inbox (
 id TINYINT PRIMARY KEY,
 settings_json TEXT NOT NULL, password_cipher TEXT NOT NULL,
 version BIGINT NOT NULL DEFAULT 0,
 source_hash CHAR(64) CHARACTER SET ascii NOT NULL DEFAULT '',
 uid_validity BIGINT UNSIGNED NOT NULL DEFAULT 0,
 last_uid BIGINT UNSIGNED NOT NULL DEFAULT 0,
 last_poll_at DATETIME(6) NULL, last_error VARCHAR(300) NOT NULL DEFAULT '',
 lease_until DATETIME(6) NULL, lease_id VARCHAR(64) CHARACTER SET ascii NULL
) ENGINE=InnoDB;
INSERT IGNORE INTO support_inbox(id,settings_json,password_cipher) VALUES(1,'{}','');

CREATE TABLE IF NOT EXISTS support_inbound (
 source_hash CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 uid_validity BIGINT UNSIGNED NOT NULL, mail_uid BIGINT UNSIGNED NOT NULL,
 fingerprint CHAR(64) CHARACTER SET ascii NOT NULL,
 ticket_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
 status VARCHAR(20) CHARACTER SET ascii NOT NULL,
 reason VARCHAR(300) NOT NULL DEFAULT '',
 received_at DATETIME(6) NOT NULL,
 PRIMARY KEY(source_hash,uid_validity,mail_uid)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS support_mail_links (
 message_hash CHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 message_id VARCHAR(998) CHARACTER SET ascii NOT NULL,
 ticket_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 sender_email VARCHAR(254) CHARACTER SET ascii NOT NULL,
 direction VARCHAR(16) CHARACTER SET ascii NOT NULL,
 INDEX ix_support_thread(ticket_id), INDEX ix_support_mail_message(message_id(200))
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS support_attachments (
 attachment_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 ticket_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 message_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 filename VARCHAR(200) NOT NULL, mime_type VARCHAR(120) NOT NULL,
 size_bytes BIGINT NOT NULL, content_cipher LONGTEXT NOT NULL,
 INDEX ix_support_attachments(ticket_id,message_id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS support_submissions (
 submission_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 user_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 sender_email VARCHAR(254) CHARACTER SET ascii NOT NULL,
 email VARCHAR(254) CHARACTER SET ascii NOT NULL,
 name VARCHAR(100) NOT NULL, category VARCHAR(24) CHARACTER SET ascii NOT NULL,
 subject VARCHAR(120) NOT NULL, body TEXT NOT NULL,
 proof_hash CHAR(64) CHARACTER SET ascii NOT NULL,
 ticket_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
 created_at DATETIME(6) NOT NULL
) ENGINE=InnoDB;

ALTER TABLE mail_outbox
 ADD COLUMN IF NOT EXISTS reply_to VARCHAR(254) CHARACTER SET ascii NOT NULL DEFAULT '',
 ADD COLUMN IF NOT EXISTS in_reply_to VARCHAR(998) CHARACTER SET ascii NOT NULL DEFAULT '',
 ADD COLUMN IF NOT EXISTS headers_cipher TEXT NULL;

CREATE TABLE IF NOT EXISTS support_ticket_days (day_key CHAR(8) CHARACTER SET ascii PRIMARY KEY) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS support_macros (
 macro_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 title VARCHAR(100) NOT NULL, body TEXT NOT NULL,
 version BIGINT NOT NULL DEFAULT 1,
 updated_at DATETIME(6) NOT NULL
) ENGINE=InnoDB;
