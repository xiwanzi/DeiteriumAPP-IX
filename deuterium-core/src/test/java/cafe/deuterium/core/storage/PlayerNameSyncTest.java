package cafe.deuterium.core.storage;

import cafe.deuterium.core.StoreFixture;
import cafe.deuterium.core.game.PlayerNameSync;
import com.google.gson.*;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;
import java.nio.file.Path;
import java.util.UUID;
import java.util.concurrent.*;
import java.util.concurrent.atomic.*;
import java.util.logging.Logger;
import static org.junit.jupiter.api.Assertions.*;

class PlayerNameSyncTest {
    @TempDir Path directory;
    @Test void guildFailureDoesNotHoldCoreCursorOrBroadcastAndRetriesIndependently() throws Exception {
        try(var fixture=new StoreFixture(directory)) {
            var players=new PlayerStore(fixture.db,"amiya");var uuid=UUID.randomUUID();
            var broadcast=new CountDownLatch(1);var guildFailed=new CountDownLatch(1);var guildRecovered=new CountDownLatch(1);
            var requests=new AtomicInteger();var attempts=new AtomicInteger();
            try(var sync=new PlayerNameSync(players,after->{
                requests.incrementAndGet();var response=new JsonObject();var items=new JsonArray();
                if(after<1){var entry=new JsonObject();entry.addProperty("sequence",1);entry.addProperty("uuid",uuid.toString());entry.addProperty("gameId","NewName");items.add(entry);}
                response.add("items",items);response.addProperty("cursor",1);response.addProperty("hasMore",false);return response;
            },items->broadcast.countDown(),entry->{
                if(attempts.incrementAndGet()==1){guildFailed.countDown();throw new IllegalStateException("guild unavailable");}
                guildRecovered.countDown();
            },Logger.getAnonymousLogger())) {
                sync.refresh();
                assertTrue(broadcast.await(2,TimeUnit.SECONDS));assertTrue(guildFailed.await(2,TimeUnit.SECONDS));
                assertEquals("NewName",players.canonicalName(uuid));assertEquals(1,players.nameCursor());assertEquals(0,players.nameCursor(":teams"));
                assertTrue(guildRecovered.await(4,TimeUnit.SECONDS));
                long deadline=System.nanoTime()+TimeUnit.SECONDS.toNanos(1);
                while(players.nameCursor(":teams")<1&&System.nanoTime()<deadline)Thread.sleep(5);
                assertEquals(1,players.nameCursor(":teams"));assertEquals(1,requests.get(),"guild retry must not re-fetch Core feed");
            }
        }
    }
}
