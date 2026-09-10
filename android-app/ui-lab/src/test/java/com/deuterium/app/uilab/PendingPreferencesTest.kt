package com.deuterium.app.uilab

import android.content.SharedPreferences
import kotlinx.coroutines.*
import org.junit.Assert.*
import org.junit.Test
import java.lang.reflect.Proxy
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicInteger

class PendingPreferencesTest {
    /** Models Android's documented commitToMemory-before-disk-completion boundary. */
    private class MemoryPreferences {
        val memory=ConcurrentHashMap<String,String>()
        var commit:()->Boolean={true}
        val prefs:SharedPreferences=Proxy.newProxyInstance(SharedPreferences::class.java.classLoader,arrayOf(SharedPreferences::class.java)){_,method,_->
            check(method.name=="edit");editor()
        } as SharedPreferences
        private fun editor():SharedPreferences.Editor {
            val changes=linkedMapOf<String,String?>()
            fun applyMemory(){changes.forEach{(key,value)->if(value==null)memory.remove(key) else memory[key]=value}}
            return Proxy.newProxyInstance(SharedPreferences.Editor::class.java.classLoader,arrayOf(SharedPreferences.Editor::class.java)){proxy,method,args->
                when(method.name){
                    "putString"->{changes[args[0] as String]=args[1] as String?;proxy}
                    "remove"->{changes[args[0] as String]=null;proxy}
                    "apply"->{applyMemory();null}
                    "commit"->{applyMemory();commit()}
                    else->error(method.name)
                }
            } as SharedPreferences.Editor
        }
    }

    @Test fun submissionAndRecoveryBothWaitForDiskWhileTheCallerCanContinue()=runBlocking {
        val fixture=MemoryPreferences();val writes=PendingPreferences(fixture.prefs)
        val entered=CountDownLatch(1);val release=CountDownLatch(1);val caller=Thread.currentThread().id
        fixture.commit={assertNotEquals(caller,Thread.currentThread().id);entered.countDown();check(release.await(5,TimeUnit.SECONDS));true}
        val sent=AtomicInteger()
        val saving=launch(start=CoroutineStart.UNDISPATCHED){writes.save("request","original-id");sent.incrementAndGet()}
        try {
            assertTrue(entered.await(5,TimeUnit.SECONDS));assertEquals("original-id",fixture.memory["request"])
            val recovery=launch(start=CoroutineStart.UNDISPATCHED){writes.awaitCommitted();sent.incrementAndGet()}
            assertEquals(0,sent.get());assertFalse(saving.isCompleted);assertFalse(recovery.isCompleted)
            release.countDown();saving.join();recovery.join();assertEquals(2,sent.get())
        } finally {release.countDown()}
    }

    @Test fun failedDiskWriteNeverPassesTheBarrierAndRetriesTheExactIdentity()=runBlocking {
        val fixture=MemoryPreferences();val writes=PendingPreferences(fixture.prefs);val values=mutableListOf<String>()
        var available=false
        fixture.commit={values.add(fixture.memory.getValue("request"));available}
        assertEquals("PERSISTENCE_UNAVAILABLE",(runCatching{writes.save("request","original-id:original-quote")}.exceptionOrNull() as ApiFailure).code)
        assertTrue(runCatching{writes.awaitCommitted()}.isFailure)
        available=true;writes.awaitCommitted()
        assertEquals(List(3){"original-id:original-quote"},values)
    }

    @Test fun cancellationDuringCommitRetainsTheSameIdentityForRecovery()=runBlocking {
        val fixture=MemoryPreferences();val writes=PendingPreferences(fixture.prefs)
        val entered=CountDownLatch(1);val release=CountDownLatch(1);val commits=AtomicInteger()
        fixture.commit={if(commits.incrementAndGet()==1){entered.countDown();check(release.await(5,TimeUnit.SECONDS))};true}
        val writer=launch(start=CoroutineStart.UNDISPATCHED){writes.save("owner-a:request","fixed-id")}
        try {
            assertTrue(entered.await(5,TimeUnit.SECONDS));writer.cancel();release.countDown();writer.join()
            writes.awaitCommitted();assertEquals("fixed-id",fixture.memory["owner-a:request"])
            assertTrue(commits.get() in 1..2)
        } finally {release.countDown()}
    }

    @Test fun removingAnAbandonedFailedWriteCannotResurrectItLater()=runBlocking {
        val fixture=MemoryPreferences();val writes=PendingPreferences(fixture.prefs)
        fixture.commit={false};runCatching{writes.save("owner-a:request","fixed-id")}
        writes.save("owner-a:request",null)
        fixture.commit={true};writes.save("owner-b:request","other-id");writes.awaitCommitted()
        assertNull(fixture.memory["owner-a:request"]);assertEquals("other-id",fixture.memory["owner-b:request"])
    }
}
