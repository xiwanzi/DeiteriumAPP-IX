CREATE TABLE IF NOT EXISTS promotion_attention_v209 (
    coupon_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    owner_uuid CHAR(36) CHARACTER SET ascii NOT NULL,
    announced_at DATETIME(6) NOT NULL,
    viewed_at DATETIME(6) NULL,
    PRIMARY KEY (coupon_id, owner_uuid),
    KEY idx_promotion_attention_owner (owner_uuid),
    CONSTRAINT fk_promotion_attention_coupon FOREIGN KEY (coupon_id) REFERENCES catalog_records_v2(resource_id)
) ENGINE=InnoDB;
