package cafe.deuterium.core.items;

import cafe.deuterium.core.api.ItemSummary;
import cafe.deuterium.core.api.ItemVersion;
import cafe.deuterium.core.compat.EditorItemStack;
import cafe.deuterium.core.config.CoreConfig;
import cafe.deuterium.core.runtime.GameThread;
import cafe.deuterium.core.storage.CatalogStore;
import cafe.deuterium.core.util.Checks;
import cafe.deuterium.core.util.CoreFailure;
import cafe.deuterium.core.util.Json;
import com.google.gson.JsonObject;

import java.nio.charset.StandardCharsets;
import java.util.Map;
import java.util.concurrent.TimeUnit;
import java.util.function.Supplier;

/** Registers the gacha editor's item selections without a player inventory. */
public final class EditorItemCatalog {
    private static final String ACTOR = "gacha-editor";
    private final Supplier<CoreConfig> config;
    private final GameThread game;
    private final ItemCodec codec;
    private final CatalogStore catalog;

    public EditorItemCatalog(Supplier<CoreConfig> config, GameThread game, ItemCodec codec, CatalogStore catalog) {
        this.config = config;
        this.game = game;
        this.codec = codec;
        this.catalog = catalog;
    }

    public JsonObject register(JsonObject payload, long deadline) throws Exception {
        Json.fields(payload, "itemId", "stackData", "name", "catalogVersion");
        String itemId = Json.string(payload, "itemId");
        String stackData = Json.string(payload, "stackData");
        String name = Json.string(payload, "name");
        long expected = Json.integer(payload, "catalogVersion");
        if (!itemId.matches("[a-z0-9_.-]+:[a-z0-9_./-]+") || itemId.length() > 256
                || stackData.length() > 26000 || stackData.indexOf(0) >= 0 || expected < 0)
            throw CoreFailure.invalid("物品入库参数无效。");
        Checks.text(name, 512, false);
        var cfg = config.get();
        String ref = reference(cfg.namespace(), itemId, stackData, name);
        ItemSummary head = catalog.findHead(ref);
        ItemVersion item;
        if (head == null) {
            if (expected != 0) throw CoreFailure.invalid("物品目录状态已变化，请重试。");
            var captured = game.call(deadline, () -> {
                var stack = EditorItemStack.read(itemId, stackData, cfg.maxItemBytes());
                var candidate = codec.capture(ref, stack, name, cfg);
                if (!itemId.equals(candidate.itemId())) throw CoreFailure.invalid("游戏服返回了不同的物品 ID。");
                return candidate;
            }).get(Math.max(1, deadline - System.currentTimeMillis()), TimeUnit.MILLISECONDS);
            item = catalog.create(captured, ACTOR);
            head = catalog.findHead(ref);
        } else {
            item = catalog.find(ref, head.latestRevision());
        }
        if (head.archived()) head = catalog.archive(ref, false, ACTOR, expected);
        return Json.tree(Map.of("item", ItemMetadata.of(item), "head", head));
    }

    public JsonObject archive(JsonObject payload) {
        Json.fields(payload, "itemRef", "catalogVersion");
        String ref = Json.string(payload, "itemRef");
        long expected = Json.integer(payload, "catalogVersion");
        if (!ref.startsWith(config.get().namespace() + ":") || !isEditorReference(ref) || expected < 1)
            throw CoreFailure.invalid("请选择抽卡编辑器登记的物品。");
        return Json.tree(Map.of("head", catalog.archive(ref, true, ACTOR, expected)));
    }

    public static boolean isEditorReference(String ref) {
        return ref.matches("[a-z][a-z0-9_-]{0,31}:gacha_[a-f0-9]{32}");
    }

    static String reference(String namespace, String itemId, String stackData, String name) {
        String input = itemId + '\0' + stackData + '\0' + name;
        return namespace + ":gacha_" + Checks.sha(input.getBytes(StandardCharsets.UTF_8)).substring(0, 32);
    }
}
