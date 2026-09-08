package cafe.deuterium.core.bridge;

import cafe.deuterium.core.config.CoreConfig;
import cafe.deuterium.core.runtime.WorkPool;
import cafe.deuterium.core.storage.OutboxStore;
import cafe.deuterium.core.util.Checks;
import cafe.deuterium.core.util.Json;
import com.google.gson.JsonObject;
import java.net.http.HttpClient;
import java.net.http.WebSocket;
import java.nio.ByteBuffer;
import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.time.Instant;
import java.util.concurrent.*;
import java.util.concurrent.atomic.*;
import java.util.function.Supplier;
import java.util.logging.Logger;

public final class BridgeClient implements AutoCloseable {
    private final Supplier<CoreConfig> config;
    private final OutboxStore outbox;
    private final BridgeEndpoint endpoint;
    private final WorkPool workers;
    private final Logger log;
    private final HttpClient http;
    private final AtomicBoolean ticking = new AtomicBoolean();
    private final AtomicLong generation = new AtomicLong();
    private volatile WebSocket socket;
    private volatile boolean welcomed, connecting, closed;
    private volatile long nextConnect, connectedAt, lastPong, lastPing, lastState, lastFlush;
    private volatile String state = "DISABLED";
    private int failures;
    public BridgeClient(Supplier<CoreConfig> config, OutboxStore outbox, BridgeEndpoint endpoint, WorkPool workers, Logger log) {
        this.config = config; this.outbox = outbox; this.endpoint = endpoint; this.workers = workers; this.log = log;
        http = HttpClient.newBuilder().connectTimeout(Duration.ofSeconds(8)).followRedirects(HttpClient.Redirect.NEVER).build();
    }
    public String state() { return state; }
    public boolean connected() { return welcomed && socket != null && !closed; }
    public void tick() {
        if (closed || !ticking.compareAndSet(false, true)) return;
        workers.submit(() -> { try { step(); } finally { ticking.set(false); } return null; })
                .exceptionally(error -> { ticking.set(false); disconnect("WORKER_OR_STORAGE_UNAVAILABLE"); return null; });
    }
    private void step() throws Exception {
        long now = System.currentTimeMillis(); CoreConfig c = config.get();
        if (!c.bridge().enabled()) { if (socket != null) disconnect("DISABLED"); state = "DISABLED"; return; }
        if (!welcomed) {
            if (connecting && now - connectedAt > 12000) disconnect("HANDSHAKE_TIMEOUT");
            if (!connecting && now >= nextConnect) connect(c);
            return;
        }
        WebSocket current = socket;
        if (current == null || current.isOutputClosed()) { disconnect("DISCONNECTED"); return; }
        if (lastPing > lastPong && now - lastPing > 10000) { disconnect("HEARTBEAT_TIMEOUT"); return; }
        if (now - lastPing >= 20000) { lastPing = now; current.sendPing(ByteBuffer.wrap(new byte[]{1})).get(5, TimeUnit.SECONDS); }
        if (now - lastState >= 10000) {
            send("core.status", null, endpoint.status());
            send("core.presence.snapshot", null, endpoint.presence());
            lastState = now;
        }
        if (now - lastFlush < 1000) return;
        lastFlush = now;
        for (OutboxStore.Event event : outbox.pending(16)) {
            JsonObject frame = envelope(event.type(), Checks.id("tx_"), Json.object(event.payload(), 28000));
            frame.addProperty("eventId", event.id()); write(frame);
        }
    }
    private synchronized void connect(CoreConfig c) {
        if (closed || connecting) return;
        connecting = true; state = "CONNECTING"; connectedAt = System.currentTimeMillis();
        long epoch = generation.incrementAndGet();
        http.newWebSocketBuilder().connectTimeout(Duration.ofSeconds(8))
                .header("Authorization", "Bearer " + c.bridge().token()).header("X-Deuterium-Node-ID", c.nodeId())
                .buildAsync(c.bridge().uri(), new Incoming(epoch)).whenComplete((ws,error) -> { if (error != null && epoch == generation.get()) disconnect("CONNECT_FAILED"); });
    }
    private void send(String type, String request, JsonObject payload) throws Exception { write(envelope(type, request, payload)); }
    private static JsonObject envelope(String type, String request, JsonObject payload) {
        JsonObject frame = new JsonObject(); frame.addProperty("type", type); frame.addProperty("sentAt", Instant.now().toString());
        if (request != null) frame.addProperty("requestId", request); frame.add("payload", payload); return frame;
    }
    private synchronized void write(JsonObject value) throws Exception {
        WebSocket ws = socket; if (ws == null || closed) throw new IllegalStateException("Bridge offline");
        String frame = Json.GSON.toJson(value);
        if (frame.getBytes(StandardCharsets.UTF_8).length > 32768) throw new IllegalArgumentException("Frame exceeds bridge limit");
        ws.sendText(frame, true).get(5, TimeUnit.SECONDS);
    }
    private void receive(JsonObject frame) throws Exception {
        String type = Json.string(frame, "type");
        JsonObject payload = frame.getAsJsonObject("payload"); if (payload == null) throw new IllegalArgumentException("Missing payload");
        String request = frame.has("requestId") ? Json.string(frame, "requestId") : null;
        switch (type) {
            case "core.welcome" -> {
                if (Json.integer(payload, "protocolVersion") != 1 || !Json.string(payload, "serverId").equals(config.get().nodeId())) throw new IllegalArgumentException("Invalid welcome identity");
                if(!payload.has("features")||!payload.getAsJsonArray("features").asList().stream().anyMatch(v->v.isJsonPrimitive()&&v.getAsString().equals("core-v1")))throw new IllegalArgumentException("Backend does not support Core runtime v1");
                welcomed = true; connecting = false; failures = 0; lastPong = System.currentTimeMillis(); lastPing = lastPong; lastState = 0; state = "CONNECTED";
            }
            case "core.event.result" -> {
                String id = Checks.operationId(Json.string(payload, "eventId")); String result = Json.string(payload, "status");
                if (result.equals("committed")) { outbox.acknowledged(id); endpoint.afterEventAcknowledged(id); }
                else if (result.equals("rejected")) {
                    String code = payload.has("error") ? Json.string(payload.getAsJsonObject("error"), "code") : "REJECTED";
                    outbox.rejected(id, code.length() <= 64 ? code : "REJECTED");
                }
            }
            case "chat.app.delivery" -> {
                endpoint.appChat(payload);
                JsonObject ack = new JsonObject(); ack.addProperty("messageId", Json.string(payload, "messageId"));
                send("chat.delivery.ack", Checks.id("ack_"), ack);
            }
            case "core.command" -> send("core.command.result", request, endpoint.command(payload));
            case "chat.delivery.ack.result", "core.status.result", "core.presence.result" -> { }
            case "error" -> log.warning("后端未接受一条 Core 消息，请核对双方协议版本。");
            default -> throw new IllegalArgumentException("Unknown backend message type");
        }
    }
    public synchronized void reconnect() { disconnect("RECONNECTING"); failures = 0; nextConnect = 0; }
    private synchronized void disconnect(String reason) {
        generation.incrementAndGet(); WebSocket old = socket; socket = null; welcomed = false; connecting = false; state = reason;
        if (old != null) old.abort();
        long backoff = Math.min(30000, 1000L << Math.min(failures++, 5));
        nextConnect = System.currentTimeMillis() + ThreadLocalRandom.current().nextLong(Math.max(500, backoff / 2), backoff + 1);
    }
    @Override public void close() { closed = true; disconnect("STOPPED"); http.shutdownNow(); }
    private final class Incoming implements WebSocket.Listener {
        private final long epoch; private final StringBuilder partial = new StringBuilder();
        private Incoming(long epoch) { this.epoch = epoch; }
        private boolean active() { return !closed && epoch == generation.get(); }
        @Override public void onOpen(WebSocket ws) {
            if (!active()) { ws.abort(); return; }
            socket = ws; ws.request(1);
            workers.submit(() -> { send("core.hello", "hello", Json.tree(java.util.Map.of("protocolVersion", 1))); return null; })
                    .exceptionally(error -> { disconnect("HELLO_FAILED"); return null; });
        }
        @Override public CompletionStage<?> onText(WebSocket ws, CharSequence data, boolean last) {
            if (!active()) { ws.abort(); return null; }
            partial.append(data);
            if (partial.length() > 32768) { disconnect("FRAME_TOO_LARGE"); return null; }
            if (!last) { ws.request(1); return null; }
            String message = partial.toString(); partial.setLength(0);
            if (message.getBytes(StandardCharsets.UTF_8).length > 32768) { disconnect("FRAME_TOO_LARGE"); return null; }
            return workers.submit(() -> { if (active()) receive(Json.object(message, 32768)); return null; })
                    .whenComplete((result,error) -> { if (error != null) { if (active()) disconnect("PROTOCOL_OR_SERVICE_ERROR"); } else if (active()) ws.request(1); });
        }
        @Override public CompletionStage<?> onBinary(WebSocket ws, ByteBuffer data, boolean last) { if (active()) disconnect("BINARY_FRAME_REJECTED"); return null; }
        @Override public CompletionStage<?> onPing(WebSocket ws, ByteBuffer message) { ws.request(1); return ws.sendPong(message); }
        @Override public CompletionStage<?> onPong(WebSocket ws, ByteBuffer message) { lastPong = System.currentTimeMillis(); ws.request(1); return null; }
        @Override public CompletionStage<?> onClose(WebSocket ws, int status, String reason) { if (active()) disconnect("DISCONNECTED"); return null; }
        @Override public void onError(WebSocket ws, Throwable error) { if (active()) disconnect("CONNECTION_ERROR"); }
    }
}
