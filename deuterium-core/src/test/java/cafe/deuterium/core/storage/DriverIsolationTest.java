package cafe.deuterium.core.storage;

import org.junit.jupiter.api.Test;
import org.mariadb.jdbc.plugin.AuthenticationPluginFactory;
import java.util.ServiceLoader;
import static org.junit.jupiter.api.Assertions.*;

class DriverIsolationTest {
    @Test void cachingSha256FactoryBelongsToTheBundledDriverLoader(){
        ClassLoader loader=org.mariadb.jdbc.Driver.class.getClassLoader();
        assertTrue(ServiceLoader.load(AuthenticationPluginFactory.class,loader).stream().map(ServiceLoader.Provider::get).anyMatch(p->p.type().equals("caching_sha2_password")));
        assertEquals(org.mariadb.jdbc.Driver.class.getName(),IsolatedMariaDataSource.driverName());
    }
}
