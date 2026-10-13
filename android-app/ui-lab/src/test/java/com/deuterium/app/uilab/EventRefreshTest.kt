package com.deuterium.app.uilab

import kotlinx.coroutines.*
import org.junit.Assert.*
import org.junit.Test

class EventRefreshTest {
    @Test fun coalescesChangesWithoutBlockingAnotherResource()=runBlocking {
        val scope=CoroutineScope(SupervisorJob()+Dispatchers.Unconfined)
        val refresh=EventRefresh(scope){false}
        val blocked=CompletableDeferred<Unit>();var calls=0;var other=0
        val action:suspend ()->Unit={calls++;if(calls==1)blocked.await()}
        try{
            refresh.request("one",action)
            repeat(100){refresh.request("one",action)}
            refresh.request("two"){other++}
            assertEquals(1,calls);assertEquals(1,other)
            blocked.complete(Unit);yield()
            assertEquals(2,calls)
            refresh.request("one",action);assertEquals(3,calls)
        }finally{refresh.close();scope.cancel()}
    }
    @Test fun stoppedOwnerCannotRestartRefresh(){
        val scope=CoroutineScope(SupervisorJob()+Dispatchers.Unconfined);var stopped=false;var calls=0
        val refresh=EventRefresh(scope){stopped}
        try{refresh.request("one"){calls++};stopped=true;refresh.close();refresh.request("one"){calls++};assertEquals(1,calls)}finally{scope.cancel()}
    }
}
