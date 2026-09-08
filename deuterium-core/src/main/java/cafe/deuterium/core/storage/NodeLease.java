package cafe.deuterium.core.storage;

import cafe.deuterium.core.util.Checks;
import java.nio.channels.FileChannel;
import java.nio.channels.FileLock;
import java.nio.charset.StandardCharsets;
import java.nio.file.Path;
import java.nio.file.StandardOpenOption;
import java.sql.Connection;
import java.sql.ResultSet;

/** Prevent two live Core instances from claiming the same persisted node identity. */
public final class NodeLease implements AutoCloseable {
    private Connection connection;
    private FileChannel file;
    private FileLock fileLock;
    private String key;
    public NodeLease(Database db, Path folder, String cluster, String node, String sqliteFile) throws Exception {
        if (db.shared()) {
            connection = db.exclusiveConnection();
            key = "dc:" + Checks.sha((cluster + ":" + node).getBytes(StandardCharsets.UTF_8)).substring(0, 48);
            try (var s = Sql.statement(connection, "SELECT GET_LOCK(?,0)", key); ResultSet r = s.executeQuery()) {
                if (!r.next() || r.getInt(1) != 1) throw new IllegalStateException("Core node already active");
            } catch (Exception error) { connection.close(); connection = null; throw error; }
        } else {
            file = FileChannel.open(folder.resolve(sqliteFile + ".lock"), StandardOpenOption.CREATE, StandardOpenOption.WRITE);
            try { fileLock = file.tryLock(); if (fileLock == null) throw new IllegalStateException("Core database already active"); }
            catch (Exception error) { file.close(); file = null; throw error; }
        }
    }
    public boolean valid() { try { return connection != null ? connection.isValid(2) : fileLock != null && fileLock.isValid(); } catch (Exception error) { return false; } }
    @Override public void close() {
        if (connection != null) {
            try (var s = Sql.statement(connection, "SELECT RELEASE_LOCK(?)", key)) { s.execute(); } catch (Exception ignored) { }
            try { connection.close(); } catch (Exception ignored) { } connection = null;
        }
        if (fileLock != null) try { fileLock.release(); } catch (Exception ignored) { }
        if (file != null) try { file.close(); } catch (Exception ignored) { }
    }
}
