package cafe.deuterium.core.items;

import cafe.deuterium.core.util.CoreFailure;
import java.io.*;
import java.util.HashSet;
import java.util.Set;
import java.util.zip.GZIPInputStream;

/** Structural bounds before the server deserializes an item; handles Paper's compressed NBT. */
public final class NbtLimits {
    public record Info(String itemId, Set<String> requiredMods) { }
    private static final int MAX_EXPANDED = 8 * 1024 * 1024;
    private final Set<String> mods = new HashSet<>();
    private int tags;
    private String rootId = "";
    private NbtLimits() { }
    public static Info inspect(byte[] data, int maxBytes) {
        if (data == null || data.length == 0 || data.length > maxBytes) throw CoreFailure.invalid("物品快照为空或过大。");
        try {
            InputStream source = new ByteArrayInputStream(data);
            if (data.length >= 2 && (data[0] & 255) == 31 && (data[1] & 255) == 139) source = new GZIPInputStream(source);
            try (DataInputStream input = new DataInputStream(new LimitedInput(source, MAX_EXPANDED))) {
                if (input.readUnsignedByte() != 10) throw new IOException("root is not compound");
                input.readUTF();
                NbtLimits reader = new NbtLimits(); reader.tag(input, 10, 0, "");
                if (input.read() != -1 || reader.rootId.isEmpty()) throw new IOException("invalid item root");
                return new Info(reader.rootId, Set.copyOf(reader.mods));
            }
        } catch (CoreFailure error) { throw error; }
        catch (Exception error) { throw new CoreFailure("ITEM_ENCODING_INVALID", "物品编码损坏或超过展开/嵌套限制。", error); }
    }
    private void tag(DataInputStream in, int type, int depth, String path) throws IOException {
        if (depth > 32 || ++tags > 100000) throw new IOException("NBT complexity limit");
        switch (type) {
            case 1 -> in.readByte(); case 2 -> in.readShort(); case 3 -> in.readInt(); case 4 -> in.readLong();
            case 5 -> in.readFloat(); case 6 -> in.readDouble();
            case 7 -> skip(in, count(in), 1);
            case 8 -> in.readUTF();
            case 9 -> {
                int element = in.readUnsignedByte(), count = count(in);
                if (count > 65536 || (element == 0 && count > 0)) throw new IOException("invalid list");
                for (int i = 0; i < count; i++) tag(in, element, depth + 1, path + "[]");
            }
            case 10 -> {
                String id = ""; boolean itemCount = false;
                for (;;) {
                    int child = in.readUnsignedByte(); if (child == 0) break;
                    if (++tags > 100000) throw new IOException("NBT complexity limit");
                    String name = in.readUTF();
                    if (name.length() > 1024) throw new IOException("NBT key limit");
                    if (path.endsWith("/components") && (name.equals("sophisticatedcore:storage_uuid") || name.equals("sophisticatedcore:contents_uuid")))
                        throw new CoreFailure("ITEM_EXTERNAL_STORAGE_UNSUPPORTED", "该容器引用外部背包存储，不能仅复制物品 NBT；请保存未初始化的空容器，或使用内容展开适配。");
                    if (path.endsWith("/components") || path.endsWith("/ForgeCaps") || path.endsWith("/neoforge:attachments")) namespace(name);
                    if (name.equals("count") || name.equals("Count")) itemCount = true;
                    if (child == 8 && name.equals("id")) { id = in.readUTF(); if (depth == 0) rootId = id; }
                    else tag(in, child, depth + 1, path + "/" + name);
                }
                if (depth == 0 || itemCount) namespace(id);
            }
            case 11 -> skip(in, count(in), 4);
            case 12 -> skip(in, count(in), 8);
            default -> throw new IOException("invalid NBT tag");
        }
    }
    private void namespace(String key) {
        int colon = key.indexOf(':');
        if (colon > 0) { String namespace = key.substring(0, colon); if (!namespace.equals("minecraft")) mods.add(namespace); }
    }
    private static int count(DataInputStream input) throws IOException { int n = input.readInt(); if (n < 0 || n > MAX_EXPANDED) throw new IOException("NBT length limit"); return n; }
    private static void skip(DataInputStream input, int count, int unit) throws IOException {
        long length = (long) count * unit; if (length > MAX_EXPANDED) throw new IOException("NBT array limit"); input.skipNBytes(length);
    }
    private static final class LimitedInput extends InputStream {
        private final InputStream source; private int remaining;
        private LimitedInput(InputStream source, int remaining) { this.source = source; this.remaining = remaining; }
        @Override public int read() throws IOException {
            int value = source.read(); if (value >= 0 && --remaining < 0) throw new IOException("NBT expansion limit"); return value;
        }
        @Override public int read(byte[] b, int off, int len) throws IOException {
            if (len == 0) return 0;
            int n = source.read(b, off, Math.min(len, Math.max(1, remaining + 1)));
            if (n > 0 && (remaining -= n) < 0) throw new IOException("NBT expansion limit"); return n;
        }
        @Override public void close() throws IOException { source.close(); }
    }
}
