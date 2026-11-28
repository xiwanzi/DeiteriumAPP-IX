package cafe.deuterium.core.supply;

import cafe.deuterium.core.runtime.Sessions;
import com.google.gson.JsonObject;
import java.util.UUID;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class SupplyChannelTest {
    @Test void revealRequestsUseTheConnectionsIdentity() {
        var session = new Sessions.PlayerSession(UUID.randomUUID(), "Actual", "current-epoch");
        var packet = new JsonObject();
        packet.addProperty("requestId", "req");
        packet.addProperty("action", "draw");
        packet.addProperty("awaitReveal", true);
        assertTrue(SupplyChannel.bindRequest(packet, session).get("awaitReveal").getAsBoolean());

        packet.addProperty("action", "reveal-complete");
        packet.addProperty("playerUuid", UUID.randomUUID().toString());
        packet.addProperty("sessionEpoch", "stale");
        packet.addProperty("orderId", "sd_original");
        var bound = SupplyChannel.bindRequest(packet, session);
        assertFalse(bound.has("awaitReveal"));
        assertEquals("sd_original", bound.get("orderId").getAsString());
        assertEquals(session.playerUuid().toString(), bound.get("playerUuid").getAsString());
        assertEquals(session.sessionEpoch(), bound.get("sessionEpoch").getAsString());
    }

    @Test void ticketCatalogCapabilityIsForwardedOnlyForState() {
        var session = new Sessions.PlayerSession(UUID.randomUUID(), "Actual", "epoch");
        var packet = new JsonObject();
        packet.addProperty("requestId", "req");
        packet.addProperty("action", "state");
        packet.addProperty("ticketCatalog", true);
        packet.addProperty("presentationVersion", 2);
        packet.addProperty("knownTicketsDigest", "0123456789abcdef");
        var known = new JsonObject();
        known.addProperty("sp_sample_up_191", 3);
        known.addProperty("invalid key!", 2);
        packet.add("knownPools", known);
        var bound = SupplyChannel.bindRequest(packet, session);
        assertTrue(bound.get("ticketCatalog").getAsBoolean());
        assertEquals(2, bound.get("presentationVersion").getAsInt());
        assertEquals("0123456789abcdef", bound.get("knownTicketsDigest").getAsString());
        assertEquals(3, bound.getAsJsonObject("knownPools").get("sp_sample_up_191").getAsLong());
        assertFalse(bound.getAsJsonObject("knownPools").has("invalid key!"));
        packet.addProperty("action", "draw");
        var drawBound = SupplyChannel.bindRequest(packet, session);
        assertFalse(drawBound.has("ticketCatalog"));
        assertFalse(drawBound.has("knownPools"));
        assertFalse(drawBound.has("presentationVersion"));
    }

    @Test void clientCannotChoosePlayerSessionOrClaimOperatorPrivileges() {
        var actual = new Sessions.PlayerSession(UUID.randomUUID(), "Actual", "current-epoch");
        var packet = new JsonObject(); packet.addProperty("requestId", "req"); packet.addProperty("action", "draw");
        packet.addProperty("playerUuid", UUID.randomUUID().toString()); packet.addProperty("sessionEpoch", "forged");
        packet.addProperty("operator", true); packet.addProperty("console", true);
        var draw = new JsonObject(); draw.addProperty("quantity", 10); packet.add("draw", draw);
        var bound = SupplyChannel.bindRequest(packet, actual);
        assertEquals(actual.playerUuid().toString(), bound.get("playerUuid").getAsString());
        assertEquals(actual.sessionEpoch(), bound.get("sessionEpoch").getAsString());
        assertFalse(bound.has("operator")); assertFalse(bound.has("console")); assertFalse(bound.has("action"));
        draw.addProperty("quantity", 1);
        assertEquals(10, bound.getAsJsonObject("draw").get("quantity").getAsInt());
    }
}
