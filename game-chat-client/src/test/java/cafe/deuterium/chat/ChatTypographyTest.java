package cafe.deuterium.chat;

import net.minecraft.network.chat.*;
import net.minecraft.resources.ResourceLocation;
import org.junit.jupiter.api.Test;
import java.time.Instant;
import java.util.*;
import static org.junit.jupiter.api.Assertions.*;

class ChatTypographyTest {
    @org.junit.jupiter.api.BeforeAll static void initialize(){
        if(net.neoforged.fml.ModList.get()==null)net.neoforged.fml.ModList.of(List.of(),List.of());
        if(net.neoforged.fml.loading.LoadingModList.get()==null)net.neoforged.fml.loading.LoadingModList.of(List.of(),List.of(),List.of(),List.of(),Map.of());
        net.minecraft.SharedConstants.tryDetectVersion();net.minecraft.server.Bootstrap.bootStrap();
    }
    @Test void namesTitlesAndMessageTextUseTheSameMediumFace(){
        var message=new ChatMessage("font","id","xiwanzi","world","amiya","",Component.literal("中文 English"),null,Instant.now(),false,false,"");
        for(var text:List.of(ChatStyle.playerName(message),ChatTitles.title(message),ChatStyle.body(message.content))){
            text.visit((style,value)->{assertEquals(DeuteriumChat.id("harmony"),style.getFont());return Optional.empty();},Style.EMPTY);
        }
    }
    @Test void normalizingRichTextRetainsColorsAndInteractions(){
        var click=new ClickEvent(ClickEvent.Action.SUGGEST_COMMAND,"/help");
        var hover=new HoverEvent(HoverEvent.Action.SHOW_TEXT,Component.literal("说明"));
        var input=Component.literal("前缀").append(Component.literal("点击").withStyle(s->s.withFont(DeuteriumChat.id("harmony_regular")).withColor(0xB5CCD9).withClickEvent(click).withHoverEvent(hover).withInsertion("insert")));
        var output=ChatStyle.body(input);assertEquals(input.getString(),output.getString());
        output.visit((style,value)->{
            assertEquals(DeuteriumChat.id("harmony"),style.getFont());
            if(value.equals("点击")){assertEquals(click,style.getClickEvent());assertEquals(hover,style.getHoverEvent());assertEquals(0xB5CCD9,style.getColor().getValue());assertEquals("insert",style.getInsertion());}
            return Optional.empty();
        },Style.EMPTY);
    }
    @Test void customServerGlyphFontIsNotReplacedWithMissingGlyphs(){
        var icon=ResourceLocation.parse("server:icons");
        var value=ChatStyle.body(Component.literal("\uE001").withStyle(s->s.withFont(icon)));
        value.visit((style,text)->{if(!text.isEmpty())assertEquals(icon,style.getFont());return Optional.empty();},Style.EMPTY);
    }
}
