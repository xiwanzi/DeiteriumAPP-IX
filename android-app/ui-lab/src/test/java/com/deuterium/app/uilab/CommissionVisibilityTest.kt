package com.deuterium.app.uilab

import java.time.LocalDateTime
import org.junit.Assert.*
import org.junit.Test

class CommissionVisibilityTest {
    private val open=Commission("open","key","Me",CommissionDraft("公开委托","内容说明","主城",100,24,Urgency.Normal,"asset:cover"),LocalDateTime.now(),serverStatus="OPEN",fundsStatus="HELD")
    @Test fun hallOnlyContainsAcceptableFundedOpenCommissions() {
        assertTrue(open.visibleInHall)
        for(status in listOf("FUNDING","ACTIVE","COMPLETED","CONFIRMED","CANCELLED")) assertFalse(status,open.copy(serverStatus=status).visibleInHall)
        assertFalse(open.copy(fundsStatus="REFUNDED").visibleInHall)
        assertFalse(open.copy(fundsStatus="UNKNOWN").visibleInHall)
        assertFalse(open.copy(pendingOperationId="pending").visibleInHall)
        assertFalse(open.copy(refund=RefundState.Requested).visibleInHall)
        assertFalse(open.copy(fundsStatus="INTERVENTION_HOLD").visibleInHall)
    }
    @Test fun cancelledRecordCanStayInHistoryWithoutReturningToHall() {
        val ended=open.copy(stage=CommissionStage.Cancelled,serverStatus="CANCELLED",fundsStatus="REFUNDED",canHideRecord=true)
        assertFalse(ended.visibleInHall)
        assertTrue(ended.canHideRecord)
        assertFalse(ended.held)
    }
}
