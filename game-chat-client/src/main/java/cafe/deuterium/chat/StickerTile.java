package cafe.deuterium.chat;
import com.lowdragmc.lowdraglib2.gui.ui.UIElement;
import com.lowdragmc.lowdraglib2.gui.ui.event.UIEvents;
import com.lowdragmc.lowdraglib2.gui.texture.ColorRectTexture;
/** Release-to-send and 550 ms hold-to-preview, including drag-away cancellation. */
final class StickerTile extends UIElement {
    private boolean held;private long since;
    StickerTile(Runnable send,Runnable preview){
        addEventListener(UIEvents.MOUSE_DOWN,e->{if(e.button==1){held=false;ChatSounds.play("card");preview.run();}else if(e.button==0){held=true;since=System.nanoTime();}e.stopPropagation();});
        addEventListener(UIEvents.MOUSE_UP,e->{if(e.button==0&&held){held=false;if(System.nanoTime()-since>=550_000_000){ChatSounds.play("card");preview.run();}else {ChatSounds.play("card");send.run();}}e.stopPropagation();});
        addEventListener(UIEvents.TICK,e->{if(held&&System.nanoTime()-since>=550_000_000){held=false;ChatSounds.play("card");preview.run();}});
        addEventListener(UIEvents.MOUSE_ENTER,e->style(s->s.background(new ColorRectTexture(0x205A6761))));
        addEventListener(UIEvents.MOUSE_LEAVE,e->{held=false;style(s->s.background(com.lowdragmc.lowdraglib2.gui.texture.IGuiTexture.EMPTY));});
        addEventListener(UIEvents.REMOVED,e->held=false);
    }
}
