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
    public void heartbeat() { db.read(c -> Sql.update(c,"UPDATE dc_players SET last_seen=? WHERE node_id=? AND online=true",System.currentTimeMillis(),node)); }
    public long nameCursor(){return db.read(c->Sql.count(c,"SELECT COALESCE(MAX(cursor_value),0) FROM dc_player_name_cursors WHERE node_id=?",node));}
    public void nameCursor(long value){if(value<0)throw CoreFailure.invalid("名称游标无效");db.read(c->Sql.update(c,db.shared()?
        "INSERT INTO dc_player_name_cursors VALUES(?,?) ON DUPLICATE KEY UPDATE cursor_value=GREATEST(cursor_value,VALUES(cursor_value))":
        "INSERT INTO dc_player_name_cursors VALUES(?,?) ON CONFLICT(node_id) DO UPDATE SET cursor_value=MAX(cursor_value,excluded.cursor_value)",node,value));}
    public long nameCursor(String consumer){return db.read(c->Sql.count(c,"SELECT COALESCE(MAX(cursor_value),0) FROM dc_player_name_consumers WHERE node_id=? AND consumer=?",node,consumer));}
    public void nameCursor(String consumer,long value){if(value<0)throw CoreFailure.invalid("名称游标无效");db.read(c->Sql.update(c,db.shared()?
        "INSERT INTO dc_player_name_consumers VALUES(?,?,?) ON DUPLICATE KEY UPDATE cursor_value=GREATEST(cursor_value,VALUES(cursor_value))":
        "INSERT INTO dc_player_name_consumers VALUES(?,?,?) ON CONFLICT(node_id,consumer) DO UPDATE SET cursor_value=MAX(cursor_value,excluded.cursor_value)",node,consumer,value));}
    public record NameChange(UUID uuid,String name,long sequence){}
    public List<NameChange> pendingNames(long after){return db.read(c->{var result=new ArrayList<NameChange>();try(var s=Sql.statement(c,"SELECT player_uuid,game_id,sequence_id FROM dc_player_names WHERE sequence_id>? ORDER BY sequence_id LIMIT 100",after);var rows=s.executeQuery()){while(rows.next())result.add(new NameChange(UUID.fromString(rows.getString(1)),rows.getString(2),rows.getLong(3)));}return result;});}
    public void canonicalName(UUID uuid,String name,long sequence){
        if(sequence<1||!name.matches("[A-Za-z0-9_]{1,32}"))throw CoreFailure.invalid("名称更新无效");
        db.transaction(c->{
            Sql.update(c,db.ignoreInsert()+" INTO dc_player_names VALUES(?,?,0)",uuid.toString(),name);
            long old=Sql.count(c,"SELECT sequence_id FROM dc_player_names WHERE player_uuid=?"+db.lock(),uuid.toString());
            if(sequence<=old)return null;
            Sql.update(c,"UPDATE dc_player_names SET game_id=?,sequence_id=? WHERE player_uuid=?",name,sequence,uuid.toString());
            Sql.update(c,"UPDATE dc_players SET game_id=? WHERE player_uuid=?",name,uuid.toString());return null;
        });
    }
    public String canonicalName(UUID uuid){return db.read(c->{try(var s=Sql.statement(c,"SELECT game_id FROM dc_player_names WHERE player_uuid=?",uuid.toString());var r=s.executeQuery()){return r.next()?r.getString(1):null;}});}
    public PlayerIdentity resolve(String name) {
        return db.read(c -> {
            List<PlayerIdentity> matches = new ArrayList<>();
            try (var s = Sql.statement(c, "SELECT p.player_uuid,COALESCE(n.game_id,p.game_id) AS game_id,p.node_id,p.online,p.last_seen FROM dc_players p LEFT JOIN dc_player_names n ON n.player_uuid=p.player_uuid WHERE LOWER(COALESCE(n.game_id,p.game_id))=LOWER(?) UNION ALL SELECT n.player_uuid,n.game_id,? AS node_id,false AS online,0 AS last_seen FROM dc_player_names n WHERE LOWER(n.game_id)=LOWER(?) AND NOT EXISTS(SELECT 1 FROM dc_players p WHERE p.player_uuid=n.player_uuid) LIMIT 2", name,node,name); ResultSet r = s.executeQuery()) {
                while (r.next()) matches.add(new PlayerIdentity(UUID.fromString(r.getString("player_uuid")), r.getString("game_id"), r.getString("node_id"),
                        r.getBoolean("online") && r.getLong("last_seen") > System.currentTimeMillis() - 60000, r.getLong("last_seen")));
            }
            if (matches.size() > 1) throw new CoreFailure("IDENTITY_AMBIGUOUS", "此名称对应多个历史身份，请使用已绑定身份核实。");
            if (matches.isEmpty()) throw new CoreFailure("PLAYER_NOT_FOUND", "玩家尚未在已接入的服务器出现。");
            return matches.getFirst();
        });
    }
}
