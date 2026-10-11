package cafe.deuterium.chat;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;
import java.awt.image.BufferedImage;
import java.nio.file.*;
import java.util.List;
import javax.imageio.ImageIO;
import static org.junit.jupiter.api.Assertions.*;

class ChatMediaTest {
    @TempDir Path temp;
    @Test void transparentStickerRoundTripKeepsSizeAlphaAndColor()throws Exception {
        var image=new BufferedImage(96,64,BufferedImage.TYPE_INT_ARGB);
        for(int y=12;y<52;y++)for(int x=16;x<80;x++)image.setRGB(x,y,0xFFE580AC);
        Path path=temp.resolve("sticker.png");ImageIO.write(image,"png",path.toFile());
        var prepared=ChatImageCompressor.prepare(path,true);
        assertEquals(96,prepared.width());assertEquals(64,prepared.height());assertFalse(prepared.animated());assertTrue(prepared.image().length<=512<<10);
        try(var decoder=new WebpCodec(prepared.image())){var frame=decoder.next();assertEquals(0,frame.image().getRGB(0,0)>>>24);assertEquals(0xFFE580AC,frame.image().getRGB(30,30));}
    }
    @Test void animationRetainsFrameTimingAndCanvas()throws Exception {
        var a=new BufferedImage(20,12,BufferedImage.TYPE_INT_ARGB);var b=new BufferedImage(20,12,BufferedImage.TYPE_INT_ARGB);
        for(int y=0;y<12;y++)for(int x=0;x<20;x++){a.setRGB(x,y,0xFFFF0000);b.setRGB(x,y,0xFF00FF00);}
        byte[] encoded=ChatImageCompressor.animatedRiff(List.of(WebpCodec.encode(a,90,true),WebpCodec.encode(b,90,true)),List.of(125,275),20,12,3);
        Path path=temp.resolve("motion.webp");Files.write(path,encoded);var prepared=ChatImageCompressor.prepare(path,false);assertTrue(prepared.animated());
        try(var decoder=new WebpCodec(prepared.image())){assertEquals(2,decoder.frames);assertEquals(3,decoder.loops);assertEquals(125,decoder.next().duration());assertEquals(275,decoder.next().duration());assertNull(decoder.next());}
    }
    @Test void limitsDoNotUpscaleAndPreserveLongScreenshots()throws Exception {
        assertEquals(new ChatImageCompressor.Size(100,60,false),ChatImageCompressor.size(100,60,false,false));
        var wide=ChatImageCompressor.size(3840,2160,false,false);assertEquals(1920,wide.width());assertEquals(1080,wide.height());
        var tall=ChatImageCompressor.size(2000,8000,false,false);assertTrue(tall.longImage());assertTrue(tall.width()<=1440);assertTrue((long)tall.width()*tall.height()<=8_000_000);
        assertThrows(java.io.IOException.class,()->ChatImageCompressor.size(100000,100000,false,false));
        assertThrows(java.io.IOException.class,()->new WebpCodec(new byte[16]));
    }
    @Test void headersAndFadeUseLatestProductRules(){
        assertEquals("Amiya",ChatPresentation.suffix("world","Aamiya"));assertEquals("封锁区",ChatPresentation.suffix("world","LOGIN"));
        assertEquals("残像区域",ChatPresentation.suffix("world","Odyssey"));assertEquals("科研中心",ChatPresentation.suffix("world","MEK"));
        assertEquals("附近",ChatPresentation.suffix("nearby","Mek"));assertEquals("私聊",ChatPresentation.suffix("private","Login"));assertEquals("团队",ChatPresentation.suffix("team","Amiya"));
        assertEquals(255,ChatPresentation.hudAlpha(8000));assertEquals(127,ChatPresentation.hudAlpha(9000));assertEquals(0,ChatPresentation.hudAlpha(10000));
    }
}
