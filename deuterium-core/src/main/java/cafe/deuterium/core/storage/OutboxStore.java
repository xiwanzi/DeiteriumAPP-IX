package cafe.deuterium.core.storage;

import cafe.deuterium.core.util.Checks;
import cafe.deuterium.core.util.CoreFailure;
import cafe.deuterium.core.util.Json;
import java.nio.charset.StandardCharsets;
import java.sql.*;
import java.util.ArrayList;
import java.util.List;

public final class OutboxStore {
    public record Event(String id, String type, String payload, String state, String mailEvent, String lease, String consumer) { }
    private final Database db;
    private final String node;
    private final int limit;
    public OutboxStore(Database db, String node, int limit) { this.db = db; this.node = node; this.limit = limit; }
    public void enqueue(String id, String type, String payload) { db.transaction(c -> { enqueue(c, id, type, payload, null, null, null); return null; }); }
    public void mail(String id, String type, String payload, String mailEvent, String lease, String consumer) {
        db.transaction(c -> { enqueue(c, id, type, payload, mailEvent, lease, consumer); return null; });
    }
    public void enqueue(Connection c, String id, String type, String payload, String mailEvent, String lease, String consumer) throws Exception {
        Checks.operationId(id);
        payload = Json.canonical(Json.object(payload, 28000));
        if (payload.getBytes(StandardCharsets.UTF_8).length > 28000) throw CoreFailure.invalid("出站事件超过大小限制。");
        String fingerprint = Checks.sha((type + ":" + payload).getBytes(StandardCharsets.UTF_8));
        try (PreparedStatement s = Sql.statement(c, "SELECT fingerprint FROM dc_outbox WHERE node_id=? AND event_id=?", node, id); ResultSet r = s.executeQuery()) {
            if (r.next()) {
                if (!r.getString(1).equals(fingerprint)) throw new CoreFailure("IDEMPOTENCY_CONFLICT", "出站事件标识对应不同内容。");
                if (mailEvent != null) Sql.update(c, "UPDATE dc_outbox SET mail_lease=?,mail_consumer=? WHERE node_id=? AND event_id=?", lease, consumer, node, id);
                return;
            }
        }
        if (Sql.count(c, "SELECT COUNT(*) FROM dc_outbox WHERE node_id=? AND state='PENDING'", node) >= limit)
            throw new CoreFailure("OUTBOX_FULL", "后端待发送队列已满，请恢复连接后重试。");
        Sql.update(c, "INSERT INTO dc_outbox(node_id,event_id,type,payload,fingerprint,state,created_at,mail_event,mail_lease,mail_consumer) VALUES(?,?,?,?,?,'PENDING',?,?,?,?)",
                node, id, type, payload, fingerprint, System.currentTimeMillis(), mailEvent, lease, consumer);
    }
    public List<Event> pending(int count) {
        return db.read(c -> rows(c, "SELECT * FROM dc_outbox WHERE node_id=? AND state='PENDING' ORDER BY sequence_id LIMIT ?", node, count));
    }
    public List<Event> mailAcknowledgements() {
        return db.read(c -> rows(c, "SELECT * FROM dc_outbox WHERE node_id=? AND state='ACKED' AND mail_lease IS NOT NULL ORDER BY sequence_id LIMIT 32", node));
    }
    private static List<Event> rows(Connection c, String sql, Object... args) throws Exception {
        List<Event> events = new ArrayList<>();
        try (PreparedStatement s = Sql.statement(c, sql, args); ResultSet r = s.executeQuery()) {
            while (r.next()) events.add(new Event(r.getString("event_id"), r.getString("type"), r.getString("payload"), r.getString("state"),
                    r.getString("mail_event"), r.getString("mail_lease"), r.getString("mail_consumer")));
        }
        return events;
    }
    public Event acknowledged(String id) {
        return db.transaction(c -> {
            Sql.update(c, "UPDATE dc_outbox SET state='ACKED',last_error=NULL WHERE node_id=? AND event_id=?", node, id);
            List<Event> events = rows(c, "SELECT * FROM dc_outbox WHERE node_id=? AND event_id=?", node, id);
            return events.isEmpty() ? null : events.getFirst();
        });
    }
    public void rejected(String id, String code) { db.read(c -> Sql.update(c, "UPDATE dc_outbox SET state='REJECTED',last_error=? WHERE node_id=? AND event_id=?", code, node, id)); }
    public void releaseMailLease(String id, String lease) { db.read(c -> Sql.update(c, "UPDATE dc_outbox SET mail_lease=NULL WHERE node_id=? AND event_id=? AND mail_lease=?", node, id, lease)); }
    public long pendingCount() { return db.read(c -> Sql.count(c, "SELECT COUNT(*) FROM dc_outbox WHERE node_id=? AND state='PENDING'", node)); }
    public long rejectedCount() { return db.read(c -> Sql.count(c, "SELECT COUNT(*) FROM dc_outbox WHERE node_id=? AND state='REJECTED'", node)); }
    public void retryRejected() { db.read(c -> Sql.update(c, "UPDATE dc_outbox SET state='PENDING',last_error=NULL WHERE node_id=? AND state='REJECTED'", node)); }
    public void republishCatalog() { db.read(c -> Sql.update(c, "UPDATE dc_outbox SET state='PENDING',last_error=NULL WHERE node_id=? AND type IN ('item.version.published','item.catalog.updated')", node)); }
}
