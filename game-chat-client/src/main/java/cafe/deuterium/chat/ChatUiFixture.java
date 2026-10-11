package cafe.deuterium.chat;

import com.google.gson.*;
import com.sun.net.httpserver.HttpServer;
import java.net.InetSocketAddress;
import java.time.Instant;
import java.util.*;

/** Loopback-only visual fixture; excluded from the delivered jar. */
final class ChatUiFixture implements AutoCloseable {
    private final HttpServer server;
    private final Map<String,byte[]> files=new HashMap<>();
    final Map<String,JsonObject> media=new HashMap<>();
    ChatUiFixture()throws Exception {
        server=HttpServer.create(new InetSocketAddress("127.0.0.1",0),0);
        for(String id:List.of("cloud","sticker","motion")){
            byte[] image=resource(id+".webp"),thumb=resource(id+"-thumb.webp");files.put("/"+id,image);files.put("/"+id+"-thumb",thumb);
            var m=new JsonObject();m.addProperty("mediaId",id);m.addProperty("sha256",ChatMediaClient.hash(image));m.addProperty("thumbnailSha256",ChatMediaClient.hash(thumb));m.addProperty("sizeBytes",image.length);m.addProperty("thumbnailBytes",thumb.length);
            try(var codec=new WebpCodec(image)){m.addProperty("width",codec.width);m.addProperty("height",codec.height);m.addProperty("animated",codec.frames>1);}
            m.addProperty("type",id.equals("cloud")?"image":"sticker");media.put(id,m);
        }
        server.createContext("/",exchange->{try{
            String path=exchange.getRequestURI().getPath();byte[] bytes;
            if(files.containsKey(path)){bytes=files.get(path);exchange.getResponseHeaders().set("Content-Type","image/webp");}
            else {JsonElement data=new JsonArray();
                if(path.endsWith("/access")){String id=path.split("/")[6];var access=new JsonObject();access.addProperty("url",base()+"/"+id);access.addProperty("thumbnailUrl",base()+"/"+id+"-thumb");access.add("media",media.get(id));data=access;}
                else if(path.startsWith("/api/v1/chat/avatars/")){var face=new java.awt.image.BufferedImage(8,8,java.awt.image.BufferedImage.TYPE_INT_ARGB);for(int y=0;y<8;y++)for(int x=0;x<8;x++)face.setRGB(x,y,y<3?0xFF8AC9C3:0xFFF5DDC1);var output=new java.io.ByteArrayOutputStream();javax.imageio.ImageIO.write(face,"png",output);var avatar=new JsonObject();avatar.addProperty("png",java.util.Base64.getEncoder().encodeToString(output.toByteArray()));data=avatar;}
                else if(path.endsWith("/stickers")){var library=new JsonArray();library.add(media.get("sticker"));library.add(media.get("motion"));data=library;}
                var root=new JsonObject();root.add("data",data);bytes=root.toString().getBytes(java.nio.charset.StandardCharsets.UTF_8);exchange.getResponseHeaders().set("Content-Type","application/json");
            }
            exchange.sendResponseHeaders(200,bytes.length);try(var out=exchange.getResponseBody()){out.write(bytes);}
        }catch(Exception error){exchange.close();}});
        server.setExecutor(java.util.concurrent.Executors.newFixedThreadPool(2,r->{var t=new Thread(r,"Chat-UI-Fixture");t.setDaemon(true);return t;}));server.start();
        var session=new JsonObject();session.addProperty("apiBase",base());session.addProperty("token","isolated-ui-fixture");session.addProperty("playerRef","fixture-owner");session.addProperty("serverId","amiya");session.addProperty("expiresAt",Instant.now().plusSeconds(3600).toString());ChatClient.INSTANCE.api=new ChatApi(session);ChatClient.INSTANCE.media.refreshStickers();
    }
    String base(){return "http://127.0.0.1:"+server.getAddress().getPort();}
    private byte[] resource(String file)throws Exception{try(var in=getClass().getResourceAsStream("/assets/deuterium_chat/ui-test/"+file)){return in.readAllBytes();}}
    @Override public void close(){server.stop(0);}
}
