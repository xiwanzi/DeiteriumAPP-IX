ALTER TABLE social_messages_v2 ADD COLUMN media_json JSON NULL;
CREATE TABLE direct_chat_media (
  message_id VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
  media_id VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  expires_at DATETIME(6) NOT NULL,
  INDEX direct_media_reference (media_id, expires_at),
  FOREIGN KEY (message_id) REFERENCES social_messages_v2(message_id) ON DELETE CASCADE
);
CREATE TABLE game_chat_messages (
  sequence_id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  message_id VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL UNIQUE,
  sender_id VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  client_message_id VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  fingerprint CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  channel VARCHAR(16) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  message_json JSON NOT NULL,
  sent_at DATETIME(6) NOT NULL,
  UNIQUE KEY game_chat_send (sender_id,client_message_id)
);
CREATE TABLE game_chat_recipients (
  message_id VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  player_uuid CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  PRIMARY KEY(message_id,player_uuid),
  INDEX game_chat_recipient(player_uuid,message_id),
  FOREIGN KEY (message_id) REFERENCES game_chat_messages(message_id) ON DELETE CASCADE
);
CREATE TABLE game_chat_media (
  message_id VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
  media_id VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  expires_at DATETIME(6) NOT NULL,
  INDEX game_media_reference (media_id, expires_at),
  FOREIGN KEY (message_id) REFERENCES game_chat_messages(message_id) ON DELETE CASCADE
);
