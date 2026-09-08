package cafe.deuterium.core.api;

import org.bukkit.entity.Player;
import org.bukkit.inventory.ItemStack;
import java.util.List;
import java.util.UUID;

/**
 * Synchronous main-thread inventory barrier implemented by the actual sync owner.
 * Never register a timer-based or optimistic implementation for a shared inventory domain.
 */
public interface PlayerDataService {
    boolean available();
    String providerName();
    Lease acquire(Player player, String inventoryDomain, UUID operationId) throws Exception;
    /**
     * Thread-safe read-only lookup; never accesses a live Player or replays a grant.
     * Null means absent/unsupported, NEVER proof that an item was not issued.
     * A positive proof must exactly match all four requested identity fields.
     */
    default SaveProof lookupSaveProof(UUID playerUuid, String inventoryDomain, UUID operationId, String sessionEpoch) throws Exception { return null; }
    record SaveProof(UUID playerUuid, String inventoryDomain, UUID operationId, String sessionEpoch,
                     String serverId, String saveReceipt, long committedAt) { }
    interface Lease extends AutoCloseable {
        String sessionEpoch();
        void verifyCurrent() throws Exception;
        void verifyItems(List<ItemStack> items) throws Exception;
        String saveAndConfirm() throws Exception;
        @Override void close() throws Exception;
    }
}
