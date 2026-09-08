package cafe.deuterium.core.storage;

import cafe.deuterium.core.runtime.Sessions.PlayerSession;
import cafe.deuterium.core.util.CoreFailure;
import java.sql.ResultSet;
import java.util.*;

public final class PlayerStore {
    public record PlayerIdentity(UUID playerUuid, String gameId, String serverId, boolean online, long lastSeen) { }
    private final Database db;
    private final String node;
    public PlayerStore(Database db, String node) { this.db = db; this.node = node; }
    public void seen(PlayerSession session) {
        db.transaction(c -> {
            String sql = db.shared()
                    ? "INSERT INTO dc_players VALUES(?,?,?,?,true,?) ON DUPLICATE KEY UPDATE game_id=VALUES(game_id),node_id=VALUES(node_id),session_epoch=VALUES(session_epoch),online=true,last_seen=VALUES(last_seen)"
                    : "INSERT INTO dc_players VALUES(?,?,?,?,true,?) ON CONFLICT(player_uuid) DO UPDATE SET game_id=excluded.game_id,node_id=excluded.node_id,session_epoch=excluded.session_epoch,online=true,last_seen=excluded.last_seen";
            Sql.update(c, sql, session.playerUuid().toString(), session.gameId(), node, session.sessionEpoch(), System.currentTimeMillis()); return null;
        });
    }
    public void left(PlayerSession session) { if (session != null) db.read(c -> Sql.update(c, "UPDATE dc_players SET online=false,last_seen=? WHERE player_uuid=? AND node_id=? AND session_epoch=?", System.currentTimeMillis(), session.playerUuid().toString(), node, session.sessionEpoch())); }
    public void offline() { db.read(c -> Sql.update(c, "UPDATE dc_players SET online=false WHERE node_id=?", node)); }
    public PlayerIdentity resolve(String name) {
        return db.read(c -> {
            List<PlayerIdentity> matches = new ArrayList<>();
            try (var s = Sql.statement(c, "SELECT * FROM dc_players WHERE LOWER(game_id)=LOWER(?) LIMIT 2", name); ResultSet r = s.executeQuery()) {
                while (r.next()) matches.add(new PlayerIdentity(UUID.fromString(r.getString("player_uuid")), r.getString("game_id"), r.getString("node_id"),
                        r.getBoolean("online") && r.getLong("last_seen") > System.currentTimeMillis() - 60000, r.getLong("last_seen")));
            }
            if (matches.size() > 1) throw new CoreFailure("IDENTITY_AMBIGUOUS", "此名称对应多个历史身份，请使用已绑定身份核实。");
            if (matches.isEmpty()) throw new CoreFailure("PLAYER_NOT_FOUND", "玩家尚未在已接入的服务器出现。");
            return matches.getFirst();
        });
    }
}
