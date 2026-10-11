package cafe.deuterium.chat;

import com.lowdragmc.lowdraglib2.registry.RegistrationEnvironment;
import com.lowdragmc.lowdraglib2.registry.annotation.LDLRegisterClient;
import com.lowdragmc.lowdraglib2.uitest.*;
import net.minecraft.client.Minecraft;
import net.minecraft.client.gui.screens.ChatScreen;
import net.minecraft.network.chat.Component;
import java.time.Instant;

@LDLRegisterClient(name="deuterium_chat_readability",group="deuterium",registry=UIScenario.REGISTRY,environment=RegistrationEnvironment.DEV_ONLY)
public final class ChatUiScenarioReadability implements UIScenario {
    private ChatUiFixture fixture;
    @Override public void configure(ScenarioOptions options){options.guiScale(Integer.getInteger("deuterium.chat.test.guiScale",3)).scenarioTimeoutMs(120000);}
    @Override public void define(ScenarioBuilder s){
        s.fill(new net.minecraft.core.BlockPos(-20,63,-20),new net.minecraft.core.BlockPos(20,63,20),net.minecraft.world.level.block.Blocks.GRASS_BLOCK.defaultBlockState())
         .teleportPlayer(0,65,0,0,0).ticks(20).step("readability messages",ctx->{
            Minecraft.getInstance().options.hideGui=false;ChatStore.INSTANCE.clear();ChatStore.INSTANCE.current="world";ChatStore.INSTANCE.server="amiya";
            Minecraft.getInstance().getTutorial().setStep(net.minecraft.client.tutorial.TutorialSteps.NONE);
            try{fixture=new ChatUiFixture();}catch(Exception e){throw new RuntimeException(e);}
            String[] messages={"Hi，晚上好！","这行用来检查文字在气泡中的上下位置。","Agjpqy 0123456789：中文、英文与标点。","小窗口保持可读字号，少显示几条消息。","长消息会正常换行，不挤出气泡，也不会把字号缩得很小。","马上调页面尺寸（测试）","一起去探索村庄吗？","收到，准备出发！","这是一条需要保留完整内容的长消息：小屏幕优先保证每条信息清楚可读，再通过滚动查看前面的聊天记录。"};
            for(int i=0;i<messages.length;i++)ChatStore.INSTANCE.add(new ChatMessage("reading_"+i,"",i%2==0?"LongPlayerName16":"xiwanzi","world","amiya","",Component.literal(messages[i]),null,Instant.now(),false,false,""));
         }).check("T remains quick chat",ctx->Minecraft.getInstance().options.keyChat.getKey().getValue()==org.lwjgl.glfw.GLFW.GLFW_KEY_T)
         .waitUntil("minimap moved to upper right",ctx->{try{var module=Class.forName("xaero.hud.minimap.BuiltInHudModules").getField("MINIMAP").get(null);var transform=module.getClass().getMethod("getConfirmedTransform").invoke(module);return transform.getClass().getField("fromRight").getBoolean(transform)&&!transform.getClass().getField("fromBottom").getBoolean(transform);}catch(ClassNotFoundException absent){return true;}catch(Exception e){throw new RuntimeException(e);}})
         .check("vanilla music blocked and custom audio retained",ctx->{
             for(String id:new String[]{"minecraft:music.menu","minecraft:music.game","minecraft:music.nether.nether_wastes","minecraft:music.end","minecraft:music.dragon","minecraft:music.creative","deuterium_bgm:day.sample","minecraft:music_disc.13","deuterium_chat:ui.send"}){
                 var sound=net.minecraft.client.resources.sounds.SimpleSoundInstance.forMusic(net.minecraft.sounds.SoundEvent.createVariableRangeEvent(net.minecraft.resources.ResourceLocation.parse(id)));
                 var event=new net.neoforged.neoforge.client.event.sound.PlaySoundEvent(null,sound);net.neoforged.neoforge.common.NeoForge.EVENT_BUS.post(event);
                 if((event.getSound()==null)!=id.startsWith("minecraft:music."))return false;
             }return true;
         }).openScreen("readable full chat",ctx->new ChatScreenFull()).awaitScreen(ChatScreenFull.class)
         .waitUntil("offline portrait ready",ctx->ChatAvatars.get("xiwanzi",null).portrait!=null)
         .frames(40).check("composer stays in window",ctx->{var input=((ChatScreenFull)Minecraft.getInstance().screen).view.input;return input.getSizeWidth()*3>=180&&(input.getPositionX()+input.getSizeWidth())*3<=Minecraft.getInstance().getWindow().getWidth();})
         .screenshot("full-readability").openScreen("quick chat",ctx->new ChatScreen(""))
         .frames(30).screenshot("hud-readability").teardown("close fixture",ctx->{if(fixture!=null)fixture.close();});
    }
}
