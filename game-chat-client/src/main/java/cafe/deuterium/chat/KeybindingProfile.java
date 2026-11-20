package cafe.deuterium.chat;

import com.google.gson.Gson;

import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.*;
import java.util.*;

final class KeybindingProfile {
    static final Gson JSON = new Gson();

    record Row(
            String label,
            String category,
            String context,
            String value,
            String observed,
            boolean conflict,
            boolean missing,
            long revision) {}

    record Change(
            String value, String before, Long ack, String label, String category, String context) {
        static Change edit(String before, String value) {
            return new Change(value, before, null, null, null, null);
        }

        static Change acknowledge(Row row) {
            return new Change(row.value(), null, row.revision(), null, null, null);
        }
    }

    long revision;
    boolean allowed;
    Map<String, Row> rows = new HashMap<>();
    Map<String, Change> pending = new HashMap<>();
    Map<String, String> undo = new HashMap<>();

    void edit(String action, String from, String to) {
        Change queued = pending.get(action);
        if (queued != null && queued.label() != null) {
            pending.put(
                    action,
                    new Change(
                            to, null, null, queued.label(), queued.category(), queued.context()));
            return;
        }
        String before = queued != null && queued.before() != null ? queued.before() : from;
        if (before.equals(to)) pending.remove(action);
        else pending.put(action, Change.edit(before, to));
    }

    void delivered(Map<String, Change> sent) {
        sent.forEach(
                (id, change) -> {
                    if (pending.remove(id, change)) return;
                    // A second edit made while the first request was in flight starts at the
                    // delivered value.
                    var remaining = pending.get(id);
                    if (remaining != null
                            && remaining.ack() == null
                            && (Objects.equals(remaining.before(), change.before())
                                    || change.label() != null)) {
                        pending.put(id, Change.edit(change.value(), remaining.value()));
                    }
                });
    }

    static KeybindingProfile load(Path file) throws IOException {
        if (!Files.exists(file)) return new KeybindingProfile();
        if (Files.size(file) > (8L << 20)) throw new IOException("键位缓存过大");
        var p = JSON.fromJson(Files.readString(file), KeybindingProfile.class);
        if (p == null || p.revision < 0 || p.rows == null || p.pending == null || p.undo == null)
            throw new IOException("键位缓存不完整");
        return p;
    }

    void save(Path file) throws IOException {
        Files.createDirectories(file.getParent());
        Path tmp = Files.createTempFile(file.getParent(), "keys-", ".tmp");
        try {
            Files.writeString(tmp, JSON.toJson(this), StandardCharsets.UTF_8);
            try {
                Files.move(
                        tmp,
                        file,
                        StandardCopyOption.ATOMIC_MOVE,
                        StandardCopyOption.REPLACE_EXISTING);
            } catch (AtomicMoveNotSupportedException e) {
                Files.move(tmp, file, StandardCopyOption.REPLACE_EXISTING);
            }
        } finally {
            Files.deleteIfExists(tmp);
        }
    }
}
