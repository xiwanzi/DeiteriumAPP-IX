package cafe.deuterium.core.items;

import cafe.deuterium.core.StoreFixture;
import cafe.deuterium.core.api.PlayerDataService;
import cafe.deuterium.core.game.CommandDispatcher;
import cafe.deuterium.core.runtime.Sessions;
import cafe.deuterium.core.storage.CatalogStore;
import cafe.deuterium.core.storage.RpcJournal;
import cafe.deuterium.core.util.Checks;
import cafe.deuterium.core.util.CoreFailure;
import cafe.deuterium.core.util.Json;
import java.nio.charset.StandardCharsets;
import java.nio.file.Path;
import java.time.Instant;
import java.util.List;
import java.util.UUID;
import org.bukkit.Bukkit;
import org.bukkit.Material;
import org.bukkit.entity.Player;
import org.bukkit.inventory.ItemStack;
import org.bukkit.inventory.PlayerInventory;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;
import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

class InventoryCatalogCaptureTest {
    @TempDir Path folder;

    @Test void snapshotShowsSlotsWithoutChangingOrSavingPlayerInventory() throws Exception {
        try (var bukkit = mockStatic(Bukkit.class)) {
            var f = fixture(bukkit);
            var slots = f.capture.snapshot("read-one", f.uuid, f.epoch);
            assertEquals(41, slots.size());
            var held = slots.getFirst();
            assertEquals("storage:0", held.slot()); assertEquals(64, held.amount()); assertTrue(held.savable());
            assertEquals(Checks.sha(new byte[]{1, 2, 3}), held.fingerprint());
            assertTrue(slots.stream().skip(1).noneMatch(InventoryCatalogCapture.Slot::savable));
            verify(f.inventory, never()).setItem(anyInt(), any());
            verify(f.inventory, never()).setStorageContents(any());
            verify(f.lease, never()).saveAndConfirm();
        }
    }

    @Test void movedOrChangedItemIsRejectedBeforeCapture() throws Exception {
        try (var bukkit = mockStatic(Bukkit.class)) {
            var f = fixture(bukkit);
            var preview = f.capture.snapshot("read-one", f.uuid, f.epoch).getFirst();
            clearInvocations(f.codec);
            when(f.item.serializeAsBytes()).thenReturn(new byte[]{9});
            var error = assertThrows(CoreFailure.class, () -> f.capture.capture("save-one", f.uuid, f.epoch, preview.slot(), preview.fingerprint(), "deuterium:save"));
            assertEquals("INVENTORY_CHANGED", error.code());
            verifyNoInteractions(f.codec);
        }
    }

    @Test void changedPlayerSessionCannotCaptureThePreviousSnapshot() throws Exception {
        try (var bukkit = mockStatic(Bukkit.class)) {
            var f = fixture(bukkit);
            f.sessions.quit(f.uuid);
            assertEquals("PLAYER_SESSION_CHANGED", assertThrows(CoreFailure.class,
                    () -> f.capture.capture("save-one", f.uuid, f.epoch, "storage:0", Checks.sha(new byte[]{1, 2, 3}), "deuterium:save")).code());
            verifyNoInteractions(f.codec);
        }
    }

    @Test void unsupportedContainerRemainsVisibleWithAnExplicitReason() throws Exception {
        try (var bukkit = mockStatic(Bukkit.class)) {
            var f = fixture(bukkit);
            when(f.codec.capture(anyString(), same(f.item), anyString(), any())).thenThrow(new CoreFailure("ITEM_EXTERNAL_STORAGE_UNSUPPORTED", "容器引用外部存储，不能复制"));
            var slot = f.capture.snapshot("read-one", f.uuid, f.epoch).getFirst();
            assertEquals(64, slot.amount()); assertFalse(slot.savable()); assertEquals("", slot.fingerprint());
            assertTrue(slot.reason().contains("外部存储"));
        }
    }

    @Test void aCommittedCaptureRecoversAfterJournalLossWithoutThePlayer() throws Exception {
        String operation = "coreop_capture_recovery";
        String ref = "deuterium:supply_" + Checks.sha(operation.getBytes(StandardCharsets.UTF_8)).substring(0, 32);
        try (var f = new StoreFixture(folder)) { f.catalog.create(StoreFixture.item(ref, new byte[]{1, 2}), "web-admin"); }
        try (var f = new StoreFixture(folder)) {
            var capture = new InventoryCatalogCapture(() -> f.config, new Sessions(), () -> { throw new AssertionError("Must not open inventory"); }, null, new ItemCodec(), f.catalog);
            var journal = new RpcJournal(f.db, "amiya");
            var dispatcher = new CommandDispatcher(() -> f.config, () -> null, null, null, null, journal, null, capture);
            var request = Json.object("{\"operationId\":\"query_capture\",\"command\":\"operation.query\",\"payload\":{\"operationId\":\"" + operation + "\"}}", 4096);
            request.addProperty("expiresAt", Instant.now().plusSeconds(30).toString());
            var reply = dispatcher.execute(request);
            assertEquals("COMPLETED", reply.get("status").getAsString());
            assertEquals("COMPLETED", reply.getAsJsonObject("data").get("state").getAsString());
            assertTrue(reply.getAsJsonObject("data").get("result").getAsString().contains(ref));
            assertEquals(List.of(1L), f.catalog.versions(ref, 1));
        }
    }

    private record Fixture(UUID uuid, String epoch, Sessions sessions, PlayerInventory inventory, ItemStack item,
                           PlayerDataService.Lease lease, ItemCodec codec, InventoryCatalogCapture capture) { }

    private Fixture fixture(org.mockito.MockedStatic<Bukkit> bukkit) throws Exception {
        UUID uuid = UUID.randomUUID();
        var player = mock(Player.class); var inventory = mock(PlayerInventory.class); var item = mock(ItemStack.class);
        when(player.getUniqueId()).thenReturn(uuid); when(player.getName()).thenReturn("SupplyPlayer");
        when(player.isOnline()).thenReturn(true); when(player.getInventory()).thenReturn(inventory);
        var material = mock(Material.class);
        when(material.getKey()).thenReturn(org.bukkit.NamespacedKey.minecraft("stone"));
        when(item.getType()).thenReturn(material); when(item.getAmount()).thenReturn(64);
        when(item.serializeAsBytes()).thenReturn(new byte[]{1, 2, 3}); when(inventory.getItem(0)).thenReturn(item);
        bukkit.when(Bukkit::isPrimaryThread).thenReturn(true); bukkit.when(() -> Bukkit.getPlayer(uuid)).thenReturn(player);
        var provider = mock(PlayerDataService.class); var lease = mock(PlayerDataService.Lease.class);
        when(provider.acquire(same(player), eq("survival"), any())).thenReturn(lease);
        var sessions = new Sessions(); var session = sessions.join(player); var codec = mock(ItemCodec.class);
        when(codec.capture(anyString(), same(item), anyString(), any())).thenReturn(StoreFixture.item("deuterium:sample", new byte[]{1, 2, 3}));
        var capture = new InventoryCatalogCapture(StoreFixture::config, sessions, () -> provider, null, codec, mock(CatalogStore.class));
        return new Fixture(uuid, session.sessionEpoch(), sessions, inventory, item, lease, codec, capture);
    }
}
