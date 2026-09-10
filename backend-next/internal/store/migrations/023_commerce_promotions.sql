CREATE TABLE IF NOT EXISTS promotion_redemptions_v209 (
 coupon_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 owner_uuid CHAR(36) CHARACTER SET ascii NOT NULL,
 resource_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 redeemed_at DATETIME(6) NULL,
 PRIMARY KEY (coupon_id, owner_uuid),
 INDEX ix_promotion_order (resource_id),
 CONSTRAINT fk_promotion_coupon FOREIGN KEY (coupon_id) REFERENCES catalog_records_v2(resource_id),
 CONSTRAINT fk_promotion_order FOREIGN KEY (resource_id) REFERENCES commerce_resources_v2(resource_id)
) ENGINE=InnoDB;

SET @dc_promotion_index_ddl = (SELECT IF(COUNT(*)=0,'CREATE INDEX ix_commerce_limits_v209 ON commerce_resources_v2(owner_uuid,channel,created_at)','DO 0') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='commerce_resources_v2' AND INDEX_NAME='ix_commerce_limits_v209');
PREPARE dc_promotion_index_statement FROM @dc_promotion_index_ddl;
EXECUTE dc_promotion_index_statement;
DEALLOCATE PREPARE dc_promotion_index_statement;
SET @dc_promotion_index_ddl = NULL;
