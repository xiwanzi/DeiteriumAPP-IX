package cafe.deuterium.chat;

import com.lowdragmc.lowdraglib2.gui.hud.ModularHudLayer;
import com.lowdragmc.lowdraglib2.gui.ui.ModularUI;
import com.lowdragmc.lowdraglib2.gui.ui.UI;
import net.minecraft.client.Minecraft;
import net.minecraft.client.KeyMapping;
import net.minecraft.client.gui.components.Button;
import net.minecraft.client.gui.screens.options.controls.KeyBindsScreen;
import net.minecraft.network.chat.Component;
import net.neoforged.neoforge.client.settings.KeyConflictContext;
import com.mojang.blaze3d.platform.InputConstants;
import net.minecraft.client.gui.screens.ChatScreen;
import net.minecraft.resources.ResourceLocation;
import net.neoforged.api.distmarker.Dist;
import net.neoforged.bus.api.IEventBus;
import net.neoforged.fml.common.Mod;
import net.neoforged.neoforge.common.NeoForge;
import net.neoforged.neoforge.client.event.*;
import net.neoforged.neoforge.network.event.RegisterPayloadHandlersEvent;
import org.lwjgl.glfw.GLFW;

@Mod(value="deuterium_chat", dist=Dist.CLIENT)
public final class DeuteriumChat {
    public static ResourceLocation id(String path) { return ResourceLocation.fromNamespaceAndPath("deuterium_chat",path); }
    private ModularUI hud;
    private final KeyMapping fullChat=new KeyMapping("key.deuterium_chat.full",KeyConflictContext.IN_GAME,
        InputConstants.Type.KEYSYM,GLFW.GLFW_KEY_ENTER,"key.categories.multiplayer");
    public DeuteriumChat(IEventBus bus) {
        bus.addListener(this::layers);
        bus.addListener(ChatBackdrop::register);
        bus.addListener(RoundTexture::register);
        bus.addListener(MaskTexture::register);
        bus.addListener(this::network);
        bus.addListener((RegisterKeyMappingsEvent e)->e.register(fullChat));
        NeoForge.EVENT_BUS.addListener(this::keyScreen);
        NeoForge.EVENT_BUS.addListener(this::chat);
        NeoForge.EVENT_BUS.addListener(this::hideVanilla);
        NeoForge.EVENT_BUS.addListener(this::tick);
        NeoForge.EVENT_BUS.addListener(this::logout);
        NeoForge.EVENT_BUS.addListener(net.neoforged.bus.api.EventPriority.LOWEST,VanillaMusicGuard::filter);
    }
    private void network(RegisterPayloadHandlersEvent e) {
        e.registrar("1").optional().playBidirectional(ChatPayload.TYPE,ChatPayload.CODEC,
                (p,ctx)->{if(ctx.flow()==net.minecraft.network.protocol.PacketFlow.CLIENTBOUND)ctx.enqueueWork(()->ChatClient.INSTANCE.receive(p.json()));});
    }
    private void layers(RegisterGuiLayersEvent e) {
        e.registerAboveAll(id("chat_hud"),(ModularHudLayer)()-> {
            Minecraft mc=Minecraft.getInstance();
            if(mc.level==null||mc.options.hideGui||mc.screen instanceof ChatScreenFull||mc.screen instanceof MediaViewer)return null;
            if(hud==null)hud=ModularUI.of(UI.of(new ChatTimeline(true),size->size));
            return hud;
        });
    }
    private void keyScreen(ScreenEvent.Init.Post e) {
        if(!(e.getScreen() instanceof KeyBindsScreen screen))return;
        CloudKeybindings.INSTANCE.prepare();
        CloudKeybindings.INSTANCE.wake();
        e.addListener(Button.builder(Component.literal("云键位"),b->Minecraft.getInstance().setScreen(new KeybindingScreen(screen)))
            .bounds(4,4,60,20).build());
    }
    private void chat(ClientChatReceivedEvent e) {
        if(Minecraft.getInstance().level==null)return;
        ChatClient.INSTANCE.nativeMessage(e);
    }
    private void hideVanilla(RenderGuiLayerEvent.Pre e) {
        if(e.getName().equals(net.neoforged.neoforge.client.gui.VanillaGuiLayers.CHAT))e.setCanceled(true);
    }
    private void tick(ClientTickEvent.Post e) {
        var mc=Minecraft.getInstance();
        while(fullChat.consumeClick())if(mc.level!=null && mc.screen==null)mc.setScreen(new ChatScreenFull());
        ChatKeyBindings.migrate();MinimapPlacement.migrate();ChatClient.INSTANCE.tick();CloudKeybindings.INSTANCE.tick();
    }
    private void logout(ClientPlayerNetworkEvent.LoggingOut e) { ChatClient.INSTANCE.reset();hud=null; }
}
