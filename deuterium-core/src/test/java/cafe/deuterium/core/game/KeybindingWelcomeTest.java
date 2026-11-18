package cafe.deuterium.core.game;

import cafe.deuterium.core.config.CoreConfig;
import cafe.deuterium.core.runtime.GameThread;
import cafe.deuterium.core.runtime.Sessions;
import cafe.deuterium.core.storage.PlayerStore;
import com.google.gson.JsonObject;
import com.google.gson.JsonParser;
import org.bukkit.Bukkit;
import org.bukkit.entity.Player;
import org.bukkit.plugin.PluginManager;
import org.bukkit.plugin.java.JavaPlugin;
import org.bukkit.plugin.messaging.Messenger;
import org.bukkit.scheduler.BukkitScheduler;
import org.bukkit.scheduler.BukkitTask;
import org.junit.jupiter.api.Test;

import java.net.URI;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.List;
import java.util.UUID;
import java.util.logging.Logger;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

class KeybindingWelcomeTest {
    @Test void welcomeSurvivesDelayedMainThreadAndAdvertisesTheLatestState() {
        var plugin = mock(JavaPlugin.class);
        var scheduler = mock(BukkitScheduler.class);
        var player = mock(Player.class);
        var config = mock(CoreConfig.class);
        when(plugin.getLogger()).thenReturn(Logger.getAnonymousLogger());
        when(config.nodeId()).thenReturn("amiya");
        when(config.bridge()).thenReturn(new CoreConfig.Bridge(false, URI.create("http://localhost"), "", true));
        when(player.getUniqueId()).thenReturn(UUID.randomUUID());
        when(player.isOnline()).thenReturn(true);
        var messages = new ArrayList<JsonObject>();
        doAnswer(call -> {
            byte[] bytes = call.getArgument(2);
            messages.add(JsonParser.parseString(new String(bytes, StandardCharsets.UTF_8)).getAsJsonObject());
            return null;
        }).when(player).sendPluginMessage(eq(plugin), eq(GameChatChannel.CHANNEL), any(byte[].class));

        var pendingTicks = new ArrayList<Runnable>();
        var task = mock(BukkitTask.class);
        when(scheduler.runTaskTimer(eq(plugin), any(Runnable.class), anyLong(), anyLong())).thenReturn(task);
        when(scheduler.runTask(eq(plugin), any(Runnable.class))).thenAnswer(call -> {
            pendingTicks.add(call.getArgument(1, Runnable.class));
            return task;
        });

        try (var bukkit = mockStatic(Bukkit.class)) {
            var messenger = mock(Messenger.class);
            var manager = mock(PluginManager.class);
            bukkit.when(Bukkit::getMessenger).thenReturn(messenger);
            bukkit.when(Bukkit::getPluginManager).thenReturn(manager);
            bukkit.when(Bukkit::getScheduler).thenReturn(scheduler);
            bukkit.when(Bukkit::getOnlinePlayers).thenAnswer(call -> List.of(player));
            try (var channel = new GameChatChannel(plugin, () -> config, mock(GameThread.class),
                    mock(ChatAccess.class), mock(Sessions.class), mock(PlayerStore.class))) {
                channel.keybindingsAvailable(true);
                channel.onPluginMessageReceived(GameChatChannel.CHANNEL, player,
                        "{\"type\":\"hello\"}".getBytes(StandardCharsets.UTF_8));
                assertTrue(messages.getFirst().get("keybindings").getAsBoolean(),
                        "the first client hello uses the welcome even if the main thread has not ticked");

                channel.keybindingsAvailable(false);
                assertEquals(2, pendingTicks.size());
                pendingTicks.forEach(Runnable::run);
                var notices = messages.stream()
                        .filter(m -> m.get("type").getAsString().equals("keybindings-ready")).toList();
                assertEquals(1, notices.size());
                assertFalse(notices.getFirst().get("enabled").getAsBoolean());
            }
        }
    }
}
