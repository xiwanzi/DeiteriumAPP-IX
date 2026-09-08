package cafe.deuterium.core.mail;

import com.google.gson.JsonObject;

public interface CoreMailbox extends AutoCloseable {
    JsonObject capabilities();
    JsonObject execute(String operationId, String type, JsonObject payload);
    /** Read a committed strict cancellation after the Core journal lost its reply. */
    default JsonObject queryCancellation(String operationId) { return null; }
    void pumpEvents();
    @Override void close();
}
