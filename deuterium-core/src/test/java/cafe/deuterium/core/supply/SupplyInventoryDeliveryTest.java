package cafe.deuterium.core.supply;

import cafe.deuterium.core.api.PlayerDataService;
import cafe.deuterium.core.runtime.Sessions;
import java.util.UUID;
import org.bukkit.Bukkit;
import org.bukkit.Material;
import org.bukkit.entity.Player;
import org.bukkit.inventory.ItemStack;
import org.bukkit.inventory.PlayerInventory;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

class SupplyInventoryDeliveryTest {
    @Test void fullBackpackChoosesMailWithoutWritingInventory() throws Exception {
        try (var bukkit = mockStatic(Bukkit.class)) {
            var f = fixture(bukkit);
            var occupied = mock(ItemStack.class); when(occupied.getType()).thenReturn(mock(Material.class));
            when(occupied.clone()).thenReturn(occupied); when(occupied.isSimilar(f.item)).thenReturn(false);
            when(f.inventory.getStorageContents()).thenReturn(new ItemStack[]{occupied});
            var result = f.deliver();
            assertEquals("MAIL_PENDING", result.state()); assertEquals("INVENTORY_FULL", result.message());
            verify(f.inventory, never()).setStorageContents(any()); verify(f.lease, never()).saveAndConfirm();
        }
    }

    @Test void committedBackpackReportsReceiptAndDoesNotChooseMail() throws Exception {
        try (var bukkit = mockStatic(Bukkit.class)) {
            var f = fixture(bukkit); var result = f.deliver();
            assertEquals("INVENTORY", result.state()); assertEquals("save-receipt", result.receipt()); assertTrue(result.committedAt()>0);
            verify(f.inventory).setStorageContents(argThat(items -> items.length==1 && items[0]==f.item));
            verify(f.lease).saveAndConfirm();
        }
    }

    @Test void unknownSaveNeverFallsBackToMail() throws Exception {
        try (var bukkit = mockStatic(Bukkit.class)) {
            var f = fixture(bukkit); when(f.lease.saveAndConfirm()).thenThrow(new IllegalStateException("connection lost after write"));
            var result = f.deliver(); assertEquals("REVIEW",result.state()); assertEquals("",result.receipt());
            verify(f.inventory).setStorageContents(any());
        }
    }

    @Test void changedInventoryOwnerRequeuesBeforeTouchingInventory() throws Exception {
        try (var bukkit = mockStatic(Bukkit.class)) {
            var f = fixture(bukkit); when(f.lease.sessionEpoch()).thenReturn("another-owner");
            assertEquals("PENDING",f.deliver().state()); verify(f.inventory,never()).setStorageContents(any());
        }
    }

    @Test void playerDisconnectBeforeWriteRoutesToMail() throws Exception {
        try (var bukkit = mockStatic(Bukkit.class)) {
            var f=fixture(bukkit);f.sessions.quit(f.uuid);
            assertEquals("MAIL_PENDING",f.deliver().state());verify(f.inventory,never()).setStorageContents(any());
        }
    }

    private record Fixture(UUID uuid,UUID operation,Sessions sessions,String epoch,PlayerInventory inventory,
                           ItemStack item,PlayerDataService.Lease lease,SupplyInventoryDelivery service) {
        SupplyInventoryDelivery.Attempt deliver(){return service.deliver(uuid,epoch,"inventory-epoch","survival",operation,item);}
    }
    private Fixture fixture(org.mockito.MockedStatic<Bukkit> bukkit) throws Exception {
        UUID uuid=UUID.randomUUID(),operation=UUID.randomUUID();var player=mock(Player.class);var inventory=mock(PlayerInventory.class);
        when(player.getUniqueId()).thenReturn(uuid);when(player.getName()).thenReturn("Player");when(player.isOnline()).thenReturn(true);when(player.getInventory()).thenReturn(inventory);
        bukkit.when(Bukkit::isPrimaryThread).thenReturn(true);bukkit.when(()->Bukkit.getPlayer(uuid)).thenReturn(player);
        var item=mock(ItemStack.class);when(item.getType()).thenReturn(mock(Material.class));when(item.getMaxStackSize()).thenReturn(1);when(item.clone()).thenReturn(item);
        when(inventory.getStorageContents()).thenReturn(new ItemStack[]{null});when(inventory.getMaxStackSize()).thenReturn(64);
        var provider=mock(PlayerDataService.class);var lease=mock(PlayerDataService.Lease.class);when(provider.acquire(player,"survival",operation)).thenReturn(lease);
        when(lease.sessionEpoch()).thenReturn("inventory-epoch");when(lease.saveAndConfirm()).thenReturn("save-receipt");
        var sessions=new Sessions();var session=sessions.join(player);
        return new Fixture(uuid,operation,sessions,session.sessionEpoch(),inventory,item,lease,new SupplyInventoryDelivery(sessions,()->provider));
    }
}
