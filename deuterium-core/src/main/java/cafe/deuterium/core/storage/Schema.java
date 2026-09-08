package cafe.deuterium.core.storage;

import java.sql.Statement;
import java.util.List;

final class Schema {
    private Schema() { }
    static void ensure(Database db) {
        String seq = db.shared() ? "BIGINT AUTO_INCREMENT PRIMARY KEY" : "INTEGER PRIMARY KEY AUTOINCREMENT";
        String text = db.shared() ? "MEDIUMTEXT" : "TEXT";
        String blob = db.shared() ? "MEDIUMBLOB" : "BLOB";
        String engine = db.shared() ? " ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin" : "";
        List<String> ddl = List.of(
                "CREATE TABLE IF NOT EXISTS dc_schema(version INTEGER PRIMARY KEY)",
                "CREATE TABLE IF NOT EXISTS dc_items(item_ref VARCHAR(96) PRIMARY KEY,latest_revision BIGINT NOT NULL,catalog_version BIGINT NOT NULL,archived BOOLEAN NOT NULL,display_name VARCHAR(128) NOT NULL,item_id VARCHAR(256) NOT NULL)",
                "CREATE TABLE IF NOT EXISTS dc_item_versions(item_ref VARCHAR(96) NOT NULL,revision BIGINT NOT NULL,metadata " + text + " NOT NULL,payload " + blob + " NOT NULL,payload_sha CHAR(64) NOT NULL,created_at BIGINT NOT NULL,PRIMARY KEY(item_ref,revision))",
                "CREATE TABLE IF NOT EXISTS dc_outbox(sequence_id " + seq + ",node_id VARCHAR(32) NOT NULL,event_id VARCHAR(128) NOT NULL,type VARCHAR(64) NOT NULL,payload " + text + " NOT NULL,fingerprint CHAR(64) NOT NULL,state VARCHAR(16) NOT NULL,created_at BIGINT NOT NULL,last_attempt BIGINT NOT NULL DEFAULT 0,mail_event VARCHAR(36),mail_lease VARCHAR(36),mail_consumer VARCHAR(128),last_error VARCHAR(64),UNIQUE(node_id,event_id))",
                "CREATE TABLE IF NOT EXISTS dc_inbox(node_id VARCHAR(32) NOT NULL,message_id VARCHAR(128) NOT NULL,fingerprint CHAR(64) NOT NULL,state VARCHAR(16) NOT NULL,expires_at BIGINT NOT NULL,PRIMARY KEY(node_id,message_id))",
                "CREATE TABLE IF NOT EXISTS dc_commands(operation_id VARCHAR(128) PRIMARY KEY,command_type VARCHAR(64) NOT NULL,fingerprint CHAR(64) NOT NULL,state VARCHAR(16) NOT NULL,result " + text + ",node_id VARCHAR(32) NOT NULL,updated_at BIGINT NOT NULL)",
                "CREATE TABLE IF NOT EXISTS dc_audit(sequence_id " + seq + ",actor VARCHAR(128) NOT NULL,action VARCHAR(64) NOT NULL,resource VARCHAR(192) NOT NULL,result VARCHAR(32) NOT NULL,created_at BIGINT NOT NULL)",
                "CREATE TABLE IF NOT EXISTS dc_players(player_uuid CHAR(36) PRIMARY KEY,game_id VARCHAR(32) NOT NULL,node_id VARCHAR(32) NOT NULL,session_epoch CHAR(36) NOT NULL,online BOOLEAN NOT NULL,last_seen BIGINT NOT NULL)"
        );
        db.read(c -> {
            try (Statement s = c.createStatement()) { for (String sql : ddl) s.execute(sql + engine); }
            if (Sql.count(c, "SELECT COUNT(*) FROM dc_schema WHERE version<>1") != 0)
                throw new IllegalStateException("Unsupported Core schema version");
            Sql.update(c, db.ignoreInsert() + " INTO dc_schema(version) VALUES(1)");
            index(c, db, "idx_dc_outbox_pending", "dc_outbox", "node_id,state,sequence_id");
            index(c, db, "idx_dc_players_name", "dc_players", "game_id");
            index(c, db, "idx_dc_audit_time", "dc_audit", "created_at");
            return null;
        });
    }
    private static void index(java.sql.Connection c, Database db, String name, String table, String fields) throws Exception {
        if (db.shared()) {
            if (Sql.count(c, "SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=? AND INDEX_NAME=?", table, name) > 0) return;
            try { Sql.update(c, "CREATE INDEX " + name + " ON " + table + "(" + fields + ")"); }
            catch (java.sql.SQLException e) { if (e.getErrorCode() != 1061) throw e; }
        } else Sql.update(c, "CREATE INDEX IF NOT EXISTS " + name + " ON " + table + "(" + fields + ")");
    }
}
