package cafe.deuterium.core.game;

import cafe.deuterium.core.config.CoreConfig;
import cafe.deuterium.core.runtime.WorkPool;
import org.bukkit.Bukkit;
import org.bukkit.OfflinePlayer;
import org.bukkit.entity.Player;
import org.bukkit.event.*;
import org.bukkit.event.player.AsyncPlayerChatEvent;
import org.bukkit.plugin.Plugin;
import java.lang.reflect.Method;
import java.util.UUID;
import java.util.concurrent.atomic.AtomicLong;
import java.util.function.Supplier;

/** Capture final approved public output. Never uses a raw pre-filter message to bypass TrChat. */
public final class ChatEvents implements Listener {
    private final Plugin plugin;
    private final Supplier<CoreConfig> config;
    private final ChatAccess chat;
    private final WorkPool workers;
    private final AtomicLong lastWarning = new AtomicLong();
    private volatile String mode = "Bukkit";
    public ChatEvents(Plugin plugin, Supplier<CoreConfig> config, ChatAccess chat, WorkPool workers) { this.plugin = plugin; this.config = config; this.chat = chat; this.workers = workers; }
    public String mode() { return mode; }
    public void register() {
        HandlerList.unregisterAll(this);
        Plugin trchat = Bukkit.getPluginManager().getPlugin("TrChat");
        if (trchat == null || !trchat.isEnabled()) { mode = "Bukkit"; Bukkit.getPluginManager().registerEvents(this, plugin); return; }
        try {
            ClassLoader loader = trchat.getClass().getClassLoader();
            Class<? extends Event> eventClass = Class.forName("me.arasple.mc.trchat.api.event.TrChatSendEvent", false, loader).asSubclass(Event.class);
            Method player = eventClass.getMethod("getPlayer"), channel = eventClass.getMethod("getChannel"), kind = eventClass.getMethod("getType"), component = eventClass.getMethod("getComponent");
            Method channelId = channel.getReturnType().getMethod("getId"), plain = component.getReturnType().getMethod("toPlainText");
            Method playerData = Class.forName("me.arasple.mc.trchat.util.BukkitUtilKt", false, loader).getMethod("getData", OfflinePlayer.class);
            Method shadow = playerData.getReturnType().getMethod("isShadowMuted");
            Bukkit.getPluginManager().registerEvent(eventClass, this, EventPriority.MONITOR, (listener,event) -> {
                // TabooLib proxy event subclasses share one HandlerList; Bukkit does not filter this raw executor.
                if (!eventClass.isInstance(event)) return;
                try {
                    if (event instanceof Cancellable cancellable && cancellable.isCancelled()) return;
                    if (!kind.invoke(event).toString().equals("COMMON") || !config.get().publicChannels().contains(channelId.invoke(channel.invoke(event)).toString())) return;
                    Player sender = (Player) player.invoke(event);
                    if ((boolean) shadow.invoke(playerData.invoke(null, sender))) return;
                    enqueue(sender.getUniqueId(), sender.getName(), (String) plain.invoke(component.invoke(event)));
                } catch (Exception failure) { warning(); }
            }, plugin, true);
            mode = "TrChat final-public API";
        } catch (ReflectiveOperationException | LinkageError unsupported) {
            mode = "TrChat API unavailable";
            plugin.getLogger().warning("TrChat 公共消息 API 不匹配，已关闭采集；不会回退采集被取消或私密消息。");
        }
    }
    @EventHandler(priority = EventPriority.MONITOR, ignoreCancelled = true)
    public void onChat(AsyncPlayerChatEvent event) { enqueue(event.getPlayer().getUniqueId(), event.getPlayer().getName(), event.getMessage()); }
    private void enqueue(UUID id, String name, String content) {
        workers.submit(() -> { chat.captured(id, name, content); return null; }).exceptionally(error -> { warning(); return null; });
    }
    private void warning() {
        long now = System.currentTimeMillis(), last = lastWarning.get();
        if (now - last > 30000 && lastWarning.compareAndSet(last, now)) plugin.getLogger().warning("公共消息未能写入 Core 队列，请检查 /dc status。游戏内聊天不受影响。");
    }
}
