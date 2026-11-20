package cafe.deuterium.core.items;

import cafe.deuterium.core.api.ItemVersion;
import cafe.deuterium.core.api.PlayerDataService;
import cafe.deuterium.core.config.CoreConfig;
import cafe.deuterium.core.runtime.GameThread;
import cafe.deuterium.core.runtime.Sessions;
import cafe.deuterium.core.storage.CatalogStore;
import cafe.deuterium.core.util.Checks;
import cafe.deuterium.core.util.CoreFailure;
import cafe.deuterium.core.util.Json;
import com.google.gson.JsonObject;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.UUID;
import java.util.concurrent.TimeUnit;
import java.util.function.Supplier;
import org.bukkit.ChatColor;
import org.bukkit.inventory.ItemStack;
import org.bukkit.inventory.PlayerInventory;

/** Captures an unchanged live inventory slot into the immutable item library without taking it. */
public final class InventoryCatalogCapture {
    public record Slot(String slot, String name, String itemId, int amount, String fingerprint, boolean savable, String reason) { }
    private final Supplier<CoreConfig> config;
    private final Sessions sessions;
    private final Supplier<PlayerDataService> inventory;
    private final GameThread game;
    private final ItemCodec codec;
    private final CatalogStore catalog;

    public InventoryCatalogCapture(Supplier<CoreConfig> config, Sessions sessions, Supplier<PlayerDataService> inventory,
                                   GameThread game, ItemCodec codec, CatalogStore catalog) {
        this.config = config;
        this.sessions = sessions;
        this.inventory = inventory;
        this.game = game;
        this.codec = codec;
        this.catalog = catalog;
    }

    public JsonObject execute(String operation, String type, JsonObject payload, long deadline) throws Exception {
        if (type.equals("items.inventory")) {
            Json.fields(payload, "playerUuid", "sessionEpoch");
            UUID player = uuid(Json.string(payload, "playerUuid"));
            String epoch = Json.string(payload, "sessionEpoch");
            List<Slot> slots = game.call(deadline, () -> snapshot(operation, player, epoch))
                    .get(Math.max(1, deadline - System.currentTimeMillis()), TimeUnit.MILLISECONDS);
            return Json.tree(Map.of("playerUuid", player.toString(), "sessionEpoch", epoch, "slots", slots));
        }
        if (!type.equals("items.capture")) throw new CoreFailure("COMMAND_NOT_ALLOWED", "不支持的物品采集操作。");
        Json.fields(payload, "playerUuid", "sessionEpoch", "slot", "fingerprint", "actor");
        UUID player = uuid(Json.string(payload, "playerUuid"));
        String epoch = Json.string(payload, "sessionEpoch"), slot = Json.string(payload, "slot");
        String fingerprint = Json.string(payload, "fingerprint"), actor = Json.string(payload, "actor");
        if (!fingerprint.matches("[a-f0-9]{64}") || actor.isBlank() || actor.length() > 80 || !validSlot(slot))
            throw CoreFailure.invalid("物品采集参数无效。");
        JsonObject committed = recover(operation);
        if (committed != null) return committed;
        String ref = reference(operation);
        CatalogStore.Captured captured = game.call(deadline, () -> capture(operation, player, epoch, slot, fingerprint, ref))
                .get(Math.max(1, deadline - System.currentTimeMillis()), TimeUnit.MILLISECONDS);
        return result(catalog.create(captured, actor));
    }

    List<Slot> snapshot(String operation, UUID uuid, String epoch) throws Exception {
        var player = sessions.require(uuid, epoch);
        var cfg = config.get();
        try (var lease = inventory.get().acquire(player, cfg.inventoryDomain(), operationUuid(operation))) {
            lease.verifyCurrent();
            List<Slot> slots = new ArrayList<>();
            for (int i = 0; i < 36; i++) slots.add(describe("storage:" + i, player.getInventory().getItem(i)));
            for (String slot : List.of("head", "chest", "legs", "feet", "offhand"))
                slots.add(describe(slot, readSlot(player.getInventory(), slot)));
            lease.verifyCurrent();
            return List.copyOf(slots);
        }
    }

