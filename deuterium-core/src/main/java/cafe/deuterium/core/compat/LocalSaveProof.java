package cafe.deuterium.core.compat;

import org.bukkit.Bukkit;
import org.bukkit.entity.Player;
import java.io.InputStream;
import java.lang.reflect.Constructor;
import java.nio.channels.FileChannel;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.StandardOpenOption;

/** Scoped 1.21.1 NMS read-back: saveData's void return alone is not a durability receipt. */
public final class LocalSaveProof {
    private LocalSaveProof() { }
    public static boolean supported() {
        try {
            Class<?> list = Class.forName("net.minecraft.nbt.ListTag"), accounter = Class.forName("net.minecraft.nbt.NbtAccounter");
            Class.forName("net.minecraft.world.entity.player.Inventory").getMethod("save", list);
            Class.forName("net.minecraft.nbt.NbtIo").getMethod("readCompressed", InputStream.class, accounter);
            return true;
        } catch (ReflectiveOperationException | LinkageError absent) { return false; }
    }
    public static void saveAndVerify(Player player) throws Exception {
        if (!Bukkit.isPrimaryThread()) throw new IllegalStateException("Local save requires server thread");
        if(player.isDead()||!player.isValid())throw new IllegalStateException("Dead or invalid player cannot receive a save proof");
        Object nms = player.getClass().getMethod("getHandle").invoke(player);
        Object inventory = nms.getClass().getMethod("getInventory").invoke(nms);
        Class<?> listClass = Class.forName("net.minecraft.nbt.ListTag");
        Object expected = inventory.getClass().getMethod("save", listClass).invoke(inventory, listClass.getConstructor().newInstance());
        player.saveData();
        var world = Bukkit.getWorld(Bukkit.getUnsafe().getMainLevelName());
        if (world == null) throw new IllegalStateException("Main player-data world unavailable");
        Path data = world.getWorldFolder().toPath().resolve("playerdata").resolve(player.getUniqueId() + ".dat");
        if (!Files.isRegularFile(data) || Files.size(data) > 33554432) throw new IllegalStateException("Saved player data unavailable or exceeds bounds");
        Class<?> accounterClass = Class.forName("net.minecraft.nbt.NbtAccounter");
        Constructor<?> constructor = accounterClass.getConstructor(long.class, int.class);
        Object root;
        try (InputStream input = Files.newInputStream(data)) {
            root = Class.forName("net.minecraft.nbt.NbtIo").getMethod("readCompressed", InputStream.class, accounterClass)
                    .invoke(null, input, constructor.newInstance(33554432L, 64));
        }
        Object actual = root.getClass().getMethod("getList", String.class, int.class).invoke(root, "Inventory", 10);
        if (!expected.equals(actual)) throw new IllegalStateException("Player inventory save did not match authoritative memory");
        try (FileChannel file = FileChannel.open(data, StandardOpenOption.READ, StandardOpenOption.WRITE)) { file.force(true); }
    }
}
