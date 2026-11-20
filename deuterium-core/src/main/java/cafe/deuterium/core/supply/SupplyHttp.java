package cafe.deuterium.core.supply;

import cafe.deuterium.core.config.CoreConfig;
import cafe.deuterium.core.util.CoreFailure;
import cafe.deuterium.core.util.Json;
import com.google.gson.JsonObject;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.util.function.Supplier;

/** Trusted-node transport. Node credentials never enter game payloads or asset requests. */
public final class SupplyHttp implements AutoCloseable {
    private final Supplier<CoreConfig> config;
    private final HttpClient client = HttpClient.newBuilder().connectTimeout(Duration.ofSeconds(3))
            .followRedirects(HttpClient.Redirect.NEVER).build();

    public SupplyHttp(Supplier<CoreConfig> config) { this.config = config; }

    public JsonObject post(String action, JsonObject input) throws Exception {
        if (!action.matches("[a-z/-]{1,80}")) throw CoreFailure.invalid("无效补给请求");
        CoreConfig settings = config.get();
        if (!settings.bridge().enabled()) throw new CoreFailure("SUPPLY_UNAVAILABLE", "补给服务尚未连接");
        URI bridge = settings.bridge().uri();
        URI base = URI.create((bridge.getScheme().equals("wss") ? "https" : "http") + "://" + bridge.getRawAuthority());
        HttpRequest request = HttpRequest.newBuilder(base.resolve("/bridge/v1/supply/" + action))
                .timeout(Duration.ofSeconds(10)).header("Authorization", "Bearer " + settings.bridge().token())
                .header("X-Deuterium-Node-ID", settings.nodeId()).header("Content-Type", "application/json")
                .POST(HttpRequest.BodyPublishers.ofString(Json.GSON.toJson(input))).build();
        HttpResponse<java.io.InputStream> response = client.send(request, HttpResponse.BodyHandlers.ofInputStream());
        byte[] bytes;
        try (var stream = response.body()) { bytes = stream.readNBytes(2_097_153); }
        if (bytes.length > 2_097_152) throw new CoreFailure("SUPPLY_RESPONSE_LIMIT", "补给目录超过传输上限，请联系管理员");
        JsonObject body = Json.object(new String(bytes, StandardCharsets.UTF_8), 2_097_152);
        if (response.statusCode() != 200) {
            JsonObject error = body.has("error") && body.get("error").isJsonObject() ? body.getAsJsonObject("error") : new JsonObject();
            String code = error.has("code") ? error.get("code").getAsString() : "RESULT_UNKNOWN";
            if (response.statusCode() >= 500) code = "RESULT_UNKNOWN";
            String message = error.has("message") ? error.get("message").getAsString() : "补给请求尚未确认，请使用原请求重试";
            throw new CoreFailure(code, message);
        }
        if (!body.has("data") || !body.get("data").isJsonObject()) throw new CoreFailure("SUPPLY_PROTOCOL", "补给服务返回无效数据");
        return body.getAsJsonObject("data");
    }

    @Override public void close() { client.shutdownNow(); }
}
