package cafe.deuterium.chat.mixin;

import cafe.deuterium.chat.QuickChatHooks;
import net.minecraft.client.gui.GuiGraphics;
import net.minecraft.client.gui.components.ChatComponent;
import net.minecraft.client.gui.components.CommandSuggestions;
import net.minecraft.client.gui.screens.ChatScreen;
import org.spongepowered.asm.mixin.Mixin;
import org.spongepowered.asm.mixin.Shadow;
import org.spongepowered.asm.mixin.injection.*;
import org.spongepowered.asm.mixin.injection.callback.*;

@Mixin(ChatScreen.class)
public abstract class ChatScreenMixin {
    @Shadow CommandSuggestions commandSuggestions;
    @org.spongepowered.asm.mixin.Unique private boolean deuteriumSoundOpened;
    @Inject(method="init",at=@At("TAIL"))
    private void openSound(CallbackInfo ci){if(!deuteriumSoundOpened){deuteriumSoundOpened=true;QuickChatHooks.sound("click");}}
    @Inject(method="removed",at=@At("HEAD"))
    private void closeSound(CallbackInfo ci){QuickChatHooks.sound("close");}
    @Redirect(method="render",at=@At(value="INVOKE",target="Lnet/minecraft/client/gui/GuiGraphics;fill(IIIII)V"))
    private void glass(GuiGraphics g,int left,int top,int right,int bottom,int color){QuickChatHooks.backdrop(g);}
    @Redirect(method="render",at=@At(value="INVOKE",target="Lnet/minecraft/client/gui/components/ChatComponent;render(Lnet/minecraft/client/gui/GuiGraphics;IIIZ)V"))
    private void retainedHud(ChatComponent chat,GuiGraphics g,int tick,int x,int y,boolean focused) {}
    @Inject(method="render",at=@At("TAIL"))
    private void hover(GuiGraphics g,int x,int y,float tick,CallbackInfo ci){QuickChatHooks.hover(g,x,y);}
    @Inject(method="mouseClicked",at=@At("HEAD"),cancellable=true)
    private void click(double x,double y,int button,CallbackInfoReturnable<Boolean> ci){if(QuickChatHooks.click(x,y,button))ci.setReturnValue(true);}
    @Inject(method="mouseScrolled",at=@At("HEAD"),cancellable=true)
    private void wheel(double x,double y,double dx,double dy,CallbackInfoReturnable<Boolean> ci){
        if(commandSuggestions.mouseScrolled(dy)){ci.setReturnValue(true);return;}
        QuickChatHooks.scroll(dy);ci.setReturnValue(true);
    }
    @Inject(method="handleChatInput",at=@At("HEAD"),cancellable=true)
    private void send(String value,boolean history,CallbackInfo ci){if(QuickChatHooks.send(value,history))ci.cancel();}
}
