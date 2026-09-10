package cafe.deuterium.core.mail;

import cafe.deuterium.core.api.PlayerDataService;
import cafe.deuterium.core.game.EconomyAccess;
import cafe.deuterium.mail.api.MailPlayerDataBarrier;
import org.bukkit.entity.Player;
import org.bukkit.inventory.ItemStack;
import java.util.List;
import java.util.UUID;
import java.util.function.Supplier;

/** Delegates every guarantee to the real sync owner; no timer or success-by-default fallback. */
public final class MailBarrierAdapter implements MailPlayerDataBarrier {
    private final Supplier<PlayerDataService> provider;
    private final EconomyAccess economy = new EconomyAccess();
    public MailBarrierAdapter(Supplier<PlayerDataService> provider) { this.provider = provider; }
    @Override public boolean available() { PlayerDataService s = provider.get(); return s != null && s.available(); }
    @Override public boolean creditRewardsAvailable() { return economy.mailCreditsAvailable(); }
    @Override public void ensureCredits(UUID player, UUID operation, long credits) {
        if (credits == 0) return;
        if (credits < 0 || credits > 1000000000000L) throw new IllegalArgumentException("Invalid mail credits");
        String id = "mailcredit_" + operation;
        var proof = economy.rewardMail(id, player, credits);
        if (!id.equals(proof.get("operationId").getAsString()) || !"COMPLETED".equals(proof.get("status").getAsString())
                || !player.toString().equals(proof.get("playerUuid").getAsString()) || !"CREDIT".equals(proof.get("currency").getAsString())
                || proof.get("amount").getAsBigDecimal().compareTo(java.math.BigDecimal.valueOf(credits)) != 0)
            throw new IllegalStateException("Mail credit commit proof mismatch");
    }
    @Override public SaveProof lookupSaveProof(UUID playerUuid, String domain, UUID operation, String epoch) throws Exception {
        PlayerDataService service = provider.get();
        if (service == null) throw new IllegalStateException("Player save proof provider unavailable");
        PlayerDataService.SaveProof proof = service.lookupSaveProof(playerUuid, domain, operation, epoch);
        if (provider.get() != service) throw new IllegalStateException("Player save proof provider changed");
        if (proof == null) return null;
        if (!playerUuid.equals(proof.playerUuid()) || !domain.equals(proof.inventoryDomain()) || !operation.equals(proof.operationId()) || !epoch.equals(proof.sessionEpoch())
                || proof.serverId() == null || !proof.serverId().matches("[a-z0-9_-]{1,64}") || proof.saveReceipt() == null || proof.saveReceipt().isBlank() || proof.saveReceipt().length() > 4096 || proof.committedAt() <= 0)
            throw new IllegalStateException("Player save proof identity or contents mismatch");
        return new SaveProof(proof.playerUuid(),proof.inventoryDomain(),proof.operationId(),proof.sessionEpoch(),proof.serverId(),proof.saveReceipt(),proof.committedAt());
    }
    @Override public Lease acquire(Player player, String domain, UUID operation) throws Exception {
        PlayerDataService service = provider.get();
        if (service == null || !service.available()) throw new IllegalStateException("Successful player-data synchronization unavailable");
        PlayerDataService.Lease lease = service.acquire(player, domain, operation);
        return new Lease() {
            @Override public String sessionEpoch() { return lease.sessionEpoch(); }
            @Override public void verifyCurrent() throws Exception { if (provider.get() != service) throw new IllegalStateException("Sync provider changed"); lease.verifyCurrent(); }
            @Override public void verifyItems(List<ItemStack> items) throws Exception { verifyCurrent(); lease.verifyItems(items); }
            @Override public String saveAndConfirm() throws Exception { verifyCurrent(); return lease.saveAndConfirm(); }
            @Override public void close() throws Exception { lease.close(); }
        };
    }
}
