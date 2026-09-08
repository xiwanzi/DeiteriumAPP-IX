package cafe.deuterium.core.bridge;

import com.google.gson.JsonObject;

/** Implementations run on a bounded Core worker, never on the HTTP selector. */
public interface BridgeEndpoint {
    void appChat(JsonObject payload) throws Exception;
    JsonObject command(JsonObject payload) throws Exception;
    JsonObject status();
    JsonObject presence();
    void afterEventAcknowledged(String eventId);
}
