package cafe.deuterium.core.game;

import net.luckperms.api.LuckPerms;
import org.bukkit.Bukkit;
import org.bukkit.entity.Player;

/** Uses LP's contextual cached result, so Bukkit's implicit OP default cannot unlock titles. */
final class TitlePermissions {
    static boolean allowed(Player player,String internalId){
        LuckPerms api=Bukkit.getServicesManager().load(LuckPerms.class);
        return api!=null&&api.getPlayerAdapter(Player.class).getPermissionData(player)
            .checkPermission("deuterium.title."+internalId).asBoolean();
    }
}
