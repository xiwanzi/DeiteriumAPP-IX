package cafe.deuterium.chat;

import net.minecraft.network.chat.Component;
import org.junit.jupiter.api.Test;
import java.time.Instant;
import java.util.List;
import static org.junit.jupiter.api.Assertions.*;

class ChatHudHistoryTest {
    private ChatMessage message(String id,boolean history){
        return new ChatMessage(id,"sender","Player","private","","conversation",
            Component.literal(id),null,Instant.now(),history,false,"");
    }
    @Test void loadingPrivateHistoryNeverReplaysItIntoQuickChat(){
        boolean sound=ChatSounds.enabled;ChatSounds.enabled=false;
        try{
            var store=new ChatStore();
            store.add(message("old-summary",true));
            store.add(message("new-private",false));
            store.add(message("older-page",true));
            assertEquals(List.of("new-private"),store.liveHud().stream().map(m->m.id).toList());
            assertEquals(3,store.inScope("private:conversation").size());
            store.add(message("new-private",true));
            assertEquals(1,store.liveHud().size());
            store.clear();assertTrue(store.liveHud().isEmpty());
        }finally{ChatSounds.enabled=sound;}
    }
}
