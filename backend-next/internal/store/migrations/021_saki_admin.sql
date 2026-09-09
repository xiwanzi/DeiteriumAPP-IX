ALTER TABLE identities ADD COLUMN admin_version BIGINT NOT NULL DEFAULT 1,
  ADD COLUMN ban_reason VARCHAR(500) NOT NULL DEFAULT '';

CREATE TABLE admin_account_guard (
  id INT PRIMARY KEY
) ENGINE=InnoDB;
INSERT INTO admin_account_guard VALUES (1);

CREATE TABLE ai_settings_v206 (
  id INT PRIMARY KEY,
  settings_json MEDIUMTEXT NOT NULL,
  version BIGINT NOT NULL,
  updated_at DATETIME(6) NOT NULL
) ENGINE=InnoDB;

ALTER TABLE ai_plans_v2 ADD COLUMN version BIGINT NOT NULL DEFAULT 1;

CREATE TABLE ai_entitlements_v206 (
  user_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
  plan_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  plan_snapshot TEXT NOT NULL,
  expires_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  CONSTRAINT fk_ai_entitlement_user FOREIGN KEY(user_id) REFERENCES identities(id)
) ENGINE=InnoDB;

ALTER TABLE social_notifications_v2 ADD COLUMN system_push BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE social_notifications_v2 ADD COLUMN inbox_visible BOOLEAN NOT NULL DEFAULT TRUE;
UPDATE social_notifications_v2 n JOIN commerce_events_v2 e ON e.event_id=n.event_key
 SET n.inbox_visible=FALSE,n.system_push=FALSE
 WHERE e.event_type LIKE 'operation.%' OR e.event_type IN ('order.payment.started','commission.funding.started','commission.accept.reserved','settlement.started','delivery.retry.started','mailbox.observed');
UPDATE social_notifications_v2 SET system_push=FALSE WHERE topic='STORE_ORDERS';

CREATE TABLE wallet_notice_cursor_v206 (
 id INT PRIMARY KEY,
 from_time DATETIME(6) NOT NULL,
 until_time DATETIME(6) NULL,
 before_time DATETIME(6) NULL,
 before_sequence VARCHAR(24) NOT NULL DEFAULT '0',
 snapshot VARCHAR(24) NOT NULL DEFAULT '0',
 version BIGINT NOT NULL DEFAULT 1
) ENGINE=InnoDB;
INSERT INTO wallet_notice_cursor_v206(id,from_time) VALUES(1,UTC_TIMESTAMP(6));
