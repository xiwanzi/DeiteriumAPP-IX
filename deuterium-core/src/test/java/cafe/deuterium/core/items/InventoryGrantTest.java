package cafe.deuterium.core.items;

import cafe.deuterium.core.util.CoreFailure;
import org.bukkit.Material;
import org.bukkit.inventory.ItemStack;
import org.bukkit.inventory.PlayerInventory;
import org.junit.jupiter.api.Test;
import java.util.concurrent.atomic.AtomicInteger;
import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

class InventoryGrantTest {
    private static final Material STONE = mock(Material.class);
    private static ItemStack stack(int count, int maximum) {
        ItemStack item = mock(ItemStack.class); AtomicInteger amount = new AtomicInteger(count);
        when(item.getType()).thenReturn(STONE); when(item.getAmount()).thenAnswer(call -> amount.get());
        doAnswer(call -> { amount.set(call.getArgument(0)); return null; }).when(item).setAmount(anyInt());
        when(item.getMaxStackSize()).thenReturn(maximum);
        when(item.isSimilar(any())).thenAnswer(call -> { ItemStack other = call.getArgument(0); return other != null && other.getType() == STONE; });
        when(item.clone()).thenAnswer(call -> stack(amount.get(), maximum)); return item;
    }
    @Test void plansFullGrantWithoutMutatingOriginalStacks() {
        ItemStack held = stack(1,64), existing = stack(60,64);
        ItemStack[] planned = InventoryGrant.plan(new ItemStack[]{existing,null}, held, 8,64);
        assertEquals(64, planned[0].getAmount()); assertEquals(4, planned[1].getAmount());
        assertEquals(60, existing.getAmount()); assertEquals(1, held.getAmount());
    }
    @Test void insufficientCapacityDoesNotWriteInventoryOrExpandHugeQuantities() {
        PlayerInventory inventory = mock(PlayerInventory.class);
        ItemStack full = stack(64,64);
        when(inventory.getStorageContents()).thenReturn(new ItemStack[]{full}); when(inventory.getMaxStackSize()).thenReturn(64);
        CoreFailure error = assertThrows(CoreFailure.class, () -> InventoryGrant.apply(inventory, stack(1,1), Integer.MAX_VALUE));
        assertEquals("INVENTORY_FULL", error.code()); verify(inventory, never()).setStorageContents(any());
    }
    @Test void writeExceptionIsUncertainNotSafeToRepeat() {
        PlayerInventory inventory = mock(PlayerInventory.class);
        when(inventory.getStorageContents()).thenReturn(new ItemStack[1]); when(inventory.getMaxStackSize()).thenReturn(64);
        doThrow(new IllegalStateException("partial write")).when(inventory).setStorageContents(any());
        assertEquals("RESULT_UNKNOWN", assertThrows(CoreFailure.class, () -> InventoryGrant.apply(inventory, stack(1,64), 1)).code());
    }
}
