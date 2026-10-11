package cafe.deuterium.core.game;

import cafe.deuterium.core.config.CoreConfig;
import cafe.deuterium.core.runtime.GameThread;
import cafe.deuterium.core.util.*;
import com.google.gson.*;
import net.kyori.adventure.text.Component;
import org.bukkit.*;
import org.bukkit.entity.Player;
import org.bukkit.event.*;
import org.bukkit.event.player.PlayerQuitEvent;
import org.bukkit.plugin.java.JavaPlugin;
import org.bukkit.plugin.messaging.PluginMessageListener;
import java.net.URI;
import java.net.http.*;
import java.nio.charset.StandardCharsets;
import java.time.*;
import java.util.*;
import java.util.concurrent.*;
import java.util.function.Supplier;

/** Authenticated game transport. No DB/network/codec operation runs on the game thread. */
public final class GameChatChannel implements PluginMessageListener,Listener,AutoCloseable {
    public static final String CHANNEL="deuterium:chat";
    private final JavaPlugin plugin;private final Supplier<CoreConfig> config;private final GameThread game;
    private final GameTitles titles;
    private final ChatAccess chat;private final TeamChatAudience teams=new TeamChatAudience();
    private final ConcurrentMap<UUID,Peer> peers=new ConcurrentHashMap<>();
    private final ExecutorService io=new ThreadPoolExecutor(1,2,20,TimeUnit.SECONDS,new ArrayBlockingQueue<>(64),r->{Thread t=new Thread(r,"Deuterium-GameChat-IO");t.setDaemon(true);return t;},new ThreadPoolExecutor.AbortPolicy());
    private final HttpClient http=HttpClient.newBuilder().connectTimeout(Duration.ofSeconds(5)).followRedirects(HttpClient.Redirect.NEVER).build();
    private final Map<UUID,LocationShare> shares=new HashMap<>();
    private final org.bukkit.scheduler.BukkitTask shareTask;
    private record LocationShare(String scope,long expires){}
    private static final class Peer { long lastHello,lastSend;final LinkedHashSet<String> requests=new LinkedHashSet<>(); }
    public GameChatChannel(JavaPlugin plugin,Supplier<CoreConfig> config,GameThread game,ChatAccess chat,cafe.deuterium.core.runtime.Sessions sessions){this.plugin=plugin;this.config=config;this.game=game;this.chat=chat;
        Bukkit.getMessenger().registerIncomingPluginChannel(plugin,CHANNEL,this);Bukkit.getMessenger().registerOutgoingPluginChannel(plugin,CHANNEL);Bukkit.getPluginManager().registerEvents(this,plugin);
        titles=new GameTitles(plugin,config,sessions,(player,packet)->{if(supports(player))deliver(player,packet);});
        shareTask=Bukkit.getScheduler().runTaskTimer(plugin,()->{for(var entry:new ArrayList<>(shares.entrySet())){Player player=Bukkit.getPlayer(entry.getKey());if(player==null||System.currentTimeMillis()>entry.getValue().expires()){shares.remove(entry.getKey());if(player!=null)locationState(player,false);continue;}publishLocation(player,entry.getValue().scope());}},100,100);
    }
    public boolean supports(Player player){return peers.containsKey(player.getUniqueId());}
    @EventHandler public void quit(PlayerQuitEvent e){peers.remove(e.getPlayer().getUniqueId());shares.remove(e.getPlayer().getUniqueId());}
    @Override public void onPluginMessageReceived(String channel,Player player,byte[] bytes){
        if(!channel.equals(CHANNEL)||bytes.length>8192)return;
        try{JsonObject p=Json.object(new String(bytes,StandardCharsets.UTF_8),8192);String type=Json.string(p,"type");
            if(type.equals("hello")){hello(player);return;}
            Peer peer=peers.get(player.getUniqueId());if(peer==null)return;
            long now=System.currentTimeMillis();if(now-peer.lastSend<750){notice(player,"发送过于频繁，请稍后重试");return;}peer.lastSend=now;
            if(type.equals("titles-list")||type.equals("title-select"))titles.request(player,p);else if(type.equals("send"))send(player,peer,p);else if(type.equals("location"))location(player,p);else if(type.equals("media"))media(player,p);
        }catch(IllegalArgumentException|IllegalStateException e){notice(player,e.getMessage()==null?"消息格式不正确":e.getMessage());}
    }
    private void hello(Player player){
        Peer peer=peers.computeIfAbsent(player.getUniqueId(),id->new Peer());long now=System.currentTimeMillis();if(now-peer.lastHello<10000)return;peer.lastHello=now;
        JsonObject hello=new JsonObject();hello.addProperty("type","hello");hello.addProperty("serverId",config.get().nodeId());deliver(player,hello);titles.hello(player);
        UUID id=player.getUniqueId();JsonObject request=new JsonObject();request.addProperty("playerUuid",id.toString());
        execute(()->{try{JsonObject session=post("/bridge/v1/game-chat/session",request);session.addProperty("type","session");session.addProperty("apiBase",apiBase().toString());game.run(()->{Player current=Bukkit.getPlayer(id);if(current!=null&&peers.get(id)==peer)deliver(current,session);});}
            catch(Exception e){game.run(()->{Player current=Bukkit.getPlayer(id);if(current!=null&&peers.get(id)==peer){JsonObject unavailable=new JsonObject();unavailable.addProperty("type","session-unavailable");unavailable.addProperty("message","云端媒体暂不可用，请确认 App 已绑定游戏账号");deliver(current,unavailable);}});}});
    }
    private void send(Player sender,Peer peer,JsonObject p){
        String channel=Json.string(p,"channel"),text=Checks.text(Json.string(p,"content"),1024,false).strip(),request=Checks.operationId(Json.string(p,"requestId"));
        if(text.codePointCount(0,text.length())>256||text.indexOf('\n')>=0)throw new IllegalArgumentException("消息最多256个字符");
        if(!peer.requests.add(request))return;while(peer.requests.size()>128)peer.requests.remove(peer.requests.iterator().next());
        JsonObject m=message("game_"+sender.getUniqueId()+"_"+request,sender.getUniqueId(),sender.getName(),channel,text);
        if(channel.equals("nearby")){
            Location origin=sender.getLocation();for(Player receiver:sender.getWorld().getPlayers())if(receiver.getLocation().distanceSquared(origin)<=2500)playerMessage(receiver,m);return;
        }
        if(channel.equals("team")){
            var team=teams.resolve(sender.getUniqueId());m.addProperty("teamId",team.id());m.addProperty("teamRevision",team.revision());
            JsonArray members=new JsonArray();team.members().forEach(id->members.add(id.toString()));m.add("members",members);
            receiveTeam(m);UUID senderId=sender.getUniqueId();execute(()->{try{post("/bridge/v1/game-chat/team",m);}catch(Exception e){game.run(()->{var player=Bukkit.getPlayer(senderId);if(player!=null)notice(player,"跨服团队消息暂未送达");});}});return;
        }
        throw new IllegalArgumentException("不支持的聊天频道");
    }
    public void receiveTeam(JsonObject message){game.run(()->{
        try{UUID sender=UUID.fromString(Json.string(message.getAsJsonObject("sender"),"playerRef"));var team=teams.resolve(sender);
            if(!team.id().equals(Json.string(message,"teamId"))||team.revision()<Json.integer(message,"teamRevision"))return;
            Set<UUID> original=new HashSet<>();for(var member:message.getAsJsonArray("members"))original.add(UUID.fromString(member.getAsString()));
            for(Player player:Bukkit.getOnlinePlayers())if(original.contains(player.getUniqueId())&&team.members().contains(player.getUniqueId()))playerMessage(player,message);
        }catch(Exception ignored){/* Fail closed while the local committed team projection is unavailable. */}
    });}
    private void location(Player sender,JsonObject p){
        if(p.has("stop")&&p.get("stop").getAsBoolean()){shares.remove(sender.getUniqueId());locationState(sender,false);return;}
        String scope=Json.string(p,"scope");
        if(!Set.of("world","nearby","team").contains(scope)&&!scope.matches("private:[A-Za-z0-9_-]{1,100}"))throw new IllegalArgumentException("请选择聊天频道");
        publishLocation(sender,scope);
        if(p.has("live")&&p.get("live").getAsBoolean()){shares.put(sender.getUniqueId(),new LocationShare(scope,System.currentTimeMillis()+300000));locationState(sender,true);}
    }
    private void locationState(Player player,boolean active){var result=new JsonObject();result.addProperty("type","location-state");result.addProperty("active",active);deliver(player,result);}
    private void publishLocation(Player sender,String scope){
        Location at=sender.getLocation();String text="位置 · "+sender.getWorld().getName()+" ("+at.getBlockX()+", "+at.getBlockY()+", "+at.getBlockZ()+")";
        if(scope.equals("world")){UUID id=sender.getUniqueId();String name=sender.getName();execute(()->chat.captured(id,name,text));}
        else if(scope.startsWith("private:")){var result=new JsonObject();result.addProperty("type","location-result");result.addProperty("scope",scope);result.addProperty("content",text);deliver(sender,result);}
        else {var packet=new JsonObject();packet.addProperty("channel",scope);packet.addProperty("content",text);packet.addProperty("requestId",UUID.randomUUID().toString());try{send(sender,peers.get(sender.getUniqueId()),packet);}catch(IllegalArgumentException|IllegalStateException e){shares.remove(sender.getUniqueId());locationState(sender,false);notice(sender,e.getMessage());}}
    }
    private void media(Player sender,JsonObject p){
        String channel=Json.string(p,"channel"),requestId=Checks.operationId(Json.string(p,"requestId"));
        JsonObject media=p.getAsJsonObject("media");String mediaId=Checks.operationId(Json.string(media,"mediaId")),kind=Json.string(media,"type");
        if(!Set.of("image","sticker").contains(kind))throw new IllegalArgumentException("媒体类型不正确");
        JsonObject request=new JsonObject();request.addProperty("playerUuid",sender.getUniqueId().toString());request.addProperty("clientMessageId",requestId);request.addProperty("channel",channel);request.add("media",media);
        Set<UUID> audience=new HashSet<>();
        if(channel.equals("nearby")){Location origin=sender.getLocation();for(Player p2:sender.getWorld().getPlayers())if(p2.getLocation().distanceSquared(origin)<=2500)audience.add(p2.getUniqueId());}
        else if(channel.equals("team")){var team=teams.resolve(sender.getUniqueId());audience.addAll(team.members());request.addProperty("teamId",team.id());request.addProperty("teamRevision",team.revision());}
        else throw new IllegalArgumentException("不支持的媒体频道");
        JsonArray members=new JsonArray();audience.forEach(id->members.add(id.toString()));request.add("members",members);
        UUID senderId=sender.getUniqueId();Peer peer=peers.get(senderId);
        execute(()->{try{JsonObject result=post("/bridge/v1/game-chat/media",request);game.run(()->{
            if(peers.get(senderId)!=peer)return;
            if(channel.equals("team"))receiveTeam(result);else{
                Set<UUID> granted=new HashSet<>();for(var id:result.getAsJsonArray("members"))granted.add(UUID.fromString(id.getAsString()));
                for(Player p2:Bukkit.getOnlinePlayers())if(granted.contains(p2.getUniqueId()))playerMessage(p2,result);
            }
            Player current=Bukkit.getPlayer(senderId);if(current!=null){JsonObject ack=new JsonObject();ack.addProperty("type","media-result");ack.addProperty("requestId",requestId);ack.add("message",result);deliver(current,ack);}
        });}catch(Exception failure){game.run(()->{Player current=Bukkit.getPlayer(senderId);if(current!=null&&peers.get(senderId)==peer){JsonObject ack=new JsonObject();ack.addProperty("type","media-result");ack.addProperty("requestId",requestId);ack.addProperty("error","图片发送未完成，请使用原消息重试");deliver(current,ack);}});}});
    }
    public void publicMessage(String id,UUID sender,String name,String content,String legacy){
        JsonObject m=message(id,sender,name,"world",content);if(legacy!=null&&!legacy.isEmpty())m.addProperty("legacyPlain",legacy);
        game.run(()->{for(Player player:Bukkit.getOnlinePlayers())if(supports(player))deliver(player,m);});
    }
    public void appMessage(Player player,JsonObject payload){
        JsonObject m=message(Json.string(payload,"messageId"),UUID.fromString(Json.string(payload,"senderUuid")),Json.string(payload,"gameId"),"world",Json.string(payload,"content"));
        if(payload.has("serverId"))m.add("serverId",payload.get("serverId"));else m.addProperty("serverId","");
        if(payload.has("media"))m.add("media",payload.get("media"));deliver(player,m);
    }
    private JsonObject message(String id,UUID sender,String name,String channel,String content){
        var m=new JsonObject();m.addProperty("type","message");m.addProperty("messageId",id);m.addProperty("channel",channel);m.addProperty("serverId",config.get().nodeId());m.addProperty("content",content);m.addProperty("sentAt",Instant.now().toString());
        var actor=new JsonObject();actor.addProperty("playerRef",sender.toString());actor.addProperty("gameId",name);m.add("sender",actor);return m;
    }
    private void playerMessage(Player player,JsonObject m){if(supports(player))deliver(player,m);else{
        String name=Json.string(m.getAsJsonObject("sender"),"gameId"),channel=Json.string(m,"channel");
        player.sendMessage(Component.text(name+" ").append(titles.title(UUID.fromString(Json.string(m.getAsJsonObject("sender"),"playerRef")))).append(Component.text(" "+(channel.equals("team")?"团队":"附近")+"\n"+Json.string(m,"content"))));}}
    private void notice(Player p,String text){JsonObject m=new JsonObject();m.addProperty("type","notice");m.addProperty("message",text);deliver(p,m);}
    private void deliver(Player player,JsonObject p){byte[] bytes=Json.GSON.toJson(p).getBytes(StandardCharsets.UTF_8);if(bytes.length<=28000&&player.isOnline())player.sendPluginMessage(plugin,CHANNEL,bytes);}
    private URI apiBase(){URI uri=config.get().bridge().uri();String scheme=uri.getScheme().equals("wss")?"https":"http";return URI.create(scheme+"://"+uri.getRawAuthority()+"/");}
    private JsonObject post(String path,JsonObject payload)throws Exception{
        CoreConfig c=config.get();URI base=apiBase();
        if(!c.bridge().enabled())throw new IllegalStateException("Bridge disabled");
        var request=HttpRequest.newBuilder(base.resolve(path)).timeout(Duration.ofSeconds(8)).header("Authorization","Bearer "+c.bridge().token()).header("X-Deuterium-Node-ID",c.nodeId()).header("Content-Type","application/json").POST(HttpRequest.BodyPublishers.ofString(Json.GSON.toJson(payload))).build();
        var response=http.send(request,HttpResponse.BodyHandlers.ofInputStream());byte[] bytes;try(var input=response.body()){bytes=input.readNBytes(32769);}
        if(bytes.length>32768||response.statusCode()!=200)throw new IllegalStateException("Chat service unavailable");return Json.object(new String(bytes,StandardCharsets.UTF_8),32768).getAsJsonObject("data");
    }
    private void execute(Runnable task){try{io.execute(task);}catch(RejectedExecutionException ignored){plugin.getLogger().warning("游戏聊天队列繁忙，请稍后重试。");}}
    @Override public void close(){titles.close();shareTask.cancel();shares.clear();peers.clear();io.shutdownNow();http.shutdownNow();Bukkit.getMessenger().unregisterIncomingPluginChannel(plugin,CHANNEL,this);Bukkit.getMessenger().unregisterOutgoingPluginChannel(plugin,CHANNEL);HandlerList.unregisterAll(this);}
}
