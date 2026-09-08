package cafe.deuterium.probe;

import cafe.deuterium.core.util.Json;
import org.bukkit.Bukkit;
import org.bukkit.OfflinePlayer;
import org.bukkit.entity.Player;
import org.bukkit.event.*;
import java.lang.reflect.*;
import java.util.*;

/** Real TrChat runtime boundary checks; only reachable through the isolated console probe. */
final class ChatProbe {
    private final CoreProbe probe;
    private final String marker = "DC_CHAT_" + UUID.randomUUID().toString().replace("-", "");
    private final List<String> observed = Collections.synchronizedList(new ArrayList<>());
    private final Listener editing = new Listener() { };
    private Class<?> eventType, channelType, sessionType, componentType, kindType;
    private Object session, normal, privateChannel, components, data;
    private Method toPlain, getComponent, makeText;
    ChatProbe(CoreProbe probe) { this.probe = probe; }

    @SuppressWarnings({"unchecked", "rawtypes"}) void run() throws Exception {
        var runtime = probe.core().runtime();
        CoreProbe.check(!runtime.config().bridge().enabled(), "chat probe requires disconnected backend");
        CoreProbe.check(runtime.config().storage().database().startsWith("dc_core_game_"), "dedicated schema required");
        CoreProbe.check("TrChat final-public API".equals(runtime.status().get("chatMode").getAsString()), "TrChat hook unavailable");
        Player player = probe.player("CoreProbePeer");
        ClassLoader loader = Objects.requireNonNull(Bukkit.getPluginManager().getPlugin("TrChat")).getClass().getClassLoader();
        eventType = Class.forName("me.arasple.mc.trchat.api.event.TrChatSendEvent", true, loader);
        channelType = Class.forName("me.arasple.mc.trchat.module.display.channel.Channel", true, loader);
        sessionType = Class.forName("me.arasple.mc.trchat.module.display.ChatSession", true, loader);
        componentType = eventType.getMethod("getComponent").getReturnType();
        kindType = eventType.getMethod("getType").getReturnType();
        Class<?> util = Class.forName("me.arasple.mc.trchat.util.BukkitUtilKt", true, loader);
        session = util.getMethod("getSession", Player.class).invoke(null, player);
        data = util.getMethod("getData", OfflinePlayer.class).invoke(null, player);
        Object companion = channelType.getField("Companion").get(null);
        Map<?, ?> channels = (Map<?, ?>) companion.getClass().getMethod("getChannels").invoke(companion);
        normal = Objects.requireNonNull(channels.get("Normal"), "Normal channel absent");
        privateChannel = Objects.requireNonNull(channels.get("Private"), "Private channel absent");
        sessionType.getMethod("setChannel", channelType).invoke(session, normal);
        components = Class.forName("me.arasple.mc.trchat.taboolib.module.chat.Components", true, loader).getField("INSTANCE").get(null);
        makeText = components.getClass().getMethod("text", String.class);
        getComponent = eventType.getMethod("getComponent");
        toPlain = componentType.getMethod("toPlainText");
        Bukkit.getPluginManager().registerEvent((Class<? extends Event>) eventType, editing, EventPriority.HIGHEST, (listener, event) -> {
            if (!eventType.isInstance(event)) return;
            try {
                String text = (String) toPlain.invoke(getComponent.invoke(event));
                if (!text.contains(marker)) return;
                observed.add(text);
                if (text.endsWith("_RAW")) eventType.getMethod("setComponent", componentType).invoke(event, makeText.invoke(components, marker + "_APPROVED"));
            } catch (Exception failure) { throw new EventException(failure); }
        }, probe, false);
        try {
            emit(normal, "COMMON", "_RAW", false);
            emit(normal, "COMMON", "_CANCELLED", true);
            emit(normal, "SENDER", "_SENDER_PRIVATE", false);
            emit(normal, "RECEIVER", "_RECEIVER_PRIVATE", false);
            emit(privateChannel, "COMMON", "_PRIVATE_CHANNEL", false);
            data.getClass().getMethod("updateShadowMuteTime", long.class).invoke(data, 60000L);
            CoreProbe.check((boolean) data.getClass().getMethod("isShadowMuted").invoke(data), "shadow mute did not activate");
            emit(normal, "COMMON", "_SHADOW", false);
        } finally {
            data.getClass().getMethod("updateShadowMuteTime", long.class).invoke(data, 0L);
        }
        // Exercise the real Bukkit -> TrChat formatting/filter pipeline as well as the final event boundary above.
        player.chat(marker + "_PIPELINE");
        Bukkit.getScheduler().runTaskLater(probe, this::verify, 100);
    }

    @SuppressWarnings({"unchecked", "rawtypes"}) private void emit(Object channel, String kind, String suffix, boolean cancelled) throws Exception {
        Object type = Enum.valueOf((Class) kindType, kind);
        Event event = (Event) eventType.getConstructor(channelType, sessionType, componentType, kindType)
                .newInstance(channel, session, makeText.invoke(components, marker + suffix), type);
        if (cancelled) ((Cancellable) event).setCancelled(true);
        Bukkit.getPluginManager().callEvent(event);
    }

    private void verify() {
        HandlerList.unregisterAll(editing);
        try {
            List<String> contents = probe.core().runtime().database.read(connection -> {
                List<String> result = new ArrayList<>();
                try (var query = connection.prepareStatement("SELECT payload FROM dc_outbox WHERE type='chat.public.event' AND payload LIKE ? ORDER BY sequence_id")) {
                    query.setString(1, "%" + marker + "%");
                    try (var rows = query.executeQuery()) { while (rows.next()) result.add(Json.object(rows.getString(1), 28000).get("content").getAsString()); }
                }
                return result;
            });
            CoreProbe.check(contents.size() == 2, "expected exactly approved event plus pipeline event: " + contents);
            CoreProbe.check(contents.stream().anyMatch(s -> s.equals(marker + "_APPROVED")), "final edited component missing");
            CoreProbe.check(contents.stream().anyMatch(s -> s.contains(marker + "_PIPELINE")), "real chat pipeline missing");
            CoreProbe.check(contents.stream().noneMatch(s -> s.matches(".*_(RAW|CANCELLED|SENDER_PRIVATE|RECEIVER_PRIVATE|PRIVATE_CHANNEL|SHADOW).*")), "private or rejected content escaped");
            probe.report("trchat", Map.of("passed", true, "marker", marker, "captured", contents, "observedFinalEvents", List.copyOf(observed), "chatMode", probe.core().runtime().status().get("chatMode").getAsString(), "privateCancelledShadowExcluded", true, "finalComponentUsed", true, "realPlayerChatPipeline", true));
        } catch (Throwable failure) {
            probe.getLogger().log(java.util.logging.Level.SEVERE, "PROBE_TRCHAT_FAILED", failure);
            probe.report("trchat", Map.of("passed", false, "marker", marker, "failure", failure.toString()));
        }
    }
}
