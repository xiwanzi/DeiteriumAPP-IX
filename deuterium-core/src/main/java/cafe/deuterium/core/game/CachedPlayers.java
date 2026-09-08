package cafe.deuterium.core.game;

import cafe.deuterium.core.storage.PlayerStore;
import org.bukkit.Bukkit;

/** Historical profiles already resolved by the game server, never name-derived UUIDs. */
public final class CachedPlayers {
    private CachedPlayers(){}
    public static PlayerStore.PlayerIdentity find(String name,String node){
        var player=Bukkit.getOfflinePlayerIfCached(name);
        if(player==null||player.getName()==null||!player.getName().equalsIgnoreCase(name))return null;
        return new PlayerStore.PlayerIdentity(player.getUniqueId(),player.getName(),node,player.isOnline(),player.getLastPlayed());
    }
}
