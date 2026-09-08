package cafe.deuterium.core.storage;

import cafe.deuterium.core.util.CoreFailure;
import java.sql.ResultSet;

public final class InboxStore {
    private final Database db;
    private final String node;
    public InboxStore(Database db, String node) { this.db = db; this.node = node; }
    /** Reserve before the non-transactional screen side effect. Crash may omit a line, never replay it endlessly. */
    public boolean reserve(String id, String fingerprint, long expiry) {
        return db.transaction(c -> {
            int inserted = Sql.update(c, db.ignoreInsert() + " INTO dc_inbox(node_id,message_id,fingerprint,state,expires_at) VALUES(?,?,?,'RESERVED',?)", node, id, fingerprint, expiry);
            if (inserted == 1) return true;
            try (var s = Sql.statement(c, "SELECT fingerprint FROM dc_inbox WHERE node_id=? AND message_id=?", node, id); ResultSet r = s.executeQuery()) {
                if (!r.next() || !fingerprint.equals(r.getString(1))) throw new CoreFailure("IDEMPOTENCY_CONFLICT", "聊天投递内容冲突。");
            }
            return false;
        });
    }
    public void shown(String id) { db.read(c -> Sql.update(c, "UPDATE dc_inbox SET state='SHOWN' WHERE node_id=? AND message_id=?", node, id)); }
    public void sweep() { db.read(c -> Sql.update(c, "DELETE FROM dc_inbox WHERE node_id=? AND expires_at<?", node, System.currentTimeMillis() - 86400000L)); }
}
