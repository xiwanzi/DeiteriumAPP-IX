package cafe.deuterium.core.commands;

import cafe.deuterium.core.CoreRuntime;
import cafe.deuterium.core.api.*;
import cafe.deuterium.core.config.CoreConfig;
import cafe.deuterium.core.items.InventoryGrant;
import cafe.deuterium.core.runtime.Sessions;
import cafe.deuterium.core.storage.CatalogStore;
import cafe.deuterium.core.storage.Sql;
import cafe.deuterium.core.util.*;
import org.bukkit.Bukkit;
import org.bukkit.command.*;
import org.bukkit.entity.Player;
import org.bukkit.inventory.ItemStack;
import org.bukkit.plugin.java.JavaPlugin;
import java.util.*;
import java.util.concurrent.*;
import java.util.function.Function;

public final class CoreCommand implements CommandExecutor, TabCompleter {
    private final JavaPlugin plugin;
    private final CoreRuntime runtime;
    private final Set<String> busy = ConcurrentHashMap.newKeySet();
    public CoreCommand(JavaPlugin plugin, CoreRuntime runtime) { this.plugin = plugin; this.runtime = runtime; }
    @Override public boolean onCommand(CommandSender sender, Command command, String label, String[] args) {
        try {
            runtime.ensureOpen(); String action = args.length == 0 ? "help" : args[0].toLowerCase(Locale.ROOT);
            String permission = switch (action) {
                case "save" -> "items.save"; case "get", "give" -> "items.give";
                case "archive", "restore", "policy", "sync", "retry" -> "items.manage";
                case "reload", "reconnect" -> "reload"; case "status" -> "status"; case "mail" -> "mail"; case "economy" -> "economy"; default -> "items.read";
            };
            require(sender, permission);
            switch (action) {
                case "help" -> help(sender);
                case "save" -> save(sender, args);
                case "get", "give" -> give(sender, args, action.equals("give"));
                case "list" -> {
                    int page = args.length > 1 ? (int) Checks.number(args[1], 1, 100000) : 1; String query = args.length > 2 ? args[2] : "";
                    async(sender, () -> runtime.catalog.search(query, page, true), rows -> rows.isEmpty() ? "本页没有物品。" : "物品库第 " + page + " 页：\n" + String.join("\n", rows.stream().map(v -> v.itemRef() + " @" + v.latestRevision() + " · " + v.displayName() + (v.archived() ? " [已归档]" : "")).toList()));
                }
                case "info" -> {
                    String ref = ref(args, 1); long revision = args.length > 2 ? Checks.number(args[2], 1, Integer.MAX_VALUE) : 0;
                    async(sender, () -> runtime.catalog.find(ref, revision), item -> item.itemRef() + " @" + item.revision() + "\n" + item.displayName() + " · " + item.itemId() + "\n范围：" + String.join(", ", item.compatibleServerIds()) + "\n单次交付上限：" + item.maxQuantity() + "\nSHA-256：" + item.payloadSha256());
                }
                case "versions" -> {
                    String ref = ref(args, 1); int page = args.length > 2 ? (int) Checks.number(args[2], 1, 100000) : 1;
                    async(sender, () -> runtime.catalog.versions(ref, page), versions -> ref + " 版本：" + versions);
                }
                case "archive", "restore" -> {
                    String ref = ref(args, 1); boolean archive = action.equals("archive");
                    async(sender, () -> runtime.catalog.archive(ref, archive, actor(sender)), item -> item.itemRef() + (archive ? " 已归档，历史版本仍保留。" : " 已恢复。"));
                }
                case "policy" -> policy(sender, args);
                case "sync" -> async(sender, () -> { runtime.catalog.republish(); runtime.outbox.republishCatalog(); return runtime.outbox.pendingCount(); }, n -> "物品索引已加入同步队列，待发事件 " + n + " 条。");
                case "retry" -> async(sender, () -> { runtime.outbox.retryRejected(); return null; }, ignored -> "已重新排队被拒绝的事件；内容冲突需先修正来源，不能覆盖历史版本。");
                case "status" -> async(sender, () -> "节点：" + runtime.config().nodeId() + "\n后端：" + runtime.bridge.state() + "\n存储：" + runtime.config().storage().kind() + "，连接 " + runtime.database.connections() + "\n待发/拒绝：" + runtime.outbox.pendingCount() + "/" + runtime.outbox.rejectedCount() + "\n在线玩家：" + runtime.sessions.snapshot().size() + "\n同步提供者：" + (runtime.dataProvider() == null ? "未连接" : runtime.dataProvider().providerName()), Function.identity());
                case "mail" -> async(sender, () -> Json.GSON.toJson(runtime.mailbox().capabilities()), info -> "独立邮箱能力：" + info);
                case "economy" -> {
                    if(!(sender instanceof ConsoleCommandSender)&&!(sender instanceof RemoteConsoleCommandSender))throw new CoreFailure("CONSOLE_ONLY","系统资金初始化仅限受信控制台。");
                    if(args.length!=2||!args[1].equals("init-system-accounts"))throw CoreFailure.invalid("用法：/dc economy init-system-accounts");
                    if(!runtime.config().nodeId().equals(runtime.config().economyAuthority()))throw new CoreFailure("NOT_ECONOMY_AUTHORITY","请在经济权威节点执行初始化。");
                    async(sender,()->runtime.economy().initializeSystemAccounts(),result->"已核验系统账号（新账号初始余额为零）："+Json.GSON.toJson(result));
                }
                case "reconnect" -> { runtime.bridge.reconnect(); reply(sender, "已请求重新连接后端。"); }
                case "reload" -> { plugin.reloadConfig(); runtime.reload(CoreConfig.load(plugin.getConfig())); reply(sender, "可热更新设置已载入，后端连接正在重建。"); }
                default -> help(sender);
            }
        } catch (Throwable error) { reply(sender, message(error)); }
        return true;
    }
    private void save(CommandSender sender, String[] args) throws Exception {
        String ref = ref(args, 1); if (!(sender instanceof Player player)) throw CoreFailure.invalid("save 需要玩家手持物品执行。");
        String name = args.length > 2 ? String.join(" ", Arrays.copyOfRange(args, 2, args.length)) : "";
        CatalogStore.Captured captured;
        try (PlayerDataService.Lease lease = runtime.inventoryAccess().acquire(player, runtime.config().inventoryDomain(), UUID.randomUUID())) {
            lease.verifyCurrent(); captured = runtime.codec.capture(ref, player.getInventory().getItemInMainHand(), name, runtime.config());
        }
        async(sender, () -> runtime.catalog.save(captured, actor(sender)), item -> "已保存 " + item.itemRef() + " @" + item.revision() + "，手持原物品未消耗。");
    }
    private void policy(CommandSender sender, String[] args) {
        String ref = ref(args, 1); if (args.length < 3) throw CoreFailure.invalid("用法：/dc policy <物品> <服务器,服务器> [单次交付上限]");
        List<String> servers = List.of(args[2].split(",", -1)); long quantity = args.length > 3 ? Checks.number(args[3], 1, 99999) : 99999;
        async(sender, () -> runtime.catalog.save(runtime.codec.policy(runtime.catalog.find(ref, 0), servers, quantity, runtime.config()), actor(sender)), item -> "策略已保存为新版本 " + item.itemRef() + " @" + item.revision() + "；旧版本与旧订单不变。");
    }
    private void give(CommandSender sender, String[] args, boolean toOther) {
        int offset = toOther ? 2 : 1;
        Player target;
        if (toOther) { if (args.length < 3) throw CoreFailure.invalid("用法：/dc give <在线玩家> <物品> [数量] [版本]"); target = Bukkit.getPlayerExact(args[1]); }
        else if (sender instanceof Player player) target = player;
        else throw CoreFailure.invalid("控制台请使用 /dc give <玩家> <物品>。");
        if (target == null) throw new CoreFailure("PLAYER_OFFLINE", "目标玩家当前不在本服。");
        String ref = ref(args, offset); int quantity = args.length > offset + 1 ? (int) Checks.number(args[offset + 1], 1, runtime.config().maxGrantQuantity()) : 1;
        long revision = args.length > offset + 2 ? Checks.number(args[offset + 2], 1, Integer.MAX_VALUE) : 0;
        Sessions.PlayerSession session = runtime.sessions.get(target.getUniqueId()); if (session == null) throw new CoreFailure("PLAYER_SESSION_CHANGED", "玩家会话未就绪。");
        String actor = actor(sender); UUID operation = UUID.randomUUID();
        async(sender, () -> {
            ItemVersion version = runtime.catalog.find(ref, revision);
            if (quantity > version.maxQuantity()) throw CoreFailure.invalid("数量超过该物品版本的单次上限。");
            runtime.database.read(c -> { Sql.audit(c, actor, "item.give", operation.toString(), "STARTED"); return null; });
            try {
                runtime.game.call(System.currentTimeMillis() + 5000, () -> {
                    require(sender, "items.give"); Player current = runtime.sessions.require(session.playerUuid(), session.sessionEpoch());
                    ItemStack item = runtime.codec.decode(version, runtime.config());
                    try (PlayerDataService.Lease lease = runtime.inventoryAccess().acquire(current, version.inventoryDomain(), operation)) {
                        lease.verifyCurrent(); lease.verifyItems(List.of(item));
                        InventoryGrant.apply(current.getInventory(), item, quantity);
                        try { lease.saveAndConfirm(); } catch (Exception e) { throw new CoreFailure("RESULT_UNKNOWN", "物品已写入，但保存未确认；请检查背包，不会自动再次给予。", e); }
                    }
                    return null;
                }).get(5, TimeUnit.SECONDS);
                runtime.database.read(c -> { Sql.audit(c, actor, "item.give", operation.toString(), "COMPLETED"); return null; });
            } catch (Exception e) {
                try { runtime.database.read(c -> { Sql.audit(c, actor, "item.give", operation.toString(), "UNKNOWN"); return null; }); } catch (Exception ignored) { }
                throw e;
            }
            return version;
        }, item -> "已给予 " + session.gameId() + " " + quantity + " × " + item.displayName() + "（" + item.itemRef() + " @" + item.revision() + "）。");
    }
    private <T> void async(CommandSender sender, Callable<T> action, Function<T,String> completed) {
        String key = actor(sender);
        if (!busy.add(key)) throw new CoreFailure("COMMAND_BUSY", "上一条操作尚未完成，请稍候。");
        if(sender instanceof RemoteConsoleCommandSender)reply(sender,"操作已受理，完成结果将写入服务器控制台日志。");
        runtime.workers.submit(action).whenComplete((value,error) -> {
            busy.remove(key); runtime.game.run(() -> {String result=error==null?completed.apply(value):message(error);if(sender instanceof RemoteConsoleCommandSender)plugin.getLogger().info(result);else if (!(sender instanceof Player player) || player.isOnline()) reply(sender,result); });
        });
    }
    private String ref(String[] args, int index) { if (args.length <= index) throw CoreFailure.invalid("缺少物品 ID，请使用 /dc help 查看用法。"); return Checks.itemRef(args[index], runtime.config().namespace()); }
    private static void require(CommandSender sender, String permission) { if (!sender.hasPermission("deuterium.core." + permission)) throw new CoreFailure("FORBIDDEN", "没有执行此操作的权限。"); }
    private static String actor(CommandSender sender) { return sender instanceof Player player ? player.getUniqueId().toString() : "CONSOLE:" + sender.getName(); }
    private static String message(Throwable error) {
        while ((error instanceof CompletionException || error instanceof ExecutionException) && error.getCause() != null) error = error.getCause();
        return error instanceof CoreFailure known ? known.getMessage() : "操作未能确认完成，请查看 Core 状态并按原记录核对。";
    }
    private static void reply(CommandSender sender, String text) { sender.sendMessage("[Deuterium Core] " + text); }
    private static void help(CommandSender sender) {
        reply(sender, "/dc save <ID> [显示名] · 保存手持物品\n/dc get <ID> [数量] [版本]\n/dc give <玩家> <ID> [数量] [版本]\n/dc list [页码] [关键词] | info <ID> [版本] | versions <ID> [页码]\n/dc policy <ID> <服务器,服务器> [单次上限]\n/dc archive <ID> | restore <ID>\n/dc sync | status | mail | reconnect | reload");
    }
    @Override public List<String> onTabComplete(CommandSender sender, Command command, String alias, String[] args) {
        if (!sender.hasPermission("deuterium.core.items.read")) return List.of();
        if (args.length == 1) return List.of("help", "save", "get", "give", "list", "info", "versions", "policy", "archive", "restore", "sync", "status", "mail", "reconnect", "reload").stream().filter(s -> s.startsWith(args[0].toLowerCase(Locale.ROOT))).toList();
        if (args.length == 2 && args[0].equalsIgnoreCase("give")) return Bukkit.getOnlinePlayers().stream().map(Player::getName).filter(n -> n.toLowerCase(Locale.ROOT).startsWith(args[1].toLowerCase(Locale.ROOT))).toList();
        return List.of();
    }
}
