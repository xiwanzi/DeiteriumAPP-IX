package cafe.deuterium.core.game;

import cafe.deuterium.core.storage.PlayerStore;
import cafe.deuterium.core.util.Json;
import com.google.gson.JsonArray;
import com.google.gson.JsonObject;
import java.util.UUID;
import java.util.concurrent.*;
import java.util.function.Consumer;
import java.util.logging.Logger;

/** Durable local name state separates the chat/Core consumer from optional guild IO. */
public final class PlayerNameSync implements AutoCloseable {
    @FunctionalInterface public interface Feed { JsonObject read(long after) throws Exception; }
    @FunctionalInterface public interface TeamSink { void apply(PlayerStore.NameChange name) throws Exception; }
    private final PlayerStore players;
    private final Feed feed;
    private final Consumer<JsonArray> broadcast;
    private final TeamSink team;
    private final Logger log;
    private final ScheduledExecutorService workers = Executors.newScheduledThreadPool(2, task -> {
        Thread thread = new Thread(task, "Deuterium-PlayerNames"); thread.setDaemon(true); return thread;
    });
    private volatile boolean closed;
    private final Task core = new Task("Core", this::readCore);
    private final Task guild = new Task("Teams", this::readTeams);

    public PlayerNameSync(PlayerStore players, Feed feed, Consumer<JsonArray> broadcast, TeamSink team, Logger log) {
        this.players=players; this.feed=feed; this.broadcast=broadcast; this.team=team; this.log=log;
    }
    public void refresh() { core.trigger(); guild.trigger(); }
    private boolean readCore() throws Exception {
        JsonObject response=feed.read(players.nameCursor());
        JsonArray entries=response.getAsJsonArray("items");
        for(var raw:entries) {
            var value=raw.getAsJsonObject();
            players.canonicalName(UUID.fromString(Json.string(value,"uuid")), Json.string(value,"gameId"), value.get("sequence").getAsLong());
        }
        if(!entries.isEmpty()) broadcast.accept(entries.deepCopy());
        players.nameCursor(response.get("cursor").getAsLong());
        guild.trigger();
        return response.get("hasMore").getAsBoolean();
    }
    private boolean readTeams() throws Exception {
        var entries=players.pendingNames(players.nameCursor(":teams"));
        for(var value:entries) { team.apply(value); players.nameCursor(":teams",value.sequence()); }
        return entries.size()==100;
    }
    public static void applyTeam(PlayerStore.NameChange value) throws Exception {
        Class<?> gateway;
        try { gateway=Class.forName("com.deuteriumix.teams.Gateway"); }
        catch(ClassNotFoundException absent) { return; }
        Object runtime=gateway.getMethod("runtime").invoke(null);
        if(runtime==null) throw new IllegalStateException("Team runtime pending");
        Object result=runtime.getClass().getMethod("syncVerifiedName",UUID.class,String.class,long.class)
                .invoke(runtime,value.uuid(),value.name(),value.sequence());
        if(result instanceof CompletableFuture<?> future) future.get(3,TimeUnit.SECONDS);
    }
    @FunctionalInterface private interface Page { boolean read() throws Exception; }
    private final class Task implements Runnable {
        private final String name; private final Page page;
        private boolean running, dirty; private int failures;
        private ScheduledFuture<?> scheduled;
        private Task(String name,Page page) { this.name=name; this.page=page; }
        synchronized void trigger() {
            if(closed) return;
            dirty=true;
            if(running || scheduled!=null) return;
            scheduled=workers.schedule(this,0,TimeUnit.SECONDS);
        }
        @Override public void run() {
            synchronized(this) { if(closed)return; scheduled=null; running=true; dirty=false; }
            boolean more=false, failed=false;
            try { more=page.read(); failures=0; }
            catch(Exception error) {
                failed=true; failures=Math.min(failures+1,8);
                log.warning("名称同步 " + name + " 延后重试：" + error.getClass().getSimpleName());
            }
            synchronized(this) {
                running=false;
                if(!closed && (failed || more || dirty)) scheduled=workers.schedule(this,failed ? 1L<<failures : 0,TimeUnit.SECONDS);
            }
        }
    }
    @Override public void close() { closed=true; workers.shutdownNow(); }
}
