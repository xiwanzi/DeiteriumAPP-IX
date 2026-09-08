import de.mkammerer.argon2.Argon2Factory;

// Generates a synthetic migration fixture using the old backend's actual library
// and parameters. Never point this program at production account data.
class LegacyArgon2Fixture {
    public static void main(String[] args) {
        System.out.println(Argon2Factory.create().hash(3, 65536, 2,
                "migration-password-123".toCharArray()));
    }
}
