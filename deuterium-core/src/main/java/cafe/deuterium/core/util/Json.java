package cafe.deuterium.core.util;

import com.google.gson.*;
import com.google.gson.stream.JsonReader;
import com.google.gson.stream.JsonToken;
import java.io.StringReader;
import java.math.BigDecimal;
import java.util.Set;

/** Bounded, strict JSON for the remote trust boundary; rejects duplicate property names. */
public final class Json {
    public static final Gson GSON = new GsonBuilder().disableHtmlEscaping().create();
    private Json() { }
    public static JsonObject object(String source, int maxCharacters) {
        if (source == null || source.length() > maxCharacters) throw CoreFailure.invalid("消息过长。");
        try (JsonReader reader = new JsonReader(new StringReader(source))) {
            reader.setStrictness(Strictness.STRICT);
            JsonElement result = read(reader, 0);
            if (!result.isJsonObject() || reader.peek() != JsonToken.END_DOCUMENT) throw CoreFailure.invalid("JSON 格式不正确。");
            return result.getAsJsonObject();
        } catch (CoreFailure e) { throw e; }
        catch (Exception e) { throw CoreFailure.invalid("JSON 格式不正确。"); }
    }
    private static JsonElement read(JsonReader reader, int depth) throws Exception {
        if (depth > 24) throw CoreFailure.invalid("JSON 嵌套过深。");
        return switch (reader.peek()) {
            case BEGIN_OBJECT -> {
                JsonObject value = new JsonObject(); reader.beginObject();
                while (reader.hasNext()) {
                    String key = reader.nextName();
                    if (value.has(key) || value.size() >= 256) throw CoreFailure.invalid("重复字段或字段过多。");
                    value.add(key, read(reader, depth + 1));
                }
                reader.endObject(); yield value;
            }
            case BEGIN_ARRAY -> {
                JsonArray value = new JsonArray(); reader.beginArray();
                while (reader.hasNext()) { if (value.size() >= 4096) throw CoreFailure.invalid("数组过长。"); value.add(read(reader, depth + 1)); }
                reader.endArray(); yield value;
            }
            case STRING -> new JsonPrimitive(reader.nextString());
            case NUMBER -> new JsonPrimitive(new BigDecimal(reader.nextString()));
            case BOOLEAN -> new JsonPrimitive(reader.nextBoolean());
            case NULL -> { reader.nextNull(); yield JsonNull.INSTANCE; }
            default -> throw CoreFailure.invalid("无效 JSON。");
        };
    }
    public static void fields(JsonObject value, String... permitted) {
        Set<String> allowed = Set.of(permitted);
        if (!allowed.containsAll(value.keySet())) throw CoreFailure.invalid("请求包含未知字段。");
    }
    public static String string(JsonObject value, String key) {
        JsonElement item = value.get(key);
        if (item == null || !item.isJsonPrimitive() || !item.getAsJsonPrimitive().isString())
            throw CoreFailure.invalid("缺少字符串字段 " + key + "。");
        return item.getAsString();
    }
    public static long integer(JsonObject value, String key) {
        try { return value.get(key).getAsBigDecimal().longValueExact(); }
        catch (Exception e) { throw CoreFailure.invalid("缺少或无效整数字段 " + key + "。"); }
    }
    public static JsonObject tree(Object value) { return GSON.toJsonTree(value).getAsJsonObject(); }
    public static String canonical(JsonElement value) { return GSON.toJson(sorted(value)); }
    private static JsonElement sorted(JsonElement value) {
        if (value.isJsonObject()) {
            JsonObject result = new JsonObject();
            value.getAsJsonObject().entrySet().stream().sorted(java.util.Map.Entry.comparingByKey())
                    .forEach(entry -> result.add(entry.getKey(), sorted(entry.getValue())));
            return result;
        }
        if (value.isJsonArray()) { JsonArray result = new JsonArray(); value.getAsJsonArray().forEach(item -> result.add(sorted(item))); return result; }
        return value;
    }
}
