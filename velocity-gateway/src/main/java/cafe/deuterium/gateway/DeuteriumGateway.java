package cafe.deuterium.gateway;

import com.google.gson.Gson;
import com.google.gson.GsonBuilder;
import com.google.inject.Inject;
import com.velocitypowered.api.command.SimpleCommand;
import com.velocitypowered.api.event.Continuation;
import com.velocitypowered.api.event.Subscribe;
import com.velocitypowered.api.event.connection.DisconnectEvent;
import com.velocitypowered.api.event.connection.LoginEvent;
import com.velocitypowered.api.event.player.KickedFromServerEvent;
import com.velocitypowered.api.event.player.PlayerChooseInitialServerEvent;
import com.velocitypowered.api.event.player.ServerConnectedEvent;
import com.velocitypowered.api.event.proxy.ProxyInitializeEvent;
import com.velocitypowered.api.event.proxy.ProxyShutdownEvent;
import com.velocitypowered.api.plugin.Plugin;
import com.velocitypowered.api.plugin.annotation.DataDirectory;
import com.velocitypowered.api.proxy.Player;
import com.velocitypowered.api.proxy.ProxyServer;
import com.velocitypowered.api.proxy.server.RegisteredServer;
import net.kyori.adventure.text.Component;
import org.slf4j.Logger;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.List;
import java.util.UUID;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.Semaphore;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicBoolean;

@Plugin(id="deuterium-gateway",name="Deuterium Gateway",version="1.0.0",authors={"Deuterium IX"},description="Unified admission and persistent last-server routing")
public final class DeuteriumGateway {
    static final String VERSION="1.0.0";
    private final ProxyServer proxy;
    private final Logger logger;
    private final Path directory;
    private final ScheduledExecutorService worker=Executors.newScheduledThreadPool(2,r->{Thread t=new Thread(r,"deuterium-gateway");t.setDaemon(true);return t;});
    private final Semaphore logins=new Semaphore(48);
    private final ConcurrentHashMap<UUID,ReconnectSession> sessions=new ConcurrentHashMap<>();
    private final ConcurrentHashMap<UUID,Player> latestLogins=new ConcurrentHashMap<>();
    private final ConcurrentHashMap<String,GatewayClient.Ack> acknowledgements=new ConcurrentHashMap<>();
    private final java.util.Set<String> processing=ConcurrentHashMap.newKeySet();
    private final AtomicBoolean polling=new AtomicBoolean();
    private final String instance=UUID.randomUUID().toString();
    private volatile boolean ready;
    private volatile long backendSeen;
    private volatile long lastWarning;
    private GatewayConfig config;
    private GatewayClient client;
    private LastServerStore history;

    @Inject public DeuteriumGateway(ProxyServer proxy,Logger logger,@DataDirectory Path directory){this.proxy=proxy;this.logger=logger;this.directory=directory;}
    @Subscribe public void initialize(ProxyInitializeEvent ignored){
        try {
            Files.createDirectories(directory);Path file=directory.resolve("config.json");Gson gson=new GsonBuilder().setPrettyPrinting().create();
            if(!Files.exists(file))Files.writeString(file,gson.toJson(new GatewayConfig()));
            if(Files.size(file)>16384)throw new IllegalStateException("configuration exceeds limit");
            config=gson.fromJson(Files.readString(file),GatewayConfig.class);if(config==null)throw new IllegalStateException("empty config");config.validate();
            if(!proxy.getConfiguration().isOnlineMode())throw new IllegalStateException("Gateway requires Velocity online-mode=true");
            if(proxy.getServer(config.defaultServer).isEmpty())throw new IllegalStateException("Default server is not registered");
            history=new LastServerStore(directory);client=new GatewayClient(config,worker);ready=true;
            worker.scheduleWithFixedDelay(this::poll,0,5,TimeUnit.SECONDS);
            worker.scheduleWithFixedDelay(this::flush,200,200,TimeUnit.MILLISECONDS);
            proxy.getCommandManager().register(proxy.getCommandManager().metaBuilder("dgate").plugin(this).build(),new SimpleCommand(){
                @Override public void execute(Invocation invocation){invocation.source().sendMessage(Component.text("Deuterium Gateway "+VERSION+" | ready="+ready+" | backend="+(System.currentTimeMillis()-backendSeen<25000)+" | default="+config.defaultServer+" | remembered="+history.size()));}
                @Override public boolean hasPermission(Invocation invocation){return invocation.source().hasPermission("deuterium.gateway.admin");}
            });
            logger.info("Deuterium Gateway {} ready. Default={}, remembered={}, admission enforced.",VERSION,config.defaultServer,history.size());
        }catch(Exception error){ready=false;logger.error("Deuterium Gateway not ready. New logins are denied: {}",error.getMessage());}
    }

