package cafe.deuterium.chat;

import net.minecraft.client.Minecraft;
import net.minecraft.client.resources.sounds.SimpleSoundInstance;
import net.minecraft.sounds.SoundEvent;

final class ChatSounds {
    static boolean enabled=true;
    private static String last="";private static long at;
    private static long receivedAt;
    static void received(ChatMessage message){
        var mc=Minecraft.getInstance();
        if(!enabled||mc.player==null||message.system||message.pending||message.receivedAt==0)return;
        if(message.senderId.equals(mc.player.getUUID().toString())||message.name.equalsIgnoreCase(mc.player.getGameProfile().getName())
                ||(!ChatStore.INSTANCE.selfRef.isBlank()&&message.senderId.equals(ChatStore.INSTANCE.selfRef)))return;
        if(System.currentTimeMillis()-message.sentMillis>30000)return;
        long now=System.nanoTime();if(now-receivedAt<650_000_000L)return;receivedAt=now;play("receive");
    }
    static void play(String role){
        if(!enabled||role.isEmpty())return;long now=System.nanoTime();if(last.equals(role)&&now-at<30_000_000)return;last=role;at=now;
        float volume=switch(role){case "open"->.22f;case "send"->.6f;case "receive"->.28f;case "card","panel"->.3f;case "close"->.12f;default->.18f;};
        Minecraft.getInstance().getSoundManager().play(SimpleSoundInstance.forUI(SoundEvent.createVariableRangeEvent(DeuteriumChat.id("ui."+role)),1,volume));
    }
}
