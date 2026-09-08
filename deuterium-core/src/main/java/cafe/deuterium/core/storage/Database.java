package cafe.deuterium.core.storage;

import cafe.deuterium.core.config.CoreConfig;
import cafe.deuterium.core.util.CoreFailure;
import com.zaxxer.hikari.HikariConfig;
import com.zaxxer.hikari.HikariDataSource;
import java.nio.file.Files;
import java.nio.file.Path;
import java.sql.*;

public final class Database implements AutoCloseable {
    @FunctionalInterface public interface Work<T> { T run(Connection c) throws Exception; }
    private final HikariDataSource pool;
    private final boolean mysql;
    public Database(CoreConfig.Storage config, Path folder) throws Exception {
        Files.createDirectories(folder);
        mysql = config.kind().equals("mysql");
        HikariConfig h = new HikariConfig();
        h.setPoolName("DeuteriumCore"); h.setMaximumPoolSize(mysql ? 4 : 1); h.setMinimumIdle(1);
        h.setConnectionTimeout(5000); h.setValidationTimeout(2000); h.setMaxLifetime(180000);
        h.setInitializationFailTimeout(5000); h.setAutoCommit(true);
        if (mysql) {
            String localAuthentication=java.util.Set.of("127.0.0.1","localhost","::1").contains(config.host())?"&allowPublicKeyRetrieval=true":"";
            h.setDataSource(new IsolatedMariaDataSource("jdbc:mariadb://" + config.host() + ":" + config.port() + "/" + config.database()
                    + "?connectTimeout=3000&socketTimeout=5000&tcpKeepAlive=true"+localAuthentication,config.username(),config.password()));
            h.setConnectionInitSql("SET time_zone='+00:00'");
        } else {
            h.setDriverClassName("org.sqlite.JDBC");
            h.setJdbcUrl("jdbc:sqlite:" + folder.resolve(config.sqliteFile()).toAbsolutePath());
            h.addDataSourceProperty("foreign_keys", "true"); h.addDataSourceProperty("busy_timeout", "5000");
            h.addDataSourceProperty("journal_mode", "WAL"); h.addDataSourceProperty("synchronous", "FULL");
        }
        pool = new HikariDataSource(h);
        try { Schema.ensure(this); } catch (Exception e) { pool.close(); throw e; }
    }
    public boolean shared() { return mysql; }
    public String lock() { return mysql ? " FOR UPDATE" : ""; }
    public String ignoreInsert() { return mysql ? "INSERT IGNORE" : "INSERT OR IGNORE"; }
    public <T> T read(Work<T> work) {
        try (Connection c = pool.getConnection()) { return work.run(c); }
        catch (CoreFailure e) { throw e; }
        catch (Exception e) { throw new CoreFailure("STORAGE_UNAVAILABLE", "Core 存储暂不可用。", e); }
    }
    public <T> T transaction(Work<T> work) {
        return read(c -> {
            c.setAutoCommit(false);
            try { T result = work.run(c); c.commit(); return result; }
            catch (Exception | Error e) { try { c.rollback(); } catch (Exception suppressed) { e.addSuppressed(suppressed); } throw e; }
            finally { c.setAutoCommit(true); }
        });
    }
    public boolean healthy() { try { return read(c -> c.isValid(2)); } catch (Exception e) { return false; } }
    public Connection exclusiveConnection() throws SQLException { return pool.getConnection(); }
    public int connections() { return pool.getHikariPoolMXBean().getTotalConnections(); }
    @Override public void close() { pool.close(); }
}
