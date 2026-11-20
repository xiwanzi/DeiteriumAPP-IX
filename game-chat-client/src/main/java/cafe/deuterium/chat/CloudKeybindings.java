package cafe.deuterium.chat;

import static cafe.deuterium.chat.KeybindingProfile.*;

import com.google.gson.JsonObject;
import com.mojang.blaze3d.platform.InputConstants;

import net.minecraft.client.KeyMapping;
import net.minecraft.client.Minecraft;
import net.minecraft.client.gui.screens.options.controls.KeyBindsScreen;
import net.minecraft.client.resources.language.I18n;
import net.minecraft.network.chat.Component;
import net.neoforged.neoforge.client.settings.KeyConflictContext;
import net.neoforged.neoforge.client.settings.KeyModifier;

import java.io.IOException;
import java.nio.file.*;
import java.util.*;

/** Game-thread state. Idle ticks do not enumerate bindings, touch the disk or send requests. */
public final class CloudKeybindings {
    static final CloudKeybindings INSTANCE = new CloudKeybindings();
    private UUID accountId;
    private Path file;
    private KeybindingProfile profile;
    private Map<String, String> lastActual = Map.of();
    private Map<String, Change> sending = Map.of();
    private KeybindingApi api;
    private boolean supported,
            busy,
            syncRequested,
            seedRequired,
            applyRequested,
            savingRemote,
            firstPull;
    private long generation, nextAttempt, retryMs = 2000;
    String status = "等待连接游戏服";

    private static Minecraft mc() {
        return Minecraft.getInstance();
    }

    private static String value(KeyMapping key) {
        if (key.isUnbound()) return InputConstants.UNKNOWN.getName();
        return key.saveString()
                + (key.getKeyModifier() == KeyModifier.NONE
                        ? ""
                        : ":" + key.getKeyModifier().name());
    }

    private static void set(KeyMapping key, String saved) {
        String[] parts = saved.split(":", 2);
        key.setKeyModifierAndCode(
                parts.length == 1 ? KeyModifier.NONE : KeyModifier.valueOf(parts[1]),
                InputConstants.getKey(parts[0]));
    }

    private static String name(String text) {
        String label = I18n.get(text).replaceAll("\\p{Cntrl}", " ").strip();
        if (label.isEmpty()) label = text;
        return label.substring(
                0,
                label.offsetByCodePoints(
                        0, Math.min(192, label.codePointCount(0, label.length()))));
    }

    private static Change describe(KeyMapping k) {
        var c = k.getKeyConflictContext();
        String scope =
                c == KeyConflictContext.IN_GAME
                        ? "game"
                        : c == KeyConflictContext.GUI
                                ? "gui"
                                : c == KeyConflictContext.UNIVERSAL ? "all" : "custom";
        return new Change(value(k), null, null, name(k.getName()), name(k.getCategory()), scope);
    }

    private static Map<String, String> snapshot() {
        var values = new HashMap<String, String>();
        for (var k : mc().options.keyMappings) values.put(k.getName(), value(k));
        return values;
    }

    void prepare() {
        UUID id = mc().getUser().getProfileId();
        if (id.equals(accountId)) return;
        disconnect();
        accountId = id;
        file = mc().gameDirectory.toPath().resolve(".deuterium/keybindings").resolve(id + ".json");
        try {
            profile = KeybindingProfile.load(file);
        } catch (Exception e) {
            profile = new KeybindingProfile();
            status = "将从云端恢复个人键位";
        }
        lastActual = snapshot();
    }

    private void persist() {
        try {
            profile.save(file);
        } catch (IOException e) {
            status = "个人键位缓存保存失败";
            org.slf4j.LoggerFactory.getLogger(CloudKeybindings.class).warn(status, e);
        }
    }

    boolean allowed() {
        return profile != null && profile.allowed;
    }

    boolean canUndo() {
        return profile != null && !profile.undo.isEmpty();
    }

    void toggle() {
        prepare();
        profile.allowed = !profile.allowed;
        persist();
        applyRequested = syncRequested = true;
        nextAttempt = 0;
    }

