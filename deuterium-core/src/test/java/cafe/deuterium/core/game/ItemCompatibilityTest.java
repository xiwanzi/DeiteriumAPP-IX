package cafe.deuterium.core.game;

import cafe.deuterium.core.util.CoreFailure;
import org.bukkit.Material;
import org.bukkit.entity.Player;
import org.bukkit.inventory.ItemStack;
import org.junit.jupiter.api.Test;
import java.io.ByteArrayOutputStream;
import java.io.DataOutputStream;
import java.util.ArrayList;
import java.util.HashSet;
import java.util.List;
import java.util.Set;
import java.util.concurrent.atomic.AtomicInteger;
import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

class ItemCompatibilityTest {
    private ItemStack item(String id) throws Exception {
        ByteArrayOutputStream bytes = new ByteArrayOutputStream();
        try (DataOutputStream out = new DataOutputStream(bytes)) {
            out.writeByte(10); out.writeUTF("");
            out.writeByte(8); out.writeUTF("id"); out.writeUTF(id); out.writeByte(0);
        }
        ItemStack stack = mock(ItemStack.class);
        when(stack.getType()).thenReturn(mock(Material.class));
        when(stack.serializeAsBytes()).thenReturn(bytes.toByteArray());
        return stack;
    }

    @Test void channelLessFurnitureAndNewNamespacesUseOneRegistryCheckPerBatch() throws Exception {
        List<ItemStack> items = new ArrayList<>();
        for (int i = 0; i < 128; i++) items.add(item(i == 0 ? "blahaj:blue_shark" : "future_furniture:chair"));
        Player player = mock(Player.class);
        try (var check = mockStatic(NeoForgeRegistrySync.class)) {
            ItemCompatibility.verify(player, items);
            check.verify(() -> NeoForgeRegistrySync.verify(player), times(1));
            check.verifyNoMoreInteractions();
        }
        NeoForgeRegistrySync.requireReady(true, true, "PLAY", "NEOFORGE", NeoForgeRegistrySync.CHANNELS);
    }

    @Test void vanillaOnlyDeliveryDoesNotRequireNeoForge() throws Exception {
        try (var check = mockStatic(NeoForgeRegistrySync.class)) {
            ItemCompatibility.verify(mock(Player.class), List.of(item("minecraft:stone")));
            check.verifyNoInteractions();
        }
    }

    @Test void everyRegistryTaskChannelAndFinishedConnectedSessionAreRequired() {
        Set<String> complete = NeoForgeRegistrySync.CHANNELS;
        for (String missing : complete) {
            Set<String> partial = new HashSet<>(complete); partial.remove(missing);
            assertThrows(CoreFailure.class, () -> NeoForgeRegistrySync.requireReady(true, true, "PLAY", "NEOFORGE", partial));
        }
        assertThrows(CoreFailure.class, () -> NeoForgeRegistrySync.requireReady(true, true, "PLAY", "OTHER", complete));
        assertThrows(CoreFailure.class, () -> NeoForgeRegistrySync.requireReady(true, true, "CONFIGURATION", "NEOFORGE", complete));
        assertThrows(CoreFailure.class, () -> NeoForgeRegistrySync.requireReady(false, true, "PLAY", "NEOFORGE", complete));
        assertThrows(CoreFailure.class, () -> NeoForgeRegistrySync.requireReady(true, false, "PLAY", "NEOFORGE", complete));
        assertThrows(CoreFailure.class, () -> NeoForgeRegistrySync.requireReady(true, true, "PLAY", "NEOFORGE", Set.of("blahaj:anything")));
    }

    @Test void reconnectIsRecheckedRatherThanReusingAPlayersPreviousResult() throws Exception {
        Player player = mock(Player.class);
        List<ItemStack> items = List.of(item("blahaj:blue_shark"));
        AtomicInteger checks = new AtomicInteger();
        try (var check = mockStatic(NeoForgeRegistrySync.class)) {
            check.when(() -> NeoForgeRegistrySync.verify(player)).thenAnswer(call -> {
                if (checks.getAndIncrement() > 0) throw new CoreFailure("CLIENT_REGISTRY_SYNC_REQUIRED", "new connection");
                return null;
            });
            ItemCompatibility.verify(player, items);
            assertThrows(CoreFailure.class, () -> ItemCompatibility.verify(player, items));
            assertEquals(2, checks.get());
        }
    }

    @Test void malformedPayloadIsRejectedBeforeCompatibilityCheck() throws Exception {
        ItemStack bad = item("minecraft:stone");
        when(bad.serializeAsBytes()).thenReturn(new byte[]{1, 2, 3});
        try (var check = mockStatic(NeoForgeRegistrySync.class)) {
            assertThrows(CoreFailure.class, () -> ItemCompatibility.verify(mock(Player.class), List.of(bad)));
            check.verifyNoInteractions();
        }
    }
}
