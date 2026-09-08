package cafe.deuterium.core.items;

import cafe.deuterium.core.api.ItemVersion;
import java.util.List;

/** Wire-safe catalog metadata. Raw NBT remains inside the Core database. */
public record ItemMetadata(String itemRef, long revision, String payloadSha256, String displayName,
                           String description, long maxQuantity, List<String> compatibleServerIds,
                           List<String> requiredMods, String codec, String itemId,
                           String inventoryDomain, String compatibilityProfile) {
    public static ItemMetadata of(ItemVersion v) {
        return new ItemMetadata(v.itemRef(), v.revision(), v.payloadSha256(), v.displayName(), v.description(),
                v.maxQuantity(), v.compatibleServerIds(), v.requiredMods(), v.codec(), v.itemId(), v.inventoryDomain(), v.compatibilityProfile());
    }
    public ItemVersion withPayload(byte[] payload, long createdAt) {
        return new ItemVersion(itemRef, revision, codec, payload, payloadSha256, itemId, displayName, description,
                maxQuantity, compatibleServerIds, requiredMods, inventoryDomain, compatibilityProfile, createdAt);
    }
}
