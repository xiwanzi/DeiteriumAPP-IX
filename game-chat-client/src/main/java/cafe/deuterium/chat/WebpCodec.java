package cafe.deuterium.chat;

import java.awt.image.BufferedImage;
import java.io.*;
import java.nio.file.*;

/** Bounded libwebp 1.5 bridge. All callers run on the dedicated media worker. */
final class WebpCodec implements AutoCloseable {
    private static boolean loaded;
    private long handle;
    final int width,height,frames,loops;
    WebpCodec(byte[] bytes)throws IOException {
        load();handle=openNative(bytes);if(handle==0)throw new IOException("无法读取图片");
        int[] info=infoNative(handle);width=info[0];height=info[1];frames=info[2];loops=info[3];
    }
    record Frame(BufferedImage image,int duration){}
    Frame next()throws IOException {
        if(handle==0)throw new IOException("Decoder closed");int[] data=nextNative(handle);if(data==null)return null;
        var image=new BufferedImage(data[0],data[1],BufferedImage.TYPE_INT_ARGB);image.setRGB(0,0,data[0],data[1],data,3,data[0]);
        return new Frame(image,Math.max(20,data[2]));
    }
    static byte[] encode(BufferedImage image,float quality,boolean lossless)throws IOException {
        load();return encodeNative(image.getRGB(0,0,image.getWidth(),image.getHeight(),null,0,image.getWidth()),image.getWidth(),image.getHeight(),quality,lossless);
    }
    private static synchronized void load()throws IOException {
        if(loaded)return;
        String platform=System.getProperty("os.name").toLowerCase().contains("win")?"windows-x86_64":System.getProperty("os.name").toLowerCase().contains("mac")?"macos":"linux-x86_64";
        String file=System.mapLibraryName("deuterium_webp");
        try(var in=WebpCodec.class.getResourceAsStream("/native/"+platform+"/"+file)) {
            if(in==null)throw new IOException("当前系统尚未提供图片编解码组件");
            Path dir=Files.createTempDirectory("deuterium-codec-");Path lib=dir.resolve(file);Files.copy(in,lib);lib.toFile().deleteOnExit();dir.toFile().deleteOnExit();
            System.load(lib.toAbsolutePath().toString());loaded=true;
        }catch(UnsatisfiedLinkError e){throw new IOException("图片编解码组件无法加载",e);}
    }
    @Override public void close(){if(handle!=0){closeNative(handle);handle=0;}}
    private static native byte[] encodeNative(int[]argb,int width,int height,float quality,boolean lossless)throws IOException;
    private static native long openNative(byte[]bytes)throws IOException;
    private static native int[] infoNative(long handle);
    private static native int[] nextNative(long handle)throws IOException;
    private static native void closeNative(long handle);
}
