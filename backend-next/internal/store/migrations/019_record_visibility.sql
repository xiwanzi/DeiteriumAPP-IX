CREATE TABLE personal_record_visibility_v203 (
    user_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    resource_kind VARCHAR(16) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    resource_id VARCHAR(96) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    hidden_at DATETIME(6) NOT NULL,
    PRIMARY KEY (user_id, resource_kind, resource_id)
) ENGINE=InnoDB;
