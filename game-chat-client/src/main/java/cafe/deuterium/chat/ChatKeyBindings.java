package cafe.deuterium.chat;

import com.mojang.blaze3d.platform.InputConstants;
import net.minecraft.client.KeyMapping;
import net.minecraft.client.Minecraft;
import net.neoforged.neoforge.client.settings.KeyModifier;
import org.lwjgl.glfw.GLFW;
import java.nio.file.Files;

final class ChatKeyBindings {
    private static boolean checked;
    static void migrate(){
        var mc=Minecraft.getInstance();if(checked||mc.options==null)return;checked=true;
        var marker=mc.gameDirectory.toPath().resolve("config/deuterium-chat/keybindings-v1");
        if(Files.exists(marker))return;
        var key=InputConstants.Type.KEYSYM.getOrCreate(GLFW.GLFW_KEY_T);
        mc.options.keyChat.setKeyModifierAndCode(KeyModifier.NONE,key);
        for(var binding:mc.options.keyMappings)
            if(binding.getName().equals("key.sparkle_morpher.animation_roulette.desc"))binding.setKeyModifierAndCode(KeyModifier.ALT,key);
        KeyMapping.resetMapping();mc.options.save();
        try{Files.createDirectories(marker.getParent());Files.writeString(marker,"T=chat; Alt+T=animation roulette; Enter=full chat\n");}
        catch(java.io.IOException failure){org.slf4j.LoggerFactory.getLogger(ChatKeyBindings.class).warn("Chat keybinding migration could not be recorded; settings were applied.");}
    }
}
