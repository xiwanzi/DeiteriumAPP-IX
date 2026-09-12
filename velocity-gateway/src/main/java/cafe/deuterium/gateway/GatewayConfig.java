package cafe.deuterium.gateway;

import java.net.URI;
import java.util.List;

final class GatewayConfig {
    String apiBaseUrl = "https://47.103.99.34";
    String apiToken = "";
    String applicationUrl = "https://47.103.99.34/admission/";
    String defaultServer = "amiya";
    List<String> fallbackServers = List.of("amiya", "login");
    int requestTimeoutSeconds = 5;

    void validate() {
        URI api = URI.create(apiBaseUrl);
        URI application = URI.create(applicationUrl);
        if (!"https".equals(api.getScheme()) || api.getHost() == null || api.getUserInfo() != null
                || api.getRawQuery() != null || api.getFragment() != null || !(api.getPath().isEmpty() || api.getPath().equals("/"))) {
            throw new IllegalArgumentException("apiBaseUrl must be an HTTPS origin");
        }
        if (!"https".equals(application.getScheme()) || application.getHost() == null || application.getUserInfo() != null) {
            throw new IllegalArgumentException("applicationUrl must be HTTPS");
        }
        if (apiToken == null || !apiToken.matches("[A-Za-z0-9_-]{43,128}")) throw new IllegalArgumentException("configure a dedicated gateway credential");
        if (!serverName(defaultServer) || fallbackServers == null || fallbackServers.isEmpty() || fallbackServers.size() > 8
                || fallbackServers.stream().anyMatch(name -> !serverName(name)) || fallbackServers.stream().distinct().count() != fallbackServers.size()) {
            throw new IllegalArgumentException("invalid fallback servers");
        }
        if (requestTimeoutSeconds < 2 || requestTimeoutSeconds > 10) throw new IllegalArgumentException("request timeout must be 2-10 seconds");
        apiBaseUrl = apiBaseUrl.replaceAll("/+$", "");
    }
    static boolean serverName(String value) { return value != null && value.matches("[a-zA-Z0-9_-]{1,64}"); }
}
