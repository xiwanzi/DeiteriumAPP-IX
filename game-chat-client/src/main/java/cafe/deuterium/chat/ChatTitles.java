package cafe.deuterium.chat;

import com.google.gson.JsonObject;
import net.minecraft.network.chat.Component;
import java.util.*;
import java.util.regex.Pattern;

final class ChatTitles {
    static long revision;
    private static final Map<String,String> names=new HashMap<>();
    private static final Pattern COLOR=Pattern.compile("#[0-9a-fA-F]{6}");
    static void receive(JsonObject packet){
        String uuid=ChatMessage.str(packet,"playerUuid"),name=ChatMessage.str(packet,"gameId"),display=ChatMessage.str(packet,"displayName");
        if(uuid.isEmpty()||display.isEmpty()||display.length()>512)return;
        names.put(uuid,display);if(!name.isEmpty())names.put("name:"+name.toLowerCase(Locale.ROOT),display);revision++;ChatStore.INSTANCE.changed();
    }
    static Component title(ChatMessage message){return title(message.senderId,message.name);}
    static Component title(String id,String name){return render(names.getOrDefault(id,names.getOrDefault("name:"+name.toLowerCase(Locale.ROOT),"溯行者")));}
    static Component render(String display){
        var result=Component.literal("[").withStyle(s->s.withColor(0xA9B3B8).withFont(DeuteriumChat.id("harmony_regular")));
        var matcher=COLOR.matcher(display);int start=0,color=0xA9B3B8;
        while(matcher.find()){
            if(matcher.start()>start)result.append(Component.literal(display.substring(start,matcher.start())).withColor(color));
            color=Integer.parseInt(matcher.group().substring(1),16);start=matcher.end();
        }
        if(start<display.length())result.append(Component.literal(display.substring(start)).withColor(color));
        return result.append(Component.literal("]").withColor(0xA9B3B8));
    }
    static void reset(){names.clear();revision++;}
}
