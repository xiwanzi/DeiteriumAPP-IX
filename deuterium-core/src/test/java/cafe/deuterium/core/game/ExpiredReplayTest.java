package cafe.deuterium.core.game;

import cafe.deuterium.core.StoreFixture;
import cafe.deuterium.core.config.CoreConfig;
import cafe.deuterium.core.mail.CoreMailbox;
import cafe.deuterium.core.storage.RpcJournal;
import cafe.deuterium.core.util.*;
import com.google.gson.JsonObject;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;
import java.nio.file.Path;
import java.time.Instant;
import java.util.*;
import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

class ExpiredReplayTest {
    @TempDir Path folder;
    JsonObject request(String type, String id, boolean expired) {
        return Json.tree(Map.of("operationId",id,"command",type,"expiresAt",Instant.now().plusSeconds(expired?-30:60).toString(),"payload",Map.of("fromUuid","b89c5edc-ec6d-467c-91dd-f93a17e1bd39","toUuid","4734fcd2-c450-4c79-b130-914504709408","amount","1.00")));
    }
    CoreConfig config() { CoreConfig config=mock(CoreConfig.class);when(config.economyEnabled()).thenReturn(true);when(config.nodeId()).thenReturn("amiya");when(config.economyAuthority()).thenReturn("amiya");return config; }
    @Test void expiredReplayReturnsCommittedWalletReceiptWithoutASecondMutation()throws Exception{
        try(var f=new StoreFixture(folder)){
            var journal=new RpcJournal(f.db,"amiya");EconomyAccess economy=mock(EconomyAccess.class);when(economy.available()).thenReturn(true);when(economy.execute(anyString(),eq("wallet.transfer"),any())).thenReturn(Json.tree(Map.of("status","COMPLETED","amount","1.00")));
            var dispatcher=new CommandDispatcher(this::config,()->null,null,null,null,journal,economy);
            var first=dispatcher.execute(request("wallet.transfer","wallet_expiry_one",false));assertEquals("COMPLETED",first.get("status").getAsString());
            var expired=dispatcher.execute(request("wallet.transfer","wallet_expiry_one",true));assertEquals(first,expired);assertEquals("COMPLETED",journal.find("wallet_expiry_one").state());verify(economy,times(1)).execute(anyString(),anyString(),any());
            var conflict=request("wallet.transfer","wallet_expiry_one",true);conflict.getAsJsonObject("payload").addProperty("amount","2.00");
            assertEquals("IDEMPOTENCY_CONFLICT",dispatcher.execute(conflict).getAsJsonObject("error").get("code").getAsString());assertEquals("COMPLETED",journal.find("wallet_expiry_one").state());
        }
    }
    @Test void expiredUnknownReplayStaysUnknownAndDoesNotReacquireExecution()throws Exception{
        try(var f=new StoreFixture(folder)){
            var journal=new RpcJournal(f.db,"amiya");EconomyAccess economy=mock(EconomyAccess.class);when(economy.available()).thenReturn(true);when(economy.execute(anyString(),anyString(),any())).thenThrow(new CoreFailure("RESULT_UNKNOWN","lost response"));
            var dispatcher=new CommandDispatcher(this::config,()->null,null,null,null,journal,economy);
            assertEquals("UNKNOWN",dispatcher.execute(request("wallet.transfer","wallet_unknown",false)).get("status").getAsString());
            assertEquals("UNKNOWN",dispatcher.execute(request("wallet.transfer","wallet_unknown",true)).get("status").getAsString());assertEquals("UNKNOWN",journal.find("wallet_unknown").state());verify(economy,times(1)).execute(anyString(),anyString(),any());verify(economy,never()).queryOperation(anyString());
        }
    }
    @Test void newExpiredIntentIsRejectedWithoutCreatingExecutionJournal()throws Exception{
        try(var f=new StoreFixture(folder)){
            var journal=new RpcJournal(f.db,"amiya");EconomyAccess economy=mock(EconomyAccess.class);
            var dispatcher=new CommandDispatcher(this::config,()->null,null,null,null,journal,economy);
            var reply=dispatcher.execute(request("wallet.transfer","wallet_new_expired",true));assertEquals("FAILED",reply.get("status").getAsString());assertEquals("COMMAND_EXPIRED",reply.getAsJsonObject("error").get("code").getAsString());assertEquals("NOT_FOUND",journal.find("wallet_new_expired").state());verifyNoInteractions(economy);
        }
    }
    @Test void expiredMailReplayUsesOriginalReceiptWithoutCallingMailAgain()throws Exception{
        try(var f=new StoreFixture(folder)){
            var journal=new RpcJournal(f.db,"amiya");CoreMailbox mail=mock(CoreMailbox.class);when(mail.execute(anyString(),anyString(),any())).thenReturn(Json.tree(Map.of("code","OK","value",Map.of("status","CREATED","mailId","77"))));
            var dispatcher=new CommandDispatcher(()->f.config,()->mail,null,null,null,journal,null);
            var first=dispatcher.execute(request("mailbox.create","mail_expiry_one",false));assertEquals("COMPLETED",first.get("status").getAsString());assertEquals(first,dispatcher.execute(request("mailbox.create","mail_expiry_one",true)));verify(mail,times(1)).execute(anyString(),anyString(),any());
        }
    }
}
