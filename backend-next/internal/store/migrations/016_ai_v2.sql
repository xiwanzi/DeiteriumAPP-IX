CREATE TABLE IF NOT EXISTS ai_plans_v2 (
  plan_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
  code VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL UNIQUE,
  name VARCHAR(80) NOT NULL,
  description VARCHAR(1000) NOT NULL,
  price DECIMAL(18,2) NOT NULL,
  quota_limit INT NOT NULL,
  window_hours INT NOT NULL,
  duration_days INT NOT NULL,
  model_tier VARCHAR(32) CHARACTER SET ascii NOT NULL,
  active BOOLEAN NOT NULL,
  sort_order INT NOT NULL
) ENGINE=InnoDB;

INSERT IGNORE INTO ai_plans_v2 VALUES ('plan_free','free','ProMax','默认',0.00,20,24,0,'flash',TRUE,0),('plan_pro','pro','Pro','暂未开放',9999999.00,40,5,30,'flash',FALSE,1),('plan_ultra','ultra','AI Ultra','暂未开放',9999999.00,100,5,30,'flash',FALSE,2);

CREATE TABLE IF NOT EXISTS ai_conversations_v2 (
  conversation_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
  user_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  active BOOLEAN NOT NULL,
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  INDEX ix_ai_conversation_user_v2(user_id,active,updated_at),
  CONSTRAINT fk_ai_conversation_user_v2 FOREIGN KEY(user_id) REFERENCES identities(id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS ai_profiles_v2 (
  user_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
  conversation_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  active_request_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
  last_request_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
  updated_at DATETIME(6) NOT NULL,
  CONSTRAINT fk_ai_profile_user_v2 FOREIGN KEY(user_id) REFERENCES identities(id),
  CONSTRAINT fk_ai_profile_conversation_v2 FOREIGN KEY(conversation_id) REFERENCES ai_conversations_v2(conversation_id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS ai_quota_v2 (
  user_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  window_start DATETIME(6) NOT NULL,
  window_hours INT NOT NULL,
  used_count INT NOT NULL DEFAULT 0,
  reserved_count INT NOT NULL DEFAULT 0,
  updated_at DATETIME(6) NOT NULL,
  PRIMARY KEY(user_id,window_start,window_hours),
  CONSTRAINT fk_ai_quota_user_v2 FOREIGN KEY(user_id) REFERENCES identities(id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS ai_requests_v2 (
  request_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
  user_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  conversation_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  client_message_id VARCHAR(128) COLLATE utf8mb4_bin NOT NULL,
  fingerprint CHAR(64) CHARACTER SET ascii NOT NULL,
  input_content TEXT NOT NULL,
  user_message_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  assistant_message_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  status VARCHAR(16) CHARACTER SET ascii NOT NULL,
  progress VARCHAR(32) CHARACTER SET ascii NOT NULL,
  answer MEDIUMTEXT NOT NULL,
  sources_json MEDIUMTEXT NOT NULL,
  search_used BOOLEAN NOT NULL DEFAULT FALSE,
  finish_reason VARCHAR(64) CHARACTER SET ascii NOT NULL,
  error_code VARCHAR(64) CHARACTER SET ascii NOT NULL,
  provider_response_id VARCHAR(128) CHARACTER SET ascii NOT NULL,
  model VARCHAR(80) CHARACTER SET ascii NOT NULL,
  window_start DATETIME(6) NOT NULL,
  window_hours INT NOT NULL,
  quota_state VARCHAR(16) CHARACTER SET ascii NOT NULL,
  input_tokens BIGINT NOT NULL DEFAULT 0,
  output_tokens BIGINT NOT NULL DEFAULT 0,
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  deadline_at DATETIME(6) NOT NULL,
  UNIQUE KEY uq_ai_request_user_key_v2(user_id,client_message_id),
  INDEX ix_ai_request_expiry_v2(status,deadline_at),
  CONSTRAINT fk_ai_request_user_v2 FOREIGN KEY(user_id) REFERENCES identities(id),
  CONSTRAINT fk_ai_request_conversation_v2 FOREIGN KEY(conversation_id) REFERENCES ai_conversations_v2(conversation_id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS ai_messages_v2 (
  sequence_id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  message_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL UNIQUE,
  user_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  conversation_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  request_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  role VARCHAR(16) CHARACTER SET ascii NOT NULL,
  content MEDIUMTEXT NOT NULL,
  status VARCHAR(16) CHARACTER SET ascii NOT NULL,
  finish_reason VARCHAR(64) CHARACTER SET ascii NOT NULL,
  sources_json MEDIUMTEXT NOT NULL,
  search_used BOOLEAN NOT NULL DEFAULT FALSE,
  created_at DATETIME(6) NOT NULL,
  UNIQUE KEY uq_ai_request_role_v2(request_id,role),
  INDEX ix_ai_message_user_conversation_v2(user_id,conversation_id,sequence_id),
  CONSTRAINT fk_ai_message_user_v2 FOREIGN KEY(user_id) REFERENCES identities(id),
  CONSTRAINT fk_ai_message_conversation_v2 FOREIGN KEY(conversation_id) REFERENCES ai_conversations_v2(conversation_id),
  CONSTRAINT fk_ai_message_request_v2 FOREIGN KEY(request_id) REFERENCES ai_requests_v2(request_id)
) ENGINE=InnoDB;
