package cafe.deuterium.core.storage;

import cafe.deuterium.core.util.CoreFailure;
import java.sql.ResultSet;

public final class RpcJournal {
    public record Entry(boolean acquired, String state, String result) { }
    private final Database db;
    private final String node;
    public RpcJournal(Database db, String node) { this.db = db; this.node = node; }
    public void recover() { db.read(c -> Sql.update(c, "UPDATE dc_commands SET state='UNKNOWN',updated_at=? WHERE node_id=? AND state='EXECUTING'", System.currentTimeMillis(), node)); }
    public Entry begin(String operation, String type, String fingerprint, boolean safeReplay) {
        return db.transaction(c -> {
            int created = Sql.update(c, db.ignoreInsert() + " INTO dc_commands(operation_id,command_type,fingerprint,state,node_id,updated_at) VALUES(?,?,?,'EXECUTING',?,?)", operation, type, fingerprint, node, System.currentTimeMillis());
            if (created == 1) return new Entry(true, "EXECUTING", null);
            try (var s = Sql.statement(c, "SELECT * FROM dc_commands WHERE operation_id=?" + db.lock(), operation); ResultSet r = s.executeQuery()) {
                if (!r.next() || !fingerprint.equals(r.getString("fingerprint")) || !type.equals(r.getString("command_type")))
                    throw new CoreFailure("IDEMPOTENCY_CONFLICT", "原操作标识已对应其他内容。");
                String state = r.getString("state"), result = r.getString("result");
                if (state.equals("UNKNOWN") && safeReplay) {
                    Sql.update(c, "UPDATE dc_commands SET state='EXECUTING',node_id=?,updated_at=? WHERE operation_id=?", node, System.currentTimeMillis(), operation);
                    return new Entry(true, "EXECUTING", null);
                }
                return new Entry(false, state, result);
            }
        });
    }
    public void complete(String id, String state, String result) {
        db.transaction(c -> {
            int changed = Sql.update(c, "UPDATE dc_commands SET state=?,result=?,updated_at=? WHERE operation_id=? AND state='EXECUTING' AND node_id=?", state, result, System.currentTimeMillis(), id, node);
            if (changed != 1) throw new CoreFailure("RESULT_UNKNOWN", "执行日志的所有权或状态已改变，不能确认结果。");
            Sql.audit(c, "backend", "command.result", id, state); return null;
        });
    }
    public Entry find(String id) {
        return db.read(c -> {
            try (var s = Sql.statement(c, "SELECT state,result FROM dc_commands WHERE operation_id=?", id); ResultSet r = s.executeQuery()) {
                return r.next() ? new Entry(false, r.getString(1), r.getString(2)) : new Entry(false, "NOT_FOUND", null);
            }
        });
    }
    /** Read an existing intent without acquiring it or changing an UNKNOWN execution. */
    public Entry match(String operation, String type, String fingerprint) {
        return db.read(c -> {
            try (var statement = Sql.statement(c, "SELECT command_type,fingerprint,state,result FROM dc_commands WHERE operation_id=?", operation); ResultSet row = statement.executeQuery()) {
                if (!row.next()) return new Entry(false, "NOT_FOUND", null);
                if (!type.equals(row.getString(1)) || !fingerprint.equals(row.getString(2))) throw new CoreFailure("IDEMPOTENCY_CONFLICT", "原操作标识已对应其他内容。");
                return new Entry(false, row.getString(3), row.getString(4));
            }
        });
    }
}
