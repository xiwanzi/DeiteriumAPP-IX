package com.deuterium.app.uilab

import org.junit.Assert.*
import org.junit.Test

class LiveBusinessBoundaryTest {
    @Test fun liveCommerceCannotMintOrdersOrCashLocally() {
        var cashChanges = 0
        val book = CommerceBook("alice", emptyList(), { 1000000 }, { _, _, _ -> cashChanges++ }, remoteOnly = true)
        val order = book.buyOfficial("same-request", listOf(OrderLine("product", "product", "", 100, 1)))
        assertNull(order); assertTrue(book.orders.isEmpty()); assertEquals(0, cashChanges)
        assertFalse(book.simulatePlatformDecision("unknown", PlatformDecision.FullRefund, 100, "cannot be executed by a player"))
    }
    @Test fun liveCommissionCannotCreateFakeEscrowAndHasNoSeedEntries() {
        var cashChanges = 0
        val book = CommissionBook("alice", { 1000000 }, { _, _, _ -> cashChanges++ }, remoteOnly = true)
        book.reset()
        assertTrue(book.entries.isEmpty())
        val id = book.publish("stable-request", CommissionDraft("valid title", "valid description", "valid location", 100, 24, Urgency.Normal, null, "garden"))
        assertNull(id); assertEquals(0, cashChanges); assertTrue(book.entries.isEmpty())
    }
    @Test fun currencyParsingRejectsRoundingAndOverflow() {
        assertEquals(12345, apiCents("123.45"))
        assertThrows(ArithmeticException::class.java) { apiCents("1.001") }
        assertThrows(ArithmeticException::class.java) { apiCents("92233720368547758.08") }
    }
    @Test fun unpaidAndInProgressFundsAreNotRefundedOrFinished(){
        val draft=CommissionDraft("title","description","location",100,24,Urgency.Normal,null,"garden")
        val unpaid=Commission("id","key","alice",draft,java.time.LocalDateTime.now(),stage=CommissionStage.Cancelled,serverStatus="CANCELLED",fundsStatus="UNPAID")
        assertFalse(unpaid.status.contains("退款"));assertTrue(unpaid.status.contains("未扣款"))
        val pending=unpaid.copy(stage=CommissionStage.Completed,serverStatus="COMPLETED",fundsStatus="SETTLING")
        assertTrue(pending.held);assertEquals("结算处理中",pending.status)
    }
}
