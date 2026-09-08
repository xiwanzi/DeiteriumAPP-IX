package cafe.deuterium.core.mail;

import cafe.deuterium.core.api.ItemVersion;
import cafe.deuterium.core.config.CoreConfig;
import cafe.deuterium.core.storage.CatalogStore;
import cafe.deuterium.core.storage.OutboxStore;
import cafe.deuterium.core.util.Checks;
import cafe.deuterium.core.util.CoreFailure;
import cafe.deuterium.core.util.Json;
import cafe.deuterium.mail.api.MailboxIntegration;
import cafe.deuterium.mail.api.MailboxItem;
import com.google.gson.*;
import org.bukkit.Bukkit;
import java.nio.charset.StandardCharsets;
import java.util.*;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.function.Supplier;

/** The only Core package importing the independent mailbox's public API. No mailbox internals. */
public final class MailboxAdapter implements CoreMailbox {
    private final Supplier<CoreConfig> config;
    private final CatalogStore catalog;
    private final OutboxStore outbox;
    private final AtomicBoolean pumping = new AtomicBoolean();
    private volatile boolean closed;
    public MailboxAdapter(Supplier<CoreConfig> config, CatalogStore catalog, OutboxStore outbox) { this.config = config; this.catalog = catalog; this.outbox = outbox; }
    private MailboxIntegration service() {
        MailboxIntegration service = closed ? null : Bukkit.getServicesManager().load(MailboxIntegration.class);
        if (service == null) throw new CoreFailure("MAILBOX_UNAVAILABLE", "独立邮箱未启用或 API 版本不匹配。");
        return service;
    }
    @Override public JsonObject capabilities() {
        try {
            MailboxIntegration.Capabilities c = service().capabilities();
            JsonObject result = Json.tree(c); result.addProperty("available", true); result.addProperty("commerceReady", c.commerceReady()); return result;
        } catch (Exception | LinkageError unavailable) { return Json.tree(Map.of("available", false, "commerceReady", false, "reason", "MAILBOX_UNAVAILABLE")); }
    }
    @Override public JsonObject execute(String operation, String type, JsonObject payload) {
        MailboxIntegration s = service();
        if (type.equals("mailbox.revoke") && payload.has("snapshotJson")) {
            return Json.tree(s.revokeOrCancel(cancellationRequest(operation, payload)));
        }
        MailboxIntegration.Result<MailboxIntegration.Receipt> result = switch (type) {
            case "mailbox.create" -> s.create(create(operation, payload));
            case "mailbox.query" -> {
                Json.fields(payload, "source", "deliveryId");
                yield s.query(source(payload), Checks.operationId(Json.string(payload, "deliveryId")));
            }
            case "mailbox.revoke" -> {
                Json.fields(payload, "source", "deliveryId", "orderId", "expectedSnapshotSha256", "reasonCode");
                yield s.revoke(new MailboxIntegration.Revoke(operation, source(payload), Checks.operationId(Json.string(payload, "deliveryId")),
                        Checks.operationId(Json.string(payload, "orderId")), hash(Json.string(payload, "expectedSnapshotSha256")),
                        Checks.text(Json.string(payload, "reasonCode"), 192, false)));
            }
            default -> throw CoreFailure.invalid("不支持的邮箱操作。");
        };
        return Json.tree(result);
    }
    @Override public JsonObject queryCancellation(String operationId) {
        try {
            MailboxIntegration service = service();
            if (!service.capabilities().missingDeliveryCancellation()) return null;
            var result = service.queryCancellation("deuterium-commerce", Checks.operationId(operationId));
            if (result.code() != MailboxIntegration.Code.OK || result.value() == null) return null;
            if (!operationId.equals(result.value().operationId()))
                throw new CoreFailure("CANCELLATION_PROOF_MISMATCH", "邮箱返回了不同操作的取消证明。");
            return Json.tree(result);
        } catch (LinkageError unavailable) {
            return null;
        } catch (CoreFailure failure) {
            if (failure.code().equals("MAILBOX_UNAVAILABLE")) return null;
            throw failure;
        }
    }
    static MailboxIntegration.Cancel cancellationRequest(String operation, JsonObject payload) {
        Json.fields(payload, "source", "deliveryId", "orderId", "expectedSnapshotSha256", "reasonCode", "snapshotJson");
        String snapshotJson = Checks.text(Json.string(payload, "snapshotJson"), 24000, false);
        String snapshotHash = hash(Json.string(payload, "expectedSnapshotSha256"));
        if (!Checks.sha(snapshotJson.getBytes(StandardCharsets.UTF_8)).equals(snapshotHash))
            throw CoreFailure.invalid("取消请求的原订单快照摘要不一致。");
        JsonObject snapshot = Json.object(snapshotJson, 24000);
        Json.fields(snapshot, "schemaVersion", "orderId", "recipientUuid", "inventoryDomain", "allowedServerIds", "attachments", "templateRef", "templateRevision");
        String order = Checks.operationId(Json.string(payload, "orderId"));
        String recipientText = Json.string(snapshot, "recipientUuid");
        UUID recipient = UUID.fromString(recipientText);
        String domain = Checks.node(Json.string(snapshot, "inventoryDomain"));
        List<String> servers = strings(snapshot.getAsJsonArray("allowedServerIds"));
        if (Json.integer(snapshot, "schemaVersion") != 1 || !order.equals(Json.string(snapshot, "orderId"))
                || !recipient.toString().equals(recipientText) || servers.isEmpty()
                || new HashSet<>(servers).size() != servers.size())
            throw CoreFailure.invalid("取消请求的原订单身份或范围不正确。");
        // Cancellation closes a delivery key. It never resolves or creates an
        // ItemStack, and must still work after a catalog item is archived.
        return new MailboxIntegration.Cancel(operation, source(payload),
                Checks.operationId(Json.string(payload, "deliveryId")), order, recipient,
                snapshotJson, snapshotHash, servers, domain,
                Checks.text(Json.string(payload, "reasonCode"), 192, false));
    }
    private MailboxIntegration.Create create(String operation, JsonObject payload) {
        Json.fields(payload, "source", "deliveryId", "orderId", "recipientUuid", "title", "body", "sender", "snapshotJson", "snapshotSha256", "allowedServerIds", "inventoryDomain");
        String snapshotJson = Checks.text(Json.string(payload, "snapshotJson"), 24000, false);
        String snapshotHash = hash(Json.string(payload, "snapshotSha256"));
        if (!Checks.sha(snapshotJson.getBytes(StandardCharsets.UTF_8)).equals(snapshotHash)) throw CoreFailure.invalid("订单快照摘要不一致。");
        JsonObject snapshot = Json.object(snapshotJson, 24000);
        Json.fields(snapshot, "schemaVersion", "orderId", "recipientUuid", "inventoryDomain", "allowedServerIds", "attachments", "templateRef", "templateRevision");
        if (Json.integer(snapshot, "schemaVersion") != 1) throw CoreFailure.invalid("不支持的订单快照版本。");
        String order = Checks.operationId(Json.string(payload, "orderId")), domain = Checks.node(Json.string(payload, "inventoryDomain"));
        String recipientText = Json.string(payload, "recipientUuid");
        UUID recipient = UUID.fromString(recipientText);
        if (!recipient.toString().equals(recipientText) || !order.equals(Json.string(snapshot, "orderId"))
                || !recipientText.equals(Json.string(snapshot, "recipientUuid")) || !domain.equals(Json.string(snapshot, "inventoryDomain")))
            throw CoreFailure.invalid("订单、收件人或背包域与快照不一致。");
        List<String> servers = strings(payload.getAsJsonArray("allowedServerIds"));
        if (!servers.equals(strings(snapshot.getAsJsonArray("allowedServerIds"))) || servers.isEmpty() || servers.size() > 32 || new HashSet<>(servers).size() != servers.size())
            throw CoreFailure.invalid("邮件领取范围不正确。");
        JsonArray attachments = snapshot.getAsJsonArray("attachments");
        if (attachments == null || attachments.isEmpty() || attachments.size() > 32) throw CoreFailure.invalid("附件数量必须为 1–32。");
        Set<String> seen = new HashSet<>(); List<MailboxItem> resolved = new ArrayList<>(); long totalBytes = 0;
        for (JsonElement value : attachments) {
            JsonObject attachment = value.getAsJsonObject(); Json.fields(attachment, "itemRef", "revision", "quantity", "payloadSha256");
            String ref = Checks.itemRef(Json.string(attachment, "itemRef"), config.get().namespace());
            long revision = Json.integer(attachment, "revision"), quantity = Json.integer(attachment, "quantity");
            if (revision < 1 || revision > Integer.MAX_VALUE || quantity < 1 || quantity > 99999 || !seen.add(ref + "@" + revision))
                throw CoreFailure.invalid("附件版本、数量或重复项不正确。");
            ItemVersion item = catalog.find(ref, revision);
            if (!item.payloadSha256().equals(hash(Json.string(attachment, "payloadSha256"))) || quantity > item.maxQuantity()
                    || !item.inventoryDomain().equals(domain) || !item.compatibleServerIds().containsAll(servers))
                throw new CoreFailure("ITEM_SNAPSHOT_MISMATCH", "物品库快照不满足该订单的内容、数量或服务器范围。");
            CoreConfig.servers(servers, config.get().nodes(), domain, item.compatibilityProfile(), true);
            byte[] bytes = item.payload(); totalBytes += bytes.length;
            if (totalBytes > 8388608 || bytes.length > config.get().maxItemBytes()) throw CoreFailure.invalid("邮件物品体积超限。");
            resolved.add(new MailboxItem(ref, revision, quantity, item.payloadSha256(), item.codec(), bytes, item.itemId(), item.displayName(), item.compatibleServerIds()));
        }
        return new MailboxIntegration.Create(operation, Checks.operationId(Json.string(payload, "deliveryId")), order, recipient,
                source(payload), Checks.text(Json.string(payload, "title"), 1024, false), Checks.text(Json.string(payload, "body"), 4096, true),
                Checks.text(Json.string(payload, "sender"), 512, false), snapshotJson, snapshotHash, resolved, servers, domain);
    }
    private static String source(JsonObject payload) {
        String source = Json.string(payload, "source");
        if (!source.equals("deuterium-commerce")) throw CoreFailure.invalid("未经授权的邮件来源。"); return source;
    }
    private static String hash(String value) { if (!value.matches("[0-9a-f]{64}")) throw CoreFailure.invalid("无效内容摘要。"); return value; }
    private static List<String> strings(JsonArray array) {
        if (array == null || array.size() > 32) throw CoreFailure.invalid("缺少或过长的服务器列表。");
        List<String> values = new ArrayList<>();
        for (JsonElement value : array) {
            if (!value.isJsonPrimitive() || !value.getAsJsonPrimitive().isString()) throw CoreFailure.invalid("服务器 ID 必须是字符串。");
            values.add(Checks.node(value.getAsString()));
        }
        return List.copyOf(values);
    }
    @Override public void pumpEvents() {
        if (closed || !pumping.compareAndSet(false, true)) return;
        try {
            MailboxIntegration s = service();
            for (OutboxStore.Event event : outbox.mailAcknowledgements()) acknowledge(s, event);
            MailboxIntegration.Result<MailboxIntegration.EventBatch> batch = s.events(config.get().mailConsumer(), 16);
            if (!batch.success() || batch.value() == null) return;
            for (MailboxIntegration.Event event : batch.value().events()) {
                String type = event.type().endsWith(".event") ? event.type() : event.type() + ".event";
                outbox.mail("mail_" + event.eventId(), type, Json.GSON.toJson(event), event.eventId().toString(), batch.value().leaseToken().toString(), config.get().mailConsumer());
            }
            // A reclaimed mailbox lease may belong to an event already committed by the backend.
            for (OutboxStore.Event event : outbox.mailAcknowledgements()) acknowledge(s, event);
        } finally { pumping.set(false); }
    }
    private void acknowledge(MailboxIntegration service, OutboxStore.Event event) {
        MailboxIntegration.Result<Boolean> result = service.acknowledge(event.consumer(), UUID.fromString(event.lease()), List.of(UUID.fromString(event.mailEvent())));
        if (result.success() || result.code() == MailboxIntegration.Code.LEASE_LOST) outbox.releaseMailLease(event.id(), event.lease());
    }
    @Override public void close() { closed = true; }
}
