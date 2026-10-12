package cafe.deuterium.core.game;

import me.clip.placeholderapi.expansion.PlaceholderExpansion;
import net.kyori.adventure.text.Component;
import net.kyori.adventure.text.format.TextColor;
import net.kyori.adventure.text.serializer.legacy.LegacyComponentSerializer;
import org.bukkit.OfflinePlayer;
import org.bukkit.plugin.java.JavaPlugin;
import java.util.UUID;
import java.util.function.Function;

/** Read-only projection of the same authoritative equipped title used by game chat. */
public final class TitlePlaceholders extends PlaceholderExpansion {
    private static final LegacyComponentSerializer LEGACY=LegacyComponentSerializer.builder()
        .hexColors().useUnusualXRepeatedCharacterHexFormat().build();
    private final String version;
    private final Function<UUID,Component> titles;

    TitlePlaceholders(String version,Function<UUID,Component> titles){this.version=version;this.titles=titles;}
    public static Runnable register(JavaPlugin plugin,Function<UUID,Component> titles){
        var expansion=new TitlePlaceholders(plugin.getDescription().getVersion(),titles);
        if(!expansion.register())throw new IllegalStateException("Cannot register deuterium title placeholders");
        return ()->expansion.unregister();
    }
    @Override public String getIdentifier(){return "deuterium";}
    @Override public String getAuthor(){return "Deuterium";}
    @Override public String getVersion(){return version;}
    @Override public boolean persist(){return true;}
    @Override public String onRequest(OfflinePlayer player,String params){
        if(!params.equals("title")&&!params.equals("title_prefix"))return null;
        if(player==null)return "";
        Component title=titles.apply(player.getUniqueId());
        if(params.equals("title_prefix"))title=Component.text("[",TextColor.color(0xA9B3B8))
            .append(title).append(Component.text("]",TextColor.color(0xA9B3B8)));
        return LEGACY.serialize(title)+"§r";
    }
}
