package cafe.deuterium.chat;

import com.lowdragmc.lowdraglib2.registry.RegistrationEnvironment;
import com.lowdragmc.lowdraglib2.registry.annotation.LDLRegisterClient;
import com.lowdragmc.lowdraglib2.uitest.*;
import com.google.gson.JsonObject;
import net.minecraft.network.chat.*;
import java.time.Instant;
import java.util.*;

@LDLRegisterClient(name="deuterium_chat_titles",group="deuterium",registry=UIScenario.REGISTRY,environment=RegistrationEnvironment.DEV_ONLY)
public final class ChatUiScenarioTitles implements UIScenario {
    private static final String ID="d97161f9-2a7c-4abd-a8e7-6fd64a64c002";
    private ChatMessage message(){return new ChatMessage("title-example",ID,"xiwanzi","world","amiya","",Component.literal("这个头衔的渐变会同步到所有佩戴者。"),null,Instant.now(),true,false,"");}
    private void update(String text){var p=new JsonObject();p.addProperty("playerUuid",ID);p.addProperty("gameId","xiwanzi");p.addProperty("displayName",text);ChatTitles.receive(p);}
    @Override public void configure(ScenarioOptions options){options.guiScale(3).scenarioTimeoutMs(120000);}
    @Override public void define(ScenarioBuilder s){
        s.step("default title",ctx->{ChatTitles.reset();ChatStore.INSTANCE.clear();})
         .check("default is grey walker",ctx->ChatTitles.title(message()).getString().equals("[溯行者]")&&ChatTitles.title(message()).getStyle().getColor().getValue()==0xA9B3B8)
         .step("receive authoritative gradient",ctx->{update("#d9afd9参#c3bddc考#adcbde这#97d9e1样");ChatStore.INSTANCE.add(message());})
         .check("inline colors preserved",ctx->{var colors=new ArrayList<Integer>();ChatTitles.title(message()).visit((style,value)->{if(!value.equals("[")&&!value.equals("]"))colors.add(style.getColor().getValue());return Optional.empty();},Style.EMPTY);return colors.equals(List.of(0xd9afd9,0xc3bddc,0xadcbde,0x97d9e1));})
         .openScreen("gradient title",ctx->new ChatScreenFull()).frames(30).screenshot("gradient-title")
         .step("rename without replacing message",ctx->update("#B8CBD9溯#C8BCCD光#D8BCC7者"))
         .check("existing message projects latest title",ctx->ChatTitles.title(message()).getString().equals("[溯光者]"))
         .frames(20).screenshot("renamed-title")
         .teardown("clear authoritative projection",ctx->ChatTitles.reset());
    }
}
