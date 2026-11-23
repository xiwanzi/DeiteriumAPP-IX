package cafe.deuterium.core.compat;

import cafe.deuterium.core.items.NbtLimits;
import cafe.deuterium.core.util.CoreFailure;
import org.bukkit.Bukkit;
import org.bukkit.inventory.ItemStack;

import java.io.ByteArrayOutputStream;
import java.io.OutputStream;
import java.lang.reflect.InvocationTargetException;
import java.util.NoSuchElementException;
import java.util.Set;

/** Converts the editor's 1.21.1 ItemStack SNBT using the running server's NBT codec. */
public final class EditorItemStack {
    private EditorItemStack() { }

    public static ItemStack read(String itemId, String snbt, int limit) {
        byte[] bytes = encode(itemId, snbt, Bukkit.getUnsafe().getDataVersion(), limit);
        try {
            return ItemStack.deserializeBytes(bytes);
        } catch (IllegalArgumentException | NoSuchElementException invalid) {
            throw new CoreFailure("ITEM_ENCODING_INVALID", "游戏服无法读取此物品或其组件。", invalid);
        }
    }

    static byte[] encode(String itemId, String snbt, int dataVersion, int limit) {
        String source = snbt.isBlank() ? "{id:\"" + itemId + "\",count:1}" : snbt;
        requireBounded(source);
        try {
            Class<?> compound = Class.forName("net.minecraft.nbt.CompoundTag");
            Object tag = Class.forName("net.minecraft.nbt.TagParser").getMethod("parseTag", String.class)
                    .invoke(null, source);
            Set<?> fields = (Set<?>) compound.getMethod("getAllKeys").invoke(tag);
            if (!fields.contains("id") || !Set.of("id", "count", "components").containsAll(fields))
                throw CoreFailure.invalid("请选择 1.21.1 格式的物品栈。");
            String actualId = (String) compound.getMethod("getString", String.class).invoke(tag, "id");
            if (!itemId.equals(actualId.contains(":") ? actualId : "minecraft:" + actualId))
                throw CoreFailure.invalid("物品 ID 与物品栈不一致。");
            var putInt = compound.getMethod("putInt", String.class, int.class);
            putInt.invoke(tag, "count", 1);
            putInt.invoke(tag, "DataVersion", dataVersion);
            var output = new ByteArrayOutputStream();
            Class.forName("net.minecraft.nbt.NbtIo").getMethod("writeCompressed", compound, OutputStream.class)
                    .invoke(null, tag, output);
            byte[] bytes = output.toByteArray();
            NbtLimits.inspect(bytes, limit);
            return bytes;
        } catch (InvocationTargetException | IllegalArgumentException invalid) {
            throw new CoreFailure("ITEM_ENCODING_INVALID", "游戏服无法读取此物品或其组件。", invalid);
        } catch (ReflectiveOperationException | LinkageError unavailable) {
            throw new CoreFailure("ITEM_IMPORT_UNAVAILABLE", "当前游戏服不支持此物品栈格式。", unavailable);
        }
    }

    static void requireBounded(String source) {
        int depth = 0;
        char quote = 0;
        for (int index = 0; index < source.length(); index++) {
            char next = source.charAt(index);
            if (quote != 0) {
                if (next == '\\') index++;
                else if (next == quote) quote = 0;
            } else if (next == '"' || next == '\'') quote = next;
            else if (next == '{' || next == '[') {
                if (++depth > 32) throw CoreFailure.invalid("物品栈嵌套过深。");
            } else if (next == '}' || next == ']') {
                if (--depth < 0) throw CoreFailure.invalid("物品栈格式不正确。");
            }
        }
        if (depth != 0 || quote != 0) throw CoreFailure.invalid("物品栈格式不完整。");
    }
}
