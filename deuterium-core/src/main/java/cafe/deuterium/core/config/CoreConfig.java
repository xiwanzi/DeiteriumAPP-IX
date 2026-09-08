package cafe.deuterium.core.config;

import cafe.deuterium.core.util.Checks;
import cafe.deuterium.core.util.CoreFailure;
import org.bukkit.configuration.ConfigurationSection;
import java.net.URI;
import java.util.*;

public record CoreConfig(String nodeId, String clusterId, String namespace, String inventoryDomain,
                         String compatibilityProfile, Storage storage, Bridge bridge, int queueLimit,
                         int maxItemBytes, int maxGrantQuantity, List<String> defaultServers,
                         Map<String, Node> nodes, Set<String> publicChannels,
                         String chatFormat, String syncMode, String mailConsumer, boolean economyEnabled, String economyAuthority) {
    public record Storage(String kind, String host, int port, String database, String username, String password, String sqliteFile) {
        @Override public String toString() { return "Storage[" + kind + "]"; }
    }
    public record Bridge(boolean enabled, URI uri, String token, boolean development) {
        @Override public String toString() { return "Bridge[enabled=" + enabled + "]"; }
    }
    public record Node(String id, String inventoryDomain, String compatibilityProfile, boolean claimEnabled) { }
    public static CoreConfig load(ConfigurationSection c) {
        String node = Checks.node(c.getString("node.id", "amiya"));
        String cluster = Checks.node(c.getString("node.cluster", "deuterium-production"));
        String namespace = Checks.node(c.getString("items.namespace", "deuterium"));
        String domain = Checks.node(c.getString("node.inventory-domain", "survival"));
        String profile = Checks.node(c.getString("node.compatibility-profile", "unverified"));
        String kind = c.getString("storage.kind", "sqlite").toLowerCase(Locale.ROOT);
        if (!Set.of("sqlite", "mysql").contains(kind)) throw CoreFailure.invalid("storage.kind 必须是 sqlite 或 mysql。");
        String database = Checks.node(c.getString("storage.mysql.database", "deuterium_core"));
        String sqlite = c.getString("storage.sqlite-file", "core.db");
        if (!sqlite.matches("[a-zA-Z0-9_-]+\\.db")) throw CoreFailure.invalid("SQLite 文件名不允许路径。");
        Storage storage = new Storage(kind, c.getString("storage.mysql.host", "127.0.0.1"),
                c.getInt("storage.mysql.port", 3306), database, c.getString("storage.mysql.username", "deuterium_core"),
                envOrValue(c, "storage.mysql.password-env", "storage.mysql.password"), sqlite);
        if (storage.port() < 1 || storage.port() > 65535 || storage.host().contains("/") || storage.host().contains("?"))
            throw CoreFailure.invalid("无效数据库地址。");
        URI uri;
        try { uri = URI.create(c.getString("bridge.url", "wss://api.deuteriumix.com/bridge/v1/connect")); }
        catch (Exception e) { throw CoreFailure.invalid("无效桥接地址。"); }
        boolean development = c.getBoolean("bridge.development", false);
        if (uri.getHost() == null || uri.getUserInfo() != null || uri.getQuery() != null || uri.getFragment() != null
                || !"/bridge/v1/connect".equals(uri.getPath())
                || !("wss".equals(uri.getScheme()) || (development && "ws".equals(uri.getScheme())
                && Set.of("127.0.0.1", "[::1]", "::1").contains(uri.getHost())))) throw CoreFailure.invalid("桥接必须使用 WSS；仅显式回环开发模式允许 WS。");
        boolean enabled = c.getBoolean("bridge.enabled", false);
        String token = envOrValue(c, "bridge.token-env", "bridge.token");
        if (enabled && (token.length() < 32 || token.length() > 256 || token.chars().anyMatch(Character::isWhitespace)))
            throw CoreFailure.invalid("启用桥接前请设置本节点独立密钥。");
        Map<String, Node> nodes = new LinkedHashMap<>();
        ConfigurationSection section = c.getConfigurationSection("servers");
        if (section != null) for (String id : section.getKeys(false)) {
            Checks.node(id); ConfigurationSection n = section.getConfigurationSection(id);
            if (n == null) throw CoreFailure.invalid("无效服务器配置。");
            nodes.put(id, new Node(id, Checks.node(n.getString("inventory-domain", "survival")),
                    Checks.node(n.getString("compatibility-profile", "unverified")), n.getBoolean("claim-enabled", false)));
        }
        if (!nodes.containsKey(node) || nodes.size() > 32) throw CoreFailure.invalid("servers 必须包含当前节点，最多 32 个。");
        if (!nodes.get(node).inventoryDomain().equals(domain) || !nodes.get(node).compatibilityProfile().equals(profile))
            throw CoreFailure.invalid("node 与 servers 中的背包域和兼容配置不一致。");
        List<String> defaults = c.getStringList("items.default-servers");
        if (defaults.isEmpty()) defaults = List.of(node);
        defaults = servers(defaults, nodes, domain, profile, false);
        int queue = c.getInt("bridge.max-pending-events", 10000);
        int bytes = c.getInt("items.max-bytes", 262144);
        int quantity = c.getInt("items.max-grant-quantity", 2304);
        if (queue < 100 || queue > 100000 || bytes < 1024 || bytes > 1048576 || quantity < 1 || quantity > 65536)
            throw CoreFailure.invalid("容量配置超出允许范围。");
        String sync = c.getString("sync.mode", "provider");
        if (!Set.of("provider", "standalone").contains(sync)) throw CoreFailure.invalid("sync.mode 必须是 provider 或 standalone。");
        if (sync.equals("standalone") && nodes.size() != 1) throw CoreFailure.invalid("standalone 模式必须只配置一个节点。");
        if (c.getBoolean("economy.enabled", false) && !nodes.containsKey(c.getString("economy.authority-node", "amiya")))
            throw CoreFailure.invalid("受控经济节点必须在 servers 中登记。");
        Set<String> channels = Set.copyOf(c.getStringList("chat.public-channels"));
        return new CoreConfig(node, cluster, namespace, domain, profile, storage,
                new Bridge(enabled, uri, token, development), queue, bytes, quantity, defaults, Map.copyOf(nodes), channels,
                c.getString("chat.app-format", "[App] {player}: {message}"),
                sync, c.getString("mail.consumer-id", "deuterium-backend"), c.getBoolean("economy.enabled", false),
                Checks.node(c.getString("economy.authority-node", "amiya")));
    }
    private static String envOrValue(ConfigurationSection c, String envKey, String valueKey) {
        String name = c.getString(envKey, "");
        if (name != null && !name.isBlank()) return Objects.requireNonNullElse(System.getenv(name), "");
        return c.getString(valueKey, "");
    }
    public static List<String> servers(List<String> ids, Map<String, Node> nodes, String domain, String profile, boolean requireClaim) {
        if (ids.isEmpty() || new HashSet<>(ids).size() != ids.size()) throw CoreFailure.invalid("服务器范围为空或重复。");
        for (String id : ids) {
            Node node = nodes.get(id);
            if (node == null || !node.inventoryDomain().equals(domain) || !node.compatibilityProfile().equals(profile)
                    || (requireClaim && !node.claimEnabled())) throw CoreFailure.invalid("服务器 " + id + " 的背包域、兼容配置或领取策略不满足要求。");
        }
        return ids.stream().sorted().toList();
    }
}
