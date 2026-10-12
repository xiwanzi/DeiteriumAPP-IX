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
    private boolean soundBefore;
    @Override public void configure(ScenarioOptions options){options.guiScale(Integer.getInteger("deuterium.chat.test.guiScale",3)).scenarioTimeoutMs(120000);}
    @Override public void define(ScenarioBuilder s){
        s.fill(new net.minecraft.core.BlockPos(-20,63,-20),new net.minecraft.core.BlockPos(20,63,20),net.minecraft.world.level.block.Blocks.GRASS_BLOCK.defaultBlockState())
         .teleportPlayer(0,65,0,0,0).ticks(20).step("readability messages",ctx->{
            Minecraft.getInstance().options.hideGui=false;ChatStore.INSTANCE.clear();ChatStore.INSTANCE.current="world";ChatStore.INSTANCE.server="amiya";
            while(ChatTextSize.level()!=0)ChatTextSize.cycle();soundBefore=ChatSounds.enabled;
            Minecraft.getInstance().getTutorial().setStep(net.minecraft.client.tutorial.TutorialSteps.NONE);
            try{fixture=new ChatUiFixture();}catch(Exception e){throw new RuntimeException(e);}
            String[] messages={"Hi，晚上好！","这行用来检查文字在气泡中的上下位置。","Agjpqy 0123456789：中文、英文与标点。","小窗口保持可读字号，少显示几条消息。","长消息会正常换行，不挤出气泡，也不会把字号缩得很小。","马上调页面尺寸（测试）","一起去探索村庄吗？","收到，准备出发！","这是一条需要保留完整内容的长消息：小屏幕优先保证每条信息清楚可读，再通过滚动查看前面的聊天记录。","表情混排 😀 ✨ Agjpqy 中文，笔画和表情都留在气泡内。","手动换行第一行：中文与标点。\n第二行 Agjpqy 0123456789，检查下伸笔画。\n最后一行：气泡底部必须保留内边距。"};
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
         .screenshot("full-readability").check("full bubbles contain measured text",ctx->fullBubbles())
         .click("#chat-text-size").frames(12).check("first click gives 110 and contained bubbles",ctx->ChatTextSize.level()==1&&fullBubbles())
         .hover("#chat-text-size-label").frames(4).screenshot("full-size-110").openScreen("quick 110",ctx->new ChatScreen(""))
         .frames(12).check("HUD follows 110 and contains text",ctx->bubblesContainText(ChatTimeline.HUD)).screenshot("hud-size-110")
         .openScreen("full for second click",ctx->new ChatScreenFull(false)).frames(15)
         .click("#chat-text-size").frames(12).check("second click gives 120 and contained bubbles",ctx->ChatTextSize.level()==2&&fullBubbles())
         .hover("#chat-text-size-label").frames(4).screenshot("full-size-120").openScreen("quick 120",ctx->new ChatScreen(""))
         .frames(12).check("HUD follows 120 and contains text",ctx->bubblesContainText(ChatTimeline.HUD)).screenshot("hud-size-120")
         .openScreen("full for third click",ctx->new ChatScreenFull(false)).frames(15)
         .click("#chat-text-size").frames(12).check("third click resets default without muting sound",ctx->ChatTextSize.level()==0&&ChatSounds.enabled==soundBefore&&fullBubbles())
         .openScreen("quick chat",ctx->new ChatScreen(""))
         .frames(30).check("HUD bubbles contain measured text",ctx->bubblesContainText(ChatTimeline.HUD))
         .screenshot("hud-readability").teardown("close fixture",ctx->{if(fixture!=null)fixture.close();});
    }
    private static boolean fullBubbles(){
        try{var f=ChatView.class.getDeclaredField("timeline");f.setAccessible(true);return bubblesContainText((ChatTimeline)f.get(((ChatScreenFull)Minecraft.getInstance().screen).view));}
        catch(ReflectiveOperationException e){throw new IllegalStateException(e);}
    }
    private static Object call(Object value,String name)throws Exception{
        var method=value.getClass().getDeclaredMethod(name);method.setAccessible(true);return method.invoke(value);
    }
    private static boolean bubblesContainText(ChatTimeline timeline){try{
        var field=ChatTimeline.class.getDeclaredField("hits");field.setAccessible(true);
        var hits=(java.util.List<?>)field.get(timeline);if(hits.isEmpty())return false;
        float indent=(float)call(timeline,"fullIndent");
        for(var hit:hits){
            var row=call(hit,"row");var message=(ChatMessage)call(hit,"message");
            var padding=ChatTimeline.class.getDeclaredMethod("inset",ChatMessage.class);padding.setAccessible(true);float pad=(float)padding.invoke(timeline,message);
            if((boolean)call(row,"emoji"))pad=0;
            var scaleMethod=ChatTimeline.class.getDeclaredMethod("rowScale",row.getClass());scaleMethod.setAccessible(true);float scale=(float)scaleMethod.invoke(timeline,row);
            var lineMethod=ChatTimeline.class.getDeclaredMethod("rowLine",row.getClass());lineMethod.setAccessible(true);float line=(float)lineMethod.invoke(timeline,row);
            var vertical=ChatTimeline.class.getDeclaredMethod("vertical",ChatMessage.class,row.getClass());vertical.setAccessible(true);float v=(float)vertical.invoke(timeline,message,row);
            var lines=(java.util.List<?>)call(row,"lines");float width=(float)call(hit,"width");
            for(var text:lines)if(com.lowdragmc.lowdraglib2.gui.LDLibFonts.font().width((net.minecraft.util.FormattedCharSequence)text)*scale+pad*2+(message.system?0:indent)>width+.02f)return false;
            if(lines.size()*line+2*v>(float)call(row,"bodyHeight")+.02f)return false;
            if(9*scale>line)return false;
        }
        return true;
        }catch(Exception e){throw new IllegalStateException(e);}
    }
}
