package cafe.deuterium.core.storage;

import cafe.deuterium.core.api.ItemSummary;
import cafe.deuterium.core.api.ItemVersion;
import cafe.deuterium.core.items.ItemMetadata;
import cafe.deuterium.core.util.Checks;
import cafe.deuterium.core.util.CoreFailure;
import cafe.deuterium.core.util.Json;
import java.nio.charset.StandardCharsets;
import java.sql.*;
import java.util.*;

public final class CatalogStore {
    public record Captured(String ref, byte[] payload, String itemId, String name, String description,
                           long maxQuantity, List<String> servers, List<String> requiredMods,
                           String domain, String profile) {
        public Captured { payload = payload.clone(); servers = List.copyOf(servers); requiredMods = List.copyOf(requiredMods); }
        @Override public byte[] payload() { return payload.clone(); }
    }
    private final Database db;
    private final OutboxStore outbox;
    public CatalogStore(Database db, OutboxStore outbox) { this.db = db; this.outbox = outbox; }
    public ItemVersion save(Captured item, String actor) {
        return db.transaction(c -> {
            Sql.update(c, db.ignoreInsert() + " INTO dc_items(item_ref,latest_revision,catalog_version,archived,display_name,item_id) VALUES(?,0,0,false,?,?)", item.ref(), item.name(), item.itemId());
            ItemSummary head = head(c, item.ref(), true);
            if (head.archived()) throw new CoreFailure("ITEM_ARCHIVED", "物品已归档，请先恢复。");
            long revision = Math.addExact(head.latestRevision(), 1), generation = Math.addExact(head.catalogVersion(), 1);
            if (revision > Integer.MAX_VALUE || generation > Integer.MAX_VALUE) throw CoreFailure.invalid("物品版本达到上限。");
            byte[] payload = item.payload();
            ItemVersion result = new ItemVersion(item.ref(), revision, "bukkit-bytes-v1", payload, Checks.sha(payload),
                    item.itemId(), item.name(), item.description(), item.maxQuantity(), item.servers(), item.requiredMods(), item.domain(), item.profile(), System.currentTimeMillis());
            String metadata = Json.GSON.toJson(ItemMetadata.of(result));
            Sql.update(c, "INSERT INTO dc_item_versions(item_ref,revision,metadata,payload,payload_sha,created_at) VALUES(?,?,?,?,?,?)",
                    item.ref(), revision, metadata, payload, result.payloadSha256(), result.createdAt());
            Sql.update(c, "UPDATE dc_items SET latest_revision=?,catalog_version=?,display_name=?,item_id=? WHERE item_ref=?", revision, generation, item.name(), item.itemId(), item.ref());
            publishVersion(c, result);
            publishHead(c, new ItemSummary(item.ref(), revision, generation, item.name(), item.itemId(), false));
            Sql.audit(c, actor, "item.save", item.ref() + "@" + revision, "COMPLETED");
            return result;
        });
    }
    public ItemVersion find(String ref, long revision) {
        return db.read(c -> {
            long resolved = revision;
            if (resolved == 0) {
                ItemSummary head = head(c, ref, false);
                if (head.archived()) throw new CoreFailure("ITEM_ARCHIVED", "物品已归档，读取历史内容需明确指定版本。");
                resolved = head.latestRevision();
            }
            try (PreparedStatement s = Sql.statement(c, "SELECT metadata,payload,payload_sha,created_at FROM dc_item_versions WHERE item_ref=? AND revision=?", ref, resolved); ResultSet r = s.executeQuery()) {
                if (!r.next()) throw new CoreFailure("ITEM_NOT_FOUND", "找不到物品或版本。");
                byte[] payload = r.getBytes("payload");
                if (!Checks.sha(payload).equals(r.getString("payload_sha"))) throw new CoreFailure("ITEM_CORRUPT", "物品快照校验失败，已阻止读取。");
                ItemMetadata metadata = Json.GSON.fromJson(r.getString("metadata"), ItemMetadata.class);
                if (!metadata.itemRef().equals(ref) || metadata.revision() != resolved || !metadata.payloadSha256().equals(r.getString("payload_sha")))
                    throw new CoreFailure("ITEM_CORRUPT", "物品元数据与内容不一致。");
                return metadata.withPayload(payload, r.getLong("created_at"));
            }
        });
    }
    public List<ItemSummary> search(String query, int page, boolean archived) {
        if (page < 1 || page > 100000 || query.length() > 96) throw CoreFailure.invalid("无效分页或关键词。");
        return db.read(c -> {
            List<ItemSummary> results = new ArrayList<>();
            String pattern = "%" + query.toLowerCase(Locale.ROOT).replace("!", "!!").replace("%", "!%").replace("_", "!_") + "%";
            try (PreparedStatement s = Sql.statement(c, "SELECT * FROM dc_items WHERE item_ref LIKE ? ESCAPE '!'" + (archived ? "" : " AND archived=false") + " ORDER BY item_ref LIMIT 10 OFFSET ?", pattern, (page - 1) * 10); ResultSet r = s.executeQuery()) {
                while (r.next()) results.add(summary(r));
            }
            return List.copyOf(results);
        });
    }
    public List<Long> versions(String ref, int page) {
        if (page < 1 || page > 100000) throw CoreFailure.invalid("无效页码。");
        return db.read(c -> {
            List<Long> versions = new ArrayList<>();
            try (PreparedStatement s = Sql.statement(c, "SELECT revision FROM dc_item_versions WHERE item_ref=? ORDER BY revision DESC LIMIT 10 OFFSET ?", ref, (page - 1) * 10); ResultSet r = s.executeQuery()) { while (r.next()) versions.add(r.getLong(1)); }
            return List.copyOf(versions);
        });
    }
    public ItemSummary archive(String ref, boolean archived, String actor) {
        return db.transaction(c -> {
            ItemSummary head = head(c, ref, true);
            if (head.archived() == archived) return head;
            long generation = Math.addExact(head.catalogVersion(), 1);
            if (generation > Integer.MAX_VALUE) throw CoreFailure.invalid("目录版本达到上限。");
            Sql.update(c, "UPDATE dc_items SET archived=?,catalog_version=? WHERE item_ref=?", archived, generation, ref);
            ItemSummary changed = new ItemSummary(ref, head.latestRevision(), generation, head.displayName(), head.itemId(), archived);
            publishHead(c, changed); Sql.audit(c, actor, archived ? "item.archive" : "item.restore", ref, "COMPLETED");
            return changed;
        });
    }
    public void republish() {
        String cursor = "";
        for (;;) {
            String after = cursor;
            List<ItemSummary> heads = db.read(c -> {
                List<ItemSummary> list = new ArrayList<>();
                try (PreparedStatement s = Sql.statement(c, "SELECT * FROM dc_items WHERE item_ref>? ORDER BY item_ref LIMIT 50", after); ResultSet r = s.executeQuery()) { while (r.next()) list.add(summary(r)); }
                return list;
            });
            if (heads.isEmpty()) return;
            for (ItemSummary head : heads) {
                ItemVersion latest = find(head.itemRef(), head.latestRevision());
                db.transaction(c -> { publishVersion(c, latest); publishHead(c, head); return null; });
                cursor = head.itemRef();
            }
        }
    }
    private void publishVersion(Connection c, ItemVersion item) throws Exception {
        outbox.enqueue(c, eventKey("item", item.itemRef(), item.revision()), "item.version.published", Json.GSON.toJson(ItemMetadata.of(item)), null, null, null);
    }
    private void publishHead(Connection c, ItemSummary head) throws Exception {
        outbox.enqueue(c, eventKey("catalog", head.itemRef(), head.catalogVersion()), "item.catalog.updated",
                Json.GSON.toJson(Map.of("itemRef", head.itemRef(), "catalogVersion", head.catalogVersion(), "latestRevision", head.latestRevision(), "archived", head.archived())), null, null, null);
    }
    private static String eventKey(String prefix, String ref, long revision) { return prefix + "_" + Checks.sha(ref.getBytes(StandardCharsets.UTF_8)).substring(0,32) + "_" + revision; }
    private ItemSummary head(Connection c, String ref, boolean lock) throws Exception {
        try (PreparedStatement s = Sql.statement(c, "SELECT * FROM dc_items WHERE item_ref=?" + (lock ? db.lock() : ""), ref); ResultSet r = s.executeQuery()) {
            if (!r.next()) throw new CoreFailure("ITEM_NOT_FOUND", "找不到物品。");
            return summary(r);
        }
    }
    private static ItemSummary summary(ResultSet r) throws SQLException { return new ItemSummary(r.getString("item_ref"), r.getLong("latest_revision"), r.getLong("catalog_version"), r.getString("display_name"), r.getString("item_id"), r.getBoolean("archived")); }
}
