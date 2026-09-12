package cafe.deuterium.gateway;

import java.util.HashSet;
import java.util.Set;

/** Per-connection state prevents an automatic fallback from replacing a remembered destination. */
final class ReconnectSession {
    final Object connection;
    private String fallbackTarget;
    private boolean fallbackStay;
    final Set<String> attempted=new HashSet<>();
    ReconnectSession(Object connection){this.connection=connection;}
    synchronized void attempting(String name){attempted.add(name);}
    synchronized boolean attempted(String name){return attempted.contains(name);}
    synchronized void fallback(String name){fallbackTarget=name;attempted.add(name);}
    synchronized boolean connected(String name){
        if(name.equals(fallbackTarget)){fallbackTarget=null;fallbackStay=true;return false;}
        fallbackTarget=null;fallbackStay=false;attempted.clear();attempted.add(name);return true;
    }
    synchronized boolean isFallbackStay(){return fallbackStay;}
}
