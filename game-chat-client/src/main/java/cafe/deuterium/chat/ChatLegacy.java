package cafe.deuterium.chat;

import net.minecraft.client.resources.language.I18n;
import net.minecraft.network.chat.Component;
import net.minecraft.network.chat.HoverEvent;
import net.minecraft.network.chat.Style;
import java.time.Instant;
import java.util.Optional;
import java.util.regex.Pattern;

/** TrChat's plain serializer leaves translatable item names as keys. Match the client-visible text. */
final class ChatLegacy {
    private static final Pattern ITEM_NAME=Pattern.compile("(?<=\\[)(?:block|item)\\.[A-Za-z0-9_.-]+");
    static String localize(String text){return ITEM_NAME.matcher(text).replaceAll(m->java.util.regex.Matcher.quoteReplacement(I18n.exists(m.group())?I18n.get(m.group()):m.group()));}
    static boolean hasItem(Component text){return text.visit((style,value)->style.getHoverEvent()!=null&&style.getHoverEvent().getAction()==HoverEvent.Action.SHOW_ITEM?Optional.of(true):Optional.empty(),Style.EMPTY).orElse(false);}
    static ChatMessage merge(ChatMessage m,String legacy,Component original){
        String body=localize(m.content.getString());
        if(!legacy.endsWith(body))return m;
        int skip=legacy.length()-body.length();int[] at={0};var content=Component.empty();
        original.visit((style,text)->{int from=Math.max(0,skip-at[0]);at[0]+=text.length();if(from<text.length())content.append(Component.literal(text.substring(from)).setStyle(style));return Optional.empty();},Style.EMPTY);
        return new ChatMessage(m.id,m.senderId,m.name,m.channel,m.server,m.conversation,content,m.media,Instant.ofEpochMilli(m.sentMillis),m.receivedAt==0,m.system,m.reply);
    }
}
