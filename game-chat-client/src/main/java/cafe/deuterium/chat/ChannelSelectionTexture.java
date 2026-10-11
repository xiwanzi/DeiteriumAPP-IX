package cafe.deuterium.chat;
import com.lowdragmc.lowdraglib2.gui.texture.IGuiTexture;
import net.minecraft.client.gui.GuiGraphics;
final class ChannelSelectionTexture implements IGuiTexture {
    @Override public void draw(GuiGraphics g,float mx,float my,float x,float y,float w,float h,float tick){
        g.pose().pushPose();g.pose().scale(1/3f,1/3f,1);
        int left=Math.round(x*3),top=Math.round(y*3),width=Math.round(w*3),height=Math.round(h*3);
        for(int i=0;i<width;i++)g.fill(left+i,top,left+i+1,top+height,Math.round(27-23f*i/width)<<24|0xEEEF60);
        g.fill(left,top+11,left+2,top+height-11,0xFFFFF100);
        int dx=left+width-8,dy=top+height/2;
        for(int row=-3;row<=3;row++){int half=3-Math.abs(row);g.fill(dx-half,dy+row,dx+half+1,dy+row+1,0xFFFFF100);}
        g.pose().popPose();
    }
}
