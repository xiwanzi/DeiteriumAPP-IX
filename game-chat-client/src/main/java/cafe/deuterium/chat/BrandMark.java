package cafe.deuterium.chat;

import com.lowdragmc.lowdraglib2.gui.LDLibFonts;
import com.lowdragmc.lowdraglib2.gui.ui.UIElement;
import com.lowdragmc.lowdraglib2.gui.ui.rendering.GUIContext;
import net.minecraft.network.chat.Component;

final class BrandMark extends UIElement {
    private static final Component NAME=Component.literal("Deuterium").withStyle(st->st.withFont(DeuteriumChat.id("geist")));
    private static final Component IX=Component.literal("IX").withStyle(st->st.withFont(DeuteriumChat.id("brand")));
    @Override public void drawBackgroundAdditional(GUIContext ctx){
        var g=ctx.graphics;g.pose().pushPose();g.pose().translate(getPositionX(),getPositionY(),0);g.pose().scale(22/27f,22/27f,1);
        var mat=g.pose().last().pose();float d=(float)ctx.mc.getWindow().getGuiScale();mat.m30(Math.round(mat.m30()*d)/d);mat.m31(Math.round(mat.m31()*d)/d);
        var font=LDLibFonts.font();float next=font.width(NAME)+3;
        LDLibFonts.drawText(g,font,NAME,0,0,0xFFF0F0F2,false);LDLibFonts.drawText(g,font,IX,next,0,0xFFF0F0F2,false);
        g.pose().popPose();
        g.pose().pushPose();g.pose().translate(getPositionX()+(next+font.width(IX))*22/27f+2,getPositionY()+17/3f,0);g.pose().scale(1/3f,1/3f,1);g.fill(0,0,3,3,ChatStyle.YELLOW);g.pose().popPose();
    }
}
