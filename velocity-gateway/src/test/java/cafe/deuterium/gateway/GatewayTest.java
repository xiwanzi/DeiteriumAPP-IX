package cafe.deuterium.gateway;

import com.google.gson.JsonParser;
import com.velocitypowered.api.event.Continuation;
import com.velocitypowered.api.event.connection.LoginEvent;
import com.velocitypowered.api.event.player.KickedFromServerEvent;
import com.velocitypowered.api.event.player.PlayerChooseInitialServerEvent;
import com.velocitypowered.api.event.player.ServerConnectedEvent;
import com.velocitypowered.api.proxy.Player;
import com.velocitypowered.api.proxy.ProxyServer;
import com.velocitypowered.api.proxy.server.RegisteredServer;
import com.velocitypowered.api.proxy.server.ServerInfo;
import net.kyori.adventure.text.Component;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;
import org.slf4j.Logger;
import java.net.InetSocketAddress;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.Optional;
import java.util.UUID;
import java.util.concurrent.CompletableFuture;
import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

class GatewayTest {
    @TempDir Path directory;
    private static final UUID PLAYER=UUID.fromString("00000000-0000-4000-8000-000000000091");

    @Test void persistsAcrossRestartAndRecoversBackup() throws Exception {
        LastServerStore store=new LastServerStore(directory);
        store.put(PLAYER,"amiya");store.flush();
        assertEquals("amiya",new LastServerStore(directory).get(PLAYER));
        store.put(PLAYER,"odyssey");store.flush();
        assertEquals("odyssey",new LastServerStore(directory).get(PLAYER));
        Files.writeString(directory.resolve("last-servers.json"),"{broken");
        assertEquals("amiya",new LastServerStore(directory).get(PLAYER));
        assertTrue(Files.list(directory).anyMatch(p->p.getFileName().toString().startsWith("last-servers.corrupt-")));
    }
    @Test void corruptFileWithoutBackupDoesNotSilentlyResetPlayers() throws Exception {
        Files.writeString(directory.resolve("last-servers.json"),"not-json");
        assertThrows(java.io.IOException.class,()->new LastServerStore(directory));
    }
    @Test void fallbackIsNotRememberedButManualTransferIs() {
        ReconnectSession s=new ReconnectSession(new Object());s.attempting("odyssey");s.fallback("amiya");
        assertFalse(s.connected("amiya"));assertTrue(s.isFallbackStay());
        assertTrue(s.connected("mek"));assertFalse(s.isFallbackStay());
        assertFalse(s.attempted("odyssey"));assertTrue(s.attempted("mek"));
    }
    @Test void configurationRequiresSeparateHttpsCredential() {
        GatewayConfig c=new GatewayConfig();assertThrows(IllegalArgumentException.class,c::validate);
        c.apiToken="a".repeat(43);c.validate();assertEquals("amiya",c.defaultServer);
        c.apiBaseUrl="http://47.103.99.34";assertThrows(IllegalArgumentException.class,c::validate);
    }
    @Test void malformedOrInconsistentAccessIsNeverAllowed() {
        for(String json:new String[]{"{}","{\"allowed\":\"true\",\"status\":\"ACTIVE\",\"version\":1}","{\"allowed\":true,\"status\":\"REVOKED\",\"version\":1}","{\"allowed\":true,\"status\":\"ACTIVE\",\"version\":0}"}) {
            assertThrows(IllegalStateException.class,()->GatewayClient.decodeAccess(JsonParser.parseString(json).getAsJsonObject()));
        }
        assertTrue(GatewayClient.decodeAccess(JsonParser.parseString("{\"allowed\":true,\"status\":\"ACTIVE\",\"version\":1}").getAsJsonObject()).allowed());
    }

