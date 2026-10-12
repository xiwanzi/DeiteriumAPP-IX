package cafe.deuterium.core.game;

import net.kyori.adventure.text.Component;
import net.kyori.adventure.text.format.TextColor;
import net.kyori.adventure.text.serializer.legacy.LegacyComponentSerializer;
import org.bukkit.OfflinePlayer;
import org.junit.jupiter.api.Test;
import java.util.UUID;
import java.util.concurrent.ConcurrentHashMap;
import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

class TitlePlaceholdersTest {
    @Test void followsEquippedTitleAndFallbackWithoutCachingOrChangingNameColor(){
        var id=UUID.randomUUID();var player=mock(OfflinePlayer.class);when(player.getUniqueId()).thenReturn(id);
        var titles=new ConcurrentHashMap<UUID,Component>();
        Component fallback=Component.text("溯行者",TextColor.color(0xA9B3B8));
        var expansion=new TitlePlaceholders("test",uuid->titles.getOrDefault(uuid,fallback));
        var codec=LegacyComponentSerializer.legacySection();
        assertEquals("[溯行者]",net.kyori.adventure.text.serializer.plain.PlainTextComponentSerializer.plainText().serialize(codec.deserialize(expansion.onRequest(player,"title_prefix"))));
        titles.put(id,Component.text("旅",TextColor.color(0x123456)).append(Component.text("者",TextColor.color(0xABCDEF))));
        String rendered=expansion.onRequest(player,"title_prefix");
        assertTrue(rendered.contains("§x§1§2§3§4§5§6旅"));
        assertTrue(rendered.contains("§x§a§b§c§d§e§f者"));
        assertTrue(rendered.endsWith("]§r"));
        assertFalse(expansion.onRequest(player,"title").contains("["));
        titles.remove(id);
        assertTrue(expansion.onRequest(player,"title").contains("溯行者"));
        assertTrue(expansion.persist());
        assertNull(expansion.onRequest(player,"missing"));assertEquals("",expansion.onRequest(null,"title"));
    }
}
