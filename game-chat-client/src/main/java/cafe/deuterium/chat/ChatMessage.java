package cafe.deuterium.chat;

import com.google.gson.JsonObject;
import net.minecraft.network.chat.Component;
import java.time.Instant;
import java.time.ZoneId;
import java.time.format.DateTimeFormatter;

public final class ChatMessage {
    private static final DateTimeFormatter CLOCK = DateTimeFormatter.ofPattern("HH:mm").withZone(ZoneId.systemDefault());
    public final String id, senderId, name, channel, server, conversation, time;
    public final Component content;
    public final JsonObject media;
    public final long receivedAt, sentMillis;
    public final boolean system;
    public final String reply;
    public String failure = "";
    public boolean pending;

    public ChatMessage(String id, String senderId, String name, String channel, String server,
                       String conversation, Component content, JsonObject media, Instant sentAt,
                       boolean historical, boolean system, String reply) {
        this.id=id; this.senderId=senderId; this.name=name; this.channel=channel; this.server=server;
        this.conversation=conversation; this.content=content; this.media=media; this.system=system;
        this.reply=reply; this.time=CLOCK.format(sentAt);this.sentMillis=sentAt.toEpochMilli();
        this.receivedAt=historical?0:System.nanoTime()/1_000_000;
    }
    public String scope() { return channel.equals("private")?"private:"+conversation:channel; }
    public static String str(JsonObject o,String k) { return o!=null&&o.has(k)&&!o.get(k).isJsonNull()?o.get(k).getAsString():""; }
    public static ChatMessage fromJson(JsonObject o, boolean historical, String channel, String conversation) {
        JsonObject sender=o.has("sender")?o.getAsJsonObject("sender"):new JsonObject();
        Instant at; try { at=Instant.parse(str(o,"sentAt")); } catch(Exception e) { at=Instant.now(); }
        return new ChatMessage(str(o,"messageId"),str(sender,"playerRef"),str(sender,"gameId"),channel,
                str(o,"serverId"),conversation,Component.literal(ChatLegacy.localize(str(o,"content"))),
                o.has("media")&&!o.get("media").isJsonNull()?o.getAsJsonObject("media"):null,at,historical,false,
                o.has("reply")&&o.get("reply").isJsonObject()?str(o.getAsJsonObject("reply"),"content"):"");
    }
}
