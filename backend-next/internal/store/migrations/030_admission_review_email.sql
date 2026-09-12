ALTER TABLE admin_email_outbox_v204
 ADD COLUMN IF NOT EXISTS application_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NULL,
 ADD COLUMN IF NOT EXISTS recipient VARCHAR(2560) CHARACTER SET ascii NOT NULL DEFAULT '',
 ADD COLUMN IF NOT EXISTS payload_json TEXT NULL;
CREATE INDEX IF NOT EXISTS ix_email_application ON admin_email_outbox_v204(application_id);
