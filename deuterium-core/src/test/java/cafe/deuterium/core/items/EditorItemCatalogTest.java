package cafe.deuterium.core.items;

import cafe.deuterium.core.StoreFixture;
import cafe.deuterium.core.compat.EditorItemStack;
import cafe.deuterium.core.runtime.GameThread;
import cafe.deuterium.core.util.CoreFailure;
import cafe.deuterium.core.util.Json;
import org.bukkit.Bukkit;
import org.bukkit.inventory.ItemStack;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;

import java.nio.file.Path;
import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

class EditorItemCatalogTest {
    @TempDir Path folder;

    @Test void editorRegistrationReusesOneVersionAndRecoversAfterArchive() throws Exception {
        String itemId = "minecraft:stone";
        String name = "石头";
        String snbt = "{id:\"minecraft:stone\",count:64}";
        String ref = EditorItemCatalog.reference("deuterium", itemId, snbt, name);
        try (var fixture = new StoreFixture(folder); var game = new GameThread(null);
             var bukkit = mockStatic(Bukkit.class); var decoder = mockStatic(EditorItemStack.class)) {
            var codec = mock(ItemCodec.class);
            var stack = mock(ItemStack.class);
            bukkit.when(Bukkit::isPrimaryThread).thenReturn(true);
            decoder.when(() -> EditorItemStack.read(itemId, snbt, fixture.config.maxItemBytes())).thenReturn(stack);
            when(codec.capture(ref, stack, name, fixture.config)).thenReturn(StoreFixture.item(ref, new byte[]{1, 2}));
            var library = new EditorItemCatalog(() -> fixture.config, game, codec, fixture.catalog);
            var selected = Json.tree(Map.of("itemId", itemId, "stackData", snbt, "name", name, "catalogVersion", 0));
            long deadline = System.currentTimeMillis() + 10000;
            var first = library.register(selected, deadline);
            assertEquals(first, library.register(selected, deadline));
            decoder.verify(() -> EditorItemStack.read(itemId, snbt, fixture.config.maxItemBytes()), times(1));
            var removal = Json.tree(Map.of("itemRef", ref, "catalogVersion", 1));
            var archived = library.archive(removal);
            assertTrue(archived.getAsJsonObject("head").get("archived").getAsBoolean());
            assertEquals(archived, library.archive(removal));
            assertThrows(CoreFailure.class, () -> library.register(selected, deadline));
            selected.addProperty("catalogVersion", 2);
            var restored = library.register(selected, deadline);
            assertEquals(3, restored.getAsJsonObject("head").get("catalogVersion").getAsLong());
            assertEquals(List.of(1L), fixture.catalog.versions(ref, 1));
            assertThrows(CoreFailure.class, () -> library.archive(removal));
            assertFalse(fixture.catalog.findHead(ref).archived());
        }
    }

    @Test void editorCannotArchiveAnItemFromAnotherSource() throws Exception {
        try (var fixture = new StoreFixture(folder)) {
            fixture.catalog.create(StoreFixture.item("deuterium:ordinary", new byte[]{1}), "operator");
            var library = new EditorItemCatalog(() -> fixture.config, null, new ItemCodec(), fixture.catalog);
            assertThrows(CoreFailure.class, () -> library.archive(Json.tree(Map.of(
                    "itemRef", "deuterium:ordinary", "catalogVersion", 1))));
            assertFalse(fixture.catalog.findHead("deuterium:ordinary").archived());
        }
    }
}
