package cafe.deuterium.core.game;

/** Removes only the installed TrChat Normal/Global formatter's trusted envelope.
 * Raw Bukkit message content never goes through this method. */
public final class ForwardedChatBody {
    private ForwardedChatBody(){}
    public static String fromTrChat(String formatted,String gameId) {
        if(formatted==null)return "";
        String s=org.bukkit.ChatColor.stripColor(formatted).strip();
        int start=0;
        // The deployed channel formatter has an optional [world/server] segment.
        if(s.startsWith("[")){int end=s.indexOf(']');if(end<0||end>80)return s;start=end+1;while(start<s.length()&&Character.isWhitespace(s.charAt(start)))start++;}
        String prefix=gameId+": ";
        if(s.startsWith(prefix,start))return s.substring(start+prefix.length());
        prefix=gameId+"： ";
        if(s.startsWith(prefix,start))return s.substring(start+prefix.length());
        return s;
    }
}
