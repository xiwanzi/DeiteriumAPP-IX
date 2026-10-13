package cafe.deuterium.chat;

import java.util.*;
import com.google.gson.JsonArray;

/** Render-thread owned retained model. Neither networking nor image decoding runs here. */
public final class ChatStore {
    public static final ChatStore INSTANCE = new ChatStore();
    private static final long NEARBY_TTL_MILLIS=5*60*1000L;
    private final java.util.function.LongSupplier clock;
    private final java.util.function.Consumer<ChatMessage> onReceived;
    public ChatStore(){this(System::currentTimeMillis,ChatSounds::received);}
    ChatStore(java.util.function.LongSupplier clock,java.util.function.Consumer<ChatMessage> onReceived){this.clock=clock;this.onReceived=onReceived;}
    private boolean expired(ChatMessage m){return m.channel.equals("nearby")&&clock.getAsLong()-m.sentMillis>=NEARBY_TTL_MILLIS;}
    public void expireNearby(){if(messages.values().removeIf(this::expired))changed();}
    private final LinkedHashMap<String,ChatMessage> messages=new LinkedHashMap<>();
    private final List<Runnable> listeners=new ArrayList<>();
    private record PlayerName(long sequence,String name) {}
    private final Map<String,PlayerName> playerNames=new HashMap<>();
    private String currentName(String ref,String fallback){var value=playerNames.get(ref);return value==null?fallback:value.name();}
    public boolean applyNames(JsonArray changes) {
        boolean updated=false;
        for(var raw:changes){try{
            var entry=raw.getAsJsonObject();String ref=ChatMessage.str(entry,"playerRef"),uuid=ChatMessage.str(entry,"uuid"),name=ChatMessage.str(entry,"gameId");long sequence=entry.get("sequence").getAsLong();
            if(ref.isBlank()||uuid.isBlank()||!name.matches("[A-Za-z0-9_]{1,32}")||sequence<1)continue;
            var previous=playerNames.get(ref);if(previous!=null&&previous.sequence()>=sequence)continue;
            var value=new PlayerName(sequence,name);playerNames.put(ref,value);playerNames.put(uuid,value);updated=true;
        }catch(RuntimeException ignored){/* An invalid name delta cannot change identity or chat content. */}}
        if(!updated)return false;
        messages.replaceAll((id,m)->m.withName(currentName(m.senderId,m.name)));
        contacts.replaceAll((id,c)->new Contact(c.id(),currentName(c.ref(),c.name()),c.ref(),c.online(),c.unread()));
        changed();return true;
    }
    public final Map<String,String> drafts=new HashMap<>();
    public final Map<String,Float> scrolls=new HashMap<>();
    public final LinkedHashMap<String,Contact> contacts=new LinkedHashMap<>();
    public final Set<String> pinned=new LinkedHashSet<>();
    public String current="world", selfRef="", server="", status="正在连接游戏聊天…";
    public long revision;
    public record Contact(String id,String name,String ref,boolean online,int unread) {}
    public void putContact(Contact c){contacts.put(c.id(),new Contact(c.id(),currentName(c.ref(),c.name()),c.ref(),c.online(),c.unread()));}
    public void add(ChatMessage m) {
        m=m.withName(currentName(m.senderId,m.name));
        if(expired(m)||messages.containsKey(m.id))return;
        messages.put(m.id,m);
        while(messages.size()>1000)messages.remove(Collections.min(messages.values(),Comparator.comparing(message->message.sentAt)).id);
        onReceived.accept(m);
        changed();
    }
    public void replace(String id,ChatMessage m) { if(id.equals(m.id)&&messages.containsKey(id)){messages.put(id,m);changed();return;}boolean removed=messages.remove(id)!=null;if(messages.containsKey(m.id)){if(removed)changed();}else add(m); }
    public net.minecraft.network.chat.Component takeLegacy(String plain) {
        ChatMessage found=null;long now=System.nanoTime()/1_000_000;
        for(var m:messages.values())if(m.system&&now-m.receivedAt<5000&&m.content.getString().equals(plain))found=m;
        if(found!=null){messages.remove(found.id);changed();return found.content;}return null;
    }
    public void changed() { revision++;for(var listener:List.copyOf(listeners))listener.run(); }
    public List<ChatMessage> inScope(String scope) { return messages.values().stream().filter(m->!expired(m)&&m.scope().equals(scope)).sorted(Comparator.comparing(m->m.sentAt)).toList(); }
    public List<ChatMessage> hud() { return messages.values().stream().filter(m->!expired(m)).sorted(Comparator.comparing(m->m.sentAt)).toList(); }
    public List<ChatMessage> liveHud() { return hud().stream().filter(m->m.receivedAt>0).toList(); }
    public AutoCloseable listen(Runnable listener) { listeners.add(listener);return ()->listeners.remove(listener); }
    public List<Contact> orderedContacts() {return contacts.values().stream().sorted(Comparator.<Contact,Boolean>comparing(c->!pinned.contains(c.id())).thenComparing(Comparator.comparingLong((Contact c)->inScope(c.id()).stream().mapToLong(m->m.sentMillis).max().orElse(0)).reversed())).toList();}
    public void clear() { messages.clear();contacts.clear();drafts.clear();scrolls.clear();pinned.clear();playerNames.clear();selfRef="";server="";current="world";changed(); }
}
