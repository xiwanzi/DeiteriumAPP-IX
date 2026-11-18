package cafe.deuterium.chat.mixin;

import cafe.deuterium.chat.CloudKeybindings;

import net.minecraft.client.Options;

import org.spongepowered.asm.mixin.Mixin;
import org.spongepowered.asm.mixin.injection.At;
import org.spongepowered.asm.mixin.injection.Inject;
import org.spongepowered.asm.mixin.injection.callback.CallbackInfo;

@Mixin(Options.class)
abstract class KeybindingSaveMixin {
    @Inject(method = "save", at = @At("RETURN"))
    private void deuterium$keybindingsSaved(CallbackInfo ci) {
        CloudKeybindings.optionsSaved();
    }
}
