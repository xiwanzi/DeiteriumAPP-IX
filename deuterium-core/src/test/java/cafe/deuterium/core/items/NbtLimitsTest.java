package cafe.deuterium.core.items;

import cafe.deuterium.core.util.CoreFailure;
import org.junit.jupiter.api.Test;
import java.io.*;
import java.util.zip.GZIPOutputStream;
import static org.junit.jupiter.api.Assertions.*;

class NbtLimitsTest {
    private byte[] item(boolean compressed, int depth) throws Exception {
        ByteArrayOutputStream bytes = new ByteArrayOutputStream();
        try (DataOutputStream out = new DataOutputStream(compressed ? new GZIPOutputStream(bytes) : bytes)) {
            out.writeByte(10); out.writeUTF(""); out.writeByte(8); out.writeUTF("id"); out.writeUTF("example:tool");
            for (int i = 0; i < depth; i++) { out.writeByte(10); out.writeUTF("nested"); }
            for (int i = 0; i <= depth; i++) out.writeByte(0);
        }
        return bytes.toByteArray();
    }
    @Test void validatesRawAndCompressedNbtAndRequiredNamespace() throws Exception {
        assertEquals("example:tool", NbtLimits.inspect(item(false,0),1024).itemId());
        assertTrue(NbtLimits.inspect(item(true,0),1024).requiredMods().contains("example"));
    }
    @Test void malformedDeepAndOversizedPayloadsAreRejected() throws Exception {
        assertThrows(CoreFailure.class, () -> NbtLimits.inspect(new byte[]{1,2,3},1024));
        assertThrows(CoreFailure.class, () -> NbtLimits.inspect(item(true,40),1024));
        assertThrows(CoreFailure.class, () -> NbtLimits.inspect(item(false,0),2));
    }
    @Test void externalStorageReferencesAreRejectedEvenInsideAnotherContainer() throws Exception {
        for (int depth : new int[]{0, 2}) {
            ByteArrayOutputStream bytes = new ByteArrayOutputStream();
            try (DataOutputStream out = new DataOutputStream(bytes)) {
                out.writeByte(10); out.writeUTF(""); out.writeByte(8); out.writeUTF("id"); out.writeUTF("minecraft:shulker_box");
                for (int i = 0; i < depth; i++) { out.writeByte(10); out.writeUTF("nested"); }
                out.writeByte(10); out.writeUTF("components"); out.writeByte(11); out.writeUTF("sophisticatedcore:storage_uuid"); out.writeInt(4);
                for (int i = 0; i < 4; i++) out.writeInt(i);
                for (int i = 0; i < depth + 2; i++) out.writeByte(0);
            }
            CoreFailure failure = assertThrows(CoreFailure.class, () -> NbtLimits.inspect(bytes.toByteArray(),1024));
            assertEquals("ITEM_EXTERNAL_STORAGE_UNSUPPORTED", failure.code());
        }
    }
}
