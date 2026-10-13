package cafe.deuterium.chat;

import net.minecraft.network.chat.Component;
import org.junit.jupiter.api.Test;
import java.time.Instant;
import java.util.*;
import java.util.concurrent.atomic.AtomicLong;
import static org.junit.jupiter.api.Assertions.*;

class ChatStoreTest {
    @Test void uuidNameDeltaRefreshesExistingHeadersWithoutReplayingSoundOrExtendingLifetime(){
        var message=message("rename","world",100,false);message.pending=true;message.failure="retry";store.add(message);
        store.putContact(new ChatStore.Contact("private:one","Alice","sender",true,2));
        var changes=com.google.gson.JsonParser.parseString("[{\"sequence\":20,\"playerRef\":\"sender\",\"uuid\":\"11111111-1111-4111-8111-111111111111\",\"gameId\":\"NewName\"}]").getAsJsonArray();
        assertTrue(store.applyNames(changes));var renamed=store.hud().getFirst();
        assertEquals("NewName",renamed.name);assertEquals(message.receivedAt,renamed.receivedAt);assertEquals(message.sentAt,renamed.sentAt);assertSame(message.content,renamed.content);
        assertTrue(renamed.pending);assertEquals("retry",renamed.failure);assertEquals(1,sounds.size());
        store.putContact(new ChatStore.Contact("private:one","Alice","sender",true,2));assertEquals("NewName",store.contacts.get("private:one").name());
        changes.get(0).getAsJsonObject().addProperty("sequence",10);changes.get(0).getAsJsonObject().addProperty("gameId","OldName");assertFalse(store.applyNames(changes));
        store.add(message("late","world",200,true));assertEquals("NewName",store.hud().getLast().name);
    }
    private final AtomicLong now=new AtomicLong(1_000_000);
    private final List<ChatMessage> sounds=new ArrayList<>();
    private final ChatStore store=new ChatStore(now::get,sounds::add);
    private ChatMessage message(String id,String channel,long sent,boolean historical){
        return new ChatMessage(id,"sender","Alice",channel,"amiya","one",Component.literal(id),null,Instant.ofEpochMilli(sent),historical,false,"");
    }
    private List<String> ids(List<ChatMessage> messages){return messages.stream().map(m->m.id).toList();}

    @Test void nearbyExpiresAtSendTimePlusFiveMinutesWithoutNewMessages(){
        store.add(message("near","nearby",now.get()-299_999,false));
        store.add(message("world","world",0,false));
        store.add(message("team","team",0,false));
        store.add(message("private","private",0,false));
        assertEquals(List.of("near"),ids(store.inScope("nearby")));
        long revision=store.revision;
        now.incrementAndGet();store.expireNearby();
        assertEquals(revision+1,store.revision);
        assertTrue(store.inScope("nearby").isEmpty());
        assertEquals(List.of("world","team","private"),ids(store.liveHud()));
        store.expireNearby();assertEquals(revision+1,store.revision);
    }
    @Test void expiredHistoryAndDelayedPacketsDoNotReappearOrPlaySound(){
        store.add(message("history","nearby",now.get()-300_000,true));
        store.add(message("packet","nearby",now.get()-300_001,false));
        assertTrue(store.hud().isEmpty());assertTrue(sounds.isEmpty());
        var fresh=message("fresh","nearby",now.get()-299_000,true);
        store.add(fresh);now.addAndGet(1000);store.expireNearby();store.add(fresh);
        assertTrue(store.hud().isEmpty());
    }
    @Test void latestSummaryThenHistoryThenRealtimeStayChronologicalAndDeduplicated(){
        store.add(message("latest","private",300,false));
        store.add(message("old","private",100,true));
        store.add(message("middle","private",200,true));
        store.add(message("latest","private",300,true));
        store.add(message("new","private",400,false));
        assertEquals(List.of("old","middle","latest","new"),ids(store.inScope("private:one")));
        assertEquals(List.of("latest","new"),ids(store.liveHud()));
        store.add(message("delayed","private",250,true));
        assertEquals(List.of("old","middle","delayed","latest","new"),ids(store.inScope("private:one")));
    }
    @Test void replacementResortsAndEqualTimesKeepStableOrder(){
        store.add(message("pending","private",500,false));
        store.add(message("other","private",300,false));
        store.replace("pending",message("confirmed","private",200,true));
        store.add(message("equal","private",300,true));
        assertEquals(List.of("confirmed","other","equal"),ids(store.inScope("private:one")));
    }
    @Test void orderingPreservesServerSubMillisecondPrecision(){
        var earlier=message("earlier","private",100,true);
        var later=new ChatMessage("later","sender","Alice","private","amiya","one",Component.literal("later"),null,earlier.sentAt.plusNanos(1000),true,false,"");
        store.add(later);store.add(earlier);
        assertEquals(List.of("earlier","later"),ids(store.inScope("private:one")));
    }
}
