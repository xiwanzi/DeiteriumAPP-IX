package cafe.deuterium.chat;

import com.lowdragmc.lowdraglib2.gui.ui.UIElement;
import com.lowdragmc.lowdraglib2.gui.ui.rendering.GUIContext;

final class PopupSurface extends UIElement {
    private final RoundTexture surface=new RoundTexture(1,0xF51C1C1E,1/3f,0x3DD5D5D7);
    @Override public void drawBackgroundAdditional(GUIContext ctx){
        float x=getPositionX(),y=getPositionY(),w=getSizeWidth(),h=getSizeHeight();var a=getWorldMouse(x,y);var b=getWorldMouse(x+w,y+h);
        ctx.graphics.pose().pushPose();ctx.graphics.pose().last().pose().identity();
        ChatBackdrop.draw(ctx.graphics,(int)a.x,(int)a.y,(int)Math.ceil(b.x-a.x),(int)Math.ceil(b.y-a.y),4,1);
        ctx.graphics.pose().popPose();surface.draw(ctx.graphics,0,0,x,y,w,h,0);
    }
}
