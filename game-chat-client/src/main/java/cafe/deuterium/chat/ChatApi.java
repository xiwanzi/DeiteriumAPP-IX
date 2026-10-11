package cafe.deuterium.chat;

import com.google.gson.*;
import java.io.*;
import java.net.URI;
import java.net.http.*;
import java.nio.charset.StandardCharsets;
import java.time.*;

/** Worker-only API operations. Session objects are immutable and never logged or persisted. */
final class ChatApi {
    final URI base;final String token,owner,server;final Instant expiry;
    private static final HttpClient HTTP=HttpClient.newBuilder().connectTimeout(Duration.ofSeconds(8)).followRedirects(HttpClient.Redirect.NEVER).build();
    ChatApi(JsonObject p) {
        base=URI.create(ChatMessage.str(p,"apiBase"));
        if(!safe(base)||base.getUserInfo()!=null||base.getQuery()!=null)throw new IllegalArgumentException("不安全的聊天服务地址");
        token=ChatMessage.str(p,"token");owner=ChatMessage.str(p,"playerRef");server=ChatMessage.str(p,"serverId");expiry=Instant.parse(ChatMessage.str(p,"expiresAt"));
    }
    boolean valid(){return !token.isBlank()&&expiry.isAfter(Instant.now().plusSeconds(25));}
    JsonElement request(String method,String path,JsonObject body)throws Exception {
        if(!path.startsWith("/api/v1/chat/"))throw new IllegalArgumentException("Chat scope only");
        var b=HttpRequest.newBuilder(base.resolve(path)).timeout(Duration.ofSeconds(25)).header("Authorization","Bearer "+token).header("Content-Type","application/json");
        b.method(method,body==null?HttpRequest.BodyPublishers.noBody():HttpRequest.BodyPublishers.ofString(body.toString(),StandardCharsets.UTF_8));
        var r=HTTP.send(b.build(),HttpResponse.BodyHandlers.ofInputStream());
        JsonObject envelope=JsonParser.parseString(new String(read(r.body(),4<<20),StandardCharsets.UTF_8)).getAsJsonObject();
        if(r.statusCode()<200||r.statusCode()>=300||envelope.has("error"))throw new IOException(envelope.has("error")?ChatMessage.str(envelope.getAsJsonObject("error"),"message"):"聊天服务暂不可用");
        return envelope.get("data");
    }
    static boolean safe(URI uri){return "https".equals(uri.getScheme())||("http".equals(uri.getScheme())&&("127.0.0.1".equals(uri.getHost())||"localhost".equals(uri.getHost())));}
    static byte[] download(String url,int max)throws Exception {
        URI uri=URI.create(url);if(!safe(uri))throw new IOException("图片地址不可用");
        var r=HTTP.send(HttpRequest.newBuilder(uri).timeout(Duration.ofSeconds(25)).GET().build(),HttpResponse.BodyHandlers.ofInputStream());
        if(r.statusCode()!=200){r.body().close();throw new IOException("图片暂不可用");}return read(r.body(),max);
    }
    static void upload(JsonObject authorization,byte[] bytes)throws Exception {
        if(!ChatMessage.str(authorization,"provider").equals("RAINYUN_S3")||!ChatMessage.str(authorization,"method").equals("PUT")||authorization.get("sizeBytes").getAsLong()!=bytes.length)throw new IOException("图片上传授权不正确");
        URI uri=URI.create(ChatMessage.str(authorization,"uploadUrl"));if(!safe(uri))throw new IOException("上传地址不可用");
        var b=HttpRequest.newBuilder(uri).timeout(Duration.ofSeconds(45)).PUT(HttpRequest.BodyPublishers.ofByteArray(bytes));
        if(authorization.has("signedHeaders"))for(var h:authorization.getAsJsonObject("signedHeaders").entrySet())
            if(!h.getKey().equalsIgnoreCase("Content-Length"))b.header(h.getKey(),h.getValue().getAsString());
        var response=HTTP.send(b.build(),HttpResponse.BodyHandlers.discarding());
        // A timed-out prior PUT may already have created this immutable object.
        // Complete still verifies its exact digest; never overwrite that object.
        if(response.statusCode()!=412&&(response.statusCode()<200||response.statusCode()>=300))throw new IOException("图片发送失败，请重试");
    }
    static byte[] read(InputStream input,int max)throws IOException {
        try(input){byte[] bytes=input.readNBytes(max+1);if(bytes.length>max)throw new IOException("响应过大");return bytes;}
    }
}
