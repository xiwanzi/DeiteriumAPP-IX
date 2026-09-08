package cafe.deuterium.core.util;

public final class CoreFailure extends RuntimeException {
    private final String code;
    public CoreFailure(String code, String message) { super(message); this.code = code; }
    public CoreFailure(String code, String message, Throwable cause) { super(message, cause); this.code = code; }
    public String code() { return code; }
    public static CoreFailure invalid(String message) { return new CoreFailure("INVALID_REQUEST", message); }
}
