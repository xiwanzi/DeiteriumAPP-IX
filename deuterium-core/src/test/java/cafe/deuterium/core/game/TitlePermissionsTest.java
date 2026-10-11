package cafe.deuterium.core.game;

import net.luckperms.api.LuckPerms;
import net.luckperms.api.platform.PlayerAdapter;
import net.luckperms.api.cacheddata.CachedPermissionData;
import net.luckperms.api.util.Tristate;
import org.bukkit.Bukkit;
import org.bukkit.entity.Player;
import org.bukkit.plugin.ServicesManager;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

class TitlePermissionsTest {
    @Test void contextualLuckPermsResultControlsAccessAndOpDoesNotBypassIt(){
        try(var bukkit=mockStatic(Bukkit.class)){
            var services=mock(ServicesManager.class);var api=mock(LuckPerms.class);var player=mock(Player.class);
            @SuppressWarnings("unchecked") PlayerAdapter<Player> adapter=mock(PlayerAdapter.class);
            var data=mock(CachedPermissionData.class);
            bukkit.when(Bukkit::getServicesManager).thenReturn(services);when(services.load(LuckPerms.class)).thenReturn(api);
            when(api.getPlayerAdapter(Player.class)).thenReturn(adapter);when(adapter.getPermissionData(player)).thenReturn(data);
            when(player.isOp()).thenReturn(true);when(player.hasPermission(anyString())).thenReturn(true);
            when(data.checkPermission("deuterium.title.dawn")).thenReturn(Tristate.UNDEFINED,Tristate.TRUE,Tristate.FALSE);
            assertFalse(TitlePermissions.allowed(player,"dawn"));assertTrue(TitlePermissions.allowed(player,"dawn"));assertFalse(TitlePermissions.allowed(player,"dawn"));
            when(services.load(LuckPerms.class)).thenReturn(null);assertFalse(TitlePermissions.allowed(player,"dawn"));
            verify(player,never()).hasPermission(anyString());
        }
    }
}
