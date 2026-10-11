package cafe.deuterium.chat;

import com.lowdragmc.lowdraglib2.gui.ui.UIElement;
import com.lowdragmc.lowdraglib2.math.Size;
import net.minecraft.client.Minecraft;

/** Full-screen layout uses physical pixels; timeline text also normalizes HUD GUI scaling. */
final class WebPixels {
    static int width(){return Math.round(Minecraft.getInstance().getWindow().getWidth()/3f);}
    static int height(){return Math.round(Minecraft.getInstance().getWindow().getHeight()/3f);}
    static Size size(){return Size.of(width(),height());}
    static boolean wide(){return Minecraft.getInstance().getWindow().getWidth()>=1800;}
    static boolean compact(){var w=Minecraft.getInstance().getWindow();return w.getWidth()<1440||w.getHeight()<820;}
    static float contentX(){return (compact()?112:wide()?158:136)/3f;}
    static float contentWidth(){float w=Minecraft.getInstance().getWindow().getWidth();return Math.min(w-contentX()*3-24,compact()?Math.min(w*.64f,820):Math.min(w*.46f,870)-19)/3;}
    static float topbar(){return (Minecraft.getInstance().getWindow().getHeight()<=620?52:wide()?88:76)/3f;}
    static float headingHeight(){return (Minecraft.getInstance().getWindow().getHeight()<=620?48:wide()?66:59)/3f;}
    static float bottom(){return (wide()?40:32)/3f;}
    static float composer(){return (wide()?115:106)/3f;}
    static void apply(UIElement root){
        root.lss("position","absolute");root.getLayout().left(0).top(0);
        root.transform(t->t.pivot(0,0).scale(3f/(float)Minecraft.getInstance().getWindow().getGuiScale()));
    }
}
