package cafe.deuterium.chat;

import net.minecraft.client.Minecraft;
import java.nio.file.Files;

/** One-time pack layout migration using Xaero's own loaded transform and save path. */
final class MinimapPlacement {
    private static boolean done;private static long nextTry;
    static void migrate(){
        var mc=Minecraft.getInstance();if(done||mc.level==null||System.currentTimeMillis()<nextTry)return;nextTry=System.currentTimeMillis()+1000;
        var marker=mc.gameDirectory.toPath().resolve("config/deuterium-chat/minimap-top-right-v1");
        if(Files.exists(marker)){done=true;return;}
        try{
            var type=Class.forName("xaero.common.HudMod");var mod=type.getField("INSTANCE").get(null);
            if(mod==null||!Boolean.TRUE.equals(type.getMethod("isLoadedClient").invoke(mod)))return;
            var module=Class.forName("xaero.hud.minimap.BuiltInHudModules").getField("MINIMAP").get(null);
            var current=module.getClass().getMethod("getConfirmedTransform").invoke(module);
            var transform=current.getClass().getMethod("copy").invoke(current);var t=transform.getClass();
            t.getField("x").setInt(transform,0);t.getField("y").setInt(transform,0);
            t.getField("centered").setBoolean(transform,false);t.getField("fromRight").setBoolean(transform,true);t.getField("fromBottom").setBoolean(transform,false);
            module.getClass().getMethod("setTransform",t).invoke(module,transform);
            var settings=type.getMethod("getSettings").invoke(mod);settings.getClass().getMethod("saveSettings").invoke(settings);
            Files.createDirectories(marker.getParent());Files.writeString(marker,"Xaero minimap: top right\n");done=true;
        }catch(ClassNotFoundException absent){done=true;}
        catch(Exception failure){done=true;org.slf4j.LoggerFactory.getLogger(MinimapPlacement.class).warn("Could not migrate Xaero minimap placement",failure);}
    }
}
