CREATE TABLE IF NOT EXISTS desktop_launcher_settings (
    id TINYINT PRIMARY KEY,
    version BIGINT NOT NULL DEFAULT 1,
    draft_json MEDIUMTEXT NOT NULL,
    published_json MEDIUMTEXT NULL,
    published_version BIGINT NOT NULL DEFAULT 0,
    published_index MEDIUMTEXT NULL,
    updated_at DATETIME(6) NOT NULL,
    published_at DATETIME(6) NULL
);
INSERT IGNORE INTO desktop_launcher_settings(id,version,draft_json,updated_at) VALUES(1,1,'{}',UTC_TIMESTAMP(6));

CREATE TABLE IF NOT EXISTS desktop_launcher_sync_runs (
    id VARCHAR(64) PRIMARY KEY,
    actor_id VARCHAR(64) NOT NULL,
    state VARCHAR(24) NOT NULL,
    message TEXT NOT NULL,
    created_at DATETIME(6) NOT NULL,
    updated_at DATETIME(6) NOT NULL,
    INDEX desktop_launcher_sync_recent(created_at)
);
