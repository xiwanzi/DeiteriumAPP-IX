package cafe.deuterium.chat;

import javax.imageio.*;
import javax.imageio.stream.*;
import java.awt.*;
import java.awt.image.BufferedImage;
import java.io.*;
import java.nio.charset.StandardCharsets;
import java.nio.file.*;
import java.util.*;
import org.w3c.dom.Node;

/** Same output limits/quality ladder as App ChatImageCompressor; worker-only. */
final class ChatImageCompressor {
    record Prepared(byte[] image,byte[] thumbnail,int width,int height,boolean animated){}
    record Size(int width,int height,boolean longImage){}
    static Size size(int w,int h,boolean sticker,boolean animated)throws IOException {
        check(w>0&&h>0&&(long)w*h<=40_000_000,"图片尺寸过大，请裁剪后再试");boolean tall=h/(double)w>=2.5;
        double factor=Math.min(1,sticker||animated?512.0/Math.max(w,h):tall?Math.min(1440.0/w,Math.min(8192.0/h,Math.sqrt(8_000_000.0/w/h))):1920.0/Math.max(w,h));
        return new Size(Math.max(1,(int)(w*factor)),Math.max(1,(int)(h*factor)),tall);
    }
    static Prepared prepare(Path path,boolean sticker)throws Exception {
        long length=Files.size(path);check(length>0&&length<=20L*1024*1024,"请选择 20MB 以内的图片");
        byte[] bytes=Files.readAllBytes(path);
        if(tag(bytes,0,3).equals("GIF"))return animation(bytes,true);
        if(tag(bytes,0,4).equals("RIFF"))try(var webp=new WebpCodec(bytes)){
            if(webp.frames>1)return animation(bytes,false);
            return still(webp.next().image(),sticker,false);
        }
        try(var stream=new MemoryCacheImageInputStream(new ByteArrayInputStream(bytes))){
            var readers=ImageIO.getImageReaders(stream);check(readers.hasNext(),"请选择 JPEG、PNG、WebP 或 GIF 图片");var reader=readers.next();
            try{reader.setInput(stream);String format=reader.getFormatName().toLowerCase(Locale.ROOT);check(format.equals("png")||format.equals("jpeg"),"请选择 JPEG、PNG、WebP 或 GIF 图片");
                int w=reader.getWidth(0),h=reader.getHeight(0);Size size=size(w,h,sticker,false);var param=reader.getDefaultReadParam();int sample=1;
                while(w/(sample*2)>=size.width&&h/(sample*2)>=size.height)sample*=2;
                param.setSourceSubsampling(sample,sample,0,0);BufferedImage image=reader.read(0,param);
                return still(ExifOrientation.apply(image,bytes),sticker,format.equals("png"));
            }finally{reader.dispose();}
        }
    }
    private static Prepared still(BufferedImage source,boolean sticker,boolean png)throws Exception {
        Size target=size(source.getWidth(),source.getHeight(),sticker,false);var image=resize(source,target.width,target.height);if(image!=source)source.flush();
        boolean text=target.longImage||png;int[] qualities=text?new int[]{90,88,84}:sticker?new int[]{85,80,76}:new int[]{82,77,72};
        int goal=sticker?200<<10:text?1<<20:800<<10,limit=sticker?512<<10:2<<20;byte[] best=null;
        try{for(int i=0;i<qualities.length;i++){interrupted();byte[] out=WebpCodec.encode(image,qualities[i],text&&i==0);if(best==null||out.length<best.length)best=out;if(best.length<=goal)break;}
            check(best.length<=limit,"图片压缩后仍然过大，请裁剪后再试");return new Prepared(best,thumbnail(image),image.getWidth(),image.getHeight(),false);
        }finally{image.flush();}
    }
    private static byte[] thumbnail(BufferedImage b)throws IOException {
        double f=Math.min(1,320.0/Math.max(b.getWidth(),b.getHeight()));var small=resize(b,Math.max(1,(int)(b.getWidth()*f)),Math.max(1,(int)(b.getHeight()*f)));
        byte[] out=WebpCodec.encode(small,70,false);if(small!=b)small.flush();check(out.length<=80<<10,"图片预览未能处理");return out;
    }
    static BufferedImage resize(BufferedImage src,int w,int h){if(src.getWidth()==w&&src.getHeight()==h)return src;var dst=new BufferedImage(w,h,BufferedImage.TYPE_INT_ARGB);var g=dst.createGraphics();g.setComposite(AlphaComposite.Src);g.setRenderingHint(RenderingHints.KEY_INTERPOLATION,RenderingHints.VALUE_INTERPOLATION_BILINEAR);g.drawImage(src,0,0,w,h,null);g.dispose();return dst;}
    private static Prepared animation(byte[] bytes,boolean gif)throws Exception {
        Prepared best=null;int[] quality={75,68,62};
        for(int attempt=0;attempt<3;attempt++)try(Source source=gif?new GifSource(bytes):new WebpSource(bytes)){
            Size size=size(source.width(),source.height(),false,true);int w=Math.max(1,(int)(size.width*(attempt==2?.75:1))),h=Math.max(1,(int)(size.height*(attempt==2?.75:1)));
            var frames=new ArrayList<byte[]>();var times=new ArrayList<Integer>();int elapsed=0,nextAt=0,lastAt=0,total=0;byte[] thumb=null;
            int interval=Math.max(50,(int)Math.ceil(source.duration()/300.0));
            for(int i=0;i<source.count();i++){interrupted();var frame=source.next();var image=frame.image();
                try{if(elapsed>=nextAt||frames.isEmpty()){
                    if(!times.isEmpty())times.set(times.size()-1,elapsed-lastAt);
                    var small=resize(image,w,h);try{byte[] encoded=WebpCodec.encode(small,quality[attempt],false);frames.add(encoded);times.add(frame.duration());total+=encoded.length;if(thumb==null)thumb=thumbnail(small);}finally{if(small!=image)small.flush();}
                    nextAt=elapsed+interval;lastAt=elapsed;
                }elapsed+=frame.duration();check(elapsed<=30000&&frames.size()<=300&&total<=6<<20,"请选择较短的动图");}finally{image.flush();}
            }
            times.set(times.size()-1,elapsed-lastAt);byte[] out=animatedRiff(frames,times,w,h,source.loops());
            if(best==null||out.length<best.image.length)best=new Prepared(out,thumb,w,h,true);if(best.image.length<=1<<20)break;
        }
        check(best!=null&&best.image.length<=2<<20,"动图压缩后仍然过大，请选择较短的动图");return best;
    }
    interface Source extends AutoCloseable {int width();int height();int count();int duration();int loops();WebpCodec.Frame next()throws Exception;void close()throws Exception;}
    private static final class WebpSource implements Source {
        final WebpCodec decoder;final int duration;
        WebpSource(byte[] bytes)throws Exception {decoder=new WebpCodec(bytes);check(decoder.width<=2048&&decoder.height<=2048&&decoder.frames<=600&&(long)decoder.width*decoder.height*decoder.frames<=160_000_000,"动图尺寸或帧数过大");int sum=0;for(int p=12;p+8<=bytes.length;){int n=le(bytes,p+4,4);check(n>=0&&n<=bytes.length-p-8,"动图内容不完整");if(tag(bytes,p,4).equals("ANMF")){check(n>=16,"动图内容不完整");sum+=Math.max(20,le(bytes,p+20,3));}p+=8+n+(n&1);}duration=sum;check(sum<=30000,"请选择30秒以内的动图");}
        public int width(){return decoder.width;}public int height(){return decoder.height;}public int count(){return decoder.frames;}public int duration(){return duration;}public int loops(){return decoder.loops;}public WebpCodec.Frame next()throws Exception{return decoder.next();}public void close(){decoder.close();}
    }
    private static final class GifSource implements Source {
        final ImageReader reader;final ImageInputStream stream;final int w,h,count,duration;final int[] delays;BufferedImage canvas,previous;Node last;int index;
        GifSource(byte[] bytes)throws Exception {
            reader=ImageIO.getImageReadersByFormatName("gif").next();stream=new MemoryCacheImageInputStream(new ByteArrayInputStream(bytes));reader.setInput(stream);
            Node logical=find(reader.getStreamMetadata().getAsTree("javax_imageio_gif_stream_1.0"),"LogicalScreenDescriptor");w=attr(logical,"logicalScreenWidth",0);h=attr(logical,"logicalScreenHeight",0);
            check(w>0&&h>0&&w<=2048&&h<=2048,"动图尺寸过大");count=reader.getNumImages(true);check(count>0&&count<=600&&(long)w*h*count<=160_000_000,"动图帧数过多");delays=new int[count];int sum=0;
            for(int i=0;i<count;i++){Node meta=reader.getImageMetadata(i).getAsTree("javax_imageio_gif_image_1.0");delays[i]=Math.max(20,attr(find(meta,"GraphicControlExtension"),"delayTime",10)*10);sum+=delays[i];}
            duration=sum;check(sum<=30000,"请选择30秒以内的动图");canvas=new BufferedImage(w,h,BufferedImage.TYPE_INT_ARGB);
        }
        public int width(){return w;}public int height(){return h;}public int count(){return count;}public int duration(){return duration;}public int loops(){return 0;}
        public WebpCodec.Frame next()throws Exception {
            if(last!=null){String disposal=value(find(last,"GraphicControlExtension"),"disposalMethod");if(disposal.equals("restoreToPrevious")&&previous!=null){canvas.flush();canvas=previous;previous=null;}else if(disposal.equals("restoreToBackgroundColor")){Node d=find(last,"ImageDescriptor");var g=canvas.createGraphics();g.setComposite(AlphaComposite.Clear);g.fillRect(attr(d,"imageLeftPosition",0),attr(d,"imageTopPosition",0),attr(d,"imageWidth",w),attr(d,"imageHeight",h));g.dispose();}}
            Node meta=reader.getImageMetadata(index).getAsTree("javax_imageio_gif_image_1.0");Node descriptor=find(meta,"ImageDescriptor");
            if(value(find(meta,"GraphicControlExtension"),"disposalMethod").equals("restoreToPrevious"))previous=copy(canvas);
            BufferedImage frame=reader.read(index);int x=attr(descriptor,"imageLeftPosition",0),y=attr(descriptor,"imageTopPosition",0);check(x>=0&&y>=0&&x+frame.getWidth()<=w&&y+frame.getHeight()<=h,"动图内容不完整");var g=canvas.createGraphics();g.drawImage(frame,x,y,null);g.dispose();frame.flush();last=meta;
            return new WebpCodec.Frame(copy(canvas),delays[index++]);
        }
        public void close()throws Exception{reader.dispose();stream.close();if(canvas!=null)canvas.flush();if(previous!=null)previous.flush();}
    }
    private static BufferedImage copy(BufferedImage src){var b=new BufferedImage(src.getWidth(),src.getHeight(),BufferedImage.TYPE_INT_ARGB);var g=b.createGraphics();g.setComposite(AlphaComposite.Src);g.drawImage(src,0,0,null);g.dispose();return b;}
    private static Node find(Node n,String name){if(n==null)return null;if(n.getNodeName().equals(name))return n;for(Node c=n.getFirstChild();c!=null;c=c.getNextSibling()){Node v=find(c,name);if(v!=null)return v;}return null;}
    private static String value(Node n,String key){return n!=null&&n.getAttributes()!=null&&n.getAttributes().getNamedItem(key)!=null?n.getAttributes().getNamedItem(key).getNodeValue():"";}
    private static int attr(Node n,String key,int def){try{return Integer.parseInt(value(n,key));}catch(Exception e){return def;}}
    static byte[] animatedRiff(java.util.List<byte[]> frames,java.util.List<Integer> times,int w,int h,int loops)throws IOException {
        var out=new ByteArrayOutputStream();out.write("WEBP".getBytes(StandardCharsets.US_ASCII));var header=new ByteArrayOutputStream();header.write(new byte[]{0x12,0,0,0});writeLE(header,w-1,3);writeLE(header,h-1,3);chunk(out,"VP8X",header.toByteArray());
        header.reset();header.write(new byte[4]);writeLE(header,Math.clamp(loops,0,65535),2);chunk(out,"ANIM",header.toByteArray());
        for(int i=0;i<frames.size();i++){var body=new ByteArrayOutputStream();body.write(new byte[6]);writeLE(body,w-1,3);writeLE(body,h-1,3);writeLE(body,times.get(i),3);body.write(2);byte[] b=frames.get(i);
            for(int p=12;p+8<=b.length;){int n=le(b,p+4,4);check(n>=0&&n<=b.length-p-8,"WebP编码无效");String kind=tag(b,p,4);if(Set.of("ALPH","VP8 ","VP8L").contains(kind))body.write(b,p,8+n+(n&1));p+=8+n+(n&1);}chunk(out,"ANMF",body.toByteArray());}
        var riff=new ByteArrayOutputStream();riff.write("RIFF".getBytes(StandardCharsets.US_ASCII));writeLE(riff,out.size(),4);out.writeTo(riff);return riff.toByteArray();
    }
    private static void chunk(ByteArrayOutputStream out,String name,byte[]data)throws IOException{out.write(name.getBytes(StandardCharsets.US_ASCII));writeLE(out,data.length,4);out.write(data);if((data.length&1)!=0)out.write(0);}
    private static void writeLE(ByteArrayOutputStream out,int v,int n){for(int i=0;i<n;i++)out.write(v>>>(8*i));}
    static int le(byte[] b,int p,int n)throws IOException{check(p>=0&&p+n<=b.length,"图片内容不完整");int v=0;for(int i=0;i<n;i++)v|=(b[p+i]&255)<<(8*i);return v;}
    static String tag(byte[] b,int p,int n){return p>=0&&p+n<=b.length?new String(b,p,n,StandardCharsets.US_ASCII):"";}
    static void check(boolean ok,String message)throws IOException{if(!ok)throw new IOException(message);}
    static void interrupted()throws InterruptedIOException{if(Thread.currentThread().isInterrupted())throw new InterruptedIOException("已取消");}
}
