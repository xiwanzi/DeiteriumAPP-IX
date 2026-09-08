package cafe.deuterium.core.config;

import cafe.deuterium.core.StoreFixture;
import cafe.deuterium.core.util.CoreFailure;
import cafe.deuterium.core.util.Json;
import org.junit.jupiter.api.Test;
import java.util.List;
import static org.junit.jupiter.api.Assertions.*;

class CoreConfigTest {
    @Test void rejectsUnsafeTransportAndScopeMisconfiguration() {
        var transport = StoreFixture.yaml(); transport.set("bridge.url", "ws://example.com/bridge/v1/connect");
        assertThrows(CoreFailure.class, () -> CoreConfig.load(transport));
        var invalid = StoreFixture.yaml(); invalid.set("bridge.enabled",true); invalid.set("bridge.token","short");
        assertThrows(CoreFailure.class, () -> CoreConfig.load(invalid));
        var defaults = StoreFixture.config();
        assertThrows(CoreFailure.class, () -> CoreConfig.servers(List.of("login"), defaults.nodes(), "survival", "ix-main-1_21_1", true));
        assertThrows(CoreFailure.class, () -> CoreConfig.servers(List.of("unknown"), defaults.nodes(), "survival", "ix-main-1_21_1", false));
    }
    @Test void settingsToStringDoesNotRevealCredentials() {
        var c = StoreFixture.yaml(); c.set("bridge.token","test-secret-that-must-never-be-printed"); c.set("storage.mysql.password","test-database-secret");
        String printed = CoreConfig.load(c).toString();
        assertFalse(printed.contains("test-secret")); assertFalse(printed.contains("test-database-secret"));
    }
    @Test void strictJsonRejectsDuplicatePropertiesAndUnboundedNesting() {
        assertThrows(CoreFailure.class, () -> Json.object("{\"x\":1,\"x\":2}",100));
        assertThrows(CoreFailure.class, () -> Json.object("{} {}",100));
        assertEquals("{\"a\":1,\"z\":2}", Json.canonical(Json.object("{\"z\":2,\"a\":1}",100)));
    }
}
