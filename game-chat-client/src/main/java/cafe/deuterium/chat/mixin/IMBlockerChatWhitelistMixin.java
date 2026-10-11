package cafe.deuterium.chat.mixin;

import cafe.deuterium.chat.ChatScreenFull;
import org.spongepowered.asm.mixin.Mixin;
import org.spongepowered.asm.mixin.Pseudo;
import org.spongepowered.asm.mixin.injection.At;
import org.spongepowered.asm.mixin.injection.Inject;
import org.spongepowered.asm.mixin.injection.callback.CallbackInfoReturnable;

/** Narrow optional compatibility: exempt this chat screen, not every LDLib2 screen. */
@Pseudo
@Mixin(targets="io.github.reserveword.imblocker.common.IMBlockerConfig",remap=false)
public abstract class IMBlockerChatWhitelistMixin {
    @Inject(method="isScreenInWhitelist",at=@At("HEAD"),cancellable=true,remap=false)
    private void deuteriumChatAllowsIME(Object screen,CallbackInfoReturnable<Boolean> result){
        if(screen instanceof ChatScreenFull)result.setReturnValue(true);
    }
}
