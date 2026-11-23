package cafe.deuterium.core.supply;

import cafe.deuterium.core.CoreRuntime;
import cafe.deuterium.core.util.CoreFailure;
import cafe.deuterium.core.util.Json;
import com.google.gson.JsonObject;
import java.util.Arrays;
import java.util.List;
import java.util.Locale;
import java.util.UUID;
import java.util.concurrent.ConcurrentHashMap;
import org.bukkit.Bukkit;
import org.bukkit.command.Command;
import org.bukkit.command.CommandExecutor;
import org.bukkit.command.CommandSender;
import org.bukkit.command.TabCompleter;
import org.bukkit.entity.Player;

/** OP-only ticket account operations; these use the same durable ledger as the Web console. */
public final class SupplyTicketCommand implements CommandExecutor, TabCompleter {
    private final CoreRuntime runtime;
    private final SupplyHttp http;
    private final java.util.Set<String> busy = ConcurrentHashMap.newKeySet();

    public SupplyTicketCommand(CoreRuntime runtime, SupplyHttp http) { this.runtime = runtime; this.http = http; }

    @Override public boolean onCommand(CommandSender sender, Command command, String label, String[] args) {
        if (!sender.isOp()) { sender.sendMessage("[补给] 仅 OP 可以管理票券"); return true; }
        if (args.length < 2) { sender.sendMessage("/supplytickets balance|history <UUID/在线玩家>；add|take|remove|set <UUID/在线玩家> <券种 ID> <数量> <原因>"); return true; }
        String actor = sender instanceof Player p ? p.getUniqueId().toString() : "CONSOLE";
        if (!busy.add(actor)) { sender.sendMessage("[补给] 上一条请求仍在处理"); return true; }
        try {
            UUID target;
            Player online = Bukkit.getPlayerExact(args[1]);
            try { target = online != null ? online.getUniqueId() : UUID.fromString(args[1]); }
            catch (IllegalArgumentException invalid) { throw CoreFailure.invalid("离线玩家请填写真实 UUID；不会从游戏名猜测 UUID"); }
            JsonObject body = new JsonObject(); body.addProperty("operator", true); body.addProperty("playerUuid", target.toString());
            if (sender instanceof Player player) {
                var session = runtime.sessions.get(player.getUniqueId()); if (session == null) throw CoreFailure.invalid("玩家会话尚未就绪");
                body.addProperty("operatorUuid", player.getUniqueId().toString()); body.addProperty("sessionEpoch", session.sessionEpoch());
            } else body.addProperty("console", true);
            String action = args[0].toLowerCase(Locale.ROOT);
            if (action.equals("history")) body.addProperty("history", true);
            else if (!action.equals("balance")) {
                if (!List.of("add", "take", "remove", "set").contains(action) || args.length < 5) throw CoreFailure.invalid("请填写券种、数量和调整原因");
                String currency = args[2];
                if (currency.equalsIgnoreCase("STANDARD") || currency.equalsIgnoreCase("LIMITED")) currency = currency.toUpperCase(Locale.ROOT);
                long amount = Long.parseLong(args[3]);
                if (!currency.matches("[A-Za-z0-9][A-Za-z0-9_-]{0,79}") || amount < 0 || amount > 1_000_000_000L) throw CoreFailure.invalid("券种或数量不正确");
                JsonObject change = new JsonObject(); change.addProperty("clientRequestId", UUID.randomUUID().toString());
                change.addProperty("playerUuid", target.toString()); change.addProperty("currency", currency); change.addProperty("action", action);
                change.addProperty("amount", amount); change.addProperty("reason", String.join(" ", Arrays.copyOfRange(args, 4, args.length)));
                body.add("adjustment", change);
            }
            runtime.workers.submit(() -> http.post("tickets", body)).whenComplete((result, failure) -> {
                busy.remove(actor);
                runtime.game.run(() -> {
                    if (failure == null) sender.sendMessage("[补给] " + Json.GSON.toJson(result));
                    else {
                        String request = body.has("adjustment") ? Json.string(body.getAsJsonObject("adjustment"), "clientRequestId") : "查询";
                        sender.sendMessage("[补给] 请求尚未确认，请按流水核对后再操作。请求编号：" + request);
                    }
                });
            });
        } catch (RuntimeException failure) { busy.remove(actor); sender.sendMessage("[补给] " + (failure instanceof CoreFailure ? failure.getMessage() : "命令参数不正确")); }
        return true;
    }

    @Override public List<String> onTabComplete(CommandSender sender, Command command, String alias, String[] args) {
        if (!sender.isOp()) return List.of();
        if (args.length == 1) return List.of("balance", "history", "add", "take", "remove", "set").stream().filter(v -> v.startsWith(args[0])).toList();
        if (args.length == 3) return List.of("standard", "limited");
        return List.of();
    }
}
