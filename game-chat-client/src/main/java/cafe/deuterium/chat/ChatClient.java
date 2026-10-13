package cafe.deuterium.chat;

import com.google.gson.*;
import net.minecraft.client.Minecraft;
import net.minecraft.client.gui.screens.ChatScreen;
import net.minecraft.network.chat.*;
import net.minecraft.network.protocol.common.ServerboundCustomPayloadPacket;
import net.neoforged.neoforge.client.event.ClientChatReceivedEvent;
import java.time.Instant;
import java.util.*;
import java.util.concurrent.*;

final class ChatClient {
    static final ChatClient INSTANCE=new ChatClient();
    private final ThreadPoolExecutor io=new ThreadPoolExecutor(1,2,20,TimeUnit.SECONDS,new ArrayBlockingQueue<>(32),r->{var t=new Thread(r,"Deuterium-Chat-IO");t.setDaemon(true);return t;},new ThreadPoolExecutor.AbortPolicy());
    final ChatMediaClient media=new ChatMediaClient(this);
    volatile ChatApi api;
    private long nextHello,nextPoll,nameCursor;
    private boolean polling,loaded,namesPending=true;
    private final Map<String,String> readThrough=new HashMap<>();
    long epoch;
    private Object connection;
    private String node="";
    private final LinkedList<ChatMessage> nativePending=new LinkedList<>();
    private record PendingRich(String legacy,ChatMessage message,long expires){}
    private final LinkedList<PendingRich> lateRich=new LinkedList<>();
    private final ConcurrentMap<String,CompletableFuture<JsonObject>> mediaRequests=new ConcurrentHashMap<>();

