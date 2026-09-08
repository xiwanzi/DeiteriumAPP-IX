package cafe.deuterium.core.mail;

import cafe.deuterium.core.util.Checks;
import cafe.deuterium.core.util.CoreFailure;
import cafe.deuterium.core.util.Json;
import com.google.gson.JsonObject;
import org.junit.jupiter.api.Test;
import java.nio.charset.StandardCharsets;
import java.util.List;
import static org.junit.jupiter.api.Assertions.*;

class MailboxCancellationTest {
    private static final String SNAPSHOT = "{\"schemaVersion\":1,\"orderId\":\"order_one\",\"recipientUuid\":\"9a8b96da-27d4-4aee-9b94-0d5ce9d51a55\",\"inventoryDomain\":\"survival\",\"allowedServerIds\":[\"amiya\",\"odyssey\"],\"attachments\":[{\"itemRef\":\"deuterium:archived\",\"revision\":1,\"quantity\":1,\"payloadSha256\":\"" + "a".repeat(64) + "\"}]}";
    private JsonObject payload() {
        JsonObject value = new JsonObject();
        value.addProperty("source", "deuterium-commerce");
        value.addProperty("deliveryId", "delivery_one");
        value.addProperty("orderId", "order_one");
        value.addProperty("snapshotJson", SNAPSHOT);
        value.addProperty("expectedSnapshotSha256", Checks.sha(SNAPSHOT.getBytes(StandardCharsets.UTF_8)));
        value.addProperty("reasonCode", "CUSTOMER_REFUND");
        return value;
    }
    @Test void cancellationPreservesOriginalBytesAndDoesNotResolveArchivedItems() {
        var cancel = MailboxAdapter.cancellationRequest("cancel_one", payload());
        assertEquals(SNAPSHOT, cancel.snapshotJson());
        assertEquals("cancel_one", cancel.operationId());
        assertEquals("order_one", cancel.orderId());
        assertEquals(List.of("amiya", "odyssey"), cancel.allowedServerIds());
        assertEquals("9a8b96da-27d4-4aee-9b94-0d5ce9d51a55", cancel.recipientUuid().toString());
    }
    @Test void cancellationRejectsChangedHashOrderOrUnexpectedFields() {
        JsonObject changedHash = payload(); changedHash.addProperty("snapshotJson", SNAPSHOT + " ");
        assertThrows(CoreFailure.class, () -> MailboxAdapter.cancellationRequest("cancel_one", changedHash));
        JsonObject changedOrder = payload(); changedOrder.addProperty("orderId", "other_order");
        assertThrows(CoreFailure.class, () -> MailboxAdapter.cancellationRequest("cancel_one", changedOrder));
        JsonObject extra = payload(); extra.addProperty("amount", "100");
        assertThrows(CoreFailure.class, () -> MailboxAdapter.cancellationRequest("cancel_one", extra));
    }
}
