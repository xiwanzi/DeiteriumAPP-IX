package cafe.deuterium.core.items;

import cafe.deuterium.core.util.CoreFailure;
import org.bukkit.inventory.ItemStack;
import org.bukkit.inventory.PlayerInventory;

/** Plan a complete grant before touching inventory; never drop overflow on the ground. */
public final class InventoryGrant {
    private InventoryGrant() { }
    public static ItemStack[] plan(ItemStack[] current, ItemStack template, int quantity, int inventoryLimit) {
        if (quantity < 1 || template == null || template.getType().isAir()) throw CoreFailure.invalid("无效给予数量或物品。");
        ItemStack[] planned = new ItemStack[current.length];
        for (int i = 0; i < current.length; i++) planned[i] = current[i] == null ? null : current[i].clone();
        int remaining = quantity;
        int max = Math.max(1, Math.min(inventoryLimit, template.getMaxStackSize()));
        for (int i = 0; i < planned.length && remaining > 0; i++) {
            ItemStack item = planned[i];
            if (item != null && !item.getType().isAir() && item.isSimilar(template)) {
                int n = Math.min(remaining, Math.max(0, max - item.getAmount()));
                item.setAmount(item.getAmount() + n); remaining -= n;
            }
        }
        for (int i = 0; i < planned.length && remaining > 0; i++) {
            if (planned[i] == null || planned[i].getType().isAir()) {
                int n = Math.min(remaining, max); planned[i] = template.clone(); planned[i].setAmount(n); remaining -= n;
            }
        }
        if (remaining > 0) throw new CoreFailure("INVENTORY_FULL", "背包空间不足，未给予任何物品。");
        return planned;
    }
    public static void apply(PlayerInventory inventory, ItemStack template, int quantity) {
        ItemStack[] planned = plan(inventory.getStorageContents(), template, quantity, inventory.getMaxStackSize());
        try { inventory.setStorageContents(planned); }
        catch (Throwable error) { throw new CoreFailure("RESULT_UNKNOWN", "背包写入出现异常，请先检查背包；系统不会自动再次发放。", error); }
    }
}
