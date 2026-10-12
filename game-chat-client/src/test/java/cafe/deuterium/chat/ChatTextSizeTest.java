package cafe.deuterium.chat;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;
import java.nio.file.*;
import static org.junit.jupiter.api.Assertions.*;

class ChatTextSizeTest {
    @TempDir Path temp;
    @Test void thirdClickReturnsToDefault(){int level=0;level=ChatTextSize.next(level);assertEquals(1,level);level=ChatTextSize.next(level);assertEquals(2,level);assertEquals(0,ChatTextSize.next(level));}
    @Test void textSizesAreModestAndRoundedToWholePhysicalPixels(){
        assertArrayEquals(new int[]{20,22,24},new int[]{ChatTextSize.pixels(20,0),ChatTextSize.pixels(20,1),ChatTextSize.pixels(20,2)});
        assertArrayEquals(new int[]{18,20,22},new int[]{ChatTextSize.pixels(18,0),ChatTextSize.pixels(18,1),ChatTextSize.pixels(18,2)});
        assertArrayEquals(new int[]{16,18,19},new int[]{ChatTextSize.pixels(16,0),ChatTextSize.pixels(16,1),ChatTextSize.pixels(16,2)});
    }
    @Test void selectionSurvivesReloadAndBadFilesUseDefault()throws Exception{
        Path file=temp.resolve("chat/text-size");assertEquals(0,ChatTextSize.read(file));
        ChatTextSize.write(file,2);assertEquals(2,ChatTextSize.read(file));
        Files.writeString(file,"999");assertEquals(0,ChatTextSize.read(file));
        Files.writeString(file,"invalid");assertEquals(0,ChatTextSize.read(file));
        ChatTextSize.write(file,0);assertEquals(0,ChatTextSize.read(file));
    }
}
