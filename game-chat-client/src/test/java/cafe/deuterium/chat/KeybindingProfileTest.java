package cafe.deuterium.chat;

import static org.junit.jupiter.api.Assertions.*;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;

import java.net.URI;
import java.nio.file.Path;
import java.util.*;

class KeybindingProfileTest {
    @Test
    void offlineEditsPersistAndRebaseAfterInflightDelivery(@TempDir Path dir) throws Exception {
        var p = new KeybindingProfile();
        p.edit("chat", "key.keyboard.t", "key.keyboard.g");
        var sent = Map.copyOf(p.pending);
        p.edit("chat", "key.keyboard.g", "key.keyboard.p");
        p.delivered(sent);
        assertEquals(
                KeybindingProfile.Change.edit("key.keyboard.g", "key.keyboard.p"),
                p.pending.get("chat"));
        p.save(dir.resolve("account.json"));
        assertEquals(p.pending, KeybindingProfile.load(dir.resolve("account.json")).pending);
        p.edit("chat", "key.keyboard.p", "key.keyboard.g");
        assertTrue(p.pending.isEmpty());
    }

    @Test
    void onlyProjectHttpsOriginsAreTrusted() {
        assertTrue(KeybindingApi.trusted(URI.create("https://chat.deuteriumix.com/")));
        for (String invalid :
                List.of(
                        "https://example.org",
                        "https://chat.deuteriumix.com.example.org",
                        "https://chat.deuteriumix.com:444",
                        "http://chat.deuteriumix.com",
                        "https://user@chat.deuteriumix.com",
                        "https://chat.deuteriumix.com/a")) {
            assertFalse(KeybindingApi.trusted(URI.create(invalid)));
        }
    }
}
