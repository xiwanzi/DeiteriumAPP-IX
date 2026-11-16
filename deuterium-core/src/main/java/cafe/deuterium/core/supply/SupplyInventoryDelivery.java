package cafe.deuterium.core.supply;

import cafe.deuterium.core.api.PlayerDataService;
import cafe.deuterium.core.items.InventoryGrant;
import cafe.deuterium.core.runtime.Sessions;
import cafe.deuterium.core.util.CoreFailure;
import java.util.List;
import java.util.UUID;
import java.util.function.Supplier;
import org.bukkit.inventory.ItemStack;

/** One-item inventory transaction with an explicit pre-write versus uncertain-write outcome. */
public final class SupplyInventoryDelivery {
    public record Attempt(String state, String message, String receipt, long committedAt) { }
    private final Sessions sessions;
    private final Supplier<PlayerDataService> provider;

    public SupplyInventoryDelivery(Sessions sessions, Supplier<PlayerDataService> provider) {
        this.sessions = sessions; this.provider = provider;
    }

    public String inspect(UUID player, String coreEpoch, String domain, UUID operation, ItemStack item) throws Exception {
        var current = sessions.require(player, coreEpoch);
        try (var lease = provider.get().acquire(current, domain, operation)) {
            lease.verifyCurrent(); lease.verifyItems(List.of(item));
            return lease.sessionEpoch();
        }
    }

    public Attempt deliver(UUID player, String coreEpoch, String inventoryEpoch, String domain, UUID operation, ItemStack item) {
        boolean touched = false;
        try {
            var current = sessions.require(player, coreEpoch);
            try (var lease = provider.get().acquire(current, domain, operation)) {
                lease.verifyCurrent(); lease.verifyItems(List.of(item));
                if (!inventoryEpoch.equals(lease.sessionEpoch())) return new Attempt("PENDING", "INVENTORY_NOT_TOUCHED", "", 0);
                ItemStack[] planned = InventoryGrant.plan(current.getInventory().getStorageContents(), item, 1, current.getInventory().getMaxStackSize());
                lease.verifyCurrent();
                touched = true;
                current.getInventory().setStorageContents(planned);
                String receipt = lease.saveAndConfirm();
                if (receipt == null || receipt.isBlank()) return new Attempt("REVIEW", "背包保存回执缺失", "", 0);
                return new Attempt("INVENTORY", "", receipt, System.currentTimeMillis());
            }
        } catch (Exception failure) {
            java.util.logging.Logger.getLogger(SupplyInventoryDelivery.class.getName()).log(
                    java.util.logging.Level.WARNING, "补给背包交付未完成：" + operation, failure);
            if (touched) return new Attempt("REVIEW", "背包写入后保存结果未确认", "", 0);
            if (failure instanceof CoreFailure known && List.of("INVENTORY_FULL", "PLAYER_OFFLINE", "PLAYER_SESSION_CHANGED").contains(known.code()))
                return new Attempt("MAIL_PENDING", known.code(), "", 0);
            return new Attempt("PENDING", "INVENTORY_NOT_TOUCHED", "", 0);
        }
    }
}