    void undo() {
        prepare();
        boolean reverted = false;
        for (var k : mc().options.keyMappings) {
            String id = k.getName(), old = profile.undo.get(id);
            Row row = profile.rows.get(id);
            if (old == null || row == null || !value(k).equals(row.value())) continue;
            set(k, old);
            reverted = true;
        }
        if (reverted) {
            KeyMapping.resetMapping();
            mc().options.save();
        }
        profile.undo.clear();
        persist();
    }

    public static void optionsSaved() {
        INSTANCE.saved();
    }

    private void saved() {
        if (savingRemote
                || profile == null
                || mc().options == null
                || !mc().getUser().getProfileId().equals(accountId)) return;
        var actual = new HashMap<String, String>();
        boolean changed = false;
        for (var k : mc().options.keyMappings) {
            String id = k.getName(), from = lastActual.get(id), to = value(k);
            actual.put(id, to);
            if (from == null || from.equals(to)) continue;
            var inFlight = sending.get(id);
            if (inFlight != null) {
                profile.pending.put(id, Change.edit(inFlight.value(), to));
            } else if (!profile.rows.containsKey(id)) {
                profile.pending.put(id, describe(k));
            } else {
                profile.edit(id, from, to);
            }
            changed = true;
        }
        lastActual = actual;
        if (changed) {
            persist();
            syncRequested = true;
            nextAttempt = 0;
        }
    }

    void advertised() {
        prepare();
        if (!supported) {
            supported = true;
            syncRequested = true;
            nextAttempt = 0;
        }
    }

    void session(JsonObject p) {
        if (!supported) return;
        api = KeybindingApi.from(p, accountId);
        syncRequested = true;
        nextAttempt = 0;
        retryMs = 2000;
    }

    void wake() {
        if (supported) {
            syncRequested = true;
            nextAttempt = 0;
        }
    }

    void disconnect() {
        generation++;
        api = null;
        supported = false;
        busy = syncRequested = seedRequired = applyRequested = false;
        sending = Map.of();
        nextAttempt = 0;
        retryMs = 2000;
        firstPull = true;
        status = "等待连接游戏服";
    }

    void tick() {
        if (accountId == null && mc().options != null) prepare();
        if (!supported || busy || (api != null && !syncRequested && !applyRequested)) return;
        if (mc().getConnection() == null || mc().level == null) return;
        long now = System.currentTimeMillis();
        if (now < nextAttempt) return;
        if (api == null) {
            nextAttempt = now + Math.max(5000, retryMs);
            retryMs = Math.min(60000, retryMs * 2);
            ChatClient.INSTANCE.packet("keybindings-hello", new JsonObject());
            return;
        }
        if (applyRequested && !(mc().screen instanceof KeyBindsScreen)) applyRows();
        if (!syncRequested || now < nextAttempt) return;
        syncRequested = false;
        busy = true;
        long ticket = generation;
        var changes = new HashMap<String, Change>();
        if (seedRequired) {
            for (var k : mc().options.keyMappings) changes.put(k.getName(), describe(k));
        } else changes.putAll(profile.pending);
        sending = changes;
        boolean seeding = seedRequired;
        var availability = seeding ? Map.<String, Boolean>of() : availabilityChanges();
        var request =
                api.sync(
                        seeding ? 0 : profile.revision,
                        profile.allowed,
                        seeding,
                        changes,
                        availability);
        request.whenComplete(
                (reply, error) ->
                        mc().execute(() -> completed(ticket, seeding, changes, reply, error)));
    }

    private Map<String, Boolean> availabilityChanges() {
        var changes = new HashMap<String, Boolean>();
        profile.rows.forEach(
                (id, row) -> {
                    boolean present = lastActual.containsKey(id);
                    if (row.missing() == present) changes.put(id, present);
                });
        return changes;
    }

