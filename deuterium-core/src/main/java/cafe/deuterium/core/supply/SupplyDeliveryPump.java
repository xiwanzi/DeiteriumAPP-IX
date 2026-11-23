package cafe.deuterium.core.supply;

import cafe.deuterium.core.CoreRuntime;
import cafe.deuterium.core.api.ItemVersion;
import cafe.deuterium.core.api.PlayerDataService;
import cafe.deuterium.core.util.CoreFailure;
import cafe.deuterium.core.util.Json;
import com.google.gson.JsonObject;
import java.util.Map;
import java.util.Set;
import java.util.UUID;
import java.util.concurrent.ArrayBlockingQueue;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.RejectedExecutionException;
import java.util.concurrent.ThreadPoolExecutor;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.function.Supplier;
import java.util.logging.Level;
import org.bukkit.Bukkit;
import org.bukkit.plugin.java.JavaPlugin;

/** Resumes the shared supply outbox and reports inventory proofs; it never rerolls rewards. */
public final class SupplyDeliveryPump implements AutoCloseable {
    private final JavaPlugin plugin;
    private final CoreRuntime runtime;
    private final SupplyHttp http;
    private final SupplyItemPreparation preparation = new SupplyItemPreparation();
    private final SupplyInventoryDelivery inventory;
    private final ThreadPoolExecutor workers = new ThreadPoolExecutor(4, 4, 0, TimeUnit.SECONDS,
            new ArrayBlockingQueue<>(64), task -> {
                Thread thread = new Thread(task, "Deuterium-Supply-Delivery"); thread.setDaemon(true); return thread;
            }, new ThreadPoolExecutor.AbortPolicy());
    private final Set<String> inFlight = ConcurrentHashMap.newKeySet();
    private final Set<String> reported = ConcurrentHashMap.newKeySet();
    private final Map<String, JsonObject> finishes = new ConcurrentHashMap<>();
    private final AtomicBoolean polling = new AtomicBoolean();
    private final AtomicBoolean pollRequested = new AtomicBoolean();
    private final org.bukkit.scheduler.BukkitTask timer;
    private volatile boolean closed;

    public SupplyDeliveryPump(JavaPlugin plugin, CoreRuntime runtime, SupplyHttp http) {
        this.plugin = plugin; this.runtime = runtime; this.http = http;
        inventory = new SupplyInventoryDelivery(runtime.sessions, runtime::inventoryAccess);
        timer = Bukkit.getScheduler().runTaskTimer(plugin, this::requestPoll, 40, 40);
    }

    public void requestPoll() {
        if (closed) return;
        pollRequested.set(true);
        if (!polling.compareAndSet(false, true)) return;
        try { workers.execute(this::poll); }
        catch (RejectedExecutionException full) { polling.set(false); }
    }

    private void poll() {
        pollRequested.set(false);
        try {
            for (var entry : finishes.entrySet()) schedule(entry.getKey(), () -> finish(entry.getKey(), entry.getValue()));
            JsonObject response = http.post("deliveries", new JsonObject());
            reported.remove("poll");
            for (var value : response.getAsJsonArray("items")) {
                JsonObject delivery = value.getAsJsonObject(); String id = Json.string(delivery, "id");
                if (!finishes.containsKey(id)) schedule(id, () -> process(delivery));
            }
        } catch (Exception failure) { report("poll", failure); }
        finally {
            polling.set(false);
            if (pollRequested.get()) requestPoll();
        }
    }

    @FunctionalInterface private interface Job { void run() throws Exception; }
    private void schedule(String id, Job job) {
        if (closed || inFlight.size() >= 32 || !inFlight.add(id)) return;
        try { workers.execute(() -> {
            try { if (!closed) job.run(); }
            catch (Exception failure) { report(id, failure); }
            finally { inFlight.remove(id); }
        }); } catch (RejectedExecutionException full) { inFlight.remove(id); }
    }

    private <T> T main(java.util.concurrent.Callable<T> job) throws Exception {
        return runtime.game.call(System.currentTimeMillis() + 8000, job).get(9, TimeUnit.SECONDS);
    }

