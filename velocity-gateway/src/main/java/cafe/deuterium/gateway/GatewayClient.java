package cafe.deuterium.gateway;

import com.google.gson.Gson;
import com.google.gson.JsonObject;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
import java.util.List;
import java.util.Map;
import java.util.UUID;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.Executor;
import java.util.concurrent.TimeUnit;

final class GatewayClient {
    record Access(boolean allowed,String status,long version,String message) {}
    record Kick(String commandId,UUID uuid,long version,String reason) {}
    record Ack(String commandId,String status) {}
    private static final Gson GSON = new Gson();
    private final HttpClient client;
    private final GatewayConfig config;
    GatewayClient(GatewayConfig config, Executor executor) {
        this.config=config;
        client=HttpClient.newBuilder().executor(executor).connectTimeout(Duration.ofSeconds(3)).followRedirects(HttpClient.Redirect.NEVER).build();
    }
    private CompletableFuture<JsonObject> post(String path,Object input) {
        var request=HttpRequest.newBuilder(URI.create(config.apiBaseUrl+path))
                .timeout(Duration.ofSeconds(config.requestTimeoutSeconds))
                .header("Authorization","Bearer "+config.apiToken).header("Content-Type","application/json")
                .header("Accept","application/json").POST(HttpRequest.BodyPublishers.ofString(GSON.toJson(input))).build();
        return client.sendAsync(request,HttpResponse.BodyHandlers.ofString())
                .orTimeout(config.requestTimeoutSeconds+1L,TimeUnit.SECONDS).thenApply(response->{
                    if(response.statusCode()!=200 || response.body().length()>65536)throw new IllegalStateException("Gateway API unavailable ("+response.statusCode()+")");
                    JsonObject envelope=GSON.fromJson(response.body(),JsonObject.class);
                    if(envelope==null||envelope.has("error")||!envelope.has("data")||!envelope.get("data").isJsonObject())throw new IllegalStateException("Invalid gateway response");
                    return envelope.getAsJsonObject("data");
                });
    }
    CompletableFuture<Access> check(UUID uuid) {
        return post("/bridge/v1/admission/check",Map.of("uuid",uuid.toString())).thenApply(GatewayClient::decodeAccess);
    }
    static Access decodeAccess(JsonObject value) {
        if(!value.has("allowed")||!value.get("allowed").isJsonPrimitive()||!value.getAsJsonPrimitive("allowed").isBoolean()
                ||!value.has("status")||!value.has("version"))throw new IllegalStateException("Incomplete access response");
        String status=value.get("status").getAsString();boolean allowed=value.get("allowed").getAsBoolean();long version=value.get("version").getAsLong();
        if(!List.of("ACTIVE","REVOKED","NONE").contains(status)||allowed!=status.equals("ACTIVE")||version<0||(allowed&&version<1))throw new IllegalStateException("Inconsistent access response");
        return new Access(allowed,status,version,value.has("message")?value.get("message").getAsString():"");
    }
    CompletableFuture<List<Kick>> poll(String instance,int online,List<Ack> acks) {
        return post("/bridge/v1/admission/poll",Map.of("instanceId",instance,"pluginVersion",DeuteriumGateway.VERSION,"onlinePlayers",online,"acks",acks))
                .thenApply(data->{
                    if(!data.has("commands")||!data.get("commands").isJsonArray()||data.getAsJsonArray("commands").size()>50)throw new IllegalStateException("Invalid gateway command list");
                    var result=new java.util.ArrayList<Kick>();
                    for(var element:data.getAsJsonArray("commands")){
                        var item=element.getAsJsonObject();String id=item.get("commandId").getAsString();
                        if(!id.matches("kick_[0-9a-f]{32}"))throw new IllegalStateException("Invalid command identifier");
                        result.add(new Kick(id,UUID.fromString(item.get("uuid").getAsString()),item.get("version").getAsLong(),item.get("reason").getAsString()));
                    }
                    return List.copyOf(result);
                });
    }
}
