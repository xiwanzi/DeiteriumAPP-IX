package com.deuterium.app.uilab

import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test

class LedgerRecordsTest {
    private fun record(id:String,at:String,status:String="success",source:String="APP")=JSONObject()
        .put("recordId",id).put("occurredAt",at).put("status",status).put("source",source)
        .put("direction","expense").put("amount","12.30").put("title",if(source=="GAME")"游戏内支出" else "玩家转账")
        .put("note",JSONObject.NULL).put("otherPlayer",JSONObject().put("gameId","Player"))
    @Test fun authoritativeEntriesAreNewestFirstAndUseServerBusinessDay() {
        val rows=ledgerEntries(listOf(record("econ_1","2026-09-08T15:59:59Z"),record("econ_3","2026-09-08T16:00:00Z",source="GAME"),record("econ_2","2026-09-08T16:00:00Z","SUCCESS"),record("econ_4","2026-09-09T18:00:00Z","unknown")))
        assertEquals(listOf(3L,2L,1L),rows.map{it.id})
        assertEquals("游戏内支出",rows[0].name);assertEquals(-1230L,rows[0].amount)
        assertEquals("2026-09-09",rows[0].at.toLocalDate().toString());assertFalse(rows.any{it.detail=="null"})
    }
    @Test fun legacyHashCollisionsDoNotRemoveDifferentTransfers() {
        val first="transfer_Aa";val second="transfer_BB";assertEquals(first.hashCode(),second.hashCode())
        assertEquals(2,ledgerEntries(listOf(record(first,"2026-09-09T01:00:00Z"),record(second,"2026-09-09T01:00:00Z"))).size)
    }
}
