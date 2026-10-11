package cafe.deuterium.chat;

import net.minecraft.sounds.SoundSource;
import net.neoforged.neoforge.client.event.sound.PlaySoundEvent;

/** Pack policy: no vanilla background tracks, without muting the shared music category. */
final class VanillaMusicGuard {
    static void filter(PlaySoundEvent event){
        var sound=event.getSound();if(sound==null)return;
        var id=sound.getLocation();
        if(sound.getSource()==SoundSource.MUSIC&&id.getNamespace().equals("minecraft")&&id.getPath().startsWith("music."))event.setSound(null);
    }
}
