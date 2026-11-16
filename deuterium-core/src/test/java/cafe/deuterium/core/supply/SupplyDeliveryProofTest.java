package cafe.deuterium.core.supply;

import cafe.deuterium.core.api.PlayerDataService;
import com.google.gson.JsonObject;
import java.util.UUID;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class SupplyDeliveryProofTest {
    @Test void recoveryRequiresEveryOriginalInventoryIdentity() {
        UUID player=UUID.randomUUID(),operation=UUID.randomUUID();var delivery=new JsonObject();
        delivery.addProperty("playerUuid",player.toString());delivery.addProperty("token",operation.toString());delivery.addProperty("nodeId","amiya");
        var proof=new PlayerDataService.SaveProof(player,"survival",operation,"inventory-epoch","amiya","saved",123);
        assertTrue(SupplyDeliveryPump.matchesProof(delivery,"survival","inventory-epoch",proof));
        assertFalse(SupplyDeliveryPump.matchesProof(delivery,"survival","core-epoch",proof));
        assertFalse(SupplyDeliveryPump.matchesProof(delivery,"another-domain","inventory-epoch",proof));
        assertFalse(SupplyDeliveryPump.matchesProof(delivery,"survival","inventory-epoch",null));
        delivery.addProperty("token",UUID.randomUUID().toString());
        assertFalse(SupplyDeliveryPump.matchesProof(delivery,"survival","inventory-epoch",proof));
    }
}
