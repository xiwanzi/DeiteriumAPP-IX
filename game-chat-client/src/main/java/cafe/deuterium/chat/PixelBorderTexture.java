package cafe.deuterium.chat;
import com.lowdragmc.lowdraglib2.gui.texture.*;
import net.minecraft.client.gui.GuiGraphics;
final class PixelBorderTexture implements IGuiTexture {
    private final ColorBorderTexture texture;
    PixelBorderTexture(int color){texture=new ColorBorderTexture(-1,color);}
    @Override public void draw(GuiGraphics g,float mx,float my,float x,float y,float w,float h,float tick){g.pose().pushPose();g.pose().scale(1/3f,1/3f,1);texture.draw(g,mx*3,my*3,x*3,y*3,w*3,h*3,tick);g.pose().popPose();}
}
