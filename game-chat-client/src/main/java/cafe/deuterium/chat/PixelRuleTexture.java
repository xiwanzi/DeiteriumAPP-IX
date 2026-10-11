package cafe.deuterium.chat;
import com.lowdragmc.lowdraglib2.gui.texture.IGuiTexture;
import net.minecraft.client.gui.GuiGraphics;
final class PixelRuleTexture implements IGuiTexture {
    private final int color;PixelRuleTexture(int color){this.color=color;}
    @Override public void draw(GuiGraphics g,float mx,float my,float x,float y,float w,float h,float tick){g.pose().pushPose();g.pose().scale(1/3f,1/3f,1);g.fill((int)(x*3),(int)(y*3),(int)((x+w)*3),(int)(y*3)+1,color);g.fill((int)(x*3),(int)(y*3)-1,(int)(x*3)+46,(int)(y*3)+2,0xFFE4E6DF);g.pose().popPose();}
}
