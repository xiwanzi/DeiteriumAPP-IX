package cafe.deuterium.chat;

import com.lowdragmc.lowdraglib2.gui.texture.IGuiTexture;
import com.mojang.blaze3d.systems.RenderSystem;
import com.mojang.blaze3d.vertex.*;
import net.minecraft.client.gui.GuiGraphics;
import net.minecraft.client.renderer.ShaderInstance;
import net.minecraft.resources.ResourceLocation;
import net.neoforged.neoforge.client.event.RegisterShadersEvent;

/** CSS mask-image equivalent: alpha from the artwork, RGB exclusively from currentColor. */
final class MaskTexture implements IGuiTexture {
    private static ShaderInstance shader;
    private final ResourceLocation art;private final int color;private final float u,v,uw,vh;
    MaskTexture(String art,int color){this(art,color,0,0,1,1);}
    MaskTexture(String art,int color,float u,float v,float uw,float vh){this.art=DeuteriumChat.id("textures/"+art);this.color=color;this.u=u;this.v=v;this.uw=uw;this.vh=vh;}
    static void register(RegisterShadersEvent e){try{e.registerShader(new ShaderInstance(e.getResourceProvider(),DeuteriumChat.id("icon_mask"),DefaultVertexFormat.POSITION_TEX_COLOR),s->shader=s);}catch(java.io.IOException ex){throw new java.io.UncheckedIOException(ex);}}
    @Override public void draw(GuiGraphics g,float mx,float my,float x,float y,float w,float h,float tick){
        if(shader==null)return;g.flush();RenderSystem.enableBlend();RenderSystem.defaultBlendFunc();RenderSystem.setShader(()->shader);RenderSystem.setShaderTexture(0,art);
        var mat=g.pose().last().pose();var b=Tesselator.getInstance().begin(VertexFormat.Mode.QUADS,DefaultVertexFormat.POSITION_TEX_COLOR);
        b.addVertex(mat,x,y+h,0).setUv(u,v+vh).setColor(color);b.addVertex(mat,x+w,y+h,0).setUv(u+uw,v+vh).setColor(color);b.addVertex(mat,x+w,y,0).setUv(u+uw,v).setColor(color);b.addVertex(mat,x,y,0).setUv(u,v).setColor(color);BufferUploader.drawWithShader(b.buildOrThrow());
    }
}
