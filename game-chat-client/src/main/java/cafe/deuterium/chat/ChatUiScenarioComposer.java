package cafe.deuterium.chat;

import com.lowdragmc.lowdraglib2.registry.RegistrationEnvironment;
import com.lowdragmc.lowdraglib2.registry.annotation.LDLRegisterClient;
import com.lowdragmc.lowdraglib2.uitest.*;
import net.minecraft.client.Minecraft;
import net.minecraft.network.chat.Component;
import org.lwjgl.glfw.GLFW;
import java.time.Instant;
import java.util.List;

@LDLRegisterClient(name="deuterium_chat_composer",group="deuterium",registry=UIScenario.REGISTRY,environment=RegistrationEnvironment.DEV_ONLY)
public final class ChatUiScenarioComposer implements UIScenario {
    private ChatUiFixture fixture;
    private static ChatView view(){return ((ChatScreenFull)Minecraft.getInstance().screen).view;}
    private static void append(String scope,String id){
        boolean direct=scope.startsWith("private:");
        ChatStore.INSTANCE.add(new ChatMessage(id,"player-fixture","Amiya",direct?"private":scope,"amiya",direct?"fixture-composer":"",Component.literal("聊天记录 · "+id+"，向上查看时保留阅读位置。"),null,Instant.now(),true,false,""));
    }
    @Override public void configure(ScenarioOptions options){options.guiScale(Integer.getInteger("deuterium.chat.test.guiScale",3)).scenarioTimeoutMs(90000);}
    @Override public void define(ScenarioBuilder s){
        s.fill(new net.minecraft.core.BlockPos(-16,63,-16),new net.minecraft.core.BlockPos(16,63,16),net.minecraft.world.level.block.Blocks.GRASS_BLOCK.defaultBlockState())
         .teleportPlayer(0,65,0,0,0).ticks(15).step("isolated chat fixture",ctx->{
            Minecraft.getInstance().options.hideGui=false;
            Minecraft.getInstance().getTutorial().setStep(net.minecraft.client.tutorial.TutorialSteps.NONE);
            ChatStore.INSTANCE.clear();ChatStore.INSTANCE.current="world";ChatStore.INSTANCE.server="amiya";
            try{fixture=new ChatUiFixture();}catch(Exception error){throw new RuntimeException(error);}
            ChatStore.INSTANCE.contacts.put("private:fixture-composer",new ChatStore.Contact("private:fixture-composer","Amiya","player-fixture",true,0));
            for(String scope:List.of("world","nearby","team","private:fixture-composer"))for(int i=0;i<40;i++)append(scope,scope+"-"+i);
         }).openScreen("detailed chat",ctx->new ChatScreenFull(false)).awaitScreen(ChatScreenFull.class).frames(8)
         .check("latest arrow hidden at bottom",ctx->!view().latest.isDisplayed())
         .step("remove input focus",ctx->view().input.blur()).key(GLFW.GLFW_KEY_ENTER)
         .check("Enter focuses composer",ctx->view().input.isFocused()).type("keyboard draft")
         .step("leave draft unfocused",ctx->view().input.blur()).keyDown(GLFW.GLFW_KEY_ENTER)
         .check("first Enter only focuses existing draft",ctx->view().input.isFocused()&&view().input.getValue().equals("keyboard draft"))
         .step("held Enter must not send",ctx->Minecraft.getInstance().screen.keyPressed(GLFW.GLFW_KEY_ENTER,0,0))
         .check("repeat preserves draft",ctx->view().input.getValue().equals("keyboard draft")).keyUp(GLFW.GLFW_KEY_ENTER)
         .key(GLFW.GLFW_KEY_KP_ENTER).check("focused keypad Enter sends once and keeps focus",ctx->view().input.getValue().isEmpty()&&view().input.isFocused());
        for(String scope:List.of("world","nearby","team","private:fixture-composer")){
            String name=scope.startsWith("private:")?"private":scope;
            s.step("switch "+name,ctx->view().switchScope(scope)).frames(6)
             .scroll("#chat-timeline",5).frames(8)
             .check("arrow shown in "+name,ctx->view().timeline.awayFromLatest()&&view().latest.isDisplayed())
             .step("incoming while reading "+name,ctx->append(scope,"incoming-"+name)).frames(3)
             .check("new message does not pull reader down "+name,ctx->view().timeline.awayFromLatest())
             .hover("#chat-jump-latest").screenshot("composer-away-"+name)
             .step("press latest "+name,ctx->{var bounds=ctx.el("#chat-jump-latest").bounds();ctx.put("jump-x",bounds.centerX());ctx.put("jump-y",bounds.centerY());ctx.input().mouseDown(bounds.centerX(),bounds.centerY(),0);})
             .step("release latest "+name,ctx->ctx.input().mouseUp(ctx.get("jump-x",0f),ctx.get("jump-y",0f),0))
             .check("jump is immediate "+name,ctx->!view().timeline.awayFromLatest()).frames(2)
             .check("arrow hidden after jump "+name,ctx->!view().latest.isDisplayed()&&view().input.isFocused());
        }
        s.screenshot("composer-latest")
         .step("inbox has no composer",ctx->view().switchScope("inbox")).key(GLFW.GLFW_KEY_ENTER)
         .check("inbox does not expose hidden input",ctx->!view().input.isDisplayed()&&!view().latest.isDisplayed())
         .teardown("close local fixture",ctx->{if(fixture!=null)fixture.close();});
    }
}
