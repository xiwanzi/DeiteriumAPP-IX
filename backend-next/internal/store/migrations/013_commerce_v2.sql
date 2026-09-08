CREATE TABLE IF NOT EXISTS commerce_resources_v2 (
 sequence_id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
 resource_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL UNIQUE,
 resource_kind VARCHAR(16) CHARACTER SET ascii NOT NULL,
 channel VARCHAR(24) CHARACTER SET ascii NOT NULL,
 owner_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 owner_uuid CHAR(36) CHARACTER SET ascii NOT NULL,
 payee_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NULL,
 payee_uuid CHAR(36) CHARACTER SET ascii NULL,
 store_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
 quote_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL UNIQUE,
 escrow_ref VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL UNIQUE,
 amount VARCHAR(20) CHARACTER SET ascii NOT NULL,
 settled_amount VARCHAR(20) CHARACTER SET ascii NOT NULL,
 refunded_amount VARCHAR(20) CHARACTER SET ascii NOT NULL,
 state VARCHAR(32) CHARACTER SET ascii NOT NULL,
 funds_state VARCHAR(32) CHARACTER SET ascii NOT NULL,
 body MEDIUMTEXT NOT NULL,
 snapshot_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 snapshot_sha256 CHAR(64) CHARACTER SET ascii NOT NULL,
 version INT NOT NULL,
 pending_operation_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
 deadline_at DATETIME(6) NULL,
 deadline_kind VARCHAR(16) CHARACTER SET ascii NULL,
 paused_remaining_seconds BIGINT NULL,
 paused_deadline_kind VARCHAR(16) CHARACTER SET ascii NULL,
 refund_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
 refund_attempts INT NOT NULL,
 intervention_case_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
 automatic BOOLEAN NOT NULL,
 next_auto_attempt DATETIME(6) NULL,
 created_at DATETIME(6) NOT NULL,
 updated_at DATETIME(6) NOT NULL,
 INDEX ix_commerce_public (resource_kind,state,sequence_id),
 INDEX ix_commerce_owner (owner_id,resource_kind,sequence_id),
 INDEX ix_commerce_payee (payee_id,resource_kind,sequence_id),
 INDEX ix_commerce_store (store_id,sequence_id),
 INDEX ix_commerce_due (funds_state,pending_operation_id,deadline_at,next_auto_attempt),
 CONSTRAINT fk_commerce_owner FOREIGN KEY (owner_id) REFERENCES identities(id),
 CONSTRAINT fk_commerce_payee FOREIGN KEY (payee_id) REFERENCES identities(id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS commerce_snapshots_v2 (
 snapshot_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 resource_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 resource_kind VARCHAR(16) CHARACTER SET ascii NOT NULL,
 resource_version INT NOT NULL,
 body MEDIUMTEXT NOT NULL,
 sha256 CHAR(64) CHARACTER SET ascii NOT NULL,
 created_at DATETIME(6) NOT NULL,
 INDEX ix_commerce_snapshot_resource (resource_id,created_at),
 CONSTRAINT fk_commerce_snapshot_resource FOREIGN KEY (resource_id) REFERENCES commerce_resources_v2(resource_id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS commerce_refunds_v2 (
 refund_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 resource_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL UNIQUE,
 version INT NOT NULL,
 state VARCHAR(24) CHARACTER SET ascii NOT NULL,
 reason_code VARCHAR(32) CHARACTER SET ascii NOT NULL,
 description VARCHAR(500) NOT NULL,
 rejection_reason VARCHAR(500) NOT NULL,
 evidence_json TEXT NOT NULL,
 amount VARCHAR(20) CHARACTER SET ascii NOT NULL,
 immediate BOOLEAN NOT NULL,
 requested_at DATETIME(6) NOT NULL,
 resolved_at DATETIME(6) NULL,
 CONSTRAINT fk_commerce_refund_resource FOREIGN KEY (resource_id) REFERENCES commerce_resources_v2(resource_id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS commerce_requests_v2 (
 actor_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 client_request_id VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 scope VARCHAR(180) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 fingerprint CHAR(64) CHARACTER SET ascii NOT NULL,
 result TEXT NOT NULL,
 created_at DATETIME(6) NOT NULL,
 PRIMARY KEY (actor_id,client_request_id),
 CONSTRAINT fk_commerce_request_actor FOREIGN KEY (actor_id) REFERENCES identities(id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS commerce_operations_v2 (
 operation_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 resource_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 actor_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 client_request_id VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 operation_kind VARCHAR(32) CHARACTER SET ascii NOT NULL,
 action VARCHAR(32) CHARACTER SET ascii NOT NULL,
 amount VARCHAR(20) CHARACTER SET ascii NOT NULL,
 state VARCHAR(24) CHARACTER SET ascii NOT NULL,
 error_code VARCHAR(80) CHARACTER SET ascii NULL,
 steps MEDIUMTEXT NOT NULL,
 step_index INT NOT NULL,
 dispatch_token VARCHAR(64) CHARACTER SET ascii NULL,
 dispatch_until DATETIME(6) NULL,
 prior_state VARCHAR(32) CHARACTER SET ascii NOT NULL,
 prior_funds_state VARCHAR(32) CHARACTER SET ascii NOT NULL,
 automatic BOOLEAN NOT NULL,
 created_at DATETIME(6) NOT NULL,
 updated_at DATETIME(6) NOT NULL,
 UNIQUE KEY uq_commerce_operation_request (actor_id,client_request_id),
 INDEX ix_commerce_operation_work (state,dispatch_until,updated_at),
 CONSTRAINT fk_commerce_operation_resource FOREIGN KEY (resource_id) REFERENCES commerce_resources_v2(resource_id),
 CONSTRAINT fk_commerce_operation_actor FOREIGN KEY (actor_id) REFERENCES identities(id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS commerce_stock_holds_v2 (
 resource_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 product_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 quantity INT NOT NULL,
 state VARCHAR(16) CHARACTER SET ascii NOT NULL,
 PRIMARY KEY (resource_id,product_id),
 INDEX ix_commerce_stock_product (product_id,state),
 CONSTRAINT fk_commerce_stock_resource FOREIGN KEY (resource_id) REFERENCES commerce_resources_v2(resource_id),
 CONSTRAINT fk_commerce_stock_product FOREIGN KEY (product_id) REFERENCES catalog_records_v2(resource_id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS commerce_events_v2 (
 sequence_id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
 event_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL UNIQUE,
 resource_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 event_type VARCHAR(80) CHARACTER SET ascii NOT NULL,
 actor_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NULL,
 summary VARCHAR(1000) NOT NULL,
 metadata MEDIUMTEXT NOT NULL,
 created_at DATETIME(6) NOT NULL,
 INDEX ix_commerce_events_resource (resource_id,sequence_id),
 CONSTRAINT fk_commerce_event_resource FOREIGN KEY (resource_id) REFERENCES commerce_resources_v2(resource_id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS commerce_interventions_v2 (
 sequence_id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
 case_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL UNIQUE,
 resource_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 applicant_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 respondent_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NULL,
 state VARCHAR(24) CHARACTER SET ascii NOT NULL,
 version INT NOT NULL,
 body MEDIUMTEXT NOT NULL,
 snapshot MEDIUMTEXT NOT NULL,
 assigned_admin_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NULL,
 funds_held BOOLEAN NOT NULL,
 desired_refund_amount VARCHAR(20) CHARACTER SET ascii NOT NULL,
 target_refund_amount VARCHAR(20) CHARACTER SET ascii NULL,
 decision VARCHAR(32) CHARACTER SET ascii NULL,
 resolution VARCHAR(3000) NOT NULL,
 evidence_entries MEDIUMTEXT NOT NULL,
 created_at DATETIME(6) NOT NULL,
 updated_at DATETIME(6) NOT NULL,
 INDEX ix_commerce_case_resource (resource_id,sequence_id),
 UNIQUE KEY uq_commerce_case_transaction (resource_id),
 INDEX ix_commerce_case_status (state,sequence_id),
 CONSTRAINT fk_commerce_case_resource FOREIGN KEY (resource_id) REFERENCES commerce_resources_v2(resource_id),
 CONSTRAINT fk_commerce_case_applicant FOREIGN KEY (applicant_id) REFERENCES identities(id)
) ENGINE=InnoDB;
