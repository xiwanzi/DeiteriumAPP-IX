package cafe.deuterium.core.game;

import cafe.deuterium.core.config.CoreConfig;
import cafe.deuterium.core.runtime.Sessions;
import cafe.deuterium.core.util.Json;
import com.google.gson.*;
import org.bukkit.Bukkit;
import org.bukkit.entity.Player;
import org.bukkit.plugin.java.JavaPlugin;
import java.net.URI;
import java.net.http.*;
import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.util.*;
import java.util.concurrent.*;
import java.util.function.*;

/** Core alone evaluates title permissions. All Player and permission access stays on the server thread. */
public final class GameTitles implements AutoCloseable, org.bukkit.plugin.messaging.PluginMessageListener {
    public static final String CHANNEL="deuterium:titles";
    private final Map<UUID,long[]> rates=new HashMap<>();
    private final JavaPlugin plugin;
    private final Supplier<CoreConfig> config;
    private final Sessions sessions;
    private final BiConsumer<Player,JsonObject> deliver;
    private final ExecutorService io=Executors.newSingleThreadExecutor(r->{var t=new Thread(r,"Deuterium-Titles");t.setDaemon(true);return t;});
    private final HttpClient http=HttpClient.newBuilder().connectTimeout(Duration.ofSeconds(3)).followRedirects(HttpClient.Redirect.NEVER).build();
    private final org.bukkit.scheduler.BukkitTask task;
    private JsonObject catalog;
    // TAB-Bridge can request placeholders asynchronously. Published snapshots are never mutated.
    private final Map<UUID,JsonObject> profiles=new ConcurrentHashMap<>();
    private Runnable unregisterPlaceholders=()->{};
    private final Set<UUID> pending=new HashSet<>();
    private boolean polling;
    private volatile boolean closed;
    private long refreshed;
    public GameTitles(JavaPlugin plugin,Supplier<CoreConfig> config,Sessions sessions,BiConsumer<Player,JsonObject> deliver){
        this.plugin=plugin;this.config=config;this.sessions=sessions;this.deliver=deliver;
        Bukkit.getMessenger().registerIncomingPluginChannel(plugin,CHANNEL,this);
        Bukkit.getMessenger().registerOutgoingPluginChannel(plugin,CHANNEL);
        task=Bukkit.getScheduler().runTaskTimer(plugin,this::poll,20,40);
        if(Bukkit.getPluginManager().isPluginEnabled("PlaceholderAPI"))
            unregisterPlaceholders=TitlePlaceholders.register(plugin,this::title);
    }
    @Override public void onPluginMessageReceived(String channel,Player player,byte[] bytes){
        if(!channel.equals(CHANNEL)||bytes.length>4096)return;
        long now=System.currentTimeMillis();var rate=rates.computeIfAbsent(player.getUniqueId(),id->new long[2]);
        if(now-rate[0]>1000){rate[0]=now;rate[1]=0;}if(++rate[1]>20){notice(player,"请求过于频繁，请稍后重试");return;}
        try{var packet=Json.object(new String(bytes,StandardCharsets.UTF_8),4096);String type=Json.string(packet,"type");
            if(!type.equals("titles-list")&&!type.equals("title-select"))return;
            if(!Json.string(packet,"requestId").matches("[A-Za-z0-9_-]{1,100}"))throw new IllegalArgumentException("请求编号不正确");
            request(player,packet);
        }catch(RuntimeException e){notice(player,"头衔请求格式不正确");}
    }
    private void reply(Player player,JsonObject packet){byte[] bytes=Json.GSON.toJson(packet).getBytes(StandardCharsets.UTF_8);if(bytes.length<=28000&&player.isOnline())player.sendPluginMessage(plugin,CHANNEL,bytes);}
    private void main(Runnable action){if(!closed&&plugin.isEnabled())Bukkit.getScheduler().runTask(plugin,()->{if(!closed)action.run();});}
    private void poll(){
        if(polling||closed)return;polling=true;rates.keySet().removeIf(id->Bukkit.getPlayer(id)==null);
        io.execute(()->{try{var result=post("catalog",new JsonObject());main(()->{
            catalog=result.getAsJsonObject("catalog");refreshed=System.currentTimeMillis();
            for(var raw:result.getAsJsonArray("profiles"))publish(raw.getAsJsonObject());
            polling=false;for(Player player:Bukkit.getOnlinePlayers())reconcile(player,null);
        });}catch(Exception e){main(()->{polling=false;if(System.currentTimeMillis()-refreshed>15000){catalog=null;for(Player player:Bukkit.getOnlinePlayers())fallback(player);}});}});
    }
    private JsonArray allowed(Player player){
        var ids=new JsonArray();
        boolean lp=Bukkit.getPluginManager().isPluginEnabled("LuckPerms");
        for(var raw:catalog.getAsJsonArray("items")){var t=raw.getAsJsonObject();String id=Json.string(t,"id");
            if(id.equals("title_default")||lp&&TitlePermissions.allowed(player,Json.string(t,"internalId")))ids.add(id);
        }return ids;
    }
    public void request(Player player,JsonObject packet){
        if(catalog==null||System.currentTimeMillis()-refreshed>15000){notice(player,"头衔服务暂不可用，请稍后重试");return;}
        if(Json.string(packet,"type").equals("titles-list")){
            var permitted=new HashSet<String>();allowed(player).forEach(v->permitted.add(v.getAsString()));
            var result=TitleCatalogPage.build(catalog,permitted,packet);
            result.add("current",profiles.get(player.getUniqueId()));reply(player,result);
        }else reconcile(player,packet);
    }
    private void reconcile(Player player,JsonObject selection){
        UUID uuid=player.getUniqueId();var session=sessions.get(uuid);if(session==null)return;
        if(pending.contains(uuid)){if(selection!=null)notice(player,"头衔状态正在同步，请稍后选择");return;}
        var request=new JsonObject();request.addProperty("playerUuid",uuid.toString());request.addProperty("sessionEpoch",session.sessionEpoch());
        request.add("catalogRevision",catalog.get("revision"));var allowed=allowed(player);request.add("allowedIds",allowed);
        if(selection!=null){
            String id=Json.string(selection,"titleId"),requestId=Json.string(selection,"requestId");
            if(!requestId.matches("[A-Za-z0-9_-]{1,100}")||!id.matches("[A-Za-z0-9_-]{1,100}"))throw new IllegalArgumentException("头衔请求不正确");
            boolean granted=false;for(var value:allowed)granted|=value.getAsString().equals(id);if(!granted){notice(player,"当前没有该头衔权限");return;}
            request.addProperty("titleId",id);request.addProperty("clientRequestId",requestId);request.add("expectedVersion",selection.get("expectedVersion"));
        }
        pending.add(uuid);
        io.execute(()->{try{var state=post(selection==null?"state":"select",request);main(()->{
            pending.remove(uuid);if(!sessions.current(uuid,session.sessionEpoch()))return;publish(state);
            if(selection!=null){var result=state.deepCopy();result.addProperty("type","title-selected");result.add("requestId",selection.get("requestId"));reply(player,result);}
        });}catch(Exception e){main(()->{pending.remove(uuid);if(sessions.current(uuid,session.sessionEpoch())&&selection!=null)notice(player,"选择未确认，请刷新后使用原请求重试");});}});
    }
    private void publish(JsonObject state){
        UUID id=UUID.fromString(Json.string(state,"playerUuid"));JsonObject previous=profiles.put(id,state.deepCopy());
        if(state.equals(previous))return;
        var packet=state.deepCopy();packet.addProperty("type","player-title");for(Player p:Bukkit.getOnlinePlayers()){deliver.accept(p,packet);if(p.getUniqueId().equals(id))reply(p,packet);}
    }
    public void hello(Player player){for(var state:profiles.values()){var packet=state.deepCopy();packet.addProperty("type","player-title");deliver.accept(player,packet);}}
    public net.kyori.adventure.text.Component title(UUID uuid){
        var state=profiles.get(uuid);String value=state==null?"溯行者":Json.string(state,"displayName");
        var out=net.kyori.adventure.text.Component.empty();var matcher=java.util.regex.Pattern.compile("#[0-9a-fA-F]{6}").matcher(value);
        int start=0,color=0xA9B3B8;
        while(matcher.find()){
            if(matcher.start()>start)out=out.append(net.kyori.adventure.text.Component.text(value.substring(start,matcher.start()),net.kyori.adventure.text.format.TextColor.color(color)));
            color=Integer.parseInt(matcher.group().substring(1),16);start=matcher.end();
        }
        return out.append(net.kyori.adventure.text.Component.text(value.substring(start),net.kyori.adventure.text.format.TextColor.color(color)));
    }
    private void fallback(Player player){var state=new JsonObject();state.addProperty("playerUuid",player.getUniqueId().toString());state.addProperty("gameId",player.getName());state.addProperty("titleId","title_default");state.addProperty("displayName","溯行者");state.addProperty("version",0);publish(state);}
    private void notice(Player p,String text){var result=new JsonObject();result.addProperty("type","title-error");result.addProperty("message",text);reply(p,result);}
    private JsonObject post(String action,JsonObject payload)throws Exception{
        var c=config.get();if(!c.bridge().enabled())throw new IllegalStateException("Bridge disabled");
        var uri=c.bridge().uri();var base=URI.create((uri.getScheme().equals("wss")?"https":"http")+"://"+uri.getRawAuthority());
        var req=HttpRequest.newBuilder(base.resolve("/bridge/v1/game-chat/titles/"+action)).timeout(Duration.ofSeconds(5)).header("Authorization","Bearer "+c.bridge().token()).header("X-Deuterium-Node-ID",c.nodeId()).header("Content-Type","application/json").POST(HttpRequest.BodyPublishers.ofString(Json.GSON.toJson(payload))).build();
        var response=http.send(req,HttpResponse.BodyHandlers.ofInputStream());byte[] bytes;try(var stream=response.body()){bytes=stream.readNBytes(1048577);}
        if(response.statusCode()!=200||bytes.length>1048576)throw new IllegalStateException("Title service unavailable");
        return Json.object(new String(bytes,StandardCharsets.UTF_8),1048576).getAsJsonObject("data");
    }
    @Override public void close(){closed=true;unregisterPlaceholders.run();Bukkit.getMessenger().unregisterIncomingPluginChannel(plugin,CHANNEL,this);Bukkit.getMessenger().unregisterOutgoingPluginChannel(plugin,CHANNEL);rates.clear();task.cancel();io.shutdownNow();http.shutdownNow();profiles.clear();pending.clear();}
}