    void tick() {
        ChatStore.INSTANCE.expireNearby();
        media.pump();
        var mc=Minecraft.getInstance();var c=mc.getConnection();if(mc.level==null||c==null)return;
        if(connection!=c){reset();connection=c;}
        long now=System.currentTimeMillis();
        while(!nativePending.isEmpty()&&System.nanoTime()/1_000_000-nativePending.getFirst().receivedAt>=(ChatLegacy.hasItem(nativePending.getFirst().content)?600:100))ChatStore.INSTANCE.add(nativePending.removeFirst());
        lateRich.removeIf(p->p.expires()<now);
        if((api==null||!api.valid())&&now>=nextHello){nextHello=now+15000;packet("hello",new JsonObject());}
        if(api!=null&&api.valid()&&!polling&&now>=nextPoll){nextPoll=now+1500;poll();}
    }
    void receive(String json) {
        try {
            JsonObject p=JsonParser.parseString(json).getAsJsonObject();
            switch(ChatMessage.str(p,"type")) {
                case "player-title" -> ChatTitles.receive(p);
                case "player-names" -> {if(ChatStore.INSTANCE.applyNames(p.getAsJsonArray("items")))ChatAvatars.clear();}
                case "session" -> {api=new ChatApi(p);node=api.server;ChatStore.INSTANCE.server=node;ChatStore.INSTANCE.selfRef=api.owner;ChatStore.INSTANCE.status="";nextPoll=0;namesPending=true;media.refreshStickers();loadPins();}
                case "media-result" -> {var future=mediaRequests.remove(ChatMessage.str(p,"requestId"));if(future!=null){if(p.has("error"))future.completeExceptionally(new java.io.IOException(ChatMessage.str(p,"error")));else future.complete(p.getAsJsonObject("message"));}}
                case "message" -> {
                    var m=ChatMessage.fromJson(p,false,ChatMessage.str(p,"channel"),ChatMessage.str(p,"conversationId"));
                    String legacy=ChatLegacy.localize(ChatMessage.str(p,"legacyPlain"));Component original=null;
                    if(!legacy.isEmpty()){
                        for(var it=nativePending.iterator();it.hasNext();){var n=it.next();if(n.content.getString().equals(legacy)){original=n.content;it.remove();break;}}
                        if(original==null)original=ChatStore.INSTANCE.takeLegacy(legacy);
                    }
                    if(original!=null)m=ChatLegacy.merge(m,legacy,original);
                    else if(!legacy.isEmpty()){lateRich.add(new PendingRich(legacy,m,System.currentTimeMillis()+5000));while(lateRich.size()>128)lateRich.removeFirst();}
                    if(original!=null)ChatStore.INSTANCE.replace(m.id,m);else ChatStore.INSTANCE.add(m);
                }
                case "location-result" -> sendTo(ChatMessage.str(p,"content"),ChatMessage.str(p,"scope"));
                case "notice" -> notice(ChatMessage.str(p,"message"));
                case "session-unavailable" -> {ChatStore.INSTANCE.status=ChatMessage.str(p,"message");ChatStore.INSTANCE.changed();}
                case "hello" -> {node=ChatMessage.str(p,"serverId");ChatStore.INSTANCE.server=node;}
            }
        }catch(Exception ignored){notice("聊天连接信息不可用，请稍后重试");}
    }
    void nativeMessage(ClientChatReceivedEvent e) {
        if(e instanceof ClientChatReceivedEvent.System s&&s.isOverlay())return;
        String name="",sender=e.getSender().toString();Component text=e.getMessage();
        boolean system=e.isSystem();
        if(e instanceof ClientChatReceivedEvent.Player p){
            var info=Minecraft.getInstance().getConnection().getPlayerInfo(e.getSender());name=info==null?p.getBoundChatType().name().getString():info.getProfile().getName();
            text=p.getPlayerChatMessage().decoratedContent();
        }
        for(var it=lateRich.iterator();it.hasNext();){var pending=it.next();if(pending.expires()>=System.currentTimeMillis()&&pending.legacy().equals(text.getString())){it.remove();ChatStore.INSTANCE.replace(pending.message().id,ChatLegacy.merge(pending.message(),pending.legacy(),text));return;}}
        var m=new ChatMessage("native_"+UUID.randomUUID(),sender,name,"world",node,"",text,null,Instant.now(),false,system,"");
        if(system){nativePending.add(m);while(nativePending.size()>128)ChatStore.INSTANCE.add(nativePending.removeFirst());}else ChatStore.INSTANCE.add(m);
    }
    void shareLocation(){String scope=ChatStore.INSTANCE.current;if(scope.equals("private")||scope.equals("inbox")){notice("请先选择一位玩家");return;}var p=new JsonObject();p.addProperty("scope",scope);p.addProperty("live",false);packet("location",p);}
    boolean shareItem(){var mc=Minecraft.getInstance();if(!ChatStore.INSTANCE.current.equals("world")){notice("当前物品暂仅支持世界频道");return false;}if(mc.player==null||mc.player.getMainHandItem().isEmpty()){notice("主手没有可分享的物品");return false;}return sendTo("[item]","world");}
    boolean send(String text) {return sendTo(text,ChatStore.INSTANCE.current);}
    private boolean sendTo(String text,String scope) {
        var mc=Minecraft.getInstance();if(mc.getConnection()==null)return false;
        if(text.length()>256){notice("消息最多 256 个字符");return false;}
        if(text.startsWith("/")){mc.getConnection().sendCommand(text.substring(1));return true;}
        if(scope.equals("world")){mc.getConnection().sendChat(text);return true;}
        if(scope.equals("nearby")||scope.equals("team")){
            JsonObject p=new JsonObject();p.addProperty("channel",scope);p.addProperty("content",text);p.addProperty("requestId",UUID.randomUUID().toString());packet("send",p);return true;
        }
        if(scope.startsWith("private:")){
            if(api==null){notice("云端聊天尚未连接");return false;}
            JsonObject body=new JsonObject();body.addProperty("clientMessageId",UUID.randomUUID().toString());body.addProperty("content",text);
            submit(a->a.request("POST","/api/v1/chat/conversations/"+scope.substring(8)+"/messages",body),r->{nextPoll=0;loadScope(scope);});return true;
        }
        notice("请先选择一位玩家");return false;
    }
    private java.nio.file.Path pinFile(ChatApi a)throws Exception {
        String key=java.util.HexFormat.of().formatHex(java.security.MessageDigest.getInstance("SHA-256").digest((a.base+"|"+a.owner).getBytes(java.nio.charset.StandardCharsets.UTF_8)));
        return Minecraft.getInstance().gameDirectory.toPath().resolve("config/deuterium-chat/pins-"+key+".json");
    }
    private void loadPins(){submit(a->{var path=pinFile(a);if(!java.nio.file.Files.exists(path))return new JsonArray();try{return JsonParser.parseString(java.nio.file.Files.readString(path));}catch(Exception ignored){return new JsonArray();}},raw->{if(raw.isJsonArray()){ChatStore.INSTANCE.pinned.clear();for(var id:raw.getAsJsonArray())if(id.isJsonPrimitive())ChatStore.INSTANCE.pinned.add(id.getAsString());ChatStore.INSTANCE.changed();}});}
    void savePins(){var ids=new JsonArray();ChatStore.INSTANCE.pinned.forEach(ids::add);submit(a->{var path=pinFile(a);java.nio.file.Files.createDirectories(path.getParent());var tmp=path.resolveSibling(path.getFileName()+".tmp");java.nio.file.Files.writeString(tmp,ids.toString());java.nio.file.Files.move(tmp,path,java.nio.file.StandardCopyOption.REPLACE_EXISTING);return JsonNull.INSTANCE;},raw->{});}
    void packet(String type,JsonObject p) {
        var c=Minecraft.getInstance().getConnection();if(c==null)return;
        p.addProperty("type",type);c.send(new ServerboundCustomPayloadPacket(new ChatPayload(p.toString())));
    }
    JsonObject sendScopedMedia(String scope,JsonObject media,String id,long generation)throws Exception {
        var future=new CompletableFuture<JsonObject>();mediaRequests.put(id,future);
        Minecraft.getInstance().execute(()->{if(epoch!=generation){future.completeExceptionally(new java.io.IOException("游戏会话已切换"));return;}var p=new JsonObject();p.addProperty("channel",scope);p.addProperty("requestId",id);p.add("media",media);packet("media",p);});
        try{return future.get(30,TimeUnit.SECONDS);}finally{mediaRequests.remove(id,future);}
    }
    interface Work { JsonElement run(ChatApi api)throws Exception; }
    void submit(Work work,java.util.function.Consumer<JsonElement> done) {
        ChatApi owner=api;long generation=epoch;
        if(owner==null||!owner.valid()){notice("云端聊天尚未连接");return;}
        try{io.execute(()->{try{var result=work.run(owner);Minecraft.getInstance().execute(()->{if(epoch==generation&&api!=null&&api.owner.equals(owner.owner)&&api.base.equals(owner.base))done.accept(result);});}
            catch(Exception e){Minecraft.getInstance().execute(()->{if(epoch==generation){polling=false;notice(e.getMessage());}});}});}catch(RejectedExecutionException busy){polling=false;notice("聊天正在处理中，请稍后重试");}
    }
    private void poll() {
        polling=true;boolean historical=!loaded,fetchNames=namesPending;
        submit(a->{var o=new JsonObject();if(fetchNames)try{o.add("names",a.request("GET","/api/v1/chat/player-names?after="+nameCursor,null));}catch(Exception ignored){}o.add("messages",a.request("GET","/api/v1/chat/messages?limit=100",null));o.add("conversations",a.request("GET","/api/v1/chat/conversations",null));return o;},raw->{
            polling=false;loaded=true;var o=raw.getAsJsonObject();
            if(o.has("names")){var names=o.getAsJsonObject("names");if(ChatStore.INSTANCE.applyNames(array(names)))ChatAvatars.clear();if(names.has("cursor"))nameCursor=names.get("cursor").getAsLong();namesPending=names.has("hasMore")&&names.get("hasMore").getAsBoolean();if(namesPending)nextPoll=0;}
            var list=array(o.get("messages"));
            for(int i=list.size()-1;i>=0;i--)ChatStore.INSTANCE.add(ChatMessage.fromJson(list.get(i).getAsJsonObject(),historical,"world",""));
            for(var e:array(o.get("conversations"))){var c=e.getAsJsonObject();String id=ChatMessage.str(c,"conversationId");
                if(c.has("lastMessage")&&c.get("lastMessage").isJsonObject())ChatStore.INSTANCE.add(ChatMessage.fromJson(c.getAsJsonObject("lastMessage"),historical,"private",id));
                JsonObject other=c.has("otherPlayer")?c.getAsJsonObject("otherPlayer"):new JsonObject();
                ChatStore.INSTANCE.putContact(new ChatStore.Contact("private:"+id,ChatMessage.str(other,"gameId"),ChatMessage.str(other,"playerRef"),false,c.has("unreadCount")?c.get("unreadCount").getAsInt():0));}
            ChatStore.INSTANCE.changed();if(ChatStore.INSTANCE.current.startsWith("private:"))loadScope(ChatStore.INSTANCE.current);
        });
    }
    static JsonArray array(JsonElement e){if(e==null||e.isJsonNull())return new JsonArray();if(e.isJsonArray())return e.getAsJsonArray();for(String k:List.of("items","messages","conversations","players","stickers"))if(e.getAsJsonObject().has(k)&&e.getAsJsonObject().get(k).isJsonArray())return e.getAsJsonObject().getAsJsonArray(k);return new JsonArray();}
    void loadScope(String scope){if(api==null)return;if(scope.equals("nearby")||scope.equals("team")){submit(a->a.request("GET","/api/v1/chat/game-messages?channel="+scope,null),r->{var list=array(r);for(int i=list.size()-1;i>=0;i--)ChatStore.INSTANCE.add(ChatMessage.fromJson(list.get(i).getAsJsonObject(),true,scope,""));});return;}if(!scope.startsWith("private:"))return;submit(a->a.request("GET","/api/v1/chat/conversations/"+scope.substring(8)+"/messages?limit=100",null),r->{var list=array(r);for(int i=list.size()-1;i>=0;i--)ChatStore.INSTANCE.add(ChatMessage.fromJson(list.get(i).getAsJsonObject(),true,"private",scope.substring(8)));if(!list.isEmpty()&&ChatStore.INSTANCE.current.equals(scope)&&Minecraft.getInstance().screen instanceof ChatScreenFull){String latest=ChatMessage.str(list.get(0).getAsJsonObject(),"messageId");if(!latest.equals(readThrough.get(scope))){var body=new JsonObject();body.addProperty("clientRequestId",UUID.randomUUID().toString());body.addProperty("lastReadMessageId",latest);submit(a->a.request("POST","/api/v1/chat/conversations/"+scope.substring(8)+"/read",body),done->{readThrough.put(scope,latest);nextPoll=0;});}}});}
    void notice(String message){ChatStore.INSTANCE.status=message==null?"聊天暂不可用":message;ChatStore.INSTANCE.changed();Minecraft.getInstance().gui.setOverlayMessage(Component.literal(ChatStore.INSTANCE.status),false);}
    boolean click(Style style){
        var e=style.getClickEvent();if(e==null)return false;var mc=Minecraft.getInstance();
        if(e.getAction()==ClickEvent.Action.SUGGEST_COMMAND){mc.setScreen(new ChatScreen(e.getValue()));return true;}
        if(e.getAction()==ClickEvent.Action.RUN_COMMAND&&e.getValue().startsWith("/")){
            mc.getConnection().sendCommand(e.getValue().substring(1));
            if(!"deuterium:keep-chat".equals(style.getInsertion())&&!e.getValue().startsWith("/redpacket"))mc.setScreen(null);return true;
        }
        return mc.screen!=null&&mc.screen.handleComponentClicked(style);
    }
    void reset(){ChatTitles.reset();readThrough.clear();epoch++;api=null;connection=null;polling=false;loaded=false;nextHello=nextPoll=nameCursor=0;node="";nativePending.clear();lateRich.clear();for(var future:mediaRequests.values())future.completeExceptionally(new java.io.IOException("游戏会话已断开"));mediaRequests.clear();io.getQueue().clear();media.clear();ChatAvatars.clear();ChatStore.INSTANCE.clear();ChatBackdrop.close();}
}
