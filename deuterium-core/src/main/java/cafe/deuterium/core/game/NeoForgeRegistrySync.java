package cafe.deuterium.core.game;

import cafe.deuterium.core.util.CoreFailure;
import org.bukkit.entity.Player;
import java.lang.reflect.Field;
import java.lang.reflect.Method;
import java.util.HashSet;
import java.util.Map;
import java.util.Set;

/**
 * For a live PLAY connection these negotiated CONFIGURATION channels force NeoForge's
 * SyncRegistries task. The client rejects unknown registry entries before acknowledging
 * completion; the server cannot finish configuration before that acknowledgement.
 * Individual mods need not register any custom payloads or use their mod ID as namespace.
 */
final class NeoForgeRegistrySync {
    static final Set<String> CHANNELS = Set.of(
            "neoforge:frozen_registry_sync_start",
            "neoforge:frozen_registry",
            "neoforge:frozen_registry_sync_completed");

    private NeoForgeRegistrySync() { }

    static void verify(Player player) {
        try {
            Access access = Holder.ACCESS;
            Object nativePlayer = access.nativePlayer.invoke(null, player);
            Object listener = access.listener.get(nativePlayer);
            Object connection = access.connection.invoke(listener);
            Object type = access.connectionType.invoke(null, connection);
            Object setup = access.payloadSetup.invoke(null, connection);
            Set<String> negotiated = new HashSet<>();
            if (setup != null) {
                Map<?, ?> channels = (Map<?, ?>) access.configurationChannels.invoke(setup, access.configuration);
                for (Object id : channels.keySet()) negotiated.add(id.toString());
            }
            requireReady(player.isOnline(), (boolean) access.connected.invoke(connection)
                            && access.currentListener.invoke(connection) == listener,
                    ((Enum<?>) access.protocol.invoke(listener)).name(), type == null ? "" : ((Enum<?>) type).name(), negotiated);
        } catch (CoreFailure rejected) {
            throw rejected;
        } catch (ReflectiveOperationException | LinkageError | RuntimeException unavailable) {
            throw new CoreFailure("CLIENT_COMPATIBILITY_UNAVAILABLE",
                    "无法确认客户端注册表同步状态，请重新连接后重试。", unavailable);
        }
    }

    static void requireReady(boolean online, boolean connected, String protocol, String type, Set<String> channels) {
        if (!online || !connected || !"PLAY".equals(protocol) || !"NEOFORGE".equals(type)
                || !channels.containsAll(CHANNELS)) {
            throw new CoreFailure("CLIENT_REGISTRY_SYNC_REQUIRED",
                    "客户端尚未完成模组物品注册表同步，请使用配套客户端重新连接。 ");
        }
    }

    // Cache immutable reflection metadata only, never a player's connection or compatibility result.
    private static final class Holder {
        static final Access ACCESS = resolve();
    }

    private record Access(Method nativePlayer, Field listener, Method connection, Method protocol,
                          Method connected, Method currentListener, Method connectionType, Method payloadSetup,
                          Method configurationChannels, Object configuration) { }

    private static Access resolve() {
        try {
            Class<?> playerApi = Class.forName("com.mohistmc.youer.api.PlayerAPI");
            Class<?> listenerApi = Class.forName("net.neoforged.neoforge.common.extensions.ICommonPacketListener");
            Class<?> connection = Class.forName("net.minecraft.network.Connection");
            Class<?> protocol = Class.forName("net.minecraft.network.ConnectionProtocol");
            Class<?> packetListener = Class.forName("net.minecraft.network.PacketListener");
            Class<?> attributes = Class.forName("net.neoforged.neoforge.network.registration.ChannelAttributes");
            Class<?> setup = Class.forName("net.neoforged.neoforge.network.registration.NetworkPayloadSetup");
            Method nativePlayer = playerApi.getMethod("getNMSPlayer", Player.class);
            return new Access(nativePlayer, nativePlayer.getReturnType().getField("connection"),
                    listenerApi.getMethod("getConnection"), packetListener.getMethod("protocol"),
                    connection.getMethod("isConnected"), connection.getMethod("getPacketListener"),
                    attributes.getMethod("getConnectionType", connection),
                    attributes.getMethod("getPayloadSetup", connection), setup.getMethod("getChannels", protocol),
                    protocol.getField("CONFIGURATION").get(null));
        } catch (ReflectiveOperationException unavailable) {
            throw new IllegalStateException("NeoForge registry synchronization API unavailable", unavailable);
        }
    }
}
