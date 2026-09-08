package com.deuterium.app.uilab

import org.junit.Assert.*
import org.junit.Test
import java.time.*

class CommerceCompletionTest {
    private var cash=100000L
    private val listing=MarketListing("service","代建小屋","主体与内装","按确认方案施工","建筑服务",10000,2,"Worker","1000101",setOf(DeliveryMethod.Worksite),"",imageUri="file:///cover.png",workHours=120)
    private fun book()=CommerceBook("Owner",listOf(listing),{cash},{delta,_,_->cash+=delta},Clock.fixed(Instant.parse("2026-09-08T00:00:00Z"),ZoneOffset.UTC))
    private fun buy(book:CommerceBook)=book.buyMarket("one","service",1,DeliveryMethod.Worksite,"主世界",projectName="湖畔小屋")!!
    @Test fun completionIsSeparateFromAcceptanceAndKeepsDeadline(){val book=book();val id=buy(book);book.ship(id,"Worker");val deadline=book.order(id)!!.deadlineMillis;assertFalse(book.confirmReceipt(id,"Owner"));assertFalse(book.completeWork(id,"Owner"));assertTrue(book.completeWork(id,"Worker"));assertEquals(deadline,book.order(id)!!.deadlineMillis);assertEquals(OrderStage.Completed,book.order(id)!!.stage);assertTrue(book.sellerPayments.isEmpty());assertTrue(book.confirmReceipt(id,"Owner"));assertEquals(10000L,book.sellerPayments["Worker"])}
    @Test fun completedWorkStillAllowsOnlyOneRefund(){val book=book();val id=buy(book);book.ship(id,"Worker");book.completeWork(id,"Worker");assertTrue(book.requestRefund(id,"Owner","工程内容不符"));assertFalse(book.resolveRefund(id,"Worker",false,""));assertEquals(RefundState.Requested,book.order(id)!!.refund);assertTrue(book.resolveRefund(id,"Worker",false,"已按双方确认方案完工"));assertEquals("已按双方确认方案完工",book.order(id)!!.rejectionReason);assertFalse(book.requestRefund(id,"Owner","再次申请"));assertTrue(book.confirmReceipt(id,"Owner"))}
    @Test fun pendingRefundPreventsCompletion(){val book=book();val id=buy(book);book.ship(id,"Worker");book.requestRefund(id,"Owner","需要协商");assertFalse(book.completeWork(id,"Worker"));assertNull(book.order(id)!!.completedAt);assertTrue(book.resolveRefund(id,"Worker",true));assertFalse(book.completeWork(id,"Worker"))}
}
