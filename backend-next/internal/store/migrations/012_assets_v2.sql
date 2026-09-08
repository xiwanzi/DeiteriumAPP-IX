CREATE TABLE IF NOT EXISTS asset_uploads_v2 (
 upload_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 asset_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL UNIQUE,
 user_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 client_request_id VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 fingerprint CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 purpose VARCHAR(32) CHARACTER SET ascii NOT NULL,
 business_type VARCHAR(32) CHARACTER SET ascii NOT NULL,
 business_ref VARCHAR(128) CHARACTER SET ascii NOT NULL,
 object_key VARCHAR(255) CHARACTER SET ascii COLLATE ascii_bin NOT NULL UNIQUE,
 content_type VARCHAR(32) CHARACTER SET ascii NOT NULL,
 size_bytes BIGINT NOT NULL,
 content_md5 VARCHAR(24) CHARACTER SET ascii NOT NULL,
 alt_text VARCHAR(200) NOT NULL,
 status VARCHAR(20) CHARACTER SET ascii NOT NULL,
 rejection_code VARCHAR(80) CHARACTER SET ascii NULL,
 width INT NULL,
 height INT NULL,
 sha256 CHAR(64) CHARACTER SET ascii NULL,
 created_at DATETIME(6) NOT NULL,
 expires_at DATETIME(6) NOT NULL,
 verification_started_at DATETIME(6) NULL,
 verified_at DATETIME(6) NULL,
 removed_at DATETIME(6) NULL,
 UNIQUE KEY uq_asset_user_request (user_id,client_request_id),
 INDEX ix_assets_user_status (user_id,status,created_at),
 CONSTRAINT fk_asset_user FOREIGN KEY (user_id) REFERENCES identities(id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS asset_bindings_v2 (
 asset_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 business_type VARCHAR(32) CHARACTER SET ascii NOT NULL,
 business_ref VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 PRIMARY KEY (asset_id,business_type,business_ref),
 INDEX ix_asset_binding_resource (business_type,business_ref),
 CONSTRAINT fk_asset_binding FOREIGN KEY (asset_id) REFERENCES asset_uploads_v2(asset_id)
) ENGINE=InnoDB;
