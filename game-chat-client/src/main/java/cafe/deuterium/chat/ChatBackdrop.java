package cafe.deuterium.chat;

import com.mojang.blaze3d.pipeline.TextureTarget;
import com.mojang.blaze3d.systems.RenderSystem;
import com.mojang.blaze3d.vertex.*;
import net.minecraft.client.Minecraft;
import net.minecraft.client.gui.GuiGraphics;
import net.minecraft.client.renderer.ShaderInstance;
import net.neoforged.neoforge.client.event.RegisterShadersEvent;
import org.lwjgl.opengl.GL30;
import java.io.IOException;

/** A reusable quarter-resolution backdrop; no CPU readback or per-frame allocation. */
final class ChatBackdrop {
    private static TextureTarget sample,horizontal;
    private static ShaderInstance shader;
    static void register(RegisterShadersEvent e) {
        try { e.registerShader(new ShaderInstance(e.getResourceProvider(),DeuteriumChat.id("chat_glass"),DefaultVertexFormat.POSITION_TEX_COLOR),s->shader=s); }
        catch(IOException error) { throw new java.io.UncheckedIOException(error); }
    }
    static void close() { if(sample!=null){sample.destroyBuffers();sample=null;} if(horizontal!=null){horizontal.destroyBuffers();horizontal=null;} }
    static void draw(GuiGraphics g,int x,int y,int w,int h,boolean full,float reveal){draw(g,x,y,w,h,full?1:0,reveal);}
    static void draw(GuiGraphics g,int x,int y,int w,int h,int mode,float reveal) {
        if(shader==null||w<=0||h<=0)return;
        var mc=Minecraft.getInstance();var main=mc.getMainRenderTarget();
        int sw=Math.max(1,main.width/4),sh=Math.max(1,main.height/4);
        if(sample==null)sample=new TextureTarget(sw,sh,false,Minecraft.ON_OSX);
        else if(sample.width!=sw||sample.height!=sh)sample.resize(sw,sh,Minecraft.ON_OSX);
        if(horizontal==null)horizontal=new TextureTarget(sw,sh,false,Minecraft.ON_OSX);
        else if(horizontal.width!=sw||horizontal.height!=sh)horizontal.resize(sw,sh,Minecraft.ON_OSX);
        sample.setFilterMode(org.lwjgl.opengl.GL11.GL_LINEAR);horizontal.setFilterMode(org.lwjgl.opengl.GL11.GL_LINEAR);
        g.flush();
        GL30.glBindFramebuffer(GL30.GL_READ_FRAMEBUFFER,main.frameBufferId);
        GL30.glBindFramebuffer(GL30.GL_DRAW_FRAMEBUFFER,sample.frameBufferId);
        GL30.glBlitFramebuffer(0,0,main.width,main.height,0,0,sw,sh,GL30.GL_COLOR_BUFFER_BIT,GL30.GL_LINEAR);
        horizontal.bindWrite(true);
        RenderSystem.disableBlend();RenderSystem.setShader(()->shader);
        shader.safeGetUniform("TexelSize").set(1f/sw,1f/sh);
        shader.safeGetUniform("BlurRadius").set(mode==1?14f:mode==2||mode==4?22f:mode==3?20f:16f);
        shader.safeGetUniform("ScreenSize").set((float)main.width,(float)main.height);
        shader.setSampler("Grain",mc.getTextureManager().getTexture(DeuteriumChat.id("textures/controls/deco_lattice_title.png")).getId());
        shader.safeGetUniform("BlurAxis").set(1f,0f);
        shader.safeGetUniform("FullPanel").set(-1f);
        shader.safeGetUniform("Reveal").set(1f);
        RenderSystem.setShaderTexture(0,sample.getColorTextureId());shader.setSampler("Sampler0",sample.getColorTextureId());
        quad(g,0,0,mc.getWindow().getGuiScaledWidth(),mc.getWindow().getGuiScaledHeight());
        main.bindWrite(true);
        RenderSystem.enableBlend();RenderSystem.defaultBlendFunc();RenderSystem.setShader(()->shader);
        RenderSystem.setShaderTexture(0,horizontal.getColorTextureId());shader.setSampler("Sampler0",horizontal.getColorTextureId());
        shader.safeGetUniform("BlurAxis").set(0f,1f);
        shader.safeGetUniform("FullPanel").set((float)mode);
        shader.safeGetUniform("Reveal").set(reveal);
        quad(g,x,y,w,h);RenderSystem.disableBlend();
    }
    private static void quad(GuiGraphics g,int x,int y,int w,int h){
        var mc=Minecraft.getInstance();
        var matrix=g.pose().last().pose();var b=Tesselator.getInstance().begin(VertexFormat.Mode.QUADS,DefaultVertexFormat.POSITION_TEX_COLOR);
        float gw=mc.getWindow().getGuiScaledWidth(),gh=mc.getWindow().getGuiScaledHeight();
        b.addVertex(matrix,x,y+h,0).setUv(x/gw,1-(y+h)/gh).setColor(-1);
        b.addVertex(matrix,x+w,y+h,0).setUv((x+w)/gw,1-(y+h)/gh).setColor(-1);
        b.addVertex(matrix,x+w,y,0).setUv((x+w)/gw,1-y/gh).setColor(-1);
        b.addVertex(matrix,x,y,0).setUv(x/gw,1-y/gh).setColor(-1);
        BufferUploader.drawWithShader(b.buildOrThrow());
    }
}
