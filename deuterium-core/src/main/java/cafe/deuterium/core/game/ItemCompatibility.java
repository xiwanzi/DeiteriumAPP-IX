package cafe.deuterium.core.game;

import cafe.deuterium.core.items.NbtLimits;
import org.bukkit.entity.Player;
import org.bukkit.inventory.ItemStack;
import java.lang.reflect.Method;
import java.util.List;

public final class ItemCompatibility {
    private ItemCompatibility() { }
    public static void verify(Player player, List<ItemStack> items) throws Exception {
        for (ItemStack item : items) {
            if (item == null || item.getType().isAir()) continue;
            NbtLimits.Info info = NbtLimits.inspect(item.serializeAsBytes(), 1048576);
            if (info.requiredMods().isEmpty()) continue;
            Class<?> api;
            try { api = Class.forName("com.mohistmc.youer.api.PlayerAPI"); }
            catch (ClassNotFoundException missing) { throw new IllegalStateException("Cannot verify modded client compatibility", missing); }
            Method hasMod = api.getMethod("hasMod", Player.class, String.class);
            for (String mod : info.requiredMods())
                if (!(boolean) hasMod.invoke(null, player, mod)) throw new IllegalStateException("Client does not advertise required mod: " + mod);
        }
    }
}
