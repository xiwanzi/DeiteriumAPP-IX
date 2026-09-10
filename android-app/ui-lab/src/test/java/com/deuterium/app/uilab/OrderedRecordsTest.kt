package com.deuterium.app.uilab

import org.junit.Assert.*
import org.junit.Test
import kotlin.random.Random

class OrderedRecordsTest {
    private data class Record(val id:Int,val time:Long,val content:String="")

    @Test fun messageInsertionMatchesEveryPrefixOfStableSortIncludingEqualTimestamps() {
        val random=Random(210)
        val order=compareBy<Record>{it.time}
        val legacy=mutableListOf<Record>();val optimized=mutableListOf<Record>()
        repeat(600){id->
            val value=Record(id,when(id){0->Long.MIN_VALUE;1->Long.MAX_VALUE;else->random.nextLong(-10,10)})
            legacy.add(value);legacy.sortWith(order);optimized.insertInOrder(value,order)
            assertEquals("After arrival $id",legacy,optimized)
        }
    }

    @Test fun identityUpdatesMatchTheOldInsertReplaceAndSortAfterEveryOperation() {
        val random=Random(2026)
        val order=compareByDescending<Record>{it.time}.thenByDescending{it.id}
        val legacy=mutableListOf<Record>();val optimized=mutableListOf<Record>()
        repeat(1000){version->
            val value=Record(random.nextInt(80),random.nextLong(10),"revision $version")
            val before=legacy.indexOfFirst{it.id==value.id}
            if(before>=0)legacy[before]=value else legacy.add(0,value)
            legacy.sortWith(order)
            optimized.replaceInOrder(optimized.indexOfFirst{it.id==value.id},value,order)
            assertEquals(legacy,optimized)
        }
    }

    @Test fun unchangedSortKeysUpdateOnlyTheirExistingPosition() {
        val order=compareByDescending<Record>{it.time}.thenByDescending{it.id}
        val rows=mutableListOf(Record(3,10),Record(2,10),Record(1,9))
        rows.replaceInOrder(1,Record(2,10,"new status"),order)
        assertEquals(listOf(3,2,1),rows.map{it.id});assertEquals("new status",rows[1].content)
    }

    @Test fun insertingHistoryDoesNotCompareTheWholeListForEveryRecord() {
        var comparisons=0
        val order=Comparator<Record>{a,b->comparisons++;a.time.compareTo(b.time)}
        val rows=mutableListOf<Record>()
        repeat(4096){rows.insertInOrder(Record(it,4096L-it),order)}
        assertTrue("Comparison count was $comparisons",comparisons<4096*13)
        assertEquals((1L..4096L).toList(),rows.map{it.time})
    }
}
