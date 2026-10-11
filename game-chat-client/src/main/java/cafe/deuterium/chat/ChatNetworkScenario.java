package cafe.deuterium.chat;

import com.lowdragmc.lowdraglib2.registry.RegistrationEnvironment;
import com.lowdragmc.lowdraglib2.registry.annotation.LDLRegisterClient;
import com.lowdragmc.lowdraglib2.uitest.*;
import net.minecraft.client.Minecraft;
import net.minecraft.client.gui.screens.*;
import net.minecraft.client.multiplayer.ServerData;
import net.minecraft.client.multiplayer.resolver.ServerAddress;
import java.nio.file.*;

/** Explicit loopback QA only. Excluded from the runtime jar. */
@LDLRegisterClient(name="deuterium_chat_network",group="deuterium",registry=UIScenario.REGISTRY,environment=RegistrationEnvironment.DEV_ONLY)
public final class ChatNetworkScenario implements UIScenario {
    @Override public void configure(ScenarioOptions o){o.requiresWorld(false).guiScale(3).defaultTimeoutMs(60000).scenarioTimeoutMs(240000);}
    @Override public void define(ScenarioBuilder s){
        s.step("connect isolated Youer",ctx->{var mc=Minecraft.getInstance();ConnectScreen.startConnecting(new TitleScreen(),mc,ServerAddress.parseString("127.0.0.1:25593"),new ServerData("Isolated chat QA","127.0.0.1:25593",ServerData.Type.OTHER),false,null);})
         .waitUntil("game player joined",ctx->Minecraft.getInstance().player!=null)
         .waitUntil("Core-issued scoped session",ctx->ChatClient.INSTANCE.api!=null&&ChatClient.INSTANCE.api.valid())
         .step("native world text",ctx->{Minecraft.getInstance().options.hideGui=false;ChatStore.INSTANCE.current="world";ChatClient.INSTANCE.send("QA 原生游戏正文，不重复玩家前缀");})
         .waitUntil("structured echo from Core",ctx->ChatStore.INSTANCE.hud().stream().anyMatch(m->m.id.startsWith("msg_")&&m.name.equals("Dev")&&m.content.getString().equals("QA 原生游戏正文，不重复玩家前缀")))
         .openScreen("network chat",ctx->new ChatScreenFull())
         .step("native codec and signed upload",ctx->ChatClient.INSTANCE.media.builtIn(2))
         .waitUntil("Go accepted image descriptor",ctx->ChatStore.INSTANCE.hud().stream().anyMatch(m->m.id.startsWith("msg_")&&m.media!=null))
         .waitUntil("uploaded sticker drawn",ctx->ChatStore.INSTANCE.hud().stream().filter(m->m.media!=null).anyMatch(m->ChatClient.INSTANCE.media.ready(ChatMessage.str(m.media,"mediaId"))))
         .screenshot("real-core-upload")
         .step("save integration evidence",ctx->{try{var out=new com.google.gson.JsonObject();out.addProperty("scope","gamechat");out.addProperty("server",ChatStore.INSTANCE.server);var list=new com.google.gson.JsonArray();for(var m:ChatStore.INSTANCE.hud())if(m.id.startsWith("msg_")){var item=new com.google.gson.JsonObject();item.addProperty("messageId",m.id);item.addProperty("content",m.content.getString());if(m.media!=null)item.add("media",m.media);list.add(item);}out.add("messages",list);Files.writeString(Path.of(System.getProperty("deuterium.chat.test.output")),out.toString());}catch(Exception e){throw new RuntimeException(e);}})
         .frames(60).closeScreen().frames(20).screenshot("real-core-hud");
    }
}
