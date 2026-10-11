package cafe.deuterium.chat;

import com.lowdragmc.lowdraglib2.gui.LDLibFonts;
import com.lowdragmc.lowdraglib2.gui.ui.UIElement;
import com.lowdragmc.lowdraglib2.gui.ui.rendering.GUIContext;
import net.minecraft.network.chat.Component;

final class BrandMark extends UIElement {
    @Override public void drawBackgroundAdditional(GUIContext ctx){
        float x=getPositionX(),y=getPositionY();var name=Component.literal("Deuterium").withStyle(st->st.withFont(DeuteriumChat.id("geist")));
        ctx.graphics.pose().pushPose();ctx.graphics.pose().translate(x,y,0);ctx.graphics.pose().scale(99.65625f/3/LDLibFonts.font().width(name),21/27f,1);LDLibFonts.drawText(ctx.graphics,LDLibFonts.font(),name,0,0,ChatStyle.TEXT,false);ctx.graphics.pose().popPose();
        ChatTimeline.drawText(ctx.graphics,Component.literal("IX").withStyle(s->s.withFont(DeuteriumChat.id("brand"))),x+106.65625f/3,y,20/27f,ChatStyle.TEXT);
        ctx.graphics.pose().pushPose();ctx.graphics.pose().translate(x+131.921875f/3,y+17/3f,0);ctx.graphics.fill(0,0,1,1,ChatStyle.YELLOW);ctx.graphics.pose().popPose();
    }
}
