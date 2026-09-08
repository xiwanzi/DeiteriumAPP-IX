package cafe.deuterium.core;

import cafe.deuterium.core.api.*;
import cafe.deuterium.core.bridge.*;
import cafe.deuterium.core.config.CoreConfig;
import cafe.deuterium.core.game.*;
import cafe.deuterium.core.items.*;
import cafe.deuterium.core.mail.*;
import cafe.deuterium.core.runtime.*;
import cafe.deuterium.core.storage.*;
import cafe.deuterium.core.util.CoreFailure;
import cafe.deuterium.core.util.Json;
import com.google.gson.JsonObject;
import org.bukkit.Bukkit;
import org.bukkit.event.*;
import org.bukkit.event.server.PluginEnableEvent;
import org.bukkit.event.server.PluginDisableEvent;
import org.bukkit.plugin.ServicePriority;
import org.bukkit.plugin.java.JavaPlugin;
import java.util.Map;
import java.util.concurrent.*;
import java.util.concurrent.atomic.*;

public final class CoreRuntime implements AutoCloseable, BridgeEndpoint, Listener {
    private final JavaPlugin plugin;
    private final AtomicReference<CoreConfig> settings;
    private final AtomicBoolean closing = new AtomicBoolean();
    private final ScheduledExecutorService timer;
    public final Database database;
    private final NodeLease lease;
    public final WorkPool workers;
    public final GameThread game;
    public final Sessions sessions = new Sessions();
    public final CatalogStore catalog;
    public final OutboxStore outbox;
    public final RpcJournal journal;
    public final ItemCodec codec = new ItemCodec();
    public final StandaloneDataService localData;
    private final InboxStore inbox;
    private final PlayerStore players;
    private final PlayerEvents playerEvents;
    private final ChatAccess chat;
    private final ChatEvents chatEvents;
    private final EconomyAccess economy = new EconomyAccess();
    private final CommandDispatcher commands;
    public final BridgeClient bridge;
    private volatile CoreMailbox mailbox = new UnavailableMailbox("独立邮箱未连接。");
    public CoreRuntime(JavaPlugin plugin, CoreConfig config) throws Exception {
        this.plugin = plugin; settings = new AtomicReference<>(config);
        java.util.List<AutoCloseable> initialized = new java.util.ArrayList<>();
        database = new Database(config.storage(), plugin.getDataFolder().toPath());
        initialized.add(database);
        try {
        lease = new NodeLease(database, plugin.getDataFolder().toPath(), config.clusterId(), config.nodeId(), config.storage().sqliteFile());
        initialized.add(lease);
        workers = new WorkPool("DeuteriumCore-IO", 2, 256); game = new GameThread(plugin);
        initialized.add(workers); initialized.add(game);
        outbox = new OutboxStore(database, config.nodeId(), config.queueLimit());
        catalog = new CatalogStore(database, outbox); inbox = new InboxStore(database, config.nodeId());
        journal = new RpcJournal(database, config.nodeId()); journal.recover();
        players = new PlayerStore(database, config.nodeId()); players.offline();
        localData = new StandaloneDataService(sessions, config.inventoryDomain());
        initialized.add(localData::disable);
        if (config.syncMode().equals("standalone")) {
            if (config.nodes().size() != 1) throw CoreFailure.invalid("standalone 模式必须只配置一个节点，不能为共享背包伪造保存屏障。");
            Bukkit.getServicesManager().register(PlayerDataService.class, localData, plugin, ServicePriority.Lowest);
        }
        Bukkit.getServicesManager().register(ItemLibrary.class, new LibraryService(catalog, workers, config.namespace()), plugin, ServicePriority.Normal);
        chat = new ChatAccess(settings::get, outbox, inbox, game);
        chatEvents = new ChatEvents(plugin, settings::get, chat, workers); chatEvents.register();
        playerEvents = new PlayerEvents(sessions, players, workers); Bukkit.getPluginManager().registerEvents(playerEvents, plugin); playerEvents.initialize();
        Bukkit.getPluginManager().registerEvents(new SystemAccountGuard(economy),plugin);
        mailbox = attachMailbox();
        initialized.add(mailbox);
        commands = new CommandDispatcher(settings::get, () -> mailbox, game, sessions, players, journal, economy);
        bridge = new BridgeClient(settings::get, outbox, this, workers, plugin.getLogger());
        initialized.add(bridge);
        timer = Executors.newSingleThreadScheduledExecutor(task -> { Thread thread = new Thread(task, "DeuteriumCore-Timer"); thread.setDaemon(true); return thread; });
        initialized.add(timer::shutdownNow);
        timer.scheduleWithFixedDelay(this::tick, 0, 1, TimeUnit.SECONDS);
        Bukkit.getPluginManager().registerEvents(this, plugin);
        } catch (Exception | LinkageError failure) {
            closing.set(true); HandlerList.unregisterAll(plugin); Bukkit.getServicesManager().unregisterAll(plugin);
            for (int i = initialized.size() - 1; i >= 0; i--) try { initialized.get(i).close(); } catch (Exception suppressed) { failure.addSuppressed(suppressed); }
            throw failure;
        }
    }
    private int ticks;
    private void tick() {
        if (closing.get()) return;
        bridge.tick();
        if (++ticks % 5 == 0) workers.submit(() -> {
            if (!lease.valid()) {
                plugin.getLogger().severe("Core 节点存储租约已失效，正在停止可变操作。");
                bridge.close(); game.close(); Bukkit.getScheduler().runTask(plugin, () -> Bukkit.getPluginManager().disablePlugin(plugin));
            }
            mailbox.pumpEvents(); return null;
        });
        if (ticks % 20 == 0) playerEvents.refresh();
        if (ticks % 300 == 0) workers.submit(() -> { inbox.sweep(); return null; });
    }
    public CoreConfig config() { return settings.get(); }
    public PlayerDataService dataProvider() { return Bukkit.getServicesManager().load(PlayerDataService.class); }
    public PlayerDataService inventoryAccess() {
        PlayerDataService provider = dataProvider();
        if (provider != null && provider.available()) return provider;
        if (localData.available()) return localData;
        throw new CoreFailure("PLAYER_DATA_UNAVAILABLE", "玩家数据同步适配未就绪，已阻止背包操作。");
    }
    public CoreMailbox mailbox() { return mailbox; }
    public EconomyAccess economy() { return economy; }
    private CoreMailbox attachMailbox() {
        try { return MailboxSupport.attach(plugin, settings::get, catalog, outbox, this::dataProvider); }
        catch (LinkageError absent) { return new UnavailableMailbox("独立邮箱未安装或 API 不匹配。"); }
    }
    public void reload(CoreConfig next) {
        CoreConfig old = config();
        if (!old.nodeId().equals(next.nodeId()) || !old.clusterId().equals(next.clusterId()) || !old.storage().equals(next.storage())
                || !old.namespace().equals(next.namespace()) || !old.inventoryDomain().equals(next.inventoryDomain())
                || !old.compatibilityProfile().equals(next.compatibilityProfile()) || !old.syncMode().equals(next.syncMode())
                || old.queueLimit() != next.queueLimit()) throw CoreFailure.invalid("节点、存储、背包域和同步模式变更需要重启服务器；当前运行设置未改变。");
        settings.set(next); chatEvents.register(); bridge.reconnect();
    }
    @Override public void appChat(JsonObject payload) throws Exception { ensureOpen(); chat.incoming(payload); }
    @Override public JsonObject command(JsonObject payload) { ensureOpen(); return commands.execute(payload); }
    @Override public JsonObject presence() { return Json.tree(Map.of("players", sessions.snapshot())); }
    @Override public JsonObject status() {
        PlayerDataService data = dataProvider();
        JsonObject result = new JsonObject();
        result.addProperty("nodeId", config().nodeId()); result.addProperty("clusterId", config().clusterId());
        result.addProperty("version", "1.0.0"); result.addProperty("storageHealthy", database.healthy());
        result.addProperty("sharedStorage", database.shared()); result.addProperty("inventoryDomain", config().inventoryDomain());
        result.addProperty("compatibilityProfile", config().compatibilityProfile()); result.addProperty("chatMode", chatEvents.mode());
        result.addProperty("playerDataReady", data != null && data.available());
        result.addProperty("playerDataProvider", data == null ? "unavailable" : data.providerName());
        result.addProperty("economy", config().economyEnabled() && economy.available());
        result.addProperty("economyAuthority", config().nodeId().equals(config().economyAuthority()));
        result.add("mailbox", mailbox.capabilities()); return result;
    }
    @Override public void afterEventAcknowledged(String eventId) { mailbox.pumpEvents(); }
    @EventHandler public void dependencyEnabled(PluginEnableEvent event) {
        if (event.getPlugin().getName().equals("DeuteriumMail")) { mailbox.close(); mailbox = attachMailbox(); }
        if (event.getPlugin().getName().equals("TrChat")) chatEvents.register();
    }
    @EventHandler public void dependencyDisabled(PluginDisableEvent event) {
        if (event.getPlugin().getName().equals("DeuteriumMail")) { mailbox.close(); mailbox = new UnavailableMailbox("独立邮箱已停用。"); }
        if (event.getPlugin().getName().equals("TrChat") && !closing.get()) game.run(chatEvents::register);
    }
    public void ensureOpen() { if (closing.get()) throw new CoreFailure("CORE_STOPPED", "Core 已停用。"); }
    @Override public void close() {
        if (!closing.compareAndSet(false, true)) return;
        bridge.close(); timer.shutdownNow(); game.close(); localData.disable(); mailbox.close();
        HandlerList.unregisterAll(plugin);
        Bukkit.getServicesManager().unregisterAll(plugin); workers.close(); sessions.clear();
        try { players.offline(); } catch (Exception ignored) { } lease.close(); database.close();
    }
}
