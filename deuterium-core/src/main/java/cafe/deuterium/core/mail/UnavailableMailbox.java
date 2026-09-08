package cafe.deuterium.core.mail;

import cafe.deuterium.core.util.CoreFailure;
import cafe.deuterium.core.util.Json;
import com.google.gson.JsonObject;
import java.util.Map;

public final class UnavailableMailbox implements CoreMailbox {
    private final String reason;
    public UnavailableMailbox(String reason) { this.reason = reason; }
    @Override public JsonObject capabilities() { return Json.tree(Map.of("available", false, "commerceReady", false, "reason", reason)); }
    @Override public JsonObject execute(String id, String type, JsonObject payload) { throw new CoreFailure("MAILBOX_UNAVAILABLE", reason); }
    @Override public void pumpEvents() { }
    @Override public void close() { }
}
