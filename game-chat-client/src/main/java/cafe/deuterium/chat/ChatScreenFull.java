package cafe.deuterium.chat;

import com.lowdragmc.lowdraglib2.gui.holder.ModularUIScreen;
import net.minecraft.client.gui.GuiGraphics;
import net.minecraft.network.chat.Component;
import org.lwjgl.glfw.GLFW;
import java.nio.file.Path;
import java.util.List;

public final class ChatScreenFull extends ModularUIScreen {
    final ChatView view;
    private final long opened;
    public ChatScreenFull() { this(true); }
    ChatScreenFull(boolean sound){this(new ChatView(),sound);}
    private ChatScreenFull(ChatView view,boolean sound) { super(view.ui,Component.literal("通讯"));this.view=view;opened=System.nanoTime()-(sound?0:180_000_000L);if(sound)ChatSounds.play("open"); }
    @Override public void init() { WebPixels.apply(view.root);super.init();view.resize(WebPixels.width(),WebPixels.height()); }
    @Override public void render(GuiGraphics g,int x,int y,float tick) {
        view.resize(WebPixels.width(),WebPixels.height());view.refreshPanelTools();
        ChatBackdrop.draw(g,0,0,width,height,true,Math.min(1,(System.nanoTime()-opened)/180_000_000f));
        super.render(g,x,y,tick);
    }
    @Override public boolean keyPressed(int key,int scan,int mods) {
        if(key==GLFW.GLFW_KEY_ESCAPE) { onClose();return true; }
        if(key==GLFW.GLFW_KEY_ENTER||key==GLFW.GLFW_KEY_KP_ENTER) { view.send();return true; }
        return super.keyPressed(key,scan,mods);
    }
    @Override public void onFilesDrop(List<Path> paths) { if(paths.size()==1)ChatClient.INSTANCE.media.preview(paths.getFirst());else ChatClient.INSTANCE.notice("请一次选择一张图片"); }
    @Override public void onClose(){ChatSounds.play("close");super.onClose();}
    @Override public void removed() { view.close();super.removed(); }
}