    CatalogStore.Captured capture(String operation, UUID uuid, String epoch, String slot, String expected, String ref) throws Exception {
        var player = sessions.require(uuid, epoch);
        var cfg = config.get();
        try (var lease = inventory.get().acquire(player, cfg.inventoryDomain(), operationUuid(operation))) {
            lease.verifyCurrent();
            ItemStack current = readSlot(player.getInventory(), slot);
            if (empty(current) || !expected.equals(Checks.sha(current.serializeAsBytes())))
                throw new CoreFailure("INVENTORY_CHANGED", "该槽位的物品已变化，请刷新背包后重新选择。");
            CatalogStore.Captured captured = codec.capture(ref, current, "", cfg);
            lease.verifyCurrent();
            return captured;
        }
    }

    private Slot describe(String slot, ItemStack item) {
        if (empty(item)) return new Slot(slot, "", "", 0, "", false, "");
        String name = item.hasItemMeta() && item.getItemMeta().hasDisplayName()
                ? ChatColor.stripColor(item.getItemMeta().getDisplayName()) : item.getType().getKey().toString();
        try {
            var captured = codec.capture(config.get().namespace() + ":inventory_preview", item, "", config.get());
            return new Slot(slot, shorten(captured.name(), 48), shorten(captured.itemId(), 128), item.getAmount(),
                    Checks.sha(item.serializeAsBytes()), true, "");
        } catch (CoreFailure unsupported) {
            return new Slot(slot, shorten(name, 48), shorten(item.getType().getKey().toString(), 128), item.getAmount(),
                    "", false, shorten(unsupported.getMessage(), 48));
        }
    }

    /** Positive committed metadata only; absence is not proof of a completed operation. */
    public JsonObject recover(String operation) {
        try { return result(catalog.find(reference(operation), 1)); }
        catch (CoreFailure missing) {
            if (missing.code().equals("ITEM_NOT_FOUND")) return null;
            throw missing;
        }
    }

    private String reference(String operation) {
        return config.get().namespace() + ":supply_" + Checks.sha(operation.getBytes(StandardCharsets.UTF_8)).substring(0, 32);
    }
    private static JsonObject result(ItemVersion item) { return Json.tree(Map.of("item", ItemMetadata.of(item))); }
    private static UUID operationUuid(String operation) { return UUID.nameUUIDFromBytes(operation.getBytes(StandardCharsets.UTF_8)); }
    private static UUID uuid(String value) {
        UUID uuid = UUID.fromString(value);
        if (!uuid.toString().equals(value)) throw CoreFailure.invalid("无效玩家 UUID。");
        return uuid;
    }
    private static boolean empty(ItemStack item) { return item == null || item.getType().isAir() || item.getAmount() < 1; }
    private static String shorten(String value, int maximum) {
        if (value == null) return "";
        int count = value.codePointCount(0, value.length());
        return count <= maximum ? value : value.substring(0, value.offsetByCodePoints(0, maximum - 1)) + "…";
    }
    static boolean validSlot(String slot) {
        return slot.matches("storage:(?:[0-9]|[12][0-9]|3[0-5])") || List.of("head", "chest", "legs", "feet", "offhand").contains(slot);
    }
    private static ItemStack readSlot(PlayerInventory items, String slot) {
        if (!validSlot(slot)) throw CoreFailure.invalid("无效背包槽位。");
        if (slot.startsWith("storage:")) return items.getItem(Integer.parseInt(slot.substring(8)));
        return switch (slot) {
            case "head" -> items.getHelmet();
            case "chest" -> items.getChestplate();
            case "legs" -> items.getLeggings();
            case "feet" -> items.getBoots();
            default -> items.getItemInOffHand();
        };
    }
}
