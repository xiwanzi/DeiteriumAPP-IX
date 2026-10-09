CREATE TABLE IF NOT EXISTS chat_media_objects (
  media_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
  sha256 CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  thumbnail_sha256 CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  canonical_hash CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL UNIQUE,
  content_md5 VARCHAR(24) CHARACTER SET ascii NOT NULL,
  thumbnail_md5 VARCHAR(24) CHARACTER SET ascii NOT NULL,
  size_bytes BIGINT NOT NULL,
  thumbnail_bytes BIGINT NOT NULL,
  width INT NOT NULL DEFAULT 0,
  height INT NOT NULL DEFAULT 0,
  animated BOOLEAN NOT NULL DEFAULT FALSE,
  frame_count INT NOT NULL DEFAULT 1,
  duration_ms INT NOT NULL DEFAULT 0,
  state VARCHAR(16) CHARACTER SET ascii NOT NULL DEFAULT 'UPLOADING',
  created_at DATETIME(6) NOT NULL,
  INDEX chat_media_gc(state,created_at)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS chat_media_uploads (
  upload_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
  user_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  client_request_id VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  fingerprint CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  media_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  staged_media_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  created_at DATETIME(6) NOT NULL,
  expires_at DATETIME(6) NOT NULL,
  UNIQUE KEY chat_media_request(user_id,client_request_id),
  INDEX chat_media_grant(media_id,expires_at),
  INDEX chat_media_upload_budget(user_id,created_at)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS chat_stickers (
  user_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  media_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  created_at DATETIME(6) NOT NULL,
  PRIMARY KEY(user_id,media_id),
  INDEX chat_sticker_media(media_id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS public_chat_media (
  message_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
  media_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  expires_at DATETIME(6) NOT NULL,
  INDEX public_chat_media_expiry(media_id,expires_at),
  CONSTRAINT fk_chat_media_message FOREIGN KEY(message_id) REFERENCES chat_messages_next(message_id) ON DELETE CASCADE
) ENGINE=InnoDB;

ALTER TABLE public_chat_metadata_v2 ADD COLUMN IF NOT EXISTS media_json TEXT NULL;
CREATE INDEX IF NOT EXISTS chat_messages_time ON chat_messages_next(sent_at);
