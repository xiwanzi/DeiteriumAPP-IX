package cafe.deuterium.core.runtime;

import cafe.deuterium.core.util.CoreFailure;
import org.bukkit.Bukkit;
import org.bukkit.plugin.java.JavaPlugin;
import java.util.concurrent.*;
import java.util.concurrent.atomic.AtomicBoolean;

public final class GameThread implements AutoCloseable {
    private final JavaPlugin plugin;
    private final AtomicBoolean open = new AtomicBoolean(true);
    public GameThread(JavaPlugin plugin) { this.plugin = plugin; }
    public <T> CompletableFuture<T> call(long deadlineMillis, Callable<T> action) {
        CompletableFuture<T> future = new CompletableFuture<>();
        Runnable task = () -> {
            if (!open.get() || future.isDone() || System.currentTimeMillis() > deadlineMillis) {
                future.completeExceptionally(new CoreFailure("COMMAND_EXPIRED", "请求已过期或 Core 已停用。")); return;
            }
            try { future.complete(action.call()); } catch (Throwable error) { future.completeExceptionally(error); }
        };
        if (!open.get()) task.run();
        else if (Bukkit.isPrimaryThread()) task.run();
        else try { Bukkit.getScheduler().runTask(plugin, task); }
        catch (Exception error) { future.completeExceptionally(error); }
        return future.orTimeout(Math.max(1, deadlineMillis - System.currentTimeMillis()), TimeUnit.MILLISECONDS);
    }
    public void run(Runnable action) { call(System.currentTimeMillis() + 5000, () -> { action.run(); return null; }); }
    @Override public void close() { open.set(false); }
}
