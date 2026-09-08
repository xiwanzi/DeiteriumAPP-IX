package cafe.deuterium.core.runtime;

import org.bukkit.Bukkit;
import org.bukkit.entity.Player;
import java.util.*;
import java.util.concurrent.ConcurrentHashMap;

public final class Sessions {
    public record PlayerSession(UUID playerUuid, String gameId, String sessionEpoch) { }
    private final Map<UUID, PlayerSession> sessions = new ConcurrentHashMap<>();
    public PlayerSession join(Player player) {
        PlayerSession state = new PlayerSession(player.getUniqueId(), player.getName(), UUID.randomUUID().toString());
        sessions.put(state.playerUuid(), state); return state;
    }
    public PlayerSession quit(UUID id) { return sessions.remove(id); }
    public PlayerSession get(UUID id) { return sessions.get(id); }
    public List<PlayerSession> snapshot() { return sessions.values().stream().sorted(Comparator.comparing(PlayerSession::gameId)).toList(); }
    public boolean current(UUID id, String epoch) { PlayerSession current = sessions.get(id); return current != null && current.sessionEpoch().equals(epoch); }
    public Player require(UUID id, String epoch) {
        if (!Bukkit.isPrimaryThread()) throw new IllegalStateException("Player access requires main thread");
        Player player = Bukkit.getPlayer(id);
        if (player == null || !player.isOnline() || !current(id, epoch)) throw new cafe.deuterium.core.util.CoreFailure("PLAYER_SESSION_CHANGED", "玩家已离线或切换会话。");
        return player;
    }
    public void clear() { sessions.clear(); }
}
