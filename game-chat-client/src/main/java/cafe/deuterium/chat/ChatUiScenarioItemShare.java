package cafe.deuterium.chat;

import com.lowdragmc.lowdraglib2.registry.RegistrationEnvironment;
import com.lowdragmc.lowdraglib2.registry.annotation.LDLRegisterClient;
import com.lowdragmc.lowdraglib2.uitest.*;
import com.google.gson.JsonObject;
import net.minecraft.client.Minecraft;
import net.minecraft.network.chat.*;
import net.minecraft.world.item.Items;
import java.time.Instant;

@LDLRegisterClient(name="deuterium_chat_item",group="deuterium",registry=UIScenario.REGISTRY,environment=RegistrationEnvironment.DEV_ONLY)
public final class ChatUiScenarioItemShare implements UIScenario {
    private ChatUiFixture fixture;
    private Component rich(){return Component.literal("xiwanzi: [").append(Component.translatable("block.minecraft.brown_terracotta").withStyle(s->s.withHoverEvent(new HoverEvent(HoverEvent.Action.SHOW_ITEM,new HoverEvent.ItemStackInfo(Items.BROWN_TERRACOTTA.getDefaultInstance()))).withClickEvent(new ClickEvent(ClickEvent.Action.RUN_COMMAND,"/trchat item fixture")))).append(" x1]");}
    private void nativeMessage(){ChatClient.INSTANCE.nativeMessage(new net.neoforged.neoforge.client.event.ClientChatReceivedEvent.System(rich(),false));}
    private void structured(String id){var p=new JsonObject();p.addProperty("type","message");p.addProperty("messageId",id);p.addProperty("channel","world");p.addProperty("serverId","amiya");p.addProperty("sentAt",Instant.now().toString());p.addProperty("legacyPlain","xiwanzi: [block.minecraft.brown_terracotta x1]");p.addProperty("content","[block.minecraft.brown_terracotta x1]");var sender=new JsonObject();sender.addProperty("gameId","xiwanzi");sender.addProperty("playerRef","fixture-player");p.add("sender",sender);ChatClient.INSTANCE.receive(p.toString());}
    private boolean singleRich(){var rows=ChatStore.INSTANCE.hud();return rows.size()==1&&rows.getFirst().content.getString().equals("["+Component.translatable("block.minecraft.brown_terracotta").getString()+" x1]")&&ChatLegacy.hasItem(rows.getFirst().content);}
    @Override public void configure(ScenarioOptions options){options.guiScale(3).scenarioTimeoutMs(120000);}
    @Override public void define(ScenarioBuilder s){
        s.fill(new net.minecraft.core.BlockPos(-20,63,-20),new net.minecraft.core.BlockPos(20,63,20),net.minecraft.world.level.block.Blocks.GRASS_BLOCK.defaultBlockState())
         .teleportPlayer(0,65,0,0,0).ticks(20).step("item fixture",ctx->{Minecraft.getInstance().options.hideGui=false;ChatStore.INSTANCE.clear();try{fixture=new ChatUiFixture();}catch(Exception e){throw new RuntimeException(e);}Minecraft.getInstance().player.setItemInHand(net.minecraft.world.InteractionHand.MAIN_HAND,Items.BROWN_TERRACOTTA.getDefaultInstance());})
         .step("native arrives first",ctx->{nativeMessage();structured("item-first");}).check("one localized rich item when native arrives first",ctx->singleRich())
         .step("bridge arrives first",ctx->{ChatStore.INSTANCE.clear();structured("bridge-first");nativeMessage();}).check("one localized rich item when bridge arrives first",ctx->singleRich())
         .step("native already displayed",ctx->{ChatStore.INSTANCE.clear();nativeMessage();}).ticks(16)
         .step("late bridge",ctx->structured("late-bridge")).check("displayed duplicate removed",ctx->singleRich())
         .openScreen("item bubble",ctx->new ChatScreenFull()).awaitScreen(ChatScreenFull.class).frames(25).screenshot("item-bubble")
         .step("share options",ctx->ChatPanels.more(((ChatScreenFull)Minecraft.getInstance().screen).view.root)).frames(15).screenshot("share-options")
         .clickAt(230/3,650/3,0).check("single current position",ctx->ChatPanels.state(((ChatScreenFull)Minecraft.getInstance().screen).view.root).equals("location")).frames(15).screenshot("current-position")
         .step("private share is not redirected to world",ctx->{ChatStore.INSTANCE.current="private:fixture";}).check("item share refuses private channel",ctx->!ChatClient.INSTANCE.shareItem())
         .teardown("close fixture",ctx->{if(fixture!=null)fixture.close();});
    }
}
