package cafe.deuterium.chat;

import net.minecraft.client.Minecraft;
import net.minecraft.client.gui.GuiGraphics;

public final class QuickChatHooks {
    private QuickChatHooks(){}
    public static void sound(String role){ChatSounds.play(role);}
    public static void backdrop(GuiGraphics g){var mc=Minecraft.getInstance();int w=mc.getWindow().getGuiScaledWidth(),h=mc.getWindow().getGuiScaledHeight();ChatBackdrop.draw(g,2,h-14,w-4,12,false,1);}
    public static void completionGlass(GuiGraphics g,int x,int y,int w,int h){ChatBackdrop.draw(g,x,y,w,h,2,1);}
    public static void hover(GuiGraphics g,int x,int y){if(ChatTimeline.HUD!=null)ChatTimeline.HUD.hover(g,x,y);}
    public static boolean click(double x,double y,int button){return ChatTimeline.HUD!=null&&ChatTimeline.HUD.click(x,y,button);}
    public static void scroll(double dy){if(ChatTimeline.HUD!=null)ChatTimeline.HUD.scrollBy(dy);}
    public static boolean send(String text,boolean history){
        if(text.strip().startsWith("/")||text.isBlank())return false;
        if(ChatClient.INSTANCE.send(text.strip())){ChatSounds.play("send");if(history)Minecraft.getInstance().gui.getChat().addRecentChat(text);}
        return true;
    }
}
