package cafe.deuterium.core.game;

import cafe.deuterium.core.items.NbtLimits;
import org.bukkit.entity.Player;
import org.bukkit.inventory.ItemStack;
import java.util.List;

public final class ItemCompatibility {
    private ItemCompatibility() { }
    public static void verify(Player player, List<ItemStack> items) throws Exception {
        boolean needsRegistrySync = false;
        for (ItemStack item : items) {
            if (item == null || item.getType().isAir()) continue;
            NbtLimits.Info info = NbtLimits.inspect(item.serializeAsBytes(), 1048576);
            needsRegistrySync |= !info.requiredMods().isEmpty();
        }
        if (needsRegistrySync) NeoForgeRegistrySync.verify(player);
    }
}
