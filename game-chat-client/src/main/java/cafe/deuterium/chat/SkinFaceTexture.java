package cafe.deuterium.chat;

import com.lowdragmc.lowdraglib2.gui.texture.IGuiTexture;
import com.mojang.blaze3d.systems.RenderSystem;
import net.minecraft.client.Minecraft;
import net.minecraft.client.gui.GuiGraphics;
import net.minecraft.client.gui.components.PlayerFaceRenderer;
import net.minecraft.client.resources.DefaultPlayerSkin;
import org.lwjgl.opengl.GL11;
import org.lwjgl.opengl.GL14;

/** Same base-face and outer-layer composition as the personal terminal. */
final class SkinFaceTexture implements IGuiTexture {
    private final String name;
    SkinFaceTexture(String name){this.name=name;}
    @Override public void draw(GuiGraphics g,float mx,float my,float x,float y,float w,float h,float tick){draw(g,name,x,y,Math.min(w,h),255);}
    static void draw(GuiGraphics g,String name,float x,float y,float size,int alpha){
        var mc=Minecraft.getInstance();var info=mc.getConnection()==null?null:mc.getConnection().getPlayerInfo(name);
        var skin=info==null?DefaultPlayerSkin.getDefaultTexture():info.getSkin().texture();
        g.flush();
        int src=GL11.glGetInteger(GL14.GL_BLEND_SRC_RGB),dst=GL11.glGetInteger(GL14.GL_BLEND_DST_RGB);
        int srcA=GL11.glGetInteger(GL14.GL_BLEND_SRC_ALPHA),dstA=GL11.glGetInteger(GL14.GL_BLEND_DST_ALPHA);
        boolean blend=GL11.glIsEnabled(GL11.GL_BLEND);
        g.pose().pushPose();
        try{
            RenderSystem.enableBlend();RenderSystem.blendFuncSeparate(GL11.GL_SRC_ALPHA,GL11.GL_ONE_MINUS_SRC_ALPHA,GL11.GL_ONE,GL11.GL_ONE_MINUS_SRC_ALPHA);
            g.setColor(1,1,1,alpha/255f);g.pose().translate(x,y,0);g.pose().scale(size/8,size/8,1);
            PlayerFaceRenderer.draw(g,skin,0,0,8,true,false);g.flush();
        }finally{
            g.pose().popPose();g.setColor(1,1,1,1);RenderSystem.blendFuncSeparate(src,dst,srcA,dstA);
            if(blend)RenderSystem.enableBlend();else RenderSystem.disableBlend();
        }
    }
}
