package cafe.deuterium.core.game;

import cafe.deuterium.core.runtime.Sessions;
import cafe.deuterium.core.runtime.WorkPool;
import cafe.deuterium.core.storage.PlayerStore;
import org.bukkit.Bukkit;
import org.bukkit.event.EventHandler;
import org.bukkit.event.EventPriority;
import org.bukkit.event.Listener;
import org.bukkit.event.player.PlayerJoinEvent;
import org.bukkit.event.player.PlayerQuitEvent;

public final class PlayerEvents implements Listener {
    private final Sessions sessions;
    private final PlayerStore players;
    private final WorkPool workers;
    private Runnable changed = () -> {};
    public void onChange(Runnable listener) { changed = listener; }
    public PlayerEvents(Sessions sessions, PlayerStore players, WorkPool workers) { this.sessions = sessions; this.players = players; this.workers = workers; }
    public void initialize() { Bukkit.getOnlinePlayers().forEach(player -> {var session=sessions.join(player);workers.submit(()->{if(sessions.current(session.playerUuid(),session.sessionEpoch()))players.seen(session);return null;});}); }
    @EventHandler(priority = EventPriority.MONITOR)
    public void joined(PlayerJoinEvent event) {
        Sessions.PlayerSession session = sessions.join(event.getPlayer());
        changed.run();
        workers.submit(() -> { if (sessions.current(session.playerUuid(), session.sessionEpoch())) players.seen(session); return null; });
    }
    @EventHandler(priority = EventPriority.MONITOR)
    public void quit(PlayerQuitEvent event) {
        Sessions.PlayerSession previous = sessions.quit(event.getPlayer().getUniqueId());
        changed.run();
        workers.submit(() -> { players.left(previous); return null; });
    }
    public void refresh() { workers.submit(()->{players.heartbeat();return null;}); }
}
