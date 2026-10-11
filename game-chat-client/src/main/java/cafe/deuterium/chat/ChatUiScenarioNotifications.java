package cafe.deuterium.chat;

import com.lowdragmc.lowdraglib2.registry.RegistrationEnvironment;
import com.lowdragmc.lowdraglib2.registry.annotation.LDLRegisterClient;
import com.lowdragmc.lowdraglib2.uitest.*;
import net.minecraft.client.Minecraft;
import net.minecraft.client.gui.screens.ChatScreen;
import net.minecraft.network.chat.*;
import net.neoforged.neoforge.client.event.sound.PlaySoundEvent;
import net.neoforged.neoforge.common.NeoForge;
import java.time.Instant;
import java.util.Optional;

@LDLRegisterClient(name="deuterium_chat_notifications",group="deuterium",registry=UIScenario.REGISTRY,environment=RegistrationEnvironment.DEV_ONLY)
public final class ChatUiScenarioNotifications implements UIScenario {
    private int sounds;
    private final java.util.function.Consumer<PlaySoundEvent> listener=event->{if(event.getOriginalSound().getLocation().equals(DeuteriumChat.id("ui.receive"))){sounds++;event.setSound(null);}};
    private ChatMessage message(String id,String sender,String name,boolean history,boolean system,Instant sent){return new ChatMessage(id,sender,name,"world","amiya","",Component.literal("测试消息"),null,sent,history,system,"");}
    private Component mail(){return Component.literal("#47 · ").withStyle(s->s.withColor(0x555555))
        .append(Component.literal("官方商城订单").withStyle(s->s.withColor(0xFF55FF)))
        .append(Component.literal(" — 购买的物品已投递，请领取附件。 ").withStyle(s->s.withColor(0x777777)))
        .append(Component.literal("[已领取] 1 项附件").withStyle(s->s.withColor(0x333333).withClickEvent(new ClickEvent(ClickEvent.Action.RUN_COMMAND,"/dmail view 47")).withHoverEvent(new HoverEvent(HoverEvent.Action.SHOW_TEXT,Component.literal("查看邮件")))));}
    @Override public void configure(ScenarioOptions options){options.guiScale(3).scenarioTimeoutMs(120000);}
    @Override public void define(ScenarioBuilder s){
        s.fill(new net.minecraft.core.BlockPos(-20,63,-20),new net.minecraft.core.BlockPos(20,63,20),net.minecraft.world.level.block.Blocks.WHITE_CONCRETE.defaultBlockState())
         .teleportPlayer(0,65,0,0,-10).ticks(20).step("listen for receive sound",ctx->{ChatStore.INSTANCE.clear();ChatSounds.enabled=true;NeoForge.EVENT_BUS.addListener(listener);Minecraft.getInstance().options.hideGui=false;ChatStore.INSTANCE.selfRef="own-ref";})
         .step("own history and system messages stay quiet",ctx->{var mc=Minecraft.getInstance();ChatStore.INSTANCE.add(message("self-uuid",mc.player.getUUID().toString(),"other",false,false,Instant.now()));ChatStore.INSTANCE.add(message("self-ref","own-ref","other",false,false,Instant.now()));ChatStore.INSTANCE.add(message("self-name","other",mc.player.getGameProfile().getName(),false,false,Instant.now()));ChatStore.INSTANCE.add(message("history","other","Other",true,false,Instant.now()));ChatStore.INSTANCE.add(message("system","","",false,true,Instant.now()));})
         .check("no incoming chime for own history or system output",ctx->sounds==0)
         .step("incoming burst",ctx->{var first=message("other-1","other","Other",false,false,Instant.now());ChatStore.INSTANCE.add(first);ChatStore.INSTANCE.add(first);ChatStore.INSTANCE.replace(first.id,first);ChatStore.INSTANCE.add(message("other-2","other","Other",false,false,Instant.now()));})
         .check("one chime for burst and no duplicate chime",ctx->sounds==1).ticks(16)
         .step("next incoming message",ctx->ChatStore.INSTANCE.add(message("other-3","other","Other",false,false,Instant.now())))
         .check("next message can notify",ctx->sounds==2).ticks(16)
         .step("mute and stale input",ctx->{ChatSounds.enabled=false;ChatStore.INSTANCE.add(message("muted","other","Other",false,false,Instant.now()));ChatSounds.enabled=true;ChatStore.INSTANCE.add(message("stale","other","Other",false,false,Instant.now().minusSeconds(120)));})
         .check("mute and stale history remain quiet",ctx->sounds==2)
         .check("system contrast preserves interaction",ctx->{var text=ChatStyle.systemBody(mail());return text.visit((style,value)->value.contains("已领取")?Optional.of(style.getColor().getValue()!=0x333333&&style.getClickEvent()!=null&&style.getHoverEvent()!=null):Optional.empty(),Style.EMPTY).orElse(false);})
         .step("mail list fixture",ctx->{ChatStore.INSTANCE.clear();for(int i=0;i<6;i++)ChatStore.INSTANCE.add(new ChatMessage("mail-"+i,"","","world","amiya","",mail(),null,Instant.now(),false,true,""));})
         .openScreen("system HUD",ctx->new ChatScreen("")).frames(25).screenshot("system-hud")
         .openScreen("system full chat",ctx->new ChatScreenFull()).frames(25).screenshot("system-full")
         .teardown("sound listener",ctx->NeoForge.EVENT_BUS.unregister(listener));
    }
}
