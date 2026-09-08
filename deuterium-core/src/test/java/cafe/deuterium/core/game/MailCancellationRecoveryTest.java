package cafe.deuterium.core.game;

import cafe.deuterium.core.StoreFixture;
import cafe.deuterium.core.mail.CoreMailbox;
import cafe.deuterium.core.storage.RpcJournal;
import cafe.deuterium.core.util.Json;
import com.google.gson.JsonObject;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;
import java.nio.file.Path;
import java.time.Instant;
import java.util.concurrent.atomic.AtomicInteger;
import static org.junit.jupiter.api.Assertions.*;

class MailCancellationRecoveryTest {
    @TempDir Path folder;
    @Test void queryRecoversCommittedCancellationWithoutExecutingItAgain() throws Exception {
        try (var fixture = new StoreFixture(folder)) {
            var journal = new RpcJournal(fixture.db, "amiya");
            journal.begin("cancel_one", "mailbox.revoke", "original-fingerprint", true);
            journal.recover();
            AtomicInteger reads = new AtomicInteger();
            CoreMailbox mail = new CoreMailbox() {
                @Override public JsonObject capabilities() { return new JsonObject(); }
                @Override public JsonObject execute(String id, String type, JsonObject payload) { fail("query invoked a mailbox mutation"); return null; }
                @Override public JsonObject queryCancellation(String id) {
                    assertEquals("cancel_one", id); reads.incrementAndGet();
                    return Json.object("{\"code\":\"OK\",\"value\":{\"operationId\":\"cancel_one\",\"proofKind\":\"CANCELLED_BEFORE_CREATE\"}}", 1024);
                }
                @Override public void pumpEvents() { }
                @Override public void close() { }
            };
            var dispatcher = new CommandDispatcher(() -> fixture.config, () -> mail, null, null, null, journal, null);
            JsonObject request = new JsonObject();
            request.addProperty("operationId", "query_one");
            request.addProperty("command", "operation.query");
            request.addProperty("expiresAt", Instant.now().plusSeconds(30).toString());
            request.add("payload", Json.object("{\"operationId\":\"cancel_one\"}", 1024));
            JsonObject reply = dispatcher.execute(request);
            assertEquals("COMPLETED", reply.get("status").getAsString());
            JsonObject persisted = Json.object(reply.getAsJsonObject("data").get("result").getAsString(), 2048);
            assertEquals("cancel_one", persisted.get("operationId").getAsString());
            assertEquals("COMPLETED", persisted.get("status").getAsString());
            assertEquals(1, reads.get());
            assertEquals("UNKNOWN", journal.find("cancel_one").state());
        }
    }
}
