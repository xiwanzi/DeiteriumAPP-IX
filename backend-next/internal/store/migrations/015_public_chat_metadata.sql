CREATE TABLE IF NOT EXISTS public_chat_metadata_v2 (
  message_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
  reply_json TEXT NULL,
  mentioned_refs_json TEXT NOT NULL,
  CONSTRAINT fk_public_metadata_message FOREIGN KEY (message_id) REFERENCES chat_messages_next(message_id)
) ENGINE=InnoDB;
