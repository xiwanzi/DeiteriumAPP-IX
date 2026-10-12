package cafe.deuterium.chat;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;
import java.awt.datatransfer.*;
import java.awt.image.BufferedImage;
import java.io.*;
import java.nio.*;
import java.nio.file.*;
import java.util.List;
import javax.imageio.ImageIO;
import static org.junit.jupiter.api.Assertions.*;

class ClipboardImageTest {
    @TempDir Path temp;
    private Transferable contents(DataFlavor flavor,Object value){return new Transferable(){
        public DataFlavor[] getTransferDataFlavors(){return new DataFlavor[]{flavor};}
        public boolean isDataFlavorSupported(DataFlavor query){return query.equals(flavor);}
        public Object getTransferData(DataFlavor query)throws UnsupportedFlavorException{if(!isDataFlavorSupported(query))throw new UnsupportedFlavorException(query);return value;}
    };}
    @Test void ordinaryTextAndImageUrlsContinueThroughNormalPaste()throws Exception{
        assertNull(ClipboardImage.from(new StringSelection("hello")));
        assertNull(ClipboardImage.from(new StringSelection("https://example.com/picture.png")));
    }
    @Test void imageFlavorKeepsTransparencyAndOnlyWritesOnPreparation()throws Exception{
        var image=new BufferedImage(2,2,BufferedImage.TYPE_INT_ARGB);image.setRGB(1,0,0x80804020);
        var source=ClipboardImage.from(contents(DataFlavor.imageFlavor,image));Path target=temp.resolve("source");
        assertFalse(Files.exists(target));source.writeTo(target);
        var read=ImageIO.read(target.toFile());assertEquals(0,read.getRGB(0,0));assertEquals(0x80804020,read.getRGB(1,0));
    }
    @Test void pngClipboardIsCopiedWithoutReencoding()throws Exception{
        byte[] data={1,2,3,4};var flavor=new DataFlavor("image/png;class=java.io.InputStream");
        var source=ClipboardImage.from(contents(flavor,new ByteArrayInputStream(data)));
        Path target=temp.resolve("source");source.writeTo(target);assertArrayEquals(data,Files.readAllBytes(target));
    }
    @Test void copiedImageFileIsPreservedButMultipleFilesAreRejected()throws Exception{
        Path image=temp.resolve("animation.gif");Files.write(image,new byte[]{1,2,3});
        var source=ClipboardImage.from(contents(DataFlavor.javaFileListFlavor,List.of(image.toFile())));
        Path target=temp.resolve("copy");source.writeTo(target);assertArrayEquals(Files.readAllBytes(image),Files.readAllBytes(target));
        assertThrows(IOException.class,()->ClipboardImage.from(contents(DataFlavor.javaFileListFlavor,List.of(image.toFile(),image.toFile()))));
        assertNull(ClipboardImage.from(contents(DataFlavor.javaFileListFlavor,List.of(temp.resolve("readme.txt").toFile()))));
    }
    private byte[] dib(int width,int height,int bits,int header,int compression){
        int stride=((width*bits+31)/32)*4;var b=ByteBuffer.allocate(header+stride*Math.abs(height)).order(ByteOrder.LITTLE_ENDIAN);
        b.putInt(0,header);b.putInt(4,width);b.putInt(8,height);b.putShort(12,(short)1);b.putShort(14,(short)bits);b.putInt(16,compression);return b.array();
    }
    @Test void windowsScreenshotSupportsBottomUpPaddingAndTopDownBgrx()throws Exception{
        byte[] bottom=dib(1,2,24,40,0);bottom[40]=(byte)255;bottom[46]=(byte)255;
        var read=ClipboardImage.decodeDib(bottom);assertEquals(0xFFFF0000,read.getRGB(0,0));assertEquals(0xFF0000FF,read.getRGB(0,1));
        byte[] top=dib(1,-1,32,40,0);ByteBuffer.wrap(top).order(ByteOrder.LITTLE_ENDIAN).putInt(40,0x00112233);
        assertEquals(0xFF112233,ClipboardImage.decodeDib(top).getRGB(0,0));
    }
    @Test void windowsV5BitfieldsKeepAlpha()throws Exception{
        byte[] data=dib(1,-1,32,124,3);var b=ByteBuffer.wrap(data).order(ByteOrder.LITTLE_ENDIAN);
        b.putInt(40,0xFF0000);b.putInt(44,0xFF00);b.putInt(48,0xFF);b.putInt(52,0xFF000000);b.putInt(124,0x80112233);
        assertEquals(0x80112233,ClipboardImage.decodeDib(data).getRGB(0,0));
    }
    @Test void invalidOrOversizedDibNeverAllocatesTheDeclaredImage(){
        assertThrows(IOException.class,()->ClipboardImage.decodeDib(new byte[10]));
        byte[] data=dib(1,1,32,40,0);var b=ByteBuffer.wrap(data).order(ByteOrder.LITTLE_ENDIAN);
        b.putInt(4,100000);b.putInt(8,100000);assertThrows(IOException.class,()->ClipboardImage.decodeDib(data));
        b.putInt(4,2);b.putInt(8,2);assertThrows(IOException.class,()->ClipboardImage.decodeDib(data));
    }
}
