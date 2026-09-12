package cafe.deuterium.gateway;

import com.google.gson.Gson;
import com.google.gson.GsonBuilder;
import com.google.gson.reflect.TypeToken;
import java.io.IOException;
import java.nio.ByteBuffer;
import java.nio.channels.FileChannel;
import java.nio.charset.StandardCharsets;
import java.nio.file.AtomicMoveNotSupportedException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.StandardCopyOption;
import java.nio.file.StandardOpenOption;
import java.util.Map;
import java.util.TreeMap;
import java.util.UUID;

/** Only successful connections are recorded. Files are replaced atomically, with a recoverable backup. */
final class LastServerStore {
    private static final Gson GSON = new GsonBuilder().setPrettyPrinting().create();
    private final Path file;
    private final Map<UUID, String> servers = new java.util.HashMap<>();
    private long revision;
    private long savedRevision;
    private final Object writeLock = new Object();

    LastServerStore(Path directory) throws IOException {
        Files.createDirectories(directory);
        file = directory.resolve("last-servers.json");
        Path backup = directory.resolve("last-servers.json.bak");
        if (Files.exists(file)) {
            try { load(file); }
            catch (IOException | RuntimeException badPrimary) {
                if (!Files.exists(backup)) throw new IOException("Last-server file cannot be read; preserve it and repair before restart", badPrimary);
                servers.clear();
                try { load(backup); } catch (IOException | RuntimeException badBackup) { throw new IOException("Both last-server files are invalid", badBackup); }
                // Keep the corrupt primary for investigation, then restore the confirmed backup.
                Files.copy(file, directory.resolve("last-servers.corrupt-" + System.currentTimeMillis() + ".json"));
                Files.copy(backup, file, StandardCopyOption.REPLACE_EXISTING);
            }
        } else if (Files.exists(backup)) { load(backup); revision = 1; }
    }

    private void load(Path path) throws IOException {
        if (Files.size(path) > 8 * 1024 * 1024) throw new IOException("last-server file exceeds limit");
        Map<String,String> data = GSON.fromJson(Files.readString(path), new TypeToken<Map<String,String>>() {}.getType());
        if (data == null || data.size() > 50000) throw new IOException("invalid last-server data");
        for (var entry : data.entrySet()) {
            UUID uuid = UUID.fromString(entry.getKey());
            if (!uuid.toString().equals(entry.getKey()) || !GatewayConfig.serverName(entry.getValue())) throw new IOException("invalid last-server entry");
            servers.put(uuid, entry.getValue());
        }
    }
    synchronized String get(UUID uuid) { return servers.get(uuid); }
    synchronized int size() { return servers.size(); }
    synchronized void put(UUID uuid, String server) {
        if (!GatewayConfig.serverName(server)) throw new IllegalArgumentException("invalid server name");
        if (!server.equals(servers.put(uuid, server))) revision++;
    }
    synchronized boolean dirty() { return revision != savedRevision; }

    // Called by one writer, never by the network event thread.
    void flush() throws IOException {
      synchronized (writeLock) {
        Map<String,String> snapshot = new TreeMap<>();
        long snapshotRevision;
        synchronized (this) {
            if (!dirty()) return;
            snapshotRevision = revision;
            servers.forEach((uuid,server) -> snapshot.put(uuid.toString(),server));
        }
        byte[] bytes = GSON.toJson(snapshot).getBytes(StandardCharsets.UTF_8);
        Path temporary = file.resolveSibling("last-servers.json.tmp");
        try (FileChannel channel = FileChannel.open(temporary, StandardOpenOption.CREATE, StandardOpenOption.TRUNCATE_EXISTING, StandardOpenOption.WRITE)) {
            ByteBuffer data = ByteBuffer.wrap(bytes);
            while (data.hasRemaining()) channel.write(data);
            channel.force(true);
        }
        if (Files.exists(file)) Files.copy(file,file.resolveSibling("last-servers.json.bak"),StandardCopyOption.REPLACE_EXISTING);
        try { Files.move(temporary,file,StandardCopyOption.ATOMIC_MOVE,StandardCopyOption.REPLACE_EXISTING); }
        catch (AtomicMoveNotSupportedException unsupported) { Files.move(temporary,file,StandardCopyOption.REPLACE_EXISTING); }
        synchronized (this) { savedRevision = snapshotRevision; }
      }
    }
}
