package cafe.deuterium.chat;

import java.util.Locale;

/** Shared header policy; location is metadata, never embedded in message content. */
public final class ChatPresentation {
    public static final int ID_COLOR = 0xE5B3D9;
    public static final String TITLE = "封弊者";
    public static final double NEARBY_RADIUS = 50;
    private ChatPresentation() {}
    public static String suffix(String channel, String server) {
        return switch (channel) {
            case "nearby" -> "附近";
            case "private" -> "私聊";
            case "team" -> "团队";
            default -> switch (server.toLowerCase(Locale.ROOT)) {
                case "amiya", "aamiya" -> "Amiya";
                case "login" -> "封锁区";
                case "odyssey" -> "残像区域";
                case "mek" -> "科研中心";
                default -> "";
            };
        };
    }
    public static int hudAlpha(long ageMillis) {
        if (ageMillis <= 8000) return 255;
        return (int)Math.clamp((10000-ageMillis)*255/2000,0,255);
    }
}
