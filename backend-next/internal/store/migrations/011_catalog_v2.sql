CREATE TABLE IF NOT EXISTS catalog_records_v2 (
  sequence_id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  resource_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL UNIQUE,
  kind VARCHAR(24) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  store_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL DEFAULT '',
  owner_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  version INT NOT NULL,
  body MEDIUMTEXT NOT NULL,
  published_body MEDIUMTEXT NULL,
  published_version INT NULL,
  published_at DATETIME(6) NULL,
  state VARCHAR(16) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  available_stock INT NULL,
  title VARCHAR(200) NOT NULL DEFAULT '',
  category_ref VARCHAR(128) NOT NULL DEFAULT '',
  brand_ref VARCHAR(128) NOT NULL DEFAULT '',
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  INDEX ix_catalog_public (kind, state, sequence_id),
  INDEX ix_catalog_store (kind, store_id, state, sequence_id),
  INDEX ix_catalog_owner (kind, owner_id, sequence_id),
  CONSTRAINT fk_catalog_owner FOREIGN KEY (owner_id) REFERENCES identities(id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS catalog_members_v2 (
  store_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  user_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  permissions TEXT NOT NULL,
  version INT NOT NULL,
  active BOOLEAN NOT NULL,
  PRIMARY KEY (store_id, user_id),
  CONSTRAINT fk_catalog_member_store FOREIGN KEY (store_id) REFERENCES catalog_records_v2(resource_id),
  CONSTRAINT fk_catalog_member_user FOREIGN KEY (user_id) REFERENCES identities(id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS catalog_mutations_v2 (
  user_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  request_id VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  scope VARCHAR(180) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  fingerprint CHAR(64) CHARACTER SET ascii NOT NULL,
  result MEDIUMTEXT NOT NULL,
  created_at DATETIME(6) NOT NULL,
  PRIMARY KEY (user_id, request_id),
  CONSTRAINT fk_catalog_mutation_user FOREIGN KEY (user_id) REFERENCES identities(id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS catalog_carts_v2 (
  user_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
  version INT NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  CONSTRAINT fk_catalog_cart_user FOREIGN KEY (user_id) REFERENCES identities(id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS catalog_cart_items_v2 (
  user_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  product_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  quantity INT NOT NULL,
  product_version INT NOT NULL,
  PRIMARY KEY (user_id, product_id),
  CONSTRAINT fk_catalog_cart_item_user FOREIGN KEY (user_id) REFERENCES catalog_carts_v2(user_id),
  CONSTRAINT fk_catalog_cart_item_product FOREIGN KEY (product_id) REFERENCES catalog_records_v2(resource_id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS catalog_quotes_v2 (
  quote_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
  user_id VARCHAR(40) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  channel VARCHAR(24) CHARACTER SET ascii NOT NULL,
  snapshot MEDIUMTEXT NOT NULL,
  created_at DATETIME(6) NOT NULL,
  expires_at DATETIME(6) NOT NULL,
  INDEX ix_catalog_quote_owner (user_id, expires_at),
  CONSTRAINT fk_catalog_quote_user FOREIGN KEY (user_id) REFERENCES identities(id)
) ENGINE=InnoDB;