    private void process(JsonObject delivery) throws Exception {
        String state = Json.string(delivery, "state");
        if (state.equals("ISSUING") || state.equals("REVIEW")) { recover(delivery); return; }
        UUID uuid = UUID.fromString(Json.string(delivery, "playerUuid"));
        var session = runtime.sessions.get(uuid); if (session == null) return;
        JsonObject claim = identity(delivery);
        claim.addProperty("sessionEpoch", session.sessionEpoch());
        JsonObject claimed = http.post("deliveries/claim", claim);
        if (!Json.string(claimed, "state").equals("PREPARING")) return;
        JsonObject reward = claimed.getAsJsonObject("reward");
        ItemVersion version = runtime.catalog.find(Json.string(reward, "itemRef"), Json.integer(reward, "itemRevision"));
        if (!version.payloadSha256().equals(Json.string(reward, "payloadSha256"))) throw new CoreFailure("SUPPLY_ITEM_CHANGED", "奖励物品版本摘要不一致");
        String id = Json.string(claimed, "id");
        SupplyItemPreparation.Prepared prepared = main(() -> {
            runtime.sessions.require(uuid, session.sessionEpoch());
            return preparation.prepare(id, uuid, runtime.codec.decode(version, runtime.config()));
        });
        if (!prepared.ready()) {
            JsonObject result = identity(claimed); result.addProperty("state", "PREPARING");
            result.addProperty("message", prepared.message().substring(0, Math.min(120, prepared.message().length())));
            http.post("deliveries/finish", result); return;
        }
        UUID operation = UUID.fromString(Json.string(claimed, "token"));
        String inventoryEpoch = main(() -> inventory.inspect(uuid, session.sessionEpoch(), version.inventoryDomain(), operation, prepared.item()));
        JsonObject begin = identity(claimed); begin.addProperty("inventoryEpoch", inventoryEpoch);
        http.post("deliveries/begin", begin);
        SupplyInventoryDelivery.Attempt attempt;
        try { attempt = main(() -> inventory.deliver(uuid, session.sessionEpoch(), inventoryEpoch, version.inventoryDomain(), operation, prepared.item())); }
        catch (Exception uncertain) {
            attempt = new SupplyInventoryDelivery.Attempt("REVIEW", "背包操作完成状态未确认", "", 0);
            report(id, uncertain);
        }
        JsonObject result = identity(claimed);
        result.addProperty("state", attempt.state()); result.addProperty("message", attempt.message());
        result.addProperty("saveReceipt", attempt.receipt()); result.addProperty("committedAt", attempt.committedAt());
        result.addProperty("inventoryDomain", version.inventoryDomain()); result.addProperty("inventoryEpoch", inventoryEpoch);
        result.addProperty("proofNode", runtime.config().nodeId());
        finishes.put(id, result); finish(id, result);
    }

    private void recover(JsonObject delivery) throws Exception {
        String inventoryEpoch = Json.string(delivery, "inventoryEpoch"); if (inventoryEpoch.isBlank()) return;
        UUID uuid = UUID.fromString(Json.string(delivery, "playerUuid"));
        UUID operation = UUID.fromString(Json.string(delivery, "token"));
        String domain = Json.string(delivery.getAsJsonObject("reward"), "inventoryDomain");
        PlayerDataService provider = main(runtime::inventoryAccess);
        PlayerDataService.SaveProof proof = provider.lookupSaveProof(uuid, domain, operation, inventoryEpoch);
        if (!matchesProof(delivery, domain, inventoryEpoch, proof)) return;
        JsonObject result = identity(delivery); result.addProperty("state", "INVENTORY"); result.addProperty("message", "");
        result.addProperty("saveReceipt", proof.saveReceipt()); result.addProperty("committedAt", proof.committedAt());
        result.addProperty("inventoryDomain", domain); result.addProperty("inventoryEpoch", inventoryEpoch); result.addProperty("proofNode", proof.serverId());
        finishes.put(Json.string(delivery, "id"), result); finish(Json.string(delivery, "id"), result);
    }

    static boolean matchesProof(JsonObject delivery, String domain, String epoch, PlayerDataService.SaveProof proof) {
        return proof != null && proof.playerUuid().toString().equals(Json.string(delivery, "playerUuid"))
                && proof.operationId().toString().equals(Json.string(delivery, "token")) && proof.inventoryDomain().equals(domain)
                && proof.sessionEpoch().equals(epoch) && proof.serverId().equals(Json.string(delivery, "nodeId"))
                && proof.saveReceipt() != null && !proof.saveReceipt().isBlank() && proof.committedAt() > 0;
    }

    private void finish(String id, JsonObject result) throws Exception {
        http.post("deliveries/finish", result); finishes.remove(id, result); reported.remove(id);
        if (Set.of("INVENTORY", "MAIL_PENDING").contains(Json.string(result, "state"))) {
            main(() -> { preparation.delivered(id, UUID.fromString(Json.string(result, "playerUuid"))); return null; });
        }
    }

    private static JsonObject identity(JsonObject d) {
        JsonObject out = new JsonObject();
        for (String key : Set.of("id", "playerUuid", "sessionEpoch", "token")) if (d.has(key)) out.add(key, d.get(key).deepCopy());
        return out;
    }

    private void report(String id, Exception failure) {
        if (!closed && reported.add(id)) plugin.getLogger().log(Level.WARNING, "补给交付等待恢复或核对：" + id, failure);
    }

    @Override public void close() { closed = true; timer.cancel(); workers.shutdownNow(); }
}
