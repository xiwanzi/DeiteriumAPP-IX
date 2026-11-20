package cafe.deuterium.core.supply;

import cafe.deuterium.core.runtime.Sessions;
import cafe.deuterium.core.util.CoreFailure;
import cafe.deuterium.core.util.Json;
import com.google.gson.JsonObject;
import java.nio.charset.StandardCharsets;
import java.util.Arrays;
import java.util.Base64;
import java.util.HashMap;
import java.util.Map;
import java.util.Set;
import java.util.UUID;
import java.util.concurrent.ArrayBlockingQueue;
import java.util.concurrent.RejectedExecutionException;
import java.util.concurrent.ThreadPoolExecutor;
import java.util.concurrent.TimeUnit;
import org.bukkit.Bukkit;
import org.bukkit.entity.Player;
import org.bukkit.plugin.java.JavaPlugin;
import org.bukkit.plugin.messaging.PluginMessageListener;

/** Binds player requests to the actual connection before forwarding to the supply authority. */
public final class SupplyChannel implements PluginMessageListener, AutoCloseable {
    public static final String CHANNEL = "deuterium:supply";
    private static final Set<String> ACTIONS = Set.of("state", "draw", "exchange", "order", "exchange-status", "history", "ledger");
    private final JavaPlugin plugin;
    private final Sessions sessions;
    private final SupplyHttp http;
    private final Map<UUID, long[]> rates = new HashMap<>();
    private final Map<UUID, Integer> pending = new HashMap<>();
    private final ThreadPoolExecutor io = new ThreadPoolExecutor(2, 2, 0, TimeUnit.SECONDS,
            new ArrayBlockingQueue<>(64), task -> {
                Thread thread = new Thread(task, "Deuterium-Supply-Requests"); thread.setDaemon(true); return thread;
            }, new ThreadPoolExecutor.AbortPolicy());
    private final org.bukkit.scheduler.BukkitTask cleanup;
    private volatile boolean closed;

    public SupplyChannel(JavaPlugin plugin, Sessions sessions, SupplyHttp http) {
        this.plugin = plugin; this.sessions = sessions; this.http = http;
        Bukkit.getMessenger().registerIncomingPluginChannel(plugin, CHANNEL, this);
        Bukkit.getMessenger().registerOutgoingPluginChannel(plugin, CHANNEL);
        cleanup = Bukkit.getScheduler().runTaskTimer(plugin, () -> {
            rates.keySet().removeIf(id -> sessions.get(id) == null);
            pending.keySet().removeIf(id -> sessions.get(id) == null);
        }, 200, 200);
    }

    @Override public void onPluginMessageReceived(String channel, Player player, byte[] bytes) {
        if (closed || !CHANNEL.equals(channel) || bytes.length > 8192) return;
        String requestId = "";
        try {
            JsonObject packet = Json.object(new String(bytes, StandardCharsets.UTF_8), 8192);
            requestId = Json.string(packet, "requestId");
            String action = Json.string(packet, "action");
            if (!requestId.matches("[A-Za-z0-9_-]{1,80}") || !ACTIONS.contains(action)) throw CoreFailure.invalid("补给请求无效");
            UUID uuid = player.getUniqueId();
            long now = System.currentTimeMillis(); long[] rate = rates.computeIfAbsent(uuid, ignored -> new long[2]);
            if (now - rate[0] >= 1000) { rate[0] = now; rate[1] = 0; }
            if (++rate[1] > 12 || pending.getOrDefault(uuid, 0) >= 4) throw new CoreFailure("SUPPLY_BUSY", "补给请求正在处理，请稍候");
            Sessions.PlayerSession session = sessions.get(uuid);
            if (session == null) throw new CoreFailure("SUPPLY_SESSION", "玩家连接尚未就绪");
            JsonObject input = bindRequest(packet, session);
            pending.merge(uuid, 1, Integer::sum);
            String id = requestId;
            try {
                io.execute(() -> {
                    JsonObject result = envelope(id);
                    try { result.add("data", http.post(action, input)); result.addProperty("success", true); }
                    catch (Exception failure) { failure(result, failure); }
                    main(() -> {
                        pending.computeIfPresent(uuid, (key, count) -> count <= 1 ? null : count - 1);
                        if (sessions.current(uuid, session.sessionEpoch()) && player.isOnline()) reply(player, result);
                    });
                });
            } catch (RejectedExecutionException full) {
                pending.computeIfPresent(uuid, (key, count) -> count <= 1 ? null : count - 1);
                throw new CoreFailure("SUPPLY_BUSY", "补给服务繁忙，请使用原请求重试");
            }
        } catch (RuntimeException failure) {
            JsonObject result = envelope(requestId); failure(result, failure); reply(player, result);
        }
    }

    static JsonObject bindRequest(JsonObject packet, Sessions.PlayerSession session) {
        JsonObject input = new JsonObject();
        input.addProperty("playerUuid", session.playerUuid().toString());
        input.addProperty("sessionEpoch", session.sessionEpoch());
        input.add("requestId", packet.get("requestId"));
        for (String key : Set.of("draw", "exchange", "orderId", "exchangeId", "filter")) {
            if (packet.has(key)) input.add(key, packet.get(key).deepCopy());
        }
        return input;
    }

    private static JsonObject envelope(String id) {
        JsonObject result = new JsonObject(); result.addProperty("type", "supply-response"); result.addProperty("requestId", id); return result;
    }

    private static void failure(JsonObject result, Throwable failure) {
        JsonObject error = new JsonObject();
        error.addProperty("code", failure instanceof CoreFailure known ? known.code() : "RESULT_UNKNOWN");
        error.addProperty("message", failure instanceof CoreFailure ? failure.getMessage() : "结果尚未确认，请使用原请求重试");
        result.addProperty("success", false); result.add("error", error);
    }

    private void reply(Player player, JsonObject result) {
        byte[] bytes = Json.GSON.toJson(result).getBytes(StandardCharsets.UTF_8);
        if (bytes.length <= 26000) { player.sendPluginMessage(plugin, CHANNEL, bytes); return; }
        int size = 18000, count = (bytes.length + size - 1) / size;
        if (count > 128) return;
        for (int part = 0; part < count; part++) {
            JsonObject chunk = new JsonObject(); chunk.addProperty("type", "supply-chunk"); chunk.add("requestId", result.get("requestId"));
            chunk.addProperty("index", part); chunk.addProperty("count", count);
            chunk.addProperty("data", Base64.getEncoder().encodeToString(Arrays.copyOfRange(bytes, part * size, Math.min(bytes.length, (part + 1) * size))));
            player.sendPluginMessage(plugin, CHANNEL, Json.GSON.toJson(chunk).getBytes(StandardCharsets.UTF_8));
        }
    }

    private void main(Runnable action) { if (!closed && plugin.isEnabled()) Bukkit.getScheduler().runTask(plugin, () -> { if (!closed) action.run(); }); }
    @Override public void close() {
        closed = true; cleanup.cancel(); io.shutdownNow(); rates.clear(); pending.clear();
        Bukkit.getMessenger().unregisterIncomingPluginChannel(plugin, CHANNEL, this);
        Bukkit.getMessenger().unregisterOutgoingPluginChannel(plugin, CHANNEL);
    }
}
