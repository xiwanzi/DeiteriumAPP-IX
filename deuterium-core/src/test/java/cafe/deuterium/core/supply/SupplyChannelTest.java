package cafe.deuterium.core.supply;

import cafe.deuterium.core.runtime.Sessions;
import com.google.gson.JsonObject;
import java.util.UUID;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class SupplyChannelTest {
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
