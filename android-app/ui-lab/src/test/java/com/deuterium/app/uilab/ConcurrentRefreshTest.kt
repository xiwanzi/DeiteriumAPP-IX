package com.deuterium.app.uilab

import kotlinx.coroutines.*
import org.junit.Assert.*
import org.junit.Test

class ConcurrentRefreshTest {
    @Test fun overlappingReadsShareCompletionButASeparateRefreshIsAlwaysFresh()=runBlocking {
        val gate=ConcurrentRefresh();var reads=0;val release=CompletableDeferred<Unit>()
        val first=launch(start=CoroutineStart.UNDISPATCHED){gate.run{reads++;release.await()}}
        val second=launch(start=CoroutineStart.UNDISPATCHED){gate.run{reads++}}
        assertEquals(1,reads);release.complete(Unit);joinAll(first,second);assertEquals(1,reads)
        gate.run{reads++};assertEquals(2,reads)
    }

    @Test fun aFailedLeaderDoesNotSuppressTheQueuedRetry()=runBlocking {
        val gate=ConcurrentRefresh();var reads=0;val release=CompletableDeferred<Unit>()
        val first=launch(start=CoroutineStart.UNDISPATCHED){runCatching{gate.run{reads++;release.await();error("offline")}}}
        val second=launch(start=CoroutineStart.UNDISPATCHED){gate.run{reads++}}
        release.complete(Unit);joinAll(first,second);assertEquals(2,reads)
    }

    @Test fun cancellationDoesNotTurnAnUnfinishedRequestIntoACompletedRefresh()=runBlocking {
        val gate=ConcurrentRefresh();var reads=0
        val first=launch(start=CoroutineStart.UNDISPATCHED){gate.run{reads++;awaitCancellation()}}
        val second=launch(start=CoroutineStart.UNDISPATCHED){gate.run{reads++}}
        first.cancelAndJoin();second.join();assertEquals(2,reads)
    }
}
