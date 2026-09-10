CREATE TABLE IF NOT EXISTS promotion_coupon_batches_v209 (
    coupon_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    batch_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    released_at DATETIME(6) NOT NULL,
    PRIMARY KEY (coupon_id),
    KEY idx_coupon_release_batch (batch_id),
    CONSTRAINT fk_coupon_release_coupon FOREIGN KEY (coupon_id) REFERENCES catalog_records_v2(resource_id)
) ENGINE=InnoDB;
