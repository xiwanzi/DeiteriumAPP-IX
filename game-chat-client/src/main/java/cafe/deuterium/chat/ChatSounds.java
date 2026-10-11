package cafe.deuterium.chat;

import net.minecraft.client.Minecraft;
import net.minecraft.client.resources.sounds.SimpleSoundInstance;
import net.minecraft.sounds.SoundEvent;

final class ChatSounds {
    static boolean enabled=true;
    private static String last="";private static long at;
    static void play(String role){
        if(!enabled||role.isEmpty())return;long now=System.nanoTime();if(last.equals(role)&&now-at<30_000_000)return;last=role;at=now;
        float volume=switch(role){case "open"->.22f;case "send"->.6f;case "card","panel"->.3f;case "close"->.12f;default->.18f;};
        Minecraft.getInstance().getSoundManager().play(SimpleSoundInstance.forUI(SoundEvent.createVariableRangeEvent(DeuteriumChat.id("ui."+role)),1,volume));
    }
}
