ALTER TABLE identities
 ADD COLUMN IF NOT EXISTS profile_uuid CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NULL,
 ADD COLUMN IF NOT EXISTS profile_observed_at DATETIME(6) NULL,
 ADD UNIQUE INDEX IF NOT EXISTS identities_profile_uuid(profile_uuid);

UPDATE identities SET profile_uuid=server_uuid
 WHERE game_verified=TRUE AND server_uuid IS NOT NULL AND status<>'deleted' AND profile_uuid IS NULL;

ALTER TABLE email_verifications
 ADD COLUMN IF NOT EXISTS profile_uuid CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NULL;

ALTER TABLE minecraft_profile_checks
 ADD COLUMN IF NOT EXISTS profile_uuid CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NULL;
