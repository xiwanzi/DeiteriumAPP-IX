ALTER TABLE identity_sessions
 MODIFY COLUMN expires_at DATETIME(6) NULL,
 ADD COLUMN IF NOT EXISTS session_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NULL,
 ADD COLUMN IF NOT EXISTS device_name VARCHAR(160) NOT NULL DEFAULT '',
 ADD COLUMN IF NOT EXISTS last_seen_at DATETIME(6) NULL;

UPDATE identity_sessions SET session_id=CONCAT('ses_',LEFT(SHA2(CONCAT('session-display:',token_hash),256),32))
 WHERE session_id IS NULL;
UPDATE identity_sessions SET last_seen_at=created_at WHERE last_seen_at IS NULL;

ALTER TABLE identity_sessions
 MODIFY COLUMN session_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 ADD UNIQUE INDEX IF NOT EXISTS ix_session_id(session_id);

-- Only still-valid sessions are extended. Revoked/expired tokens stay invalid.
UPDATE identity_sessions s JOIN identities i ON i.id=s.user_id SET s.expires_at=NULL
 WHERE s.client_kind='app' AND s.revoked_at IS NULL AND s.expires_at>UTC_TIMESTAMP(6) AND i.status='active';
UPDATE identity_sessions s JOIN identities i ON i.id=s.user_id SET s.expires_at=DATE_ADD(s.created_at,INTERVAL 30 DAY)
 WHERE s.client_kind='web' AND s.revoked_at IS NULL AND s.expires_at>UTC_TIMESTAMP(6) AND i.status='active'
 AND s.expires_at<DATE_ADD(s.created_at,INTERVAL 30 DAY);
