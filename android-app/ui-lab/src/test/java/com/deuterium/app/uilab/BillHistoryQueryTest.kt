package com.deuterium.app.uilab

import okhttp3.HttpUrl.Companion.toHttpUrl
import org.junit.Assert.*
import org.junit.Test
import java.time.LocalDate

class BillHistoryQueryTest {
    private val today=LocalDate.of(2026,9,9)
    private fun query(start:LocalDate=today,end:LocalDate=today,type:String="all",cursor:String?=null)=
        ("https://example.test"+billHistoryPath(start.toEpochDay(),end.toEpochDay(),type,today,cursor)).toHttpUrl()

    @Test fun currentDayUsesServerCutoffInsteadOfPhoneClock() {
        for(type in listOf("all","income","expense")) {
            val url=query(type=type)
            assertEquals("2026-09-08T16:00:00Z",url.queryParameter("from"))
            assertNull(url.queryParameter("to"))
            assertEquals(if(type=="all")null else type,url.queryParameter("direction"))
        }
    }
    @Test fun pastRangeIncludesWholeFinalDayAtBeijingMidnight() {
        val url=query(today.minusDays(7),today.minusDays(1))
        assertEquals("2026-09-01T16:00:00Z",url.queryParameter("from"))
        assertEquals("2026-09-08T16:00:00Z",url.queryParameter("to"))
    }
    @Test fun paginationKeepsOnlySnapshotCursorAndLimit() {
        val url=query(cursor="cursor+/:=")
        assertEquals(setOf("cursor","limit"),url.queryParameterNames)
        assertEquals("cursor+/:=",url.queryParameter("cursor"))
    }
    @Test(expected=IllegalArgumentException::class) fun reversedDatesAreRejected() {query(today,today.minusDays(1))}
    @Test(expected=IllegalArgumentException::class) fun oversizedRangeIsRejected() {query(today.minusDays(366))}
}
