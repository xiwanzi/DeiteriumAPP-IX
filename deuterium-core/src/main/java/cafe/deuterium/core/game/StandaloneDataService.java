package cafe.deuterium.core.game;

import cafe.deuterium.core.api.PlayerDataService;
import cafe.deuterium.core.runtime.Sessions;
import cafe.deuterium.core.compat.LocalSaveProof;
import org.bukkit.Bukkit;
import org.bukkit.entity.Player;
import org.bukkit.inventory.ItemStack;
import java.util.*;
import java.util.concurrent.ConcurrentHashMap;

/** Explicit single-server mode only. It cannot certify a shared YouerModSync inventory. */
public final class StandaloneDataService implements PlayerDataService {
    private final Sessions sessions;
    private final String domain;
    private final Set<UUID> held = ConcurrentHashMap.newKeySet();
    private volatile boolean enabled = true;
    public StandaloneDataService(Sessions sessions, String domain) { this.sessions = sessions; this.domain = domain; }
    @Override public String providerName() { return "standalone-local-save"; }
    @Override public boolean available() {
        if (!enabled || !LocalSaveProof.supported()) return false;
        return Bukkit.getPluginManager().getPlugin("YouerModSync") == null && Bukkit.getPluginManager().getPlugin("HuskSync") == null;
    }
    @Override public Lease acquire(Player player, String requestedDomain, UUID operation) {
        if (!Bukkit.isPrimaryThread() || !available() || !domain.equals(requestedDomain)||player.isDead()||!player.isValid()) throw new IllegalStateException("Standalone barrier unavailable or player not alive");
        Sessions.PlayerSession session = sessions.get(player.getUniqueId());
        if (session == null || !held.add(player.getUniqueId())) throw new IllegalStateException("Player unavailable or inventory already held");
        return new Lease() {
            private boolean open = true;
            @Override public String sessionEpoch() { return session.sessionEpoch(); }
            @Override public void verifyCurrent() {
                if (!open || !available() || player.isDead()||!player.isValid()||sessions.require(player.getUniqueId(), session.sessionEpoch()) != player) throw new IllegalStateException("Inventory lease lost");
            }
            @Override public void verifyItems(List<ItemStack> items) throws Exception { verifyCurrent(); ItemCompatibility.verify(player, items); }
            @Override public String saveAndConfirm() throws Exception {
                verifyCurrent(); LocalSaveProof.saveAndVerify(player); verifyCurrent();
                return "local:" + operation + ":" + System.currentTimeMillis();
            }
            @Override public void close() { if (open) { open = false; held.remove(player.getUniqueId()); } }
        };
    }
    public void disable() { enabled = false; held.clear(); }
}
