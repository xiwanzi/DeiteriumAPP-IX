package cafe.deuterium.chat;

import net.minecraft.client.gui.GuiGraphics;
import com.mojang.blaze3d.systems.RenderSystem;

/** Same Endfield nine-slice and destination border widths as the approved Web preview. */
final class NativeBubble {
    private static final net.minecraft.resources.ResourceLocation ART=DeuteriumChat.id("textures/chat-native/bg_snscontenttextorpic_chat_03.png");
    private static final int[] SX={0,38,51,82}, SY={0,31,38,76};
    static void draw(GuiGraphics g,float x,float y,float w,float h,int alpha) {
        int width=Math.round(w*3),height=Math.round(h*3);
        int[] dx={0,12,Math.max(12,width-10),width},dy={0,8,Math.max(8,height-10),height};
        g.pose().pushPose();g.pose().translate(x,y,0);g.pose().scale(1/3f,1/3f,1);
        RenderSystem.enableBlend();g.setColor(1,1,1,alpha/255f*.055f);
        for(int row=0;row<3;row++)for(int col=0;col<3;col++)if(dx[col+1]>dx[col]&&dy[row+1]>dy[row])
            g.blit(ART,dx[col],dy[row],dx[col+1]-dx[col],dy[row+1]-dy[row],SX[col],SY[row],SX[col+1]-SX[col],SY[row+1]-SY[row],82,76);
        g.setColor(1,1,1,1);g.pose().popPose();
    }
}
