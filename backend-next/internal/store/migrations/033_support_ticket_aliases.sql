CREATE TABLE IF NOT EXISTS support_ticket_aliases (
 alias_ticket_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 ticket_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 created_at DATETIME(6) NOT NULL,
 INDEX ix_support_alias_target(ticket_id)
) ENGINE=InnoDB;
