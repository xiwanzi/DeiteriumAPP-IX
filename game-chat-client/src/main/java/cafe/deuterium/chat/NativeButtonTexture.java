package cafe.deuterium.chat;

import com.lowdragmc.lowdraglib2.gui.texture.IGuiTexture;
import com.mojang.blaze3d.systems.RenderSystem;
import net.minecraft.client.gui.GuiGraphics;
import net.minecraft.resources.ResourceLocation;

/** Exact Web send-button slice widths and optical bounds, in physical pixels. */
final class NativeButtonTexture implements IGuiTexture {
    private final boolean hover;private final int tint;
    NativeButtonTexture(boolean hover){this(hover,hover?0xFFF0F0F0:0xFFF7F7F7);}
    NativeButtonTexture(boolean hover,int tint){this.hover=hover;this.tint=tint;}
    private static void slice(GuiGraphics g,String name,float x,float y,float w,float h,int sw,int sh,int edge,float target,int color){
        var art=DeuteriumChat.id("textures/"+name);int[] sx={0,edge,sw-edge,sw};float[] dx={x,x+target,x+w-target,x+w};
        g.setColor(((color>>16)&255)/255f,((color>>8)&255)/255f,(color&255)/255f,(color>>>24)/255f);
        for(int i=0;i<3;i++){g.pose().pushPose();g.pose().translate(dx[i],y,0);g.pose().scale(dx[i+1]-dx[i],h,1);g.blit(art,0,0,1,1,sx[i],0,sx[i+1]-sx[i],sh,sw,sh);g.pose().popPose();}
    }
    @Override public void draw(GuiGraphics g,float mx,float my,float x,float y,float w,float h,float tick){
        g.pose().pushPose();g.pose().scale(1/3f,1/3f,1);RenderSystem.enableBlend();RenderSystem.defaultBlendFunc();x*=3;y*=3;w*=3;h*=3;
        if(hover)slice(g,"chat-native/btn_common_hover_bg_short.png",x-2,y-2.873f,w+4.873f,h+4.873f,68,56,33,28.8f,0xCCBFBFBF);
        slice(g,"controls/btn_common_normal_bg_short.png",x-5.923f,y-4.231f,w+13.538f,h+16.923f,84,72,40,33.846f,tint);
        float ch=h-14,cw=ch*148/48f;g.setColor(.1f,.1f,.1f,((tint&255)<100)?.04f:.12f);
        g.pose().pushPose();g.pose().translate(x+w-14-cw,y+7,0);g.pose().scale(cw,ch,1);g.blit(DeuteriumChat.id("textures/controls/deco_btn_contour_lines_short.png"),0,0,1,1,0,0,148,48,148,48);g.pose().popPose();
        g.setColor(1,1,1,1);
        if(hover)new MaskTexture("chat-native/focus-arrow.png",0xFFC5CBC8).draw(g,mx,my,x+w+6,y+h/2-8,16,16,tick);
        g.pose().popPose();
    }
}
