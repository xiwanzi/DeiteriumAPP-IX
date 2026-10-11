package cafe.deuterium.chat;

import com.lowdragmc.lowdraglib2.gui.texture.IGuiTexture;
import com.mojang.blaze3d.systems.RenderSystem;
import com.mojang.blaze3d.vertex.*;
import net.minecraft.client.gui.GuiGraphics;
import net.minecraft.client.renderer.ShaderInstance;
import net.neoforged.neoforge.client.event.RegisterShadersEvent;

/** Subpixel coverage in physical pixels, with straight-alpha compositing. */
final class RoundTexture implements IGuiTexture {
    private static ShaderInstance shader;
    private final float radius,stroke;private int fill;private final int border;
    RoundTexture(float radius,int fill){this(radius,fill,0,0);}
    RoundTexture(float radius,int fill,float stroke,int border){this.radius=radius;this.fill=fill;this.stroke=stroke;this.border=border;}
    @Override public RoundTexture setColor(int color){fill=color;return this;}
    static void register(RegisterShadersEvent e){try{e.registerShader(new ShaderInstance(e.getResourceProvider(),DeuteriumChat.id("round_shape"),DefaultVertexFormat.POSITION_TEX_COLOR),s->shader=s);}catch(java.io.IOException ex){throw new java.io.UncheckedIOException(ex);}}
    private static void color(String name,int c){shader.safeGetUniform(name).set(((c>>16)&255)/255f,((c>>8)&255)/255f,(c&255)/255f,(c>>>24)/255f);}
    @Override public void draw(GuiGraphics g,float mx,float my,float x,float y,float w,float h,float tick){
        if(shader==null||w<=0||h<=0)return;g.flush();RenderSystem.enableBlend();RenderSystem.defaultBlendFunc();RenderSystem.setShader(()->shader);
        shader.safeGetUniform("ShapeSize").set(w,h);shader.safeGetUniform("Radius").set(Math.min(radius,Math.min(w,h)/2));shader.safeGetUniform("Stroke").set(stroke);color("FillColor",fill);color("BorderColor",border);
        float p=1;var mat=g.pose().last().pose();var b=Tesselator.getInstance().begin(VertexFormat.Mode.QUADS,DefaultVertexFormat.POSITION_TEX_COLOR);
        b.addVertex(mat,x-p,y+h+p,0).setUv(-p/w,1+p/h).setColor(-1);b.addVertex(mat,x+w+p,y+h+p,0).setUv(1+p/w,1+p/h).setColor(-1);
        b.addVertex(mat,x+w+p,y-p,0).setUv(1+p/w,-p/h).setColor(-1);b.addVertex(mat,x-p,y-p,0).setUv(-p/w,-p/h).setColor(-1);BufferUploader.drawWithShader(b.buildOrThrow());
    }
}
