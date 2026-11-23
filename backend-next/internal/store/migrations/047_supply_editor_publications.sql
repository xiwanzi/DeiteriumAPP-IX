CREATE TABLE IF NOT EXISTS supply_editor_publications (
 source_revision BIGINT PRIMARY KEY,
 catalog_json LONGTEXT NOT NULL,
 created_at DATETIME(6) NOT NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS supply_editor_artwork (
 sha256 CHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 picture MEDIUMBLOB NOT NULL,
 width INT NOT NULL,
 height INT NOT NULL,
 size_bytes INT NOT NULL,
 published BOOLEAN NOT NULL DEFAULT FALSE
) ENGINE=InnoDB;
