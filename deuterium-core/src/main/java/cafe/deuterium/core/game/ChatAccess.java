package cafe.deuterium.core.game;

import cafe.deuterium.core.config.CoreConfig;
import cafe.deuterium.core.runtime.GameThread;
import cafe.deuterium.core.storage.InboxStore;
import cafe.deuterium.core.storage.OutboxStore;
import cafe.deuterium.core.util.Checks;
import cafe.deuterium.core.util.CoreFailure;
import cafe.deuterium.core.util.Json;
import com.google.gson.JsonObject;
import net.kyori.adventure.text.Component;
import org.bukkit.Bukkit;
import org.bukkit.ChatColor;
import java.nio.charset.StandardCharsets;
import java.time.Instant;
import java.util.Map;
import java.util.UUID;
import java.util.concurrent.TimeUnit;
import java.util.function.Supplier;

public final class ChatAccess {
    private final Supplier<CoreConfig> config;
    private final OutboxStore outbox;
    private final InboxStore inbox;
    private final GameThread game;
    private volatile GameChatChannel gameChat;
    public void bindGameChat(GameChatChannel value) { gameChat=value; }
    public boolean gameClient(org.bukkit.entity.Player player) { return gameChat!=null&&gameChat.supports(player); }
    public ChatAccess(Supplier<CoreConfig> config, OutboxStore outbox, InboxStore inbox, GameThread game) { this.config = config; this.outbox = outbox; this.inbox = inbox; this.game = game; }
    public void captured(UUID player, String name, String text) { captured(player,name,text,""); }
    public void captured(UUID player, String name, String text, String legacy) {
        text = ChatColor.stripColor(text);
        if (text == null) return; text = text.strip();
        if (text.isBlank()) return;
        int points = text.codePointCount(0, text.length());
        if (points > 256) text = text.substring(0, text.offsetByCodePoints(0, 255)) + "…";
        Checks.text(text, 1024, false);
        String payload = Json.GSON.toJson(Map.of("playerUuid", player.toString(), "gameId", name, "content", text));
        String eventId=Checks.id("chat_");
        outbox.enqueue(eventId, "chat.public.event", payload);
        if(gameChat!=null)gameChat.publicMessage("msg_"+Checks.sha((config.get().nodeId()+":"+eventId).getBytes(StandardCharsets.UTF_8)).substring(0,32),player,name,text,legacy);
    }
    public void incoming(JsonObject payload) throws Exception {
        Json.fields(payload, "messageId", "senderUuid", "gameId", "content", "origin", "expiresAt", "suppressRebroadcast", "media", "serverId", "sentAt");
        String id = Checks.operationId(Json.string(payload, "messageId")), name = Json.string(payload, "gameId"), text = Json.string(payload, "content");
        if (!name.matches("[A-Za-z0-9_]{1,32}") || !java.util.Set.of("app","game").contains(Json.string(payload, "origin"))
                || !payload.has("suppressRebroadcast") || !payload.get("suppressRebroadcast").getAsBoolean()
                || text.codePointCount(0, text.length()) > 256) throw CoreFailure.invalid("不正确的公共聊天投递。");
        Checks.text(text, 1024, false); UUID.fromString(Json.string(payload, "senderUuid"));
        long expiry = Instant.parse(Json.string(payload, "expiresAt")).toEpochMilli();
        if (expiry <= System.currentTimeMillis()) return;
        String fingerprint = Checks.sha(Json.canonical(payload).getBytes(StandardCharsets.UTF_8));
        if (!inbox.reserve(id, fingerprint, expiry)) return;
        String line = config.get().chatFormat().replace("{player}", name).replace("{message}", text);
        game.call(Math.min(expiry, System.currentTimeMillis() + 5000), () -> {
            Component component = Component.text(line);
            Bukkit.getOnlinePlayers().forEach(player -> {
                if(gameChat!=null&&gameChat.supports(player))gameChat.appMessage(player,payload);
                else if(Json.string(payload,"origin").equals("app"))player.sendMessage(component);
            });
            return null;
        }).get(5, TimeUnit.SECONDS);
        inbox.shown(id);
    }
}
