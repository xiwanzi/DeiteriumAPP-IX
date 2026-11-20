package cafe.deuterium.chat;

import com.google.gson.*;

import java.io.IOException;
import java.net.URI;
import java.net.http.*;
import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.util.*;
import java.util.concurrent.CompletableFuture;

record KeybindingApi(URI endpoint, String token, UUID player) {
    private static final Set<String> HOSTS =
            Set.of("api.deuteriumix.com", "chat.deuteriumix.com", "47.103.99.34");

    record Reply(
            UUID playerUuid,
            long revision,
            boolean full,
            boolean seeded,
            Map<String, KeybindingProfile.Row> rows) {}

    static KeybindingApi from(JsonObject message, UUID player) {
        if (!player.toString().equals(ChatMessage.str(message, "playerUuid")))
            throw new IllegalArgumentException("键位身份不匹配");
        URI origin = URI.create(ChatMessage.str(message, "apiBase"));
        if (!trusted(origin)) throw new IllegalArgumentException("未知的键位服务");
        String token = ChatMessage.str(message, "token");
        if (!token.matches("[0-9a-f]{64}")) throw new IllegalArgumentException("键位会话无效");
        return new KeybindingApi(origin.resolve("/api/v1/game/keybindings/sync"), token, player);
    }

    static boolean trusted(URI origin) {
        return "https".equals(origin.getScheme())
                && origin.getHost() != null
                && HOSTS.contains(origin.getHost())
                && (origin.getPort() == -1 || origin.getPort() == 443)
                && origin.getUserInfo() == null
                && origin.getQuery() == null
                && origin.getFragment() == null
                && Set.of("", "/").contains(origin.getPath());
    }

    CompletableFuture<Reply> sync(
            long revision,
            boolean allowed,
            boolean seed,
            Map<String, KeybindingProfile.Change> changes,
            Map<String, Boolean> availability) {
        var body = new JsonObject();
        body.addProperty("playerUuid", player.toString());
        body.addProperty("since", revision);
        body.addProperty("allowRemote", allowed);
        if (!availability.isEmpty())
            body.add("availability", KeybindingProfile.JSON.toJsonTree(availability));
        if (seed) body.addProperty("seed", true);
        if (!changes.isEmpty()) body.add("changes", KeybindingProfile.JSON.toJsonTree(changes));
        var request =
                HttpRequest.newBuilder(endpoint)
                        .timeout(Duration.ofSeconds(15))
                        .header("Content-Type", "application/json")
                        .header("X-Deuterium-Player", player.toString())
                        .header("Authorization", "Bearer " + token)
                        .POST(HttpRequest.BodyPublishers.ofString(body.toString()))
                        .build();
        return ChatApi.HTTP
                .sendAsync(request, HttpResponse.BodyHandlers.ofInputStream())
                .thenApplyAsync(
                        response -> {
                            try (var stream = response.body()) {
                                if (response.statusCode() == 401 || response.statusCode() == 403)
                                    throw new Expired();
                                if (response.statusCode() != 200) throw new IOException("键位服务暂不可用");
                                var envelope =
                                        JsonParser.parseString(
                                                        new String(
                                                                ChatApi.read(stream, 4 << 20),
                                                                StandardCharsets.UTF_8))
                                                .getAsJsonObject();
                                Reply reply =
                                        KeybindingProfile.JSON.fromJson(
                                                envelope.get("data"), Reply.class);
                                if (reply == null
                                        || !player.equals(reply.playerUuid())
                                        || reply.rows() == null
                                        || reply.rows().size() > 2048)
                                    throw new IOException("键位响应无效");
                                return reply;
                            } catch (IOException e) {
                                throw new java.util.concurrent.CompletionException(e);
                            }
                        });
    }

    static final class Expired extends RuntimeException {}
}
