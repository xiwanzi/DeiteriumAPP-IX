package cafe.deuterium.core.items;

import cafe.deuterium.core.api.ItemVersion;
import cafe.deuterium.core.config.CoreConfig;
import cafe.deuterium.core.storage.CatalogStore;
import cafe.deuterium.core.util.Checks;
import cafe.deuterium.core.util.CoreFailure;
import org.bukkit.Bukkit;
import org.bukkit.ChatColor;
import org.bukkit.inventory.ItemStack;
import java.util.List;

public final class ItemCodec {
    public CatalogStore.Captured capture(String ref, ItemStack original, String overrideName, CoreConfig config) {
        mainThread();
        if (original == null || original.getType().isAir() || original.getAmount() < 1) throw CoreFailure.invalid("请手持要保存的物品。");
        ItemStack unit = original.clone(); unit.setAmount(1);
        byte[] bytes = unit.serializeAsBytes();
        NbtLimits.Info info = NbtLimits.inspect(bytes, config.maxItemBytes());
        ItemStack restored = ItemStack.deserializeBytes(bytes);
        if (restored.getAmount() != 1 || !restored.isSimilar(unit)) throw new CoreFailure("ITEM_ROUNDTRIP_FAILED", "当前物品无法无损保存，已停止写入物品库。");
        String name = overrideName == null || overrideName.isBlank()
                ? unit.hasItemMeta() && unit.getItemMeta().hasDisplayName() ? ChatColor.stripColor(unit.getItemMeta().getDisplayName()) : info.itemId()
                : overrideName;
        if (name == null || name.codePointCount(0, name.length()) > 128) throw CoreFailure.invalid("显示名称最多 128 个字符。");
        Checks.text(name, 512, false);
        return new CatalogStore.Captured(ref, bytes, info.itemId(), name, "", 99999,
                config.defaultServers(), info.requiredMods().stream().sorted().toList(), config.inventoryDomain(), config.compatibilityProfile());
    }
    public ItemStack decode(ItemVersion version, CoreConfig config) {
        mainThread();
        if (!version.codec().equals("bukkit-bytes-v1") || !version.compatibleServerIds().contains(config.nodeId())
                || !version.inventoryDomain().equals(config.inventoryDomain())
                || !version.compatibilityProfile().equals(config.compatibilityProfile()))
            throw new CoreFailure("ITEM_INCOMPATIBLE", "当前服务器不在该物品版本的兼容范围内。");
        byte[] bytes = version.payload();
        if (!Checks.sha(bytes).equals(version.payloadSha256())) throw new CoreFailure("ITEM_CORRUPT", "物品内容校验失败。");
        NbtLimits.Info info = NbtLimits.inspect(bytes, config.maxItemBytes());
        if (!info.itemId().equals(version.itemId())) throw new CoreFailure("ITEM_CORRUPT", "物品标识与内容不一致。");
        ItemStack item = ItemStack.deserializeBytes(bytes);
        if (item == null || item.getType().isAir() || item.getAmount() != 1) throw new CoreFailure("ITEM_INCOMPATIBLE", "物品无法在当前服务器完整还原。");
        return item;
    }
    public CatalogStore.Captured policy(ItemVersion old, List<String> servers, long quantity, CoreConfig config) {
        List<String> checked = CoreConfig.servers(servers, config.nodes(), old.inventoryDomain(), old.compatibilityProfile(), false);
        if (quantity < 1 || quantity > 99999) throw CoreFailure.invalid("单次交付数量上限必须在 1–99999 之间。");
        return new CatalogStore.Captured(old.itemRef(), old.payload(), old.itemId(), old.displayName(), old.description(), quantity,
                checked, old.requiredMods(), old.inventoryDomain(), old.compatibilityProfile());
    }
    private static void mainThread() { if (!Bukkit.isPrimaryThread()) throw new IllegalStateException("Game item access requires main thread"); }
}
