package cafe.deuterium.core;

import cafe.deuterium.core.config.CoreConfig;
import cafe.deuterium.core.storage.*;
import org.bukkit.configuration.file.YamlConfiguration;
import java.io.InputStreamReader;
import java.nio.charset.StandardCharsets;
import java.nio.file.Path;
import java.util.List;

public final class StoreFixture implements AutoCloseable {
    public final CoreConfig config;
    public final Database db;
    public final OutboxStore outbox;
    public final CatalogStore catalog;
    public StoreFixture(Path folder) throws Exception {
        config = config(); db = new Database(config.storage(), folder);
        outbox = new OutboxStore(db, config.nodeId(), 10000); catalog = new CatalogStore(db, outbox);
    }
    public static YamlConfiguration yaml() {
        return YamlConfiguration.loadConfiguration(new InputStreamReader(StoreFixture.class.getResourceAsStream("/config.yml"), StandardCharsets.UTF_8));
    }
    public static CoreConfig config() { return CoreConfig.load(yaml()); }
    public static CatalogStore.Captured item(String id, byte[] bytes) {
        return new CatalogStore.Captured(id, bytes, "minecraft:stone", "测试物品", "", 64, List.of("amiya"), List.of(), "survival", "ix-main-1_21_1");
    }
    @Override public void close() { db.close(); }
}
