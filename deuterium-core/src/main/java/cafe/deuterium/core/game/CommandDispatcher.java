package cafe.deuterium.core.game;

import cafe.deuterium.core.config.CoreConfig;
import cafe.deuterium.core.mail.CoreMailbox;
import cafe.deuterium.core.runtime.GameThread;
import cafe.deuterium.core.runtime.Sessions;
import cafe.deuterium.core.storage.PlayerStore;
import cafe.deuterium.core.storage.RpcJournal;
import cafe.deuterium.core.util.Checks;
import cafe.deuterium.core.util.CoreFailure;
import cafe.deuterium.core.util.Json;
import com.google.gson.JsonObject;
import org.bukkit.Bukkit;
import net.kyori.adventure.text.Component;
import java.nio.charset.StandardCharsets;
import java.time.Instant;
import java.util.*;
import java.util.concurrent.*;
import java.util.function.Supplier;

public final class CommandDispatcher {
    private static final Set<String> TYPES = Set.of("player.resolve", "verification.deliver", "wallet.balance", "wallet.transfer", "operation.query", "mailbox.create", "mailbox.query", "mailbox.revoke","wallet.escrow.reserve","wallet.escrow.bind","wallet.escrow.settle","wallet.escrow.refund","wallet.escrow.query");
    private static final Set<String> READS = Set.of("player.resolve", "wallet.balance", "operation.query", "mailbox.query","wallet.escrow.query");
    private final Supplier<CoreConfig> config;
    private final Supplier<CoreMailbox> mailbox;
    private final GameThread game;
    private final Sessions sessions;
    private final PlayerStore players;
    private final RpcJournal journal;
    private final EconomyAccess economy;
    public CommandDispatcher(Supplier<CoreConfig> config, Supplier<CoreMailbox> mailbox, GameThread game, Sessions sessions,
                             PlayerStore players, RpcJournal journal, EconomyAccess economy) {
        this.config = config; this.mailbox = mailbox; this.game = game; this.sessions = sessions; this.players = players; this.journal = journal; this.economy = economy;
    }
    public JsonObject execute(JsonObject request) {
        String operation = ""; boolean started = false;
        try {
            Json.fields(request, "operationId", "command", "expiresAt", "payload");
            operation = Checks.operationId(Json.string(request, "operationId"));
            String type = Json.string(request, "command");
            if (!TYPES.contains(type)) throw new CoreFailure("COMMAND_NOT_ALLOWED", "不允许执行此远程操作。");
            long expiry = Instant.parse(Json.string(request, "expiresAt")).toEpochMilli();
            JsonObject payload = request.getAsJsonObject("payload"); if (payload == null) throw CoreFailure.invalid("缺少请求内容。");
            boolean read = READS.contains(type);
            boolean invalidDeadline = expiry < System.currentTimeMillis() || expiry > System.currentTimeMillis() + 120000;
            if (read && invalidDeadline) throw new CoreFailure("COMMAND_EXPIRED", "请求已过期或有效期不正确。");
            if (!read) {
                String fingerprint = Checks.sha((type + ":" + Json.canonical(payload)).getBytes(StandardCharsets.UTF_8));
                if (invalidDeadline) {
                    RpcJournal.Entry existing = journal.match(operation, type, fingerprint);
                    if (existing.result() != null && Set.of("COMPLETED", "FAILED").contains(existing.state())) return Json.object(existing.result(),32768);
                    if (!existing.state().equals("NOT_FOUND")) return result(operation,existing.state().equals("EXECUTING") ? "PROCESSING" : "UNKNOWN",null,
                            new CoreFailure("RESULT_UNKNOWN","本次投递已过期，原操作结果仍待核实；请查询原操作，不会重新执行。"));
                    throw new CoreFailure("COMMAND_EXPIRED", "请求已过期或有效期不正确。");
                }
                boolean committedEconomyReplay=type.startsWith("wallet.")&&config.get().economyEnabled()&&config.get().nodeId().equals(config.get().economyAuthority())&&economy.available();
                RpcJournal.Entry entry = journal.begin(operation, type, fingerprint, type.startsWith("mailbox.")||committedEconomyReplay);
                if (!entry.acquired()) {
                    if (entry.state().equals("COMPLETED") && type.startsWith("mailbox."))
                        return result(operation, "COMPLETED", invoke(operation, type, payload, expiry), null);
                    if (entry.result() != null && Set.of("COMPLETED","FAILED").contains(entry.state())) return Json.object(entry.result(), 32768);
                    return result(operation, entry.state().equals("EXECUTING") ? "PROCESSING" : entry.state(), null,
                            new CoreFailure("RESULT_UNKNOWN", "请查询原操作；不会自动重复执行。"));
                }
                started = true;
            }
            JsonObject output = result(operation, "COMPLETED", invoke(operation, type, payload, expiry), null);
            if (started) journal.complete(operation, "COMPLETED", Json.GSON.toJson(output));
            return output;
        } catch (Throwable failure) {
            while (failure instanceof CompletionException || failure instanceof ExecutionException) failure = failure.getCause();
            CoreFailure error = failure instanceof CoreFailure known ? known : new CoreFailure("RESULT_UNKNOWN", "操作结果无法确认，请按原标识查询。");
            String state = Set.of("RESULT_UNKNOWN","STORAGE_UNAVAILABLE","ECONOMY_UNAVAILABLE","SYSTEM_IDENTITY_UNAVAILABLE").contains(error.code()) ? "UNKNOWN" : "FAILED";
            JsonObject output = result(operation, state, null, error);
            if (started) try { journal.complete(operation, state, Json.GSON.toJson(output)); } catch (Exception ignored) { }
            return output;
        }
    }
    private JsonObject invoke(String operation, String type, JsonObject payload, long deadline) throws Exception {
        if (type.startsWith("mailbox.")) return mailbox.get().execute(operation, type, payload);
        return switch (type) {
            case "operation.query" -> {
                Json.fields(payload, "operationId");String id=Checks.operationId(Json.string(payload,"operationId"));RpcJournal.Entry entry=journal.find(id);
                if(Set.of("UNKNOWN","NOT_FOUND").contains(entry.state())) {
                    JsonObject cancellation = mailbox.get().queryCancellation(id);
                    if(cancellation != null) {
                        JsonObject reply = result(id,"COMPLETED",cancellation,null);
                        yield Json.tree(new RpcJournal.Entry(false,"COMPLETED",Json.GSON.toJson(reply)));
                    }
                }
                if(Set.of("UNKNOWN","NOT_FOUND").contains(entry.state())&&economy.available()){
                    JsonObject proof=economy.queryOperation(id);String state=proof.get("state").getAsString();
                    if(Set.of("COMPLETED","FAILED").contains(state)){
                        JsonObject committed=proof.getAsJsonObject("result");
                        CoreFailure failure=state.equals("FAILED")?new CoreFailure(committed.get("code").getAsString(),committed.get("message").getAsString()):null;
                        JsonObject reply=result(id,state,state.equals("COMPLETED")?committed:null,failure);
                        yield Json.tree(new RpcJournal.Entry(false,state,Json.GSON.toJson(reply)));
                    }
                }
                yield Json.tree(entry);
            }
            case "player.resolve" -> {
                Json.fields(payload, "gameId"); String name = Json.string(payload, "gameId");
                if (!name.matches("[A-Za-z0-9_]{1,32}")) throw CoreFailure.invalid("无效游戏名。");
                if(name.equalsIgnoreCase("DIMA")||name.equalsIgnoreCase("DaoYu"))throw new CoreFailure("SYSTEM_ACCOUNT_PROTECTED","系统账号不是普通收款人。");
                PlayerStore.PlayerIdentity known;
                try{known=players.resolve(name);}catch(CoreFailure absent){if(!absent.code().equals("PLAYER_NOT_FOUND"))throw absent;known=null;}
                PlayerStore.PlayerIdentity cached=game.call(deadline,()->{
                    // This API only reads a profile already resolved by this game
                    // server. Never use getOfflinePlayer(String), which may derive
                    // an offline UUID for an identity the server has never seen.
                    return CachedPlayers.find(name,config.get().nodeId());
                }).get(Math.max(1,deadline-System.currentTimeMillis()),TimeUnit.MILLISECONDS);
                if(known!=null&&cached!=null&&!known.playerUuid().equals(cached.playerUuid()))throw new CoreFailure("IDENTITY_AMBIGUOUS","历史目录与服务器缓存的 UUID 不一致，请人工核实该名称。");
                if(cached==null&&known==null)throw new CoreFailure("PLAYER_NOT_FOUND","服务器缓存与 Core 目录均未记录此玩家。");
                yield Json.tree(known!=null&&known.online()?known:cached!=null?cached:known);
            }
            case "verification.deliver" -> {
                Json.fields(payload, "playerUuid", "code", "purpose");
                UUID id = uuid(Json.string(payload, "playerUuid")); String code = Json.string(payload, "code"), purpose = Json.string(payload, "purpose");
                if (!code.matches("[0-9]{6}") || !Set.of("register", "password_reset").contains(purpose)) throw CoreFailure.invalid("无效验证码请求。");
                Sessions.PlayerSession session = sessions.get(id); if (session == null) throw new CoreFailure("PLAYER_OFFLINE", "玩家当前不在本服。");
                yield game.call(deadline, () -> {
                    sessions.require(id, session.sessionEpoch()).sendMessage(Component.text("[Deuterium ID] " + (purpose.equals("register") ? "注册" : "重置密码") + "验证码：" + code + "。请勿向他人提供。"));
                    return Json.tree(Map.of("delivered", true));
                }).get(Math.max(1, deadline - System.currentTimeMillis()), TimeUnit.MILLISECONDS);
            }
            case "wallet.balance", "wallet.transfer" -> {
                if (!config.get().economyEnabled()) throw new CoreFailure("ECONOMY_DISABLED", "受控经济能力尚未启用。");
                if (type.equals("wallet.balance")) {
                    Json.fields(payload, "playerUuid"); UUID player = uuid(Json.string(payload, "playerUuid"));
                    yield economy.balance(player);
                }
                // Only the configured authority accepts mutations. Backend configuration must use the same authority.
                if (!config.get().nodeId().equals(config.get().economyAuthority())) throw new CoreFailure("NOT_ECONOMY_AUTHORITY", "本节点不是受控经济执行节点。");
                Json.fields(payload, "fromUuid", "toUuid", "amount");
                UUID from = uuid(Json.string(payload, "fromUuid")), to = uuid(Json.string(payload, "toUuid")); String amount = Json.string(payload, "amount");
                yield economy.execute(operation,type,payload);
            }
            case "wallet.escrow.reserve","wallet.escrow.bind","wallet.escrow.settle","wallet.escrow.refund","wallet.escrow.query" -> {
                if(!config.get().economyEnabled()||!config.get().nodeId().equals(config.get().economyAuthority()))throw new CoreFailure("ECONOMY_DISABLED","请通过已启用的经济权威节点执行。");
                yield economy.execute(operation,type,payload);
            }
            default -> throw new CoreFailure("COMMAND_NOT_ALLOWED", "不支持的远程操作。");
        };
    }
    private static UUID uuid(String value) { UUID id = UUID.fromString(value); if (!id.toString().equals(value)) throw CoreFailure.invalid("无效 UUID。"); return id; }
    private static JsonObject result(String operation, String status, JsonObject data, CoreFailure error) {
        JsonObject result = new JsonObject(); result.addProperty("operationId", operation); result.addProperty("status", status);
        if (data != null) result.add("data", data);
        if (error != null) result.add("error", Json.tree(Map.of("code", error.code(), "message", error.getMessage())));
        return result;
    }
}
