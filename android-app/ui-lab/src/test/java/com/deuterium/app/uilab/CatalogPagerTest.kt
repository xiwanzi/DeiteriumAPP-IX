package com.deuterium.app.uilab

import kotlinx.coroutines.*
import org.json.JSONArray
import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test

class CatalogPagerTest {
    private fun page(ids:List<String>,cursor:String?=null)=JSONObject()
        .put("items",JSONArray(ids.map{JSONObject().put("id",it)}))
        .put("_page",JSONObject().put("nextCursor",cursor ?: JSONObject.NULL).put("hasMore",cursor!=null))
    private fun pager(read:suspend (String)->JSONObject)=CatalogPager(read,{it.getString("id")},{it})

    @Test fun thousandsOfProductsDoNotBlockFirstPageAndFreshOverlappingRefreshDoesNotRestart()=runBlocking {
        var calls=0
        val pager=pager{calls++;page((0 until 40).map{"product-$it"},"page2-of-3084")}
        pager.refresh();assertEquals(40,pager.items.size);assertEquals(1,calls);assertTrue(pager.hasMore)
        pager.refresh();assertEquals(1,calls)
    }

    @Test fun failedNextPageKeepsProductsAndRetriesSameCursorWithoutDuplicates()=runBlocking {
        val paths=mutableListOf<String>();var fail=true
        val pager=pager{path->paths+=path
            if("cursor=" !in path)page(listOf("a","b"),"second")
            else if(fail)throw IllegalStateException("offline") else page(listOf("b","c"))
        }
        pager.refresh();pager.more();assertEquals(listOf("a","b"),pager.items.toList());assertEquals("offline",pager.error)
        fail=false;pager.retry();assertEquals(listOf("a","b","c"),pager.items.toList());assertEquals(paths[1],paths[2]);assertFalse(pager.hasMore)
    }

    @Test fun slowOldSearchCannotReplaceNewSearch()=runBlocking {
        val release=CompletableDeferred<Unit>();val started=CompletableDeferred<Unit>()
        val pager=pager{path->if("q=old" in path){started.complete(Unit);release.await();page(listOf("old"),"old-next")}else page(listOf("new"))}
        val old=launch{pager.refresh("old")};started.await()
        pager.refresh("new");release.complete(Unit);old.join()
        assertEquals(listOf("new"),pager.items.toList());assertFalse(pager.hasMore);assertFalse(pager.loading)
    }

    @Test fun cancelledRequestIsNotAnErrorAndCanBeRetried()=runBlocking {
        val started=CompletableDeferred<Unit>();var cancel=true
        val pager=pager{if(cancel){started.complete(Unit);awaitCancellation()}else page(listOf("ok"))}
        val job=launch{pager.refresh()};started.await()
        job.cancel(CancellationException("The coroutine scope left the composition"));job.join()
        assertNull(pager.error);assertFalse(pager.loading)
        cancel=false;pager.refresh();assertEquals(listOf("ok"),pager.items.toList())
    }

    @Test fun malformedRepeatedCursorDoesNotAppendOrLosePreviousPage()=runBlocking {
        var calls=0
        val pager=pager{calls++;page(listOf("p$calls"),"same")}
        pager.refresh();pager.more()
        assertEquals(listOf("p1"),pager.items.toList());assertNotNull(pager.error)
    }

    @Test fun searchAndCategoryAreEncodedAndQueryChangeDiscardsOldCursor()=runBlocking {
        val paths=mutableListOf<String>()
        val pager=pager{paths+=it;page(listOf("a"),"next")}
        pager.refresh("木 & 灯","category_one");pager.refresh("new","category_two")
        assertTrue(paths[0].contains("q=%E6%9C%A8+%26+%E7%81%AF"));assertTrue(paths[0].contains("categoryId=category_one"))
        assertFalse(paths[1].contains("cursor="));assertTrue(paths[1].contains("categoryId=category_two"))
    }
}