    @Test void firstConnectionUsesAmiyaAndReturningPlayerUsesRememberedServer() throws Exception {
        try(Fixture f=fixture()) {
            Player player=f.player();f.allow(player);
            var initial=new PlayerChooseInitialServerEvent(player,f.login);f.plugin.chooseInitial(initial);assertEquals(f.amiya,initial.getInitialServer().orElseThrow());
            f.history.put(PLAYER,"odyssey");initial=new PlayerChooseInitialServerEvent(player,f.login);f.plugin.chooseInitial(initial);assertEquals(f.odyssey,initial.getInitialServer().orElseThrow());
        }
    }
    @Test void unavailableRememberedServerFallsBackWithoutLosingRecord() throws Exception {
        try(Fixture f=fixture()) {
            f.history.put(PLAYER,"odyssey");Player player=f.player();f.allow(player);
            var initial=new PlayerChooseInitialServerEvent(player,f.login);f.plugin.chooseInitial(initial);
            var kicked=new KickedFromServerEvent(player,f.odyssey,Component.text("Offline"),true,KickedFromServerEvent.DisconnectPlayer.create(Component.text("Offline")));
            f.plugin.kicked(kicked);assertInstanceOf(KickedFromServerEvent.RedirectPlayer.class,kicked.getResult());
            assertEquals(f.amiya,((KickedFromServerEvent.RedirectPlayer)kicked.getResult()).getServer());
            f.plugin.connected(new ServerConnectedEvent(player,f.amiya,null));assertEquals("odyssey",f.history.get(PLAYER));
            f.plugin.connected(new ServerConnectedEvent(player,f.login,f.amiya));assertEquals("login",f.history.get(PLAYER));
        }
    }
    @Test void backendFailureAndUninitializedPluginDenyLogin() throws Exception {
        try(Fixture f=fixture()) {
            Player player=f.player();var event=new LoginEvent(player);when(f.client.check(PLAYER)).thenReturn(CompletableFuture.failedFuture(new java.io.IOException("offline")));
            Continuation continuation=mock(Continuation.class);f.plugin.login(event,continuation);assertFalse(event.getResult().isAllowed());verify(continuation).resume();
            set(f.plugin,"ready",false);event=new LoginEvent(player);f.plugin.login(event,mock(Continuation.class));assertFalse(event.getResult().isAllowed());
        }
    }
    @Test void staleLoginCallbackCannotReplaceNewConnection() throws Exception {
        try(Fixture f=fixture()) {
            Player old=f.player(),latest=f.player();CompletableFuture<GatewayClient.Access> first=new CompletableFuture<>(),second=new CompletableFuture<>();
            when(f.client.check(PLAYER)).thenReturn(first,second);
            var oldEvent=new LoginEvent(old);var latestEvent=new LoginEvent(latest);
            f.plugin.login(oldEvent,mock(Continuation.class));f.plugin.login(latestEvent,mock(Continuation.class));
            second.complete(new GatewayClient.Access(true,"ACTIVE",1,""));f.plugin.connected(new ServerConnectedEvent(latest,f.amiya,null));
            first.complete(new GatewayClient.Access(true,"ACTIVE",1,""));f.plugin.connected(new ServerConnectedEvent(old,f.odyssey,null));
            assertFalse(oldEvent.getResult().isAllowed());assertEquals("amiya",f.history.get(PLAYER));
        }
    }
    @Test void explicitOnlineKickDoesNotRedirectIntoAnotherServer() throws Exception {
        try(Fixture f=fixture()) {
            Player p=f.player();f.allow(p);
            var kick=new KickedFromServerEvent(p,f.amiya,Component.text("Ban"),false,KickedFromServerEvent.DisconnectPlayer.create(Component.text("Ban")));
            f.plugin.kicked(kick);assertInstanceOf(KickedFromServerEvent.DisconnectPlayer.class,kick.getResult());
        }
    }

    private Fixture fixture() throws Exception {return new Fixture();}
    private final class Fixture implements AutoCloseable {
        final ProxyServer proxy=mock(ProxyServer.class);final GatewayClient client=mock(GatewayClient.class);
        final RegisteredServer amiya=server("amiya",25566),odyssey=server("odyssey",25567),login=server("login",25565);
        final DeuteriumGateway plugin=new DeuteriumGateway(proxy,mock(Logger.class),directory);
        final LastServerStore history=new LastServerStore(directory);
        Fixture()throws Exception{GatewayConfig config=new GatewayConfig();config.apiToken="a".repeat(43);set(plugin,"config",config);set(plugin,"client",client);set(plugin,"history",history);set(plugin,"ready",true);}
        RegisteredServer server(String name,int port){RegisteredServer s=mock(RegisteredServer.class);when(s.getServerInfo()).thenReturn(new ServerInfo(name,new InetSocketAddress("127.0.0.1",port)));when(proxy.getServer(name)).thenReturn(Optional.of(s));return s;}
        Player player(){Player p=mock(Player.class);when(p.getUniqueId()).thenReturn(PLAYER);when(p.isActive()).thenReturn(true);when(p.getCurrentServer()).thenReturn(Optional.empty());return p;}
        void allow(Player p){when(client.check(PLAYER)).thenReturn(CompletableFuture.completedFuture(new GatewayClient.Access(true,"ACTIVE",1,"")));plugin.login(new LoginEvent(p),mock(Continuation.class));}
        public void close(){plugin.shutdown(null);}
    }
    private static void set(Object object,String field,Object value)throws Exception{var f=object.getClass().getDeclaredField(field);f.setAccessible(true);f.set(object,value);}
}
