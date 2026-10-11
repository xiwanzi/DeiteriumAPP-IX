package cafe.deuterium.chat;

import com.lowdragmc.lowdraglib2.gui.LDLibFonts;
import net.minecraft.client.Minecraft;
import net.minecraft.client.gui.GuiGraphics;
import net.minecraft.network.chat.Component;
import net.minecraft.util.FormattedCharSequence;
import java.util.*;

/** SDF text and native RGBA bitmap glyphs share the same measured advances. */
final class ChatTextLine {
    private record Run(FormattedCharSequence text,float x,boolean color){}
    private final List<Run> runs=new ArrayList<>();
    static boolean colorGlyph(int c){return c>=0x1F000||c>=0x2600&&c<=0x27FF||c==0xFE0F;}
    static boolean hasColor(String text){return text.codePoints().anyMatch(ChatTextLine::colorGlyph);}
    ChatTextLine(FormattedCharSequence line){
        List<FormattedCharSequence> pending=new ArrayList<>();float[] x={0},start={0};
        line.accept((index,style,c)->{var part=Component.literal(new String(Character.toChars(c))).withStyle(style).getVisualOrderText();
            if(colorGlyph(c)){if(!pending.isEmpty()){runs.add(new Run(FormattedCharSequence.composite(List.copyOf(pending)),start[0],false));pending.clear();}runs.add(new Run(part,x[0],true));}
            else{if(pending.isEmpty())start[0]=x[0];pending.add(part);}
            x[0]+=LDLibFonts.font().getSplitter().stringWidth(part);return true;
        });
        if(!pending.isEmpty())runs.add(new Run(FormattedCharSequence.composite(List.copyOf(pending)),start[0],false));
    }
    void draw(GuiGraphics g,int color,boolean shadow){for(var r:runs)LDLibFonts.drawText(g,r.color?Minecraft.getInstance().font:LDLibFonts.font(),r.text,r.x,0,r.color?(color&0xFF000000)|0xFFFFFF:color,shadow);}
}
