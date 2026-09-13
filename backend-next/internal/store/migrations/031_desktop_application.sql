CREATE TABLE IF NOT EXISTS desktop_launcher_application (
    id TINYINT PRIMARY KEY,
    revision BIGINT NOT NULL DEFAULT 0,
    version_name VARCHAR(32) NOT NULL DEFAULT '',
    version_code BIGINT NOT NULL DEFAULT 0,
    envelope_json MEDIUMTEXT NULL,
    published_at DATETIME(6) NULL
);
INSERT IGNORE INTO desktop_launcher_application(id) VALUES(1);
