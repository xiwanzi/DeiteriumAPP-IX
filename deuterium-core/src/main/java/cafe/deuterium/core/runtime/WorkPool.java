package cafe.deuterium.core.runtime;

import cafe.deuterium.core.util.CoreFailure;
import java.util.concurrent.*;
import java.util.concurrent.atomic.AtomicInteger;

public final class WorkPool implements AutoCloseable {
    private final ThreadPoolExecutor executor;
    private final java.util.Set<CompletableFuture<?>> pending = ConcurrentHashMap.newKeySet();
    public WorkPool(String name, int threads, int queue) {
        AtomicInteger index = new AtomicInteger();
        executor = new ThreadPoolExecutor(threads, threads, 30, TimeUnit.SECONDS, new ArrayBlockingQueue<>(queue), task -> {
            Thread thread = new Thread(task, name + "-" + index.incrementAndGet()); thread.setDaemon(true); return thread;
        }, new ThreadPoolExecutor.AbortPolicy());
    }
    public <T> CompletableFuture<T> submit(Callable<T> action) {
        CompletableFuture<T> future = new CompletableFuture<>();
        pending.add(future); future.whenComplete((result,error) -> pending.remove(future));
        try { executor.execute(() -> { if (!future.isCancelled()) try { future.complete(action.call()); } catch (Throwable error) { future.completeExceptionally(error); } }); }
        catch (RejectedExecutionException e) { future.completeExceptionally(new CoreFailure("CORE_BUSY", "Core 繁忙，请稍后重试。")); }
        return future;
    }
    public Executor executor() { return executor; }
    public int queued() { return executor.getQueue().size(); }
    @Override public void close() {
        executor.shutdownNow();
        for (CompletableFuture<?> future : pending) future.completeExceptionally(new CoreFailure("CORE_STOPPED", "Core 已停用，未确认的操作请按原标识查询。"));
    }
}
