package com.deuterium.app.uilab

import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.ensureActive
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import java.util.concurrent.atomic.AtomicLong

/** Only overlapping successful reads share a result. Later, failed and cancelled reads retry normally. */
internal class ConcurrentRefresh {
    private val lock=Mutex()
    private val completed=AtomicLong()
    suspend fun run(read:suspend ()->Unit) {
        val observed=completed.get()
        lock.withLock {
            if(completed.get()!=observed)return
            read();currentCoroutineContext().ensureActive();completed.incrementAndGet()
        }
    }
}
