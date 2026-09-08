package cafe.deuterium.core.storage;

import cafe.deuterium.core.StoreFixture;
import cafe.deuterium.core.util.CoreFailure;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;
import java.nio.file.Path;
import static org.junit.jupiter.api.Assertions.*;

class JournalTest {
    @TempDir Path folder;
    @Test void eventReplayIsCanonicalAndAckPersists() throws Exception {
        try (var f = new StoreFixture(folder)) {
            f.outbox.enqueue("same", "event", "{\"b\":2,\"a\":1}");
            f.outbox.enqueue("same", "event", "{\"a\":1,\"b\":2}");
            assertEquals(1, f.outbox.pendingCount());
            assertThrows(CoreFailure.class, () -> f.outbox.enqueue("same", "event", "{\"a\":9}"));
            f.outbox.acknowledged("same"); assertEquals(0, f.outbox.pendingCount());
        }
        try (var f = new StoreFixture(folder)) { f.outbox.enqueue("same", "event", "{\"b\":2,\"a\":1}"); assertEquals(0, f.outbox.pendingCount()); }
    }
    @Test void sameChatReachesEachNodeExactlyOnceInNormalOperation() throws Exception {
        try (var f = new StoreFixture(folder)) {
            InboxStore amiya = new InboxStore(f.db, "amiya"), odyssey = new InboxStore(f.db, "odyssey");
            assertTrue(amiya.reserve("message", "hash", 10000)); assertFalse(amiya.reserve("message", "hash", 10000));
            assertTrue(odyssey.reserve("message", "hash", 10000));
            assertThrows(CoreFailure.class, () -> odyssey.reserve("message", "different", 10000));
        }
    }
    @Test void unknownEconomicActionsNeverAutomaticallyReexecute() throws Exception {
        try (var f = new StoreFixture(folder)) {
            RpcJournal j = new RpcJournal(f.db, "amiya");
            assertTrue(j.begin("op", "wallet.transfer", "hash", false).acquired());
            assertFalse(j.begin("op", "wallet.transfer", "hash", false).acquired());
            j.recover(); assertEquals("UNKNOWN", j.find("op").state());
            assertFalse(j.begin("op", "wallet.transfer", "hash", false).acquired());
            assertThrows(CoreFailure.class, () -> j.begin("op", "wallet.transfer", "changed", false));
            assertTrue(j.begin("mail", "mailbox.create", "mailhash", true).acquired());
            j.recover(); assertTrue(j.begin("mail", "mailbox.create", "mailhash", true).acquired());
        }
    }
    @Test void duplicateLocalRuntimeCannotClaimSameSqliteFile() throws Exception {
        try (var f = new StoreFixture(folder); var lease = new NodeLease(f.db, folder, "cluster", "amiya", "core.db")) {
            assertTrue(lease.valid());
            assertThrows(Exception.class, () -> new NodeLease(f.db, folder, "cluster", "other", "core.db"));
        }
    }
}
