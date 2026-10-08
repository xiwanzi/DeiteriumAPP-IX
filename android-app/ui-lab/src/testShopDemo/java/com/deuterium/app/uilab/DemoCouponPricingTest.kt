package com.deuterium.app.uilab

import org.junit.Assert.*
import org.junit.Test
import java.time.Instant

class DemoCouponPricingTest {
    private val now=Instant.parse("2026-10-08T12:00:00Z")
    @Test fun selectsLowestPayableRatherThanLargestFaceValue() {
        val best=bestDemoCoupon(10000,8000,"p","s",listOf(DemoCoupon("large","大额不叠加",amountOff=3000,stack=false),DemoCoupon("stack","小额可叠加",amountOff=2000)),now)!!
        assertEquals("stack",best.couponId);assertEquals(6000L,best.finalPrice);assertEquals(2000L,best.saving)
    }
    @Test fun respectsThresholdTimeAndProductScope() {
        val coupons=listOf(DemoCoupon("scope","其他商品",amountOff=9000,productIds=setOf("other")),DemoCoupon("future","未开始",amountOff=9000,starts=now.plusSeconds(1)),DemoCoupon("expired","已过期",amountOff=9000,ends=now),DemoCoupon("threshold","未达门槛",minimum=10000,amountOff=9000))
        assertNull(bestDemoCoupon(12000,9000,"p","s",coupons,now))
    }
    @Test fun percentRoundingCapAndZeroFloor() {
        assertEquals(1L,demoRate(1,5000))
        assertEquals(Long.MAX_VALUE,demoRate(Long.MAX_VALUE,10000))
        val capped=bestDemoCoupon(999,999,"p","s",listOf(DemoCoupon("percent","折扣券",rate=9000,cap=30)),now)!!
        assertEquals(969L,capped.finalPrice)
        assertEquals(0L,bestDemoCoupon(999,999,"p","s",listOf(DemoCoupon("fixed","立减券",amountOff=2000)),now)!!.finalPrice)
    }
    @Test fun equalSavingsPreferEarlierExpiryAndIgnoreWorseCoupons() {
        val best=bestDemoCoupon(10000,7000,"p","s",listOf(DemoCoupon("later","晚到期",amountOff=1000),DemoCoupon("earlier","早到期",amountOff=1000,ends=now.plusSeconds(60)),DemoCoupon("worse","不叠加",amountOff=2000,stack=false)),now)!!
        assertEquals("earlier",best.couponId)
    }
}
