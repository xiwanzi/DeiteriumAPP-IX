CREATE TABLE IF NOT EXISTS social_profiles_v2 (
  user_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
  bio VARCHAR(200) NOT NULL DEFAULT '',
  avatar_json TEXT NULL,
  version BIGINT NOT NULL DEFAULT 1,
  updated_at DATETIME(6) NOT NULL,
  CONSTRAINT fk_social_profile_user FOREIGN KEY (user_id) REFERENCES identities(id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS social_follows_v2 (
  follower_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  followed_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  created_at DATETIME(6) NOT NULL,
  PRIMARY KEY (follower_id, followed_id),
  INDEX ix_social_followers (followed_id, follower_id),
  CONSTRAINT fk_social_follow_from FOREIGN KEY (follower_id) REFERENCES identities(id),
  CONSTRAINT fk_social_follow_to FOREIGN KEY (followed_id) REFERENCES identities(id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS social_requests_v2 (
  actor_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  action VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  request_id VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  fingerprint CHAR(64) CHARACTER SET ascii NOT NULL,
  response_json MEDIUMTEXT NULL,
  created_at DATETIME(6) NOT NULL,
  PRIMARY KEY (actor_id, action, request_id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS social_conversations_v2 (
  conversation_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
  user_lo VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  user_hi VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  version BIGINT NOT NULL DEFAULT 1,
  UNIQUE KEY uq_social_pair (user_lo, user_hi),
  CONSTRAINT fk_social_conversation_lo FOREIGN KEY (user_lo) REFERENCES identities(id),
  CONSTRAINT fk_social_conversation_hi FOREIGN KEY (user_hi) REFERENCES identities(id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS social_members_v2 (
  conversation_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  user_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  last_read_sequence BIGINT NOT NULL DEFAULT 0,
  PRIMARY KEY (conversation_id, user_id),
  INDEX ix_social_member_user (user_id, conversation_id),
  CONSTRAINT fk_social_member_conversation FOREIGN KEY (conversation_id) REFERENCES social_conversations_v2(conversation_id),
  CONSTRAINT fk_social_member_user FOREIGN KEY (user_id) REFERENCES identities(id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS social_messages_v2 (
  sequence_id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  message_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL UNIQUE,
  conversation_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  sender_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  client_message_id VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  content VARCHAR(256) NOT NULL,
  reply_json TEXT NULL,
  forwarded_json TEXT NULL,
  sent_at DATETIME(6) NOT NULL,
  UNIQUE KEY uq_social_message (sender_id, client_message_id),
  INDEX ix_social_conversation_messages (conversation_id, sequence_id),
  CONSTRAINT fk_social_message_conversation FOREIGN KEY (conversation_id) REFERENCES social_conversations_v2(conversation_id),
  CONSTRAINT fk_social_message_sender FOREIGN KEY (sender_id) REFERENCES identities(id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS social_notifications_v2 (
  sequence_id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  notification_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL UNIQUE,
  user_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  event_key VARCHAR(180) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  topic VARCHAR(32) CHARACTER SET ascii NOT NULL,
  title VARCHAR(100) NOT NULL,
  body VARCHAR(500) NOT NULL,
  target_json TEXT NOT NULL,
  created_at DATETIME(6) NOT NULL,
  read_at DATETIME(6) NULL,
  UNIQUE KEY uq_social_notification_event (user_id, event_key),
  INDEX ix_social_notification_user (user_id, sequence_id),
  CONSTRAINT fk_social_notification_user FOREIGN KEY (user_id) REFERENCES identities(id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS social_notification_preferences_v2 (
  user_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
  preferences_json TEXT NOT NULL,
  version BIGINT NOT NULL DEFAULT 1,
  CONSTRAINT fk_social_preferences_user FOREIGN KEY (user_id) REFERENCES identities(id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS social_announcements_v2 (
  sequence_id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  announcement_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL UNIQUE,
  draft_json MEDIUMTEXT NOT NULL,
  published_json MEDIUMTEXT NULL,
  state VARCHAR(16) CHARACTER SET ascii NOT NULL DEFAULT 'DRAFT',
  version BIGINT NOT NULL DEFAULT 1,
  published_version BIGINT NOT NULL DEFAULT 0,
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS social_send_budgets_v2 (
  user_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
  attempts INT NOT NULL,
  expires_at DATETIME(6) NOT NULL
) ENGINE=InnoDB;
