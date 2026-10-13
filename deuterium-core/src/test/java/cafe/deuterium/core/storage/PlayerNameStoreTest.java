package cafe.deuterium.core.storage;

import cafe.deuterium.core.StoreFixture;
import cafe.deuterium.core.runtime.Sessions.PlayerSession;
import cafe.deuterium.core.util.CoreFailure;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;
import java.nio.file.Path;
import java.util.UUID;
import static org.junit.jupiter.api.Assertions.*;

class PlayerNameStoreTest {
    @TempDir Path directory;
    @Test void canonicalNamesSurviveOldSessionsCursorReplayAndRestart() throws Exception {
        UUID uuid=UUID.randomUUID();
        try(var fixture=new StoreFixture(directory)){
            var players=new PlayerStore(fixture.db,"amiya");
            players.seen(new PlayerSession(uuid,"OldName",UUID.randomUUID().toString()));
            players.canonicalName(uuid,"NewName",20);players.canonicalName(uuid,"OldName",10);
            players.seen(new PlayerSession(uuid,"OldName",UUID.randomUUID().toString()));
            assertEquals(uuid,players.resolve("NewName").playerUuid());
            assertThrows(CoreFailure.class,()->players.resolve("OldName"));
            players.nameCursor(20);players.nameCursor(10);assertEquals(20,players.nameCursor());
            assertEquals(0,new PlayerStore(fixture.db,"odyssey").nameCursor());
        }
        try(var fixture=new StoreFixture(directory)){
            var players=new PlayerStore(fixture.db,"amiya");assertEquals(20,players.nameCursor());assertEquals("NewName",players.canonicalName(uuid));
        }
    }
    @Test void verifiedOfflineIdentityDoesNotRequireASecondGameLogin() throws Exception {
        try(var fixture=new StoreFixture(directory)){
            var players=new PlayerStore(fixture.db,"amiya");UUID uuid=UUID.randomUUID();
            players.canonicalName(uuid,"OfflineNew",1);
            var result=players.resolve("OfflineNew");assertEquals(uuid,result.playerUuid());assertFalse(result.online());
        }
    }
}
