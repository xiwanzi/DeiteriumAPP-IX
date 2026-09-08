CREATE TABLE IF NOT EXISTS social_reconcile_v2 (
  worker_name VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
  last_sequence BIGINT NOT NULL DEFAULT 0,
  updated_at DATETIME(6) NOT NULL
) ENGINE=InnoDB;
