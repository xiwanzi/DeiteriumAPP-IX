package cafe.deuterium.chat;

import com.lowdragmc.lowdraglib2.registry.RegistrationEnvironment;
import com.lowdragmc.lowdraglib2.registry.annotation.LDLRegisterClient;
import com.lowdragmc.lowdraglib2.uitest.*;
import net.minecraft.network.chat.*;
import net.minecraft.client.gui.screens.ChatScreen;
import java.time.Instant;

/** Synthetic content in an isolated real game world; never registered in production. */
@LDLRegisterClient(name="deuterium_chat",group="deuterium",registry=UIScenario.REGISTRY,environment=RegistrationEnvironment.DEV_ONLY)
public final class ChatUiScenario implements UIScenario {
    private ChatUiFixture fixture;
    private static int testWidth(){return Integer.parseInt(System.getProperty("deuterium.chat.test.window","1440x900").split("x")[0]);}
    private static int testHeight(){return Integer.parseInt(System.getProperty("deuterium.chat.test.window","1440x900").split("x")[1]);}
    private final java.util.List<Double> frames=new java.util.ArrayList<>();
    private long frameAt;
    private final java.util.function.Consumer<net.neoforged.neoforge.client.event.RenderFrameEvent.Post> timer=e->{long now=System.nanoTime();if(frameAt!=0)frames.add((now-frameAt)/1e6);frameAt=now;};
    @Override public void configure(ScenarioOptions options){options.guiScale(Integer.getInteger("deuterium.chat.test.guiScale",3)).scenarioTimeoutMs(180000);}
    @Override public void define(ScenarioBuilder s){
        s.fill(new net.minecraft.core.BlockPos(-32,63,-32),new net.minecraft.core.BlockPos(32,63,32),net.minecraft.world.level.block.Blocks.GRASS_BLOCK.defaultBlockState())
         .fill(new net.minecraft.core.BlockPos(-12,63,12),new net.minecraft.core.BlockPos(12,69,12),net.minecraft.world.level.block.Blocks.STONE_BRICKS.defaultBlockState())
         .fill(new net.minecraft.core.BlockPos(-3,64,12),new net.minecraft.core.BlockPos(3,68,12),net.minecraft.world.level.block.Blocks.AIR.defaultBlockState())
         .fill(new net.minecraft.core.BlockPos(-10,68,11),new net.minecraft.core.BlockPos(-5,73,16),net.minecraft.world.level.block.Blocks.OAK_LEAVES.defaultBlockState())
         .teleportPlayer(0,65,0,0,0).ticks(20).step("fixture messages",ctx->{
            net.minecraft.client.Minecraft.getInstance().options.hideGui=false;net.minecraft.client.Minecraft.getInstance().getTutorial().setStep(net.minecraft.client.tutorial.TutorialSteps.NONE);
            ChatStore.INSTANCE.clear();ChatStore.INSTANCE.server="amiya";
            try{fixture=new ChatUiFixture();}catch(Exception e){throw new RuntimeException(e);}
            String[] names={"North","Yuzu","Mori","xiwanzi","Amiya","Odyssey","L1ghT_"};
            String[] content={"有人缺附魔书吗？今天钓了不少。","收一本经验修补。","仓库门口，给你留着了。","新修的桥终于通了！","刚走过，好看。","有人一起去东边的村庄吗？","等我一下，我回去拿点火把。","村庄旁边有樱花林。","我在钟楼下面，准备好了就出发。","收到，出发！😀 ✨","😀"};
            for(int i=0;i<content.length;i++)ChatStore.INSTANCE.add(new ChatMessage("fixture_"+i,"",names[i%names.length],"world",i%2==0?"amiya":"login","",Component.literal(content[i]),null,Instant.now(),false,false,""));
            var rich=Component.literal("[红包] ").withStyle(s2->s2.withColor(0xDAAAAD)).append(Component.literal("点击领取").withStyle(s2->s2.withUnderlined(true).withClickEvent(new ClickEvent(ClickEvent.Action.RUN_COMMAND,"/redpacket open test")).withInsertion("deuterium:keep-chat").withHoverEvent(new HoverEvent(HoverEvent.Action.SHOW_TEXT,Component.literal("服务器插件交互，保持聊天")))));
            ChatStore.INSTANCE.add(new ChatMessage("fixture_plugin","","","world","amiya","",rich,null,Instant.now(),false,true,""));
            for(String id:java.util.List.of("cloud","sticker","motion"))ChatStore.INSTANCE.add(new ChatMessage("fixture_"+id,"","Amiya","world","amiya","",Component.literal(id.equals("cloud")?"这是刚拍的云海。":""),fixture.media.get(id),Instant.now(),false,false,""));
            ChatStore.INSTANCE.pinned.add("private:fixture-a");ChatStore.INSTANCE.pinned.add("private:fixture-b");
            ChatStore.INSTANCE.contacts.put("private:fixture-a",new ChatStore.Contact("private:fixture-a","Amiya","",true,2));
            ChatStore.INSTANCE.contacts.put("private:fixture-b",new ChatStore.Contact("private:fixture-b","L1ghT_","",true,1));
        ChatStore.INSTANCE.add(new ChatMessage("private_preview_a","","Amiya","private","","fixture-a",Component.literal("仓库门口见。"),null,Instant.now().minusSeconds(60),true,false,""));
            ChatStore.INSTANCE.add(new ChatMessage("private_preview_b","","L1ghT_","private","","fixture-b",Component.empty(),fixture.media.get("cloud"),Instant.now(),true,false,""));
        }).frames(8).screenshot("background").openScreen("full chat",ctx->new ChatScreenFull()).awaitScreen(ChatScreenFull.class)
          .waitUntil("media decoded and uploaded",ctx->ChatClient.INSTANCE.media.ready("cloud")&&ChatClient.INSTANCE.media.ready("sticker")&&ChatClient.INSTANCE.media.ready("motion"))
          .check("IMBlocker exempts only the full chat screen",ctx->{try{var c=Class.forName("io.github.reserveword.imblocker.common.IMBlockerConfig");var config=c.getField("INSTANCE").get(null);var method=c.getMethod("isScreenInWhitelist",Object.class);return Boolean.TRUE.equals(method.invoke(config,net.minecraft.client.Minecraft.getInstance().screen))&&!Boolean.TRUE.equals(method.invoke(config,new Object()));}catch(ClassNotFoundException absent){return true;}catch(Exception failure){throw new RuntimeException(failure);}})
          .waitUntil("offline avatar loaded from server",ctx->ChatAvatars.get("xiwanzi",null).portrait!=null)
          .check("offline face retains observed skin",ctx->{var skin=net.minecraft.resources.ResourceLocation.parse("minecraft:textures/entity/player/wide/steve.png");ChatAvatars.get("OfflineFixture",skin);return ChatAvatars.get("OfflineFixture",null).observed.equals(skin);})
          .frames(80).screenshot("full-chat")
          .step("compose hover sample",ctx->((ChatScreenFull)net.minecraft.client.Minecraft.getInstance().screen).view.input.setText("准备出发！"))
          .hover("#chat-send").frames(12).screenshot("composer-hover")
          .step("clear sample",ctx->((ChatScreenFull)net.minecraft.client.Minecraft.getInstance().screen).view.input.setText(""))
          .check("media remain static first-frame",ctx->ChatClient.INSTANCE.media.ready("motion"))
          .step("start frame sample",ctx->{frames.clear();frameAt=0;net.neoforged.neoforge.common.NeoForge.EVENT_BUS.addListener(timer);})
          .frames(240).step("save frame sample",ctx->{net.neoforged.neoforge.common.NeoForge.EVENT_BUS.unregister(timer);frames.sort(Double::compare);try{java.nio.file.Files.writeString(java.nio.file.Path.of("../build/ldlib2-uitest/frame-times.json"),"{\"samples\":"+frames.size()+",\"p50Ms\":"+frames.get(frames.size()/2)+",\"p95Ms\":"+frames.get((int)(frames.size()*.95))+",\"maxMs\":"+frames.getLast()+"}");}catch(Exception ex){throw new RuntimeException(ex);}})
          .step("stickers",ctx->ChatPanels.stickers(((ChatScreenFull)net.minecraft.client.Minecraft.getInstance().screen).view.root)).frames(30).screenshot("stickers")
          .click("#sticker-personal-tab").frames(20).screenshot("personal-stickers")
          .clickAt((testWidth()>=1800?215:193)/Integer.getInteger("deuterium.chat.test.guiScale",3),(testHeight()-(testWidth()>=1800?221:204))/Integer.getInteger("deuterium.chat.test.guiScale",3),1).awaitScreen(MediaViewer.class).frames(20).screenshot("viewer")
          .click("#viewer-plus").frames(10).screenshot("viewer-zoom")
          .click("#viewer-fit").clickAt((testWidth()-57)/Integer.getInteger("deuterium.chat.test.guiScale",3),45/Integer.getInteger("deuterium.chat.test.guiScale",3),0).awaitScreen(ChatScreenFull.class)
          .check("return restores personal panel",ctx->ChatPanels.state(((ChatScreenFull)net.minecraft.client.Minecraft.getInstance().screen).view.root).equals("personal"))
          .click("#emoji-category-2").frames(10).screenshot("common-emoji")
          .step("select middle draft",ctx->{var f=((ChatScreenFull)net.minecraft.client.Minecraft.getInstance().screen).view.input;f.setText("ABC");f.setCursor(2);f.setSelection(1,2);})
          .click("#emoji-cell-0").check("emoji replaces selected draft",ctx->((ChatScreenFull)net.minecraft.client.Minecraft.getInstance().screen).view.input.getValue().equals("A😀C"))
          .click("#emoji-category-3").frames(10).screenshot("adventure-emoji")
          .step("share panel",ctx->ChatPanels.more(((ChatScreenFull)net.minecraft.client.Minecraft.getInstance().screen).view.root)).frames(10).screenshot("share-panel")
          .clickAt((testWidth()>=1800?232:210)/Integer.getInteger("deuterium.chat.test.guiScale",3),(testHeight()-(testWidth()>=1800?262:245))/Integer.getInteger("deuterium.chat.test.guiScale",3),0).frames(10).screenshot("position-preview")
          .step("private inbox",ctx->((ChatScreenFull)net.minecraft.client.Minecraft.getInstance().screen).view.switchScope("inbox"))
          .frames(20).screenshot("private-inbox")
          .step("private conversation",ctx->((ChatScreenFull)net.minecraft.client.Minecraft.getInstance().screen).view.switchScope("private:fixture-a")).frames(12).screenshot("private-conversation")
          .clickAt((testWidth()>=1800?174:152)/Integer.getInteger("deuterium.chat.test.guiScale",3),(testWidth()>=1800?107:95)/Integer.getInteger("deuterium.chat.test.guiScale",3),0).frames(8)
          .click("#private-batch").frames(5).click("#private-open-Amiya").frames(5).screenshot("private-selection")
          .click("#private-batch").frames(5).check("batch unpins deselected contact",ctx->!ChatStore.INSTANCE.pinned.contains("private:fixture-a"))
          .step("restore world",ctx->((ChatScreenFull)net.minecraft.client.Minecraft.getInstance().screen).view.switchScope("world"))
          .openScreen("quick chat",ctx->new ChatScreen("/ti" )).frames(35).screenshot("quick-chat")
          .closeScreen().frames(20).screenshot("hud")
          .teardown("fixture server",ctx->{net.neoforged.neoforge.common.NeoForge.EVENT_BUS.unregister(timer);if(fixture!=null)fixture.close();});
    }
}
