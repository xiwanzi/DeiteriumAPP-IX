package cafe.deuterium.chat;

import net.minecraft.client.Minecraft;
import java.io.IOException;
import java.nio.file.*;

/** Local three-step text preference; never changes the screen or game GUI scale. */
final class ChatTextSize {
    private static int current=-1;
    private ChatTextSize(){}
    private static Path file(){return Minecraft.getInstance().gameDirectory.toPath().resolve("config/deuterium-chat/text-size");}
    static int level(){if(current<0)current=read(file());return current;}
    static int next(int level){return level>=0&&level<2?level+1:0;}
    static int pixels(int base,int level){return Math.round(base*(1+.1f*Math.clamp(level,0,2)));}
    static int pixels(int base){return pixels(base,level());}
    static String label(){return (100+10*level())+"%";}
    static void cycle(){
        current=next(level());
        try{write(file(),current);}catch(IOException e){org.slf4j.LoggerFactory.getLogger(ChatTextSize.class).warn("Chat text size could not be saved",e);}
        ChatStore.INSTANCE.changed();
    }
    static int read(Path file){
        try{if(!Files.isRegularFile(file)||Files.size(file)>16)return 0;int value=Integer.parseInt(Files.readString(file).strip());return value>=0&&value<=2?value:0;}
        catch(IOException|NumberFormatException e){return 0;}
    }
    static void write(Path file,int level)throws IOException{
        Files.createDirectories(file.getParent());Path pending=file.resolveSibling(file.getFileName()+".tmp");
        Files.writeString(pending,Integer.toString(level));Files.move(pending,file,StandardCopyOption.REPLACE_EXISTING);
    }
}
