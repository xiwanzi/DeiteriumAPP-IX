package cafe.deuterium.chat;

import com.lowdragmc.lowdraglib2.gui.ui.UIElement;
import com.lowdragmc.lowdraglib2.math.Size;
import net.minecraft.client.Minecraft;

/** Full-screen social UI uses CSS pixels; Minecraft GUI scale only affects the HUD. */
final class WebPixels {
    static int width(){return Math.round(Minecraft.getInstance().getWindow().getWidth()/3f);}
    static int height(){return Math.round(Minecraft.getInstance().getWindow().getHeight()/3f);}
    static Size size(){return Size.of(width(),height());}
    static boolean wide(){return Minecraft.getInstance().getWindow().getWidth()>=1800;}
    static float contentX(){return (wide()?158:136)/3f;}
    static float contentWidth(){return (Math.min(Minecraft.getInstance().getWindow().getWidth()*.46f,870)-19)/3;}
    static float topbar(){return (Minecraft.getInstance().getWindow().getHeight()<=620?52:wide()?88:76)/3f;}
    static float headingHeight(){return (Minecraft.getInstance().getWindow().getHeight()<=620?48:wide()?66:59)/3f;}
    static float bottom(){return (wide()?40:32)/3f;}
    static float composer(){return (wide()?115:106)/3f;}
    static void apply(UIElement root){
        root.lss("position","absolute");root.getLayout().left(0).top(0);
        root.transform(t->t.pivot(0,0).scale(3f/(float)Minecraft.getInstance().getWindow().getGuiScale()));
    }
}
