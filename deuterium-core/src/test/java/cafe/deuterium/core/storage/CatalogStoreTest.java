package cafe.deuterium.core.storage;

import cafe.deuterium.core.StoreFixture;
import cafe.deuterium.core.util.CoreFailure;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;
import java.nio.file.Path;
import java.util.*;
import java.util.concurrent.*;
import static org.junit.jupiter.api.Assertions.*;

class CatalogStoreTest {
    @TempDir Path folder;
    @Test void completeSnapshotsSurviveRestartAndArchiveKeepsHistory() throws Exception {
        try (var f = new StoreFixture(folder)) {
            byte[] original = {1,2,3,4};
            var v1 = f.catalog.save(StoreFixture.item("deuterium:sword", original), "test-actor");
            original[0] = 9; byte[] returned = v1.payload(); returned[1] = 8;
            assertArrayEquals(new byte[]{1,2,3,4}, f.catalog.find("deuterium:sword", 1).payload());
            f.catalog.save(StoreFixture.item("deuterium:sword", new byte[]{5,6}), "test-actor");
            assertEquals(2, f.catalog.find("deuterium:sword", 0).revision());
            f.catalog.archive("deuterium:sword", true, "test-actor");
            assertThrows(CoreFailure.class, () -> f.catalog.find("deuterium:sword", 0));
            assertThrows(CoreFailure.class, () -> f.catalog.save(StoreFixture.item("deuterium:sword", new byte[]{7}), "test-actor"));
            assertArrayEquals(new byte[]{1,2,3,4}, f.catalog.find("deuterium:sword", 1).payload());
            assertTrue(f.catalog.search("sword", 1, false).isEmpty());
            assertTrue(f.catalog.search("sword", 1, true).getFirst().archived());
        }
        try (var f = new StoreFixture(folder)) {
            assertEquals(List.of(2L,1L), f.catalog.versions("deuterium:sword", 1));
            f.catalog.archive("deuterium:sword", false, "test-actor");
            assertEquals(2, f.catalog.find("deuterium:sword", 0).revision());
            assertEquals(6, f.outbox.pendingCount());
        }
    }
    @Test void concurrentSavesAllocateUniqueMonotonicVersions() throws Exception {
        try (var f = new StoreFixture(folder); var pool = Executors.newFixedThreadPool(8)) {
            List<Future<Long>> jobs = new ArrayList<>();
            for (int i = 0; i < 24; i++) { final byte content = (byte) i; jobs.add(pool.submit(() -> f.catalog.save(StoreFixture.item("deuterium:concurrent", new byte[]{content}), "test").revision())); }
            Set<Long> versions = new HashSet<>(); for (Future<Long> job : jobs) versions.add(job.get(10, TimeUnit.SECONDS));
            assertEquals(24, versions.size()); assertEquals(24, f.catalog.find("deuterium:concurrent", 0).revision());
        }
    }
    @Test void corruptionAndSqlWildcardsDoNotEscapeCatalogBoundary() throws Exception {
        try (var f = new StoreFixture(folder)) {
            f.catalog.save(StoreFixture.item("deuterium:sample", new byte[]{1,2}), "test");
            assertTrue(f.catalog.search("%", 1, false).isEmpty());
            assertThrows(CoreFailure.class, () -> f.catalog.search("", 0, false));
            f.db.read(c -> Sql.update(c, "UPDATE dc_item_versions SET payload=? WHERE item_ref=?", new byte[]{9}, "deuterium:sample"));
            assertEquals("ITEM_CORRUPT", assertThrows(CoreFailure.class, () -> f.catalog.find("deuterium:sample", 1)).code());
        }
    }
    @Test void anOutboxFailureRollsBackTheItemVersion() throws Exception {
        try (var f = new StoreFixture(folder)) {
            OutboxStore tiny = new OutboxStore(f.db, "amiya", 1); CatalogStore catalog = new CatalogStore(f.db, tiny);
            assertThrows(CoreFailure.class, () -> catalog.save(StoreFixture.item("deuterium:rollback", new byte[]{1}), "test"));
            assertTrue(catalog.search("", 1, true).isEmpty()); assertEquals(0, tiny.pendingCount());
        }
    }
}
