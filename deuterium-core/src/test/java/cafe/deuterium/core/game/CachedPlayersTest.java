package cafe.deuterium.core.game;

import org.bukkit.Bukkit;
import org.bukkit.OfflinePlayer;
import org.junit.jupiter.api.Test;
import java.util.UUID;
import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

class CachedPlayersTest {
    @Test void historicalPlayerWithoutCoreJoinUsesExactCachedUuidAndRealLastSeen(){
        UUID id=UUID.randomUUID();OfflinePlayer player=mock(OfflinePlayer.class);when(player.getName()).thenReturn("HistoricalPlayer");when(player.getUniqueId()).thenReturn(id);when(player.getLastPlayed()).thenReturn(1725000000000L);when(player.isOnline()).thenReturn(false);
        try(var bukkit=mockStatic(Bukkit.class)){
            bukkit.when(()->Bukkit.getOfflinePlayerIfCached("historicalplayer")).thenReturn(player);
            var found=CachedPlayers.find("historicalplayer","amiya");assertNotNull(found);assertEquals(id,found.playerUuid());assertEquals(1725000000000L,found.lastSeen());assertFalse(found.online());
            bukkit.verify(()->Bukkit.getOfflinePlayerIfCached("historicalplayer"));bukkit.verifyNoMoreInteractions();
        }
    }
    @Test void missingCacheDoesNotInventAnOfflineUuid(){try(var bukkit=mockStatic(Bukkit.class)){assertNull(CachedPlayers.find("NobodyHasSeenMe","amiya"));bukkit.verify(()->Bukkit.getOfflinePlayerIfCached("NobodyHasSeenMe"));bukkit.verifyNoMoreInteractions();}}
}
