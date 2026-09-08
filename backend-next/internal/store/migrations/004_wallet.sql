CREATE TABLE IF NOT EXISTS wallet_transfers_next (
 transfer_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 actor_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 client_request_id VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 fingerprint CHAR(64) CHARACTER SET ascii NOT NULL,
 from_uuid CHAR(36) CHARACTER SET ascii NOT NULL,
 to_uuid CHAR(36) CHARACTER SET ascii NOT NULL,
 recipient_ref VARCHAR(64) CHARACTER SET ascii NOT NULL,
 amount DECIMAL(16,2) NOT NULL,
 note VARCHAR(80) NULL,
 operation_id VARCHAR(64) CHARACTER SET ascii NOT NULL UNIQUE,
 created_at DATETIME(6) NOT NULL,
 UNIQUE KEY uq_wallet_transfer_request(actor_id,client_request_id),
 INDEX ix_wallet_transfer_from(from_uuid,created_at),
 INDEX ix_wallet_transfer_to(to_uuid,created_at)
) ENGINE=InnoDB;
