ALTER TABLE asset_uploads_v2
 ADD COLUMN lifecycle_state VARCHAR(16) CHARACTER SET ascii NOT NULL DEFAULT 'ACTIVE',
 ADD COLUMN was_bound BOOLEAN NOT NULL DEFAULT FALSE,
 ADD COLUMN source_object_key VARCHAR(255) CHARACTER SET ascii COLLATE ascii_bin NULL,
 ADD COLUMN pending_object_key VARCHAR(255) CHARACTER SET ascii COLLATE ascii_bin NULL,
 ADD COLUMN retired_at DATETIME(6) NULL,
 ADD COLUMN retain_until DATETIME(6) NULL,
 ADD COLUMN lifecycle_keep_until DATETIME(6) NULL,
 ADD COLUMN lifecycle_checked_at DATETIME(6) NULL,
 ADD COLUMN lifecycle_retry_at DATETIME(6) NULL,
 ADD COLUMN lifecycle_error VARCHAR(64) CHARACTER SET ascii NULL,
 ADD INDEX ix_asset_lifecycle_work (lifecycle_state,lifecycle_checked_at,lifecycle_retry_at);

UPDATE asset_uploads_v2 a JOIN asset_bindings_v2 b ON b.asset_id=a.asset_id SET a.was_bound=TRUE;

UPDATE asset_uploads_v2 a
 SET a.was_bound=TRUE
 WHERE a.purpose='AVATAR' AND a.status='READY' AND EXISTS (
  SELECT 1 FROM social_requests_v2 r
  WHERE r.actor_id=a.user_id AND r.action='profile.patch' AND JSON_VALID(r.response_json)
   AND JSON_UNQUOTE(JSON_EXTRACT(r.response_json,'$.avatar.assetId'))=a.asset_id
 );
