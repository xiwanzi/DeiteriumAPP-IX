package cafe.deuterium.chat;

import com.google.gson.*;
import com.mojang.blaze3d.platform.NativeImage;
import com.mojang.blaze3d.systems.RenderSystem;
import net.minecraft.client.Minecraft;
import net.minecraft.client.gui.GuiGraphics;
import net.minecraft.client.renderer.texture.DynamicTexture;
import net.minecraft.network.chat.Component;
import net.minecraft.resources.ResourceLocation;
import org.lwjgl.system.MemoryStack;
import org.lwjgl.util.tinyfd.TinyFileDialogs;
import java.awt.image.BufferedImage;
import java.io.*;
import java.nio.file.*;
import java.security.MessageDigest;
import java.time.Instant;
import java.util.*;
import java.util.concurrent.*;

/** Single bounded codec queue, immutable captured session, bounded decoded/GPU/disk caches. */
final class ChatMediaClient {
    private final ChatClient client;
    private final ThreadPoolExecutor worker=new ThreadPoolExecutor(1,1,0,TimeUnit.SECONDS,new ArrayBlockingQueue<>(24),r->{var t=new Thread(r,"Deuterium-Chat-Media");t.setDaemon(true);return t;},new ThreadPoolExecutor.AbortPolicy());
    private final LinkedHashMap<String,ImageEntry> images=new LinkedHashMap<>(32,.75f,true);
    private final Map<String,Job> jobs=new LinkedHashMap<>();
    private final ConcurrentLinkedQueue<Runnable> completed=new ConcurrentLinkedQueue<>();
    private JsonArray stickers=new JsonArray();
    private long resident;
    private String previewRequest="";
    private static final long MEMORY_LIMIT=64L<<20,DISK_LIMIT=500L<<20;
    private record Pixels(int width,int height,NativeImage image,long bytes){}
    private static final class ImageEntry {
        Pixels pixels;ResourceLocation texture;DynamicTexture dynamic;boolean busy=true,disposed;String failure="";long used=System.nanoTime();
    }
    private record Job(String id,Path dir,ChatApi api,long epoch,String scope,boolean importing,boolean sticker){}
    ChatMediaClient(ChatClient client){this.client=client;}
    JsonArray stickers(){return stickers;}
    boolean ready(String id){var e=images.get(id+":thumb");return e!=null&&e.texture!=null;}
    void refreshStickers(){client.submit(a->a.request("GET","/api/v1/chat/stickers",null),r->{stickers=ChatClient.array(r);ChatStore.INSTANCE.changed();if(Minecraft.getInstance().screen instanceof ChatScreenFull full){String state=ChatPanels.state(full.view.root);if(state.equals("personal")||state.equals("manage"))ChatPanels.restore(full.view.root,state,full.view::switchScope);}});}
    void choose(boolean importing){
        queue(()->{try(var stack=MemoryStack.stackPush()){
            var patterns=stack.mallocPointer(4);patterns.put(stack.UTF8("*.png")).put(stack.UTF8("*.jpg")).put(stack.UTF8("*.webp")).put(stack.UTF8("*.gif")).flip();
            String file=TinyFileDialogs.tinyfd_openFileDialog(importing?"添加个人贴纸":"发送图片","",patterns,"图片",false);
            if(file!=null)Minecraft.getInstance().execute(()->{if(importing)pick(Path.of(file),true);else preview(Path.of(file));});
        }});
    }
    void pick(Path path,boolean importing){start(path,importing,importing);}
    void preview(Path path){
        preview(target->{try(var in=Files.newInputStream(path)){Files.write(target,ChatApi.read(in,20<<20));}});
    }
    boolean pasteImage(){
        try{var source=ClipboardImage.capture();if(source==null)return false;preview(source);}
        catch(Exception e){client.notice(e instanceof IllegalStateException?"剪贴板正忙，请稍后再粘贴":e.getMessage()==null?"无法读取剪贴板图片":e.getMessage());}
        return true;
    }
    private void preview(ClipboardImage.Source source){
        if(!previewRequest.isEmpty()){client.notice("正在准备上一张图片，请稍候");return;}
        ChatApi api=client.api;if(api==null||!api.valid()){client.notice("云端聊天尚未连接");return;}
        String scope=ChatStore.INSTANCE.current;if(scope.equals("inbox")){client.notice("请先选择一位玩家");return;}
        if(jobs.size()>=4){client.notice("请等待当前图片处理完成");return;}
        String id=UUID.randomUUID().toString();Job job=new Job(id,cache(api).resolve("pending").resolve(id),api,client.epoch,scope,false,false);
        var previous=Minecraft.getInstance().screen;previewRequest=id;
        client.notice("正在准备图片预览…");
        if(!queue(()->{try{
            Files.createDirectories(job.dir);source.writeTo(job.dir.resolve("source"));
            var result=ChatImageCompressor.prepare(job.dir.resolve("source"),false);Files.write(job.dir.resolve("image.webp"),result.image());Files.write(job.dir.resolve("thumbnail.webp"),result.thumbnail());
            saveCache(api,hash(result.image()),result.image());saveCache(api,hash(result.thumbnail()),result.thumbnail());
            JsonObject media=new JsonObject();media.addProperty("mediaId",id);media.addProperty("sha256",hash(result.image()));media.addProperty("thumbnailSha256",hash(result.thumbnail()));media.addProperty("width",result.width());media.addProperty("height",result.height());media.addProperty("type","image");media.addProperty("animated",result.animated());
            completed.add(()->{if(previewRequest.equals(id))previewRequest="";if(job.epoch!=client.epoch||Minecraft.getInstance().screen!=previous||!scope.equals(ChatStore.INSTANCE.current)){discard(job);return;}var message=new ChatMessage(id,api.owner,Minecraft.getInstance().getUser().getName(),scope.startsWith("private:")?"private":scope,api.server,scope.startsWith("private:")?scope.substring(8):"",Component.literal("[图片]"),media,Instant.now(),false,false,"");
                Minecraft.getInstance().setScreen(new MediaViewer(message,Minecraft.getInstance().screen,()->{if(job.epoch!=client.epoch){client.notice("游戏会话已切换，请重新选择图片");discard(job);return;}jobs.put(id,job);message.pending=true;ChatStore.INSTANCE.add(message);queue(()->{try{writeJob(job);process(job);}catch(Exception e){failed(job,e);}});},()->discard(job)));
            });
        }catch(Exception e){discard(job);completed.add(()->{if(previewRequest.equals(id))previewRequest="";if(job.epoch==client.epoch)client.notice(e.getMessage());});}}))previewRequest="";
    }
    private void writeJob(Job job)throws IOException{var meta=new JsonObject();meta.addProperty("id",job.id);meta.addProperty("scope",job.scope);meta.addProperty("importing",job.importing);meta.addProperty("sticker",job.sticker);meta.addProperty("server",job.api.server);Files.writeString(job.dir.resolve("job.json"),meta.toString());}
    private void discard(Job job){queue(()->{try{for(String name:List.of("source","image.webp","thumbnail.webp","job.json"))Files.deleteIfExists(job.dir.resolve(name));Files.deleteIfExists(job.dir);}catch(IOException ignored){}});}
    private void start(Path path,boolean importing,boolean sticker){
        var api=client.api;if(api==null||!api.valid()){client.notice("云端聊天尚未连接");return;}
        String scope=ChatStore.INSTANCE.current;
        if(!importing&&scope.equals("inbox")){client.notice("请先选择一位玩家");return;}
        if(jobs.size()>=4){client.notice("请等待当前图片处理完成");return;}
        String id=UUID.randomUUID().toString();Path dir=cache(api).resolve("pending").resolve(id);
        Job job=new Job(id,dir,api,client.epoch,scope,importing,sticker);jobs.put(id,job);
        if(!importing){var m=new ChatMessage(id,api.owner,Minecraft.getInstance().getUser().getName(),scope.startsWith("private:")?"private":scope,api.server,scope.startsWith("private:")?scope.substring(8):"",Component.literal("正在准备图片…"),null,Instant.now(),false,false,"");m.pending=true;ChatStore.INSTANCE.add(m);}
        queue(()->{try{Files.createDirectories(dir);try(var in=Files.newInputStream(path)){Files.write(dir.resolve("source"),ChatApi.read(in,20<<20));}
                writeJob(job);process(job);
            }catch(Exception e){failed(job,e);}});
    }
    void builtIn(int index){builtIn(index,false);}
    void builtIn(int index,boolean importing){
        var api=client.api;if(api==null){client.notice("云端聊天尚未连接");return;}
        queue(()->{try(var in=ChatMediaClient.class.getResourceAsStream("/assets/deuterium_chat/textures/chat-native/sns_sticker_%03d.png".formatted(index))){
            if(in==null)throw new IOException("贴纸资源不存在");Path file=cache(api).resolve("builtin-"+index+".png");Files.createDirectories(file.getParent());Files.write(file,in.readAllBytes());Minecraft.getInstance().execute(()->start(file,importing,true));
        }catch(Exception e){Minecraft.getInstance().execute(()->client.notice(e.getMessage()));}});
    }
    private void process(Job job)throws Exception {
        if(job.epoch!=client.epoch)return;
        Path main=job.dir.resolve("image.webp"),thumb=job.dir.resolve("thumbnail.webp");
        if(!Files.exists(main)||!Files.exists(thumb)){var result=ChatImageCompressor.prepare(job.dir.resolve("source"),job.sticker);Files.write(main,result.image());Files.write(thumb,result.thumbnail());}
        byte[] bytes=Files.readAllBytes(main),preview=Files.readAllBytes(thumb);
        JsonObject input=new JsonObject();input.addProperty("clientRequestId",job.id);input.addProperty("sha256",hash(bytes));input.addProperty("thumbnailSha256",hash(preview));input.addProperty("contentMd5",md5(bytes));input.addProperty("thumbnailMd5",md5(preview));input.addProperty("sizeBytes",bytes.length);input.addProperty("thumbnailBytes",preview.length);
        JsonObject upload=job.api.request("POST","/api/v1/chat/media/uploads",input).getAsJsonObject();
        if(!ChatMessage.str(upload,"status").equals("READY")){
            ChatApi.upload(upload.getAsJsonObject("authorization"),bytes);ChatApi.upload(upload.getAsJsonObject("thumbnailAuthorization"),preview);
            upload=job.api.request("POST","/api/v1/chat/media/uploads/"+ChatMessage.str(upload,"uploadId")+"/complete",new JsonObject()).getAsJsonObject();
        }
        JsonObject media=upload.getAsJsonObject("media");media.addProperty("type",job.sticker?"sticker":"image");
        saveCache(job.api,hash(bytes),bytes);saveCache(job.api,hash(preview),preview);
        if(job.epoch!=client.epoch)return;
        JsonObject body=new JsonObject();
        if(job.importing){body.addProperty("mediaId",ChatMessage.str(media,"mediaId"));job.api.request("POST","/api/v1/chat/stickers",body);}
        else {
            body.addProperty("clientMessageId",job.id);body.addProperty("content","");JsonObject attachment=new JsonObject();attachment.addProperty("mediaId",ChatMessage.str(media,"mediaId"));attachment.addProperty("type",job.sticker?"sticker":"image");body.add("media",attachment);
            JsonObject result;
            if(job.scope.equals("world")){
                result=job.api.request("POST","/api/v1/chat/messages",body).getAsJsonObject();
                if(!ChatMessage.str(result,"status").equals("accepted")||!result.has("message"))throw new IOException(result.has("error")?ChatMessage.str(result.getAsJsonObject("error"),"message"):"发送结果待确认，请重试");
                result=result.getAsJsonObject("message");
            }else if(job.scope.startsWith("private:"))result=job.api.request("POST","/api/v1/chat/conversations/"+job.scope.substring(8)+"/messages",body).getAsJsonObject();
            else result=client.sendScopedMedia(job.scope,attachment,job.id,job.epoch);
            var message=ChatMessage.fromJson(result,false,job.scope.startsWith("private:")?"private":job.scope,job.scope.startsWith("private:")?job.scope.substring(8):"");
            completed.add(()->{if(job.epoch==client.epoch)ChatStore.INSTANCE.replace(job.id,message);});
        }
        for(String name:List.of("source","image.webp","thumbnail.webp","job.json"))Files.deleteIfExists(job.dir.resolve(name));Files.deleteIfExists(job.dir);
        completed.add(()->{if(job.epoch==client.epoch){jobs.remove(job.id);if(job.importing)refreshStickers();}});
    }
    private void failed(Job job,Exception e){completed.add(()->{if(job.epoch!=client.epoch)return;for(var m:ChatStore.INSTANCE.hud())if(m.id.equals(job.id)){m.pending=false;m.failure="发送失败 · 点击重试";}client.notice(e.getMessage());ChatStore.INSTANCE.changed();});}
    void retry(String id){Job old=jobs.get(id);ChatApi api=client.api;if(old==null||api==null)return;if(!old.api.owner.equals(api.owner)||!old.api.base.equals(api.base)){client.notice("请切回原账号重试");return;}Job job=new Job(old.id,old.dir,api,client.epoch,old.scope,old.importing,old.sticker);jobs.put(id,job);for(var m:ChatStore.INSTANCE.hud())if(m.id.equals(id)){m.pending=true;m.failure="";}ChatStore.INSTANCE.changed();queue(()->{try{process(job);}catch(Exception e){failed(job,e);}});}
    void sendSticker(JsonObject media){
        String scope=ChatStore.INSTANCE.current,id=UUID.randomUUID().toString();long generation=client.epoch;
        if(scope.equals("inbox")){client.notice("请先选择一位玩家");return;}
        JsonObject b=new JsonObject();b.addProperty("clientMessageId",id);b.addProperty("content","");var attachment=new JsonObject();attachment.addProperty("mediaId",ChatMessage.str(media,"mediaId"));attachment.addProperty("type","sticker");b.add("media",attachment);
        client.submit(a->{
            if(scope.equals("world")){var response=a.request("POST","/api/v1/chat/messages",b).getAsJsonObject();if(!response.has("message"))throw new IOException("发送未完成，请稍后重试");return response.get("message");}
            if(scope.startsWith("private:"))return a.request("POST","/api/v1/chat/conversations/"+scope.substring(8)+"/messages",b);
            return client.sendScopedMedia(scope,attachment,id,generation);
        },r->ChatStore.INSTANCE.add(ChatMessage.fromJson(r.getAsJsonObject(),false,scope.startsWith("private:")?"private":scope,scope.startsWith("private:")?scope.substring(8):"")));
    }
    void collect(JsonObject media){if(media.has("builtin")){builtIn(media.get("builtin").getAsInt(),true);return;}JsonObject b=new JsonObject();b.addProperty("mediaId",ChatMessage.str(media,"mediaId"));client.submit(a->a.request("POST","/api/v1/chat/stickers",b),r->refreshStickers());}
    void removeSticker(String id){client.submit(a->a.request("DELETE","/api/v1/chat/stickers/"+id,null),r->refreshStickers());}
    void draw(GuiGraphics g,ChatMessage message,float x,float y,float w,float h,int alpha,boolean full){
        JsonObject media=message.media;if(media==null)return;
        if(media.has("builtin")){int index=media.get("builtin").getAsInt();if(index>=1&&index<=14){RenderSystem.enableBlend();g.setColor(1,1,1,alpha/255f);g.blit(DeuteriumChat.id("textures/chat-native/sns_sticker_%03d.png".formatted(index)),(int)x,(int)y,(int)w,(int)h,0,0,256,256,256,256);g.setColor(1,1,1,1);}return;}
        String id=ChatMessage.str(media,"mediaId");
        boolean main=full;
        String key=id+(full?":full":":thumb");ImageEntry entry=images.get(key);
        if(entry==null){entry=new ImageEntry();images.put(key,entry);load(entry,message,main,full);}
        entry.used=System.nanoTime();
        if(entry.texture!=null){RenderSystem.enableBlend();g.setColor(1,1,1,alpha/255f);g.blit(entry.texture,(int)x,(int)y,(int)w,(int)h,0,0,entry.pixels.width,entry.pixels.height,entry.pixels.width,entry.pixels.height);g.setColor(1,1,1,1);}
        else {g.fill((int)x,(int)y,(int)(x+w),(int)(y+h),alpha/5<<24|0x71807C);ChatTimeline.drawText(g,ChatStyle.body(Component.literal(entry.failure.isEmpty()?"图片加载中…":"图片不可用")),x+2,y+3,.45f,alpha<<24|0xB1BDB6);}
    }
    private void load(ImageEntry entry,ChatMessage message,boolean main,boolean full){
        ChatApi api=client.api;long epoch=client.epoch;if(api==null){entry.busy=false;entry.failure="尚未连接";return;}
        queue(()->{try{
            String hash=ChatMessage.str(message.media,main?"sha256":"thumbnailSha256");ChatImageCompressor.check(hash.matches("[0-9a-f]{64}"),"图片信息无效");Path file=cache(api).resolve("media").resolve(hash);byte[] bytes;
            if(Files.exists(file))bytes=Files.readAllBytes(file);else{
                JsonObject access=api.request("GET","/api/v1/chat/media/files/"+ChatMessage.str(message.media,"mediaId")+"/access"+(message.id.startsWith("own_")?"":"?messageId="+message.id),null).getAsJsonObject();
                bytes=ChatApi.download(ChatMessage.str(access,main?"url":"thumbnailUrl"),main?2<<20:80<<10);ChatImageCompressor.check(hash(bytes).equals(hash),"图片完整性校验失败");saveCache(api,hash,bytes);
            }
            ChatImageCompressor.check(hash(bytes).equals(hash),"本地图片不完整");Pixels pixels=decode(bytes,full?1920:256);
            if(epoch!=client.epoch){disposePixels(pixels);return;}
            completed.add(()->{if(epoch==client.epoch&&!entry.disposed){entry.pixels=pixels;entry.busy=false;resident+=pixels.bytes;evict();}else disposePixels(pixels);});
        }catch(Exception e){completed.add(()->{if(epoch==client.epoch){entry.failure=e.getMessage();entry.busy=false;}});}});
    }
    private static Pixels decode(byte[] bytes,int max)throws Exception {
        // All game surfaces intentionally display only frame one, including the viewer.
        try(var source=new WebpCodec(bytes)){
            int bound=max==1920&&source.height/(double)source.width>=2.5?8192:max;
            double f=Math.min(1,bound/(double)Math.max(source.width,source.height));int w=Math.max(1,(int)(source.width*f)),h=Math.max(1,(int)(source.height*f));
            ChatImageCompressor.check((long)w*h<=8_000_000,"图片显示尺寸过大");
            var first=source.next();if(first==null)throw new IOException("图片内容不完整");
            try{return new Pixels(w,h,pixels(first.image(),w,h),(long)w*h*4);}finally{first.image().flush();}
        }
    }
    private static NativeImage pixels(BufferedImage source,int width,int height){
        var scaled=ChatImageCompressor.resize(source,width,height);var image=new NativeImage(width,height,false);
        int[] row=new int[width];for(int y=0;y<height;y++){scaled.getRGB(0,y,width,1,row,0,width);for(int x=0;x<width;x++){int v=row[x];image.setPixelRGBA(x,y,(v&0xFF00FF00)|((v>>>16)&255)|((v&255)<<16));}}
        if(scaled!=source)scaled.flush();return image;
    }
    void pump(){
        for(int i=0;i<4;i++){Runnable r=completed.poll();if(r==null)break;r.run();}
        long start=System.nanoTime();int count=0;
        for(var entry:images.values())if(entry.pixels!=null&&entry.texture==null){
            if(count++>=1||System.nanoTime()-start>2_000_000)break;Pixels p=entry.pixels;
            entry.dynamic=new DynamicTexture(p.image);entry.texture=Minecraft.getInstance().getTextureManager().register("deuterium_chat",entry.dynamic);entry.dynamic.setFilter(true,false);
        }
    }
    private static void disposePixels(Pixels p){p.image.close();}
    private void release(ImageEntry e){e.disposed=true;if(e.texture!=null)Minecraft.getInstance().getTextureManager().release(e.texture);if(e.pixels!=null){resident-=e.pixels.bytes;disposePixels(e.pixels);e.pixels=null;}}
    private void evict(){var it=images.entrySet().iterator();while((resident>MEMORY_LIMIT||images.size()>48)&&it.hasNext()){var entry=it.next().getValue();if(entry.busy)continue;release(entry);it.remove();}}
    void clear(){previewRequest="";worker.getQueue().clear();for(Runnable r;(r=completed.poll())!=null;)r.run();for(var e:images.values())release(e);images.clear();jobs.clear();stickers=new JsonArray();resident=0;}
    void clearCachedMedia(){ChatApi api=client.api;for(var e:images.values())release(e);images.clear();resident=0;if(api==null)return;queue(()->{try{Path dir=cache(api).resolve("media");if(Files.isDirectory(dir))try(var files=Files.list(dir)){for(Path p:files.filter(Files::isRegularFile).toList())if(p.getFileName().toString().matches("[0-9a-f]{64}"))Files.deleteIfExists(p);}completed.add(()->client.notice("本机图片缓存已清理，待发送内容已保留"));}catch(IOException e){completed.add(()->client.notice("缓存未能完全清理，请稍后重试"));}});}
    private boolean queue(Runnable task){try{worker.execute(task);return true;}catch(RejectedExecutionException e){Minecraft.getInstance().execute(()->client.notice("图片正在处理中，请稍后重试"));return false;}}
    private static Path cache(ChatApi api){return Minecraft.getInstance().gameDirectory.toPath().resolve("deuterium-chat").resolve(hash((api.base+"|"+api.owner).getBytes(java.nio.charset.StandardCharsets.UTF_8)).substring(0,32));}
    private static void saveCache(ChatApi api,String hash,byte[] bytes)throws IOException{Path dir=cache(api).resolve("media");Files.createDirectories(dir);Path file=dir.resolve(hash);if(!Files.exists(file)){Path temp=Files.createTempFile(dir,"incoming-",".tmp");Files.write(temp,bytes);Files.move(temp,file,StandardCopyOption.REPLACE_EXISTING);}
        try(var paths=Files.list(dir)){var list=paths.filter(Files::isRegularFile).sorted(Comparator.comparingLong(p->{try{return Files.getLastModifiedTime(p).toMillis();}catch(IOException e){return 0;}})).toList();long size=0;for(Path p:list)size+=Files.size(p);for(Path p:list){if(size<=DISK_LIMIT)break;long n=Files.size(p);Files.deleteIfExists(p);size-=n;}}}
    static String hash(byte[] b){try{return HexFormat.of().formatHex(MessageDigest.getInstance("SHA-256").digest(b));}catch(Exception e){throw new IllegalStateException(e);}}
    private static String md5(byte[] b){try{return Base64.getEncoder().encodeToString(MessageDigest.getInstance("MD5").digest(b));}catch(Exception e){throw new IllegalStateException(e);}}
}
