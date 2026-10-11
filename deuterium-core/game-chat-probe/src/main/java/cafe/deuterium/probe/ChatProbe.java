package cafe.deuterium.probe;
import org.bukkit.*;
import org.bukkit.entity.Player;
import org.bukkit.plugin.java.JavaPlugin;
import java.nio.file.*;
import java.util.*;
import com.google.gson.Gson;

/** Read-only, console-only probe; not part of the Core artifact or release. */
public final class ChatProbe extends JavaPlugin {
    @Override public void onEnable(){
        String root=Path.of(".").toAbsolutePath().normalize().toString().replace('\\','/').toLowerCase(Locale.ROOT);
        if(!Bukkit.getIp().equals("127.0.0.1")||!Set.of("c:/deiteriumapp-ix/.worktrees/game-chat-ldlib/output/game-server","c:/deiteriumapp-ix/.worktrees/game-chat-ldlib/output/game-server-odyssey").contains(root)){Bukkit.getPluginManager().disablePlugin(this);return;}
        getCommand("chatprobe").setExecutor((sender,command,label,args)->{if(sender instanceof Player)return true;try{
            Object result;
            if(args.length==1&&args[0].equals("channels")){
                var registry=Class.forName("net.neoforged.neoforge.network.registration.NetworkRegistry");var field=registry.getDeclaredField("PAYLOAD_REGISTRATIONS");field.setAccessible(true);var map=(Map<?,?>)field.get(null);List<Object> protocols=new ArrayList<>();
                for(var protocol:map.entrySet()){List<Object> channels=new ArrayList<>();for(Object registration:((Map<?,?>)protocol.getValue()).values()){Optional<?> flow=(Optional<?>)call(registration,"flow");channels.add(Map.of("id",call(registration,"id").toString(),"version",call(registration,"version"),"optional",call(registration,"optional"),"flow",flow.isPresent()?((Enum<?>)flow.get()).ordinal():-1));}protocols.add(Map.of("ordinal",((Enum<?>)protocol.getKey()).ordinal(),"name",protocol.getKey().toString(),"channels",channels));}result=Map.of("protocols",protocols,"purpose","synthetic protocol fixture for isolated server tests");
            }else if(args.length==2&&args[0].equals("team")){
                var player=Bukkit.getPlayerExact(args[1]);if(player==null)throw new IllegalStateException("test player offline");var loader=Bukkit.getPluginManager().getPlugin("DeuteriumCore").getClass().getClassLoader();var type=loader.loadClass("cafe.deuterium.core.game.TeamChatAudience");var team=type.getMethod("resolve",UUID.class).invoke(type.getConstructor().newInstance(),player.getUniqueId());result=Map.of("team",call(team,"id"),"revision",call(team,"revision"),"members",call(team,"members").toString());
            }else return true;
            getDataFolder().mkdirs();Files.writeString(getDataFolder().toPath().resolve(args[0]+".json"),new Gson().toJson(result));sender.sendMessage("chat probe recorded "+args[0]);
        }catch(Exception e){getLogger().log(java.util.logging.Level.SEVERE,"Chat probe failed",e);sender.sendMessage("chat probe failed");}return true;});
    }
    private static Object call(Object o,String method)throws Exception{return o.getClass().getMethod(method).invoke(o);}
}
