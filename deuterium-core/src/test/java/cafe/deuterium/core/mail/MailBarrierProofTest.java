package cafe.deuterium.core.mail;

import cafe.deuterium.core.api.PlayerDataService;
import org.junit.jupiter.api.Test;
import java.util.UUID;
import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

class MailBarrierProofTest {
    @Test void proofLookupIsReadOnlyAndPreservesOriginalServerAndSession()throws Exception {
        var provider=mock(PlayerDataService.class);UUID player=UUID.randomUUID(),operation=UUID.randomUUID();
        when(provider.lookupSaveProof(player,"survival",operation,"old-session")).thenReturn(new PlayerDataService.SaveProof(player,"survival",operation,"old-session","odyssey","committed-receipt",1788844000));
        var proof=new MailBarrierAdapter(()->provider).lookupSaveProof(player,"survival",operation,"old-session");
        assertEquals(player,proof.playerUuid());assertEquals(operation,proof.operationId());assertEquals("old-session",proof.sessionEpoch());assertEquals("odyssey",proof.serverId());assertEquals("committed-receipt",proof.saveReceipt());assertEquals(1788844000,proof.committedAt());
        verify(provider).lookupSaveProof(player,"survival",operation,"old-session");verifyNoMoreInteractions(provider);
    }
    @Test void absentProofCannotBecomeSuccessAndMismatchedProofIsRejected()throws Exception {
        var provider=mock(PlayerDataService.class);UUID player=UUID.randomUUID(),operation=UUID.randomUUID();var adapter=new MailBarrierAdapter(()->provider);
        assertNull(adapter.lookupSaveProof(player,"survival",operation,"session"));
        when(provider.lookupSaveProof(player,"survival",operation,"session")).thenReturn(new PlayerDataService.SaveProof(UUID.randomUUID(),"survival",operation,"session","amiya","receipt",1788844000));
        assertThrows(IllegalStateException.class,()->adapter.lookupSaveProof(player,"survival",operation,"session"));
    }
}
