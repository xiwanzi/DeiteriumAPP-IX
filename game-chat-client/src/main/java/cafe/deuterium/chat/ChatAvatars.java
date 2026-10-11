package cafe.deuterium.chat;

import com.mojang.blaze3d.platform.NativeImage;
import net.minecraft.client.Minecraft;
import net.minecraft.client.renderer.texture.DynamicTexture;
import net.minecraft.resources.ResourceLocation;
import java.util.*;
import java.util.concurrent.*;

/** Render-thread cache; durable portraits live on the server and survive players going offline. */
final class ChatAvatars {
    private static final Map<String,Entry> entries=new LinkedHashMap<>(32,.75f,true);
    private static final ExecutorService io=new ThreadPoolExecutor(1,1,0,TimeUnit.SECONDS,new ArrayBlockingQueue<>(256),r->{var t=new Thread(r,"Deuterium-Chat-Avatars");t.setDaemon(true);return t;},new ThreadPoolExecutor.AbortPolicy());
    static final class Entry {ResourceLocation observed,portrait;boolean busy;long retry;}
    static Entry get(String name,ResourceLocation online){
        String key=name.toLowerCase(Locale.ROOT);var e=entries.computeIfAbsent(key,k->new Entry());if(online!=null)e.observed=online;
        var client=ChatClient.INSTANCE;var api=client.api;long epoch=client.epoch,now=System.currentTimeMillis();
        if(name.matches("[A-Za-z0-9_]{1,16}")&&e.portrait==null&&!e.busy&&now>=e.retry&&api!=null&&api.valid()){
            e.busy=true;
            try{io.execute(()->{
                byte[] png=null;
                try{var raw=api.request("GET","/api/v1/chat/avatars/"+name,null).getAsJsonObject();String encoded=ChatMessage.str(raw,"png");if(encoded.length()>5500)throw new IllegalArgumentException("Avatar too large");png=Base64.getDecoder().decode(encoded);}catch(Exception ignored){}
                byte[] result=png;
                Minecraft.getInstance().execute(()->{
                    if(client.epoch!=epoch||entries.get(key)!=e)return;
                    e.busy=false;e.retry=System.currentTimeMillis()+300000;
                    if(result==null)return;
                    try{var image=NativeImage.read(result);if(image.getWidth()!=8||image.getHeight()!=8){image.close();return;}
                        var texture=new DynamicTexture(image);texture.setFilter(false,false);e.portrait=Minecraft.getInstance().getTextureManager().register("deuterium_avatar",texture);
                    }catch(Exception ignored){}
                });
            });}catch(RejectedExecutionException full){e.busy=false;e.retry=now+10000;}
        }
        while(entries.size()>256){var it=entries.entrySet().iterator();var oldest=it.next().getValue();it.remove();if(oldest.portrait!=null)Minecraft.getInstance().getTextureManager().release(oldest.portrait);}
        return e;
    }
    static void clear(){for(var e:entries.values())if(e.portrait!=null)Minecraft.getInstance().getTextureManager().release(e.portrait);entries.clear();((ThreadPoolExecutor)io).getQueue().clear();}
}