    private void completed(
            long ticket,
            boolean seeding,
            Map<String, Change> sent,
            KeybindingApi.Reply reply,
            Throwable error) {
        if (ticket != generation) return;
        busy = false;
        sending = Map.of();
        if (error != null) {
            if (error instanceof KeybindingApi.Expired
                    || error.getCause() instanceof KeybindingApi.Expired) api = null;
            status = "连接中断，稍后重试";
            syncRequested = true;
            nextAttempt = System.currentTimeMillis() + retryMs;
            retryMs = Math.min(60000, retryMs * 2);
            return;
        }
        retryMs = 2000;
        if (reply.revision() == 0) {
            seedRequired = true;
            syncRequested = true;
            return;
        }
        seedRequired = false;
        boolean changed = profile.revision != reply.revision();
        if (!seeding) profile.delivered(sent);
        if (reply.full() || reply.seeded()) profile.rows.clear();
        if (reply.seeded()) {
            for (var entry : sent.entrySet()) {
                String id = entry.getKey();
                Change c = entry.getValue();
                profile.rows.put(
                        id,
                        new Row(
                                c.label(),
                                c.category(),
                                c.context(),
                                c.value(),
                                c.value(),
                                false,
                                false,
                                reply.revision()));
                // Leave edits that occurred while the initial upload was in flight in the outbox.
                Change pending = profile.pending.get(id);
                if (pending != null && c.value().equals(pending.value()))
                    profile.pending.remove(id);
            }
        } else {
            profile.rows.putAll(reply.rows());
        }
        profile.revision = reply.revision();
        if (firstPull || reply.full() || reply.seeded()) {
            for (var k : mc().options.keyMappings) {
                if (!profile.rows.containsKey(k.getName()))
                    profile.pending.putIfAbsent(k.getName(), describe(k));
            }
            if (!availabilityChanges().isEmpty()) syncRequested = true;
        }
        applyRequested |= firstPull || changed || !reply.rows().isEmpty();
        firstPull = false;
        if (changed || !sent.isEmpty() || !profile.pending.isEmpty()) persist();
        syncRequested |= !profile.pending.isEmpty();
        status = "已连接 · " + profile.rows.size() + " 个动作";
    }

    private void applyRows() {
        applyRequested = false;
        var acknowledgements = new HashMap<String, Change>();
        var undo = new HashMap<String, String>();
        boolean modified = false;
        for (var k : mc().options.keyMappings) {
            String id = k.getName();
            Row row = profile.rows.get(id);
            if (row == null || profile.pending.containsKey(id)) continue;
            boolean remote =
                    !row.conflict() && !row.value().equals(row.observed()) && profile.allowed;
            String target = remote ? row.value() : row.observed();
            String old = value(k);
            try {
                if (!target.equals(old)
                        || (k.isUnbound() && k.getKeyModifier() != KeyModifier.NONE)) {
                    set(k, target);
                    modified = true;
                    if (remote) undo.put(id, old);
                }
                if (remote && target.equals(value(k)))
                    acknowledgements.put(id, Change.acknowledge(row));
            } catch (IllegalArgumentException invalid) {
                status = "此客户端不支持该键位";
            }
        }
        if (!modified && acknowledgements.isEmpty()) return;
        savingRemote = true;
        try {
            KeyMapping.resetMapping();
            mc().options.save();
        } finally {
            savingRemote = false;
        }
        lastActual = snapshot();
        if (!undo.isEmpty()) {
            profile.undo = undo;
            mc().gui
                    .setOverlayMessage(
                            Component.literal("已调整 " + undo.size() + " 个键位，可在「按键绑定 → 云键位」撤销"),
                            false);
        }
        // Options.save logs IO failures internally. Only acknowledge values actually persisted.
        if (!acknowledgements.isEmpty()) {
            try (var lines = Files.lines(mc().gameDirectory.toPath().resolve("options.txt"))) {
                var verified = new HashSet<String>();
                lines.filter(l -> l.startsWith("key_"))
                        .forEach(
                                l -> {
                                    int colon = l.indexOf(':');
                                    if (colon <= 4) return;
                                    String id = l.substring(4, colon);
                                    Change ack = acknowledgements.get(id);
                                    if (ack != null && ack.value().equals(l.substring(colon + 1)))
                                        verified.add(id);
                                });
                for (String id : verified) profile.pending.put(id, acknowledgements.get(id));
                if (verified.size() < acknowledgements.size()) retrySave();
            } catch (IOException e) {
                retrySave();
            }
        }
        persist();
        syncRequested = syncRequested || !profile.pending.isEmpty();
    }

    private void retrySave() {
        status = "键位尚未写入磁盘，将重试";
        applyRequested = true;
        nextAttempt = System.currentTimeMillis() + 10000;
    }
}
