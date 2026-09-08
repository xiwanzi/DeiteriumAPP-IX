package cafe.deuterium.core.util;

import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.util.HexFormat;
import java.util.Locale;
import java.util.UUID;
import java.util.regex.Pattern;

public final class Checks {
    private static final Pattern ITEM = Pattern.compile("[a-z][a-z0-9_-]{0,31}:[a-z0-9_./-]{1,63}");
    private static final Pattern NODE = Pattern.compile("[a-z][a-z0-9_-]{0,31}");
    private Checks() { }
    public static String itemRef(String value, String namespace) {
        if (value == null) throw CoreFailure.invalid("缺少物品 ID。");
        String ref = value.toLowerCase(Locale.ROOT);
        if (!ref.contains(":")) ref = namespace + ":" + ref;
        if (ref.length() > 96 || !ITEM.matcher(ref).matches() || ref.contains("..") || ref.endsWith("/"))
            throw CoreFailure.invalid("物品 ID 只能包含小写字母、数字、下划线、短横线和目录分隔符。");
        return ref;
    }
    public static String node(String value) {
        if (value == null || !NODE.matcher(value).matches()) throw CoreFailure.invalid("无效的节点或命名空间标识。");
        return value;
    }
    public static long number(String value, long min, long max) {
        try { long n = Long.parseLong(value); if (n >= min && n <= max) return n; }
        catch (NumberFormatException ignored) { }
        throw CoreFailure.invalid("数量或版本不在允许范围内。");
    }
    public static String text(String value, int maxBytes, boolean blankAllowed) {
        if (value == null || (!blankAllowed && value.isBlank()) || value.getBytes(StandardCharsets.UTF_8).length > maxBytes)
            throw CoreFailure.invalid("文本为空或超过长度限制。");
        for (int i = 0; i < value.length(); i++) {
            char c = value.charAt(i);
            if (Character.isHighSurrogate(c)) {
                if (++i >= value.length() || !Character.isLowSurrogate(value.charAt(i))) throw CoreFailure.invalid("无效文本编码。");
            } else if (Character.isLowSurrogate(c) || (Character.isISOControl(c) && c != '\n'))
                throw CoreFailure.invalid("文本包含不允许的控制字符。");
        }
        return value;
    }
    public static String operationId(String value) {
        if (value == null || !value.matches("[a-zA-Z0-9_:.-]{1,128}")) throw CoreFailure.invalid("无效请求标识。");
        return value;
    }
    public static String sha(byte[] value) {
        try { return HexFormat.of().formatHex(MessageDigest.getInstance("SHA-256").digest(value)); }
        catch (Exception impossible) { throw new IllegalStateException(impossible); }
    }
    public static String id(String prefix) { return prefix + UUID.randomUUID().toString().replace("-", ""); }
}
