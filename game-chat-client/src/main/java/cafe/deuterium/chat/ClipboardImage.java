package cafe.deuterium.chat;

import java.awt.Image;
import java.awt.Toolkit;
import java.awt.datatransfer.DataFlavor;
import java.awt.datatransfer.Transferable;
import java.awt.image.BufferedImage;
import java.io.*;
import java.nio.file.*;
import java.util.List;
import java.util.Locale;
import javax.imageio.ImageIO;
import com.sun.jna.*;
import com.sun.jna.win32.StdCallLibrary;
import com.sun.jna.platform.win32.BaseTSD;
import java.nio.ByteBuffer;
import java.nio.ByteOrder;
import java.nio.charset.StandardCharsets;

/** Captures image clipboard formats; never treats text or URLs as an image download. */
final class ClipboardImage {
    private static final long MAX_BYTES=20L<<20;
    @FunctionalInterface interface Source { void writeTo(Path target) throws Exception; }
    private ClipboardImage(){}
    static Source capture() throws Exception {
        // Minecraft explicitly enables AWT headless mode. Use the existing JNA runtime on Windows.
        if(Platform.isWindows())return windows();
        if(java.awt.GraphicsEnvironment.isHeadless())return null;
        return from(Toolkit.getDefaultToolkit().getSystemClipboard().getContents(null));
    }
    private interface ClipApi extends StdCallLibrary {
        ClipApi API=Native.load("user32",ClipApi.class);
        boolean OpenClipboard(Pointer owner);boolean CloseClipboard();
        boolean IsClipboardFormatAvailable(int format);Pointer GetClipboardData(int format);
        int RegisterClipboardFormatW(WString name);
    }
    private interface MemoryApi extends StdCallLibrary {
        MemoryApi API=Native.load("kernel32",MemoryApi.class);
        BaseTSD.SIZE_T GlobalSize(Pointer handle);Pointer GlobalLock(Pointer handle);boolean GlobalUnlock(Pointer handle);
    }
    private static Source windows() throws Exception {
        var api=ClipApi.API;int png=api.RegisterClipboardFormatW(new WString("PNG"));
        if(!api.IsClipboardFormatAvailable(png)&&!api.IsClipboardFormatAvailable(17)&&!api.IsClipboardFormatAvailable(8)&&!api.IsClipboardFormatAvailable(15))return null;
        if(!api.OpenClipboard(null))throw new IllegalStateException("剪贴板正忙，请稍后再粘贴");
        try{
            if(api.IsClipboardFormatAvailable(15)){
                byte[] drop=nativeBytes(15,1<<20);var b=ByteBuffer.wrap(drop).order(ByteOrder.LITTLE_ENDIAN);
                if(drop.length<20)throw new IOException("剪贴板文件列表无效");
                int offset=b.getInt(0);if(offset<20||offset>=drop.length)throw new IOException("剪贴板文件列表无效");
                String names=new String(drop,offset,drop.length-offset,b.getInt(16)!=0?StandardCharsets.UTF_16LE:java.nio.charset.Charset.forName(Native.getDefaultStringEncoding()));
                var files=java.util.Arrays.stream(names.split("\u0000")).filter(s->!s.isEmpty()).map(File::new).toList();
                if(files.stream().noneMatch(f->imageName(f.getName())))return null;
                if(files.size()!=1)throw new IOException("请一次粘贴一张图片");
                Path path=files.getFirst().toPath();return target->{try(var in=Files.newInputStream(path)){copy(in,target);}};
            }
            if(api.IsClipboardFormatAvailable(png)){byte[] bytes=nativeBytes(png,MAX_BYTES);return target->Files.write(target,bytes);}
            byte[] dib=nativeBytes(api.IsClipboardFormatAvailable(17)?17:8,160_001_024L);
            return target->{var image=decodeDib(dib);try{writePng(image,target);}finally{image.flush();}};
        }finally{api.CloseClipboard();}
    }
    private static byte[] nativeBytes(int format,long limit)throws IOException{
        Pointer handle=ClipApi.API.GetClipboardData(format);if(handle==null)throw new IOException("无法读取剪贴板图片");
        long size=MemoryApi.API.GlobalSize(handle).longValue();if(size<=0||size>limit)throw new IOException("剪贴板图片过大或无效");
        Pointer data=MemoryApi.API.GlobalLock(handle);if(data==null)throw new IOException("剪贴板图片暂不可用");
        try{return data.getByteArray(0,(int)size);}finally{MemoryApi.API.GlobalUnlock(handle);}
    }
    static BufferedImage decodeDib(byte[] data)throws IOException{
        if(data.length<40)throw new IOException("剪贴板位图不完整");
        var b=ByteBuffer.wrap(data).order(ByteOrder.LITTLE_ENDIAN);
        int header=b.getInt(0),width=b.getInt(4),signedHeight=b.getInt(8),bits=Short.toUnsignedInt(b.getShort(14)),compression=b.getInt(16);
        long height=Math.abs((long)signedHeight),colors=Integer.toUnsignedLong(b.getInt(32));
        if(header<40||header>data.length||width<=0||height<=0||(long)width*height>40_000_000||b.getShort(12)!=1
            ||(bits!=24&&bits!=32)||(compression!=0&&compression!=3&&compression!=6)||(bits==24&&compression!=0))throw new IOException("不支持此剪贴板位图，请复制为 PNG 图片");
        int masks=header==40?(compression==3?12:compression==6?16:0):0;
        long offset=header+(long)masks+colors*4,stride=(((long)width*bits+31)/32)*4;
        if(offset>data.length||offset+stride*height>data.length)throw new IOException("剪贴板位图不完整");
        int red=0xFF0000,green=0xFF00,blue=0xFF,alpha=0;
        if(compression!=0){
            if(header!=40&&header<52)throw new IOException("剪贴板颜色格式无效");
            red=b.getInt(40);green=b.getInt(44);blue=b.getInt(48);if(header>=56||masks==16)alpha=b.getInt(52);
            if(red==0||green==0||blue==0)throw new IOException("剪贴板颜色格式无效");
        }
        var image=new BufferedImage(width,(int)height,BufferedImage.TYPE_INT_ARGB);
        for(int y=0;y<height;y++){
            int row=(int)(offset+(signedHeight>0?height-1-y:y)*stride);
            for(int x=0;x<width;x++){
                int at=row+x*(bits/8),pixel=bits==32?b.getInt(at):(data[at]&255)|((data[at+1]&255)<<8)|((data[at+2]&255)<<16);
                image.setRGB(x,y,((alpha==0?255:channel(pixel,alpha))<<24)|(channel(pixel,red)<<16)|(channel(pixel,green)<<8)|channel(pixel,blue));
            }
        }
        return image;
    }
    private static int channel(int pixel,int mask){int shift=Integer.numberOfTrailingZeros(mask);long max=Integer.toUnsignedLong(mask>>>shift);return (int)(Integer.toUnsignedLong((pixel&mask)>>>shift)*255/max);}
    static Source from(Transferable contents) throws Exception {
        if(contents==null)return null;
        if(contents.isDataFlavorSupported(DataFlavor.javaFileListFlavor)){
            var files=(List<?>)contents.getTransferData(DataFlavor.javaFileListFlavor);
            boolean images=files.stream().anyMatch(v->v instanceof File f&&imageName(f.getName()));
            if(!images)return null;
            if(files.size()!=1)throw new IOException("请一次粘贴一张图片");
            Path path=((File)files.getFirst()).toPath();
            return target->{try(var in=Files.newInputStream(path)){copy(in,target);}};
        }
        for(var flavor:contents.getTransferDataFlavors()){
            if(flavor.isMimeTypeEqual("image/png")&&InputStream.class.isAssignableFrom(flavor.getRepresentationClass()))
                return target->{try(var in=(InputStream)contents.getTransferData(flavor)){copy(in,target);}};
        }
        if(!contents.isDataFlavorSupported(DataFlavor.imageFlavor))return null;
        return target->{
            var image=(Image)contents.getTransferData(DataFlavor.imageFlavor);
            int width=image.getWidth(null),height=image.getHeight(null);
            if(width<=0||height<=0||(long)width*height>40_000_000)throw new IOException("剪贴板图片尺寸过大或尚未就绪");
            var pixels=new BufferedImage(width,height,BufferedImage.TYPE_INT_ARGB);
            var graphics=pixels.createGraphics();
            try{if(!graphics.drawImage(image,0,0,null))throw new IOException("剪贴板图片尚未就绪，请重新复制");}
            finally{graphics.dispose();}
            try{writePng(pixels,target);}finally{pixels.flush();}
        };
    }
    private static void writePng(BufferedImage pixels,Path target)throws IOException{
            try(var out=Files.newOutputStream(target)){
                var bounded=new FilterOutputStream(out){
                    long count;
                    private void reserve(int length)throws IOException{if((count+=length)>MAX_BYTES)throw new IOException("请粘贴 20MB 以内的图片");}
                    @Override public void write(int b)throws IOException{reserve(1);out.write(b);}
                    @Override public void write(byte[] b,int offset,int length)throws IOException{reserve(length);out.write(b,offset,length);}
                };
                if(!ImageIO.write(pixels,"png",bounded))throw new IOException("无法读取剪贴板图片");
            }
    }
    private static boolean imageName(String name){return name.toLowerCase(Locale.ROOT).matches(".*\\.(png|jpe?g|gif|webp)");}
    private static void copy(InputStream in,Path target)throws IOException{
        try(var out=Files.newOutputStream(target)){
            byte[] buffer=new byte[8192];long total=0;int read;
            while((read=in.read(buffer))!=-1){if((total+=read)>MAX_BYTES)throw new IOException("请粘贴 20MB 以内的图片");out.write(buffer,0,read);}
        }
    }
}
