package cafe.deuterium.core.api;

import java.util.List;

/** Immutable complete unit-item snapshot. Quantity belongs to the grant/delivery request. */
public record ItemVersion(String itemRef, long revision, String codec, byte[] payload,
                          String payloadSha256, String itemId, String displayName, String description,
                          long maxQuantity, List<String> compatibleServerIds, List<String> requiredMods,
                          String inventoryDomain, String compatibilityProfile, long createdAt) {
    public ItemVersion {
        payload = payload.clone();
        compatibleServerIds = List.copyOf(compatibleServerIds);
        requiredMods = List.copyOf(requiredMods);
    }
    @Override public byte[] payload() { return payload.clone(); }
}
