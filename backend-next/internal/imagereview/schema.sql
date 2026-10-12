CREATE TABLE IF NOT EXISTS image_review_items (
 id CHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 source_kind VARCHAR(16) CHARACTER SET ascii NOT NULL,
 source_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 media_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL DEFAULT '',
 source_key VARCHAR(512) NOT NULL DEFAULT '',
 sha256 CHAR(64) CHARACTER SET ascii NOT NULL DEFAULT '',
 verified BOOLEAN NOT NULL DEFAULT FALSE,
 content_type VARCHAR(32) CHARACTER SET ascii NOT NULL DEFAULT '',
 size_bytes BIGINT NOT NULL DEFAULT 0,
 animated BOOLEAN NOT NULL DEFAULT FALSE,
 uploader_id VARCHAR(64) NOT NULL DEFAULT '',
 player_ref VARCHAR(64) NOT NULL DEFAULT '',
 game_id VARCHAR(64) NOT NULL DEFAULT '',
 display_name VARCHAR(128) NOT NULL DEFAULT '',
 purpose VARCHAR(32) NOT NULL DEFAULT '',
 uploaded_at DATETIME(6) NOT NULL,
 expires_at DATETIME(6) NOT NULL,
 source_state VARCHAR(20) NOT NULL DEFAULT '',
 state VARCHAR(20) CHARACTER SET ascii NOT NULL DEFAULT 'PENDING',
 score INT NULL,
 result_json JSON NULL,
 policy_version VARCHAR(40) NOT NULL DEFAULT '',
 model VARCHAR(64) NOT NULL DEFAULT '',
 evidence_key VARCHAR(512) NOT NULL DEFAULT '',
 evidence_etag VARCHAR(128) NOT NULL DEFAULT '',
 error_code VARCHAR(64) NOT NULL DEFAULT '',
 attempts INT NOT NULL DEFAULT 0,
 next_attempt_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
 reviewed_at DATETIME(6) NULL,
 events_checked_at DATETIME(6) NULL,
 created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
 updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
 UNIQUE KEY image_review_source(source_kind,source_id),
 INDEX image_review_work(state,next_attempt_at),
 INDEX image_review_list(state,score,uploaded_at,id),
 INDEX image_review_digest(sha256,policy_version,state)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS image_review_events (
 item_id CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 event_id VARCHAR(160) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 channel VARCHAR(32) NOT NULL,
 sender_id VARCHAR(64) NOT NULL DEFAULT '',
 player_ref VARCHAR(64) NOT NULL DEFAULT '',
 game_id VARCHAR(64) NOT NULL DEFAULT '',
 display_name VARCHAR(128) NOT NULL DEFAULT '',
 sent_at DATETIME(6) NOT NULL,
 PRIMARY KEY(item_id,event_id),
 INDEX image_review_event_time(item_id,sent_at)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS image_review_runtime (
 id INT PRIMARY KEY,
 started_at DATETIME(6) NULL,
 last_scan_at DATETIME(6) NULL,
 last_work_at DATETIME(6) NULL,
 scan_error VARCHAR(64) NOT NULL DEFAULT ''
) ENGINE=InnoDB;
INSERT IGNORE INTO image_review_runtime(id) VALUES(1);

CREATE TABLE IF NOT EXISTS image_review_settings (
 id INT PRIMARY KEY,
 threshold INT NOT NULL,
 prompt TEXT NOT NULL,
 version BIGINT NOT NULL DEFAULT 1,
 updated_at DATETIME(6) NULL,
 updated_by VARCHAR(64) NOT NULL DEFAULT ''
) ENGINE=InnoDB;
ALTER TABLE image_review_items ADD COLUMN IF NOT EXISTS threshold_used INT NOT NULL DEFAULT 50;
