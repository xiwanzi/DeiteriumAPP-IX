package cafe.deuterium.chat.mixin;
import cafe.deuterium.chat.QuickChatHooks;
import net.minecraft.client.gui.GuiGraphics;
import net.minecraft.client.renderer.Rect2i;
import org.spongepowered.asm.mixin.*;
import org.spongepowered.asm.mixin.injection.*;
import org.spongepowered.asm.mixin.injection.callback.CallbackInfo;
/** Keep native Brigadier selection, scrolling and completion; change only its backing material. */
@Mixin(targets="net.minecraft.client.gui.components.CommandSuggestions$SuggestionsList")
public abstract class CommandSuggestionsMixin {
    @Shadow @Final private Rect2i rect;
    @Inject(method="render",at=@At("HEAD"))
    private void glass(GuiGraphics g,int x,int y,CallbackInfo ci){QuickChatHooks.completionGlass(g,rect.getX(),rect.getY()-1,rect.getWidth(),rect.getHeight()+2);}
    @Redirect(method="render",at=@At(value="INVOKE",target="Lnet/minecraft/client/gui/GuiGraphics;fill(IIIII)V"))
    private void quietBackground(GuiGraphics g,int x1,int y1,int x2,int y2,int color){if(color==-1)g.fill(x1,y1,x2,y2,color);}
}