    @Subscribe(priority=-100) public void login(LoginEvent event,Continuation continuation){
        if(!event.getResult().isAllowed()){continuation.resume();return;}
        if(!ready||!logins.tryAcquire()){event.setResult(LoginEvent.ComponentResult.denied(Component.text("通行服务暂不可用，请稍后重试。")));continuation.resume();return;}
        latestLogins.put(event.getPlayer().getUniqueId(),event.getPlayer());
        try {
            client.check(event.getPlayer().getUniqueId()).whenComplete((access,error)->{
                try {
                    if(latestLogins.get(event.getPlayer().getUniqueId())!=event.getPlayer()){event.setResult(LoginEvent.ComponentResult.denied(Component.text("登录连接已更新，请使用最新连接。")));}
                    else if(error!=null||!ready){event.setResult(LoginEvent.ComponentResult.denied(Component.text("暂时无法核对通行许可，请稍后重试。")));warn("Admission check unavailable");}
                    else if(!access.allowed()){event.setResult(LoginEvent.ComponentResult.denied(Component.text(access.message()+"\n申请入口："+config.applicationUrl)));}
                    else if(event.getPlayer().isActive()){sessions.put(event.getPlayer().getUniqueId(),new ReconnectSession(event.getPlayer()));backendSeen=System.currentTimeMillis();}
                }finally{if(!event.getResult().isAllowed()||!event.getPlayer().isActive())latestLogins.remove(event.getPlayer().getUniqueId(),event.getPlayer());logins.release();continuation.resume();}
            });
        }catch(Exception error){latestLogins.remove(event.getPlayer().getUniqueId(),event.getPlayer());logins.release();event.setResult(LoginEvent.ComponentResult.denied(Component.text("通行服务暂不可用，请稍后重试。")));continuation.resume();}
    }
    private ReconnectSession session(Player player){ReconnectSession s=sessions.get(player.getUniqueId());return s!=null&&s.connection==player?s:null;}

    @Subscribe(priority=-100) public void chooseInitial(PlayerChooseInitialServerEvent event){
        ReconnectSession session=session(event.getPlayer());if(!ready||session==null){event.setInitialServer(null);return;}
        String remembered=history.get(event.getPlayer().getUniqueId());
        RegisteredServer target=remembered==null?null:proxy.getServer(remembered).orElse(null);
        if(target==null){target=proxy.getServer(config.defaultServer).orElse(null);if(target!=null&&remembered!=null)session.fallback(target.getServerInfo().getName());}
        if(target!=null)session.attempting(target.getServerInfo().getName());event.setInitialServer(target);
    }

    @Subscribe(priority=-100) public void connected(ServerConnectedEvent event){
        ReconnectSession session=session(event.getPlayer());if(!ready||session==null)return;
        String server=event.getServer().getServerInfo().getName();
        if(session.connected(server))history.put(event.getPlayer().getUniqueId(),server);
    }

    @Subscribe(priority=-100) public void kicked(KickedFromServerEvent event){
        ReconnectSession session=session(event.getPlayer());if(!ready||session==null)return;
        // A failed manual transfer leaves the player on their current server.
        if(event.kickedDuringServerConnect()&&event.getPlayer().getCurrentServer().isPresent())return;
        // Respect explicit kicks while online. Only track a redirect that Velocity already chose.
        if(!event.kickedDuringServerConnect()){
            if(event.getResult() instanceof KickedFromServerEvent.RedirectPlayer redirect)session.fallback(redirect.getServer().getServerInfo().getName());
            return;
        }
        String failed=event.getServer().getServerInfo().getName();session.attempting(failed);
        for(String name:config.fallbackServers){
            if(session.attempted(name))continue;
            RegisteredServer target=proxy.getServer(name).orElse(null);if(target==null)continue;
            session.fallback(name);
            event.setResult(KickedFromServerEvent.RedirectPlayer.create(target,Component.text("上次游玩的服务器暂时不可用，已为你连接备用服务器。")));
            return;
        }
    }
    @Subscribe public void disconnected(DisconnectEvent event){latestLogins.remove(event.getPlayer().getUniqueId(),event.getPlayer());ReconnectSession s=session(event.getPlayer());if(s!=null)sessions.remove(event.getPlayer().getUniqueId(),s);}

    private void poll(){
        if(!ready||!polling.compareAndSet(false,true))return;
        List<GatewayClient.Ack> sent=acknowledgements.values().stream().limit(25).toList();
        try{client.poll(instance,proxy.getPlayerCount(),sent).whenComplete((commands,error)->{
            try{
                if(error!=null){warn("Gateway heartbeat unavailable");return;}backendSeen=System.currentTimeMillis();
                sent.forEach(ack->acknowledgements.remove(ack.commandId(),ack));
                for(var command:commands)executeKick(command);
            }finally{polling.set(false);}
        });}catch(Exception error){polling.set(false);warn("Gateway heartbeat unavailable");}
    }
    private void executeKick(GatewayClient.Kick command){
        if(acknowledgements.containsKey(command.commandId())||!processing.add(command.commandId()))return;
        Player player=proxy.getPlayer(command.uuid()).orElse(null);
        if(player==null){acknowledgements.put(command.commandId(),new GatewayClient.Ack(command.commandId(),"NOT_ONLINE"));processing.remove(command.commandId());return;}
        client.check(command.uuid()).whenComplete((access,error)->{
            try{
                if(error!=null)return;
                if(access.allowed()||access.version()!=command.version()){acknowledgements.put(command.commandId(),new GatewayClient.Ack(command.commandId(),"CANCELLED"));return;}
                Player current=proxy.getPlayer(command.uuid()).orElse(null);
                if(current!=null)current.disconnect(Component.text("你的通行许可已被移除。\n"+command.reason()+"\n如有疑问，请联系管理组。"));
                acknowledgements.put(command.commandId(),new GatewayClient.Ack(command.commandId(),current==null?"NOT_ONLINE":"DISCONNECTED"));
            }finally{processing.remove(command.commandId());}
        });
    }
    private void warn(String message){long now=System.currentTimeMillis();if(now-lastWarning>30000){lastWarning=now;logger.warn("{}; unknown admission results remain denied.",message);}}
    private void flush(){try{if(history!=null)history.flush();}catch(Exception error){warn("Last-server persistence failed");}}
    @Subscribe public void shutdown(ProxyShutdownEvent ignored){ready=false;worker.shutdown();try{worker.awaitTermination(3,TimeUnit.SECONDS);}catch(InterruptedException e){Thread.currentThread().interrupt();}flush();worker.shutdownNow();}
}
