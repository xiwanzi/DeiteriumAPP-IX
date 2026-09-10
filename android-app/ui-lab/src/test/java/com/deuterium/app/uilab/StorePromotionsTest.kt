package com.deuterium.app.uilab

import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test
import java.time.Instant

class StorePromotionsTest {
    private fun quote()=JSONObject("""{"channel":"OFFICIAL_STORE","totalAmount":"14.76","originalTotal":"22.20","productDiscount":"4.44","couponDiscount":"3.00","discountTotal":"7.44","storeName":"EOS Lab旗舰店","coupon":{"name":"开业礼遇"},"items":[{"title":"探索补给包","quantity":2,"unitPrice":"8.88","subtotal":"17.76"}]}""")

    @Test fun discountedQuoteAndOrderUseTheSameExactAmounts() {
        val q=quote();val summary=checkoutSummary(q)
        assertEquals(1476L,summary.total);assertEquals(2220L,summary.originalTotal);assertEquals(444L,summary.productDiscount);assertEquals(300L,summary.couponDiscount);assertEquals("开业礼遇",summary.couponName)
        q.put("amount","14.76");assertEquals(1476L,checkoutAmounts(q,1776,"amount").total)
    }
    @Test fun tamperedDiscountAndOriginalTotalsCannotProduceSuccess() {
        assertThrows(IllegalArgumentException::class.java){checkoutSummary(quote().put("couponDiscount","8.00"))}
        assertThrows(IllegalArgumentException::class.java){checkoutSummary(quote().put("originalTotal","100.00"))}
        assertThrows(IllegalArgumentException::class.java){checkoutSummary(quote().put("discountTotal","1.00"))}
    }
    @Test fun fullCouponDeductionAllowsZeroWithoutInventingACharge() {
        val q=quote().put("totalAmount","0.00").put("couponDiscount","17.76").put("discountTotal","22.20")
        assertEquals(0L,checkoutSummary(q).total)
    }
    @Test fun expiredCouponDisappearsAtTheExactDeadline() {
        val value=JSONObject("""{"couponId":"test","name":"单品礼遇","type":"ITEM","benefit":"PERCENT","amountOff":"0.00","discountRate":8500,"minimumSpend":"0.00","maxDiscount":"20.00","stackWithProductDiscount":true,"storeIds":["shop"],"productIds":[],"scopeDescription":"EOS Lab旗舰店 · 全部商品","startsAt":"2026-09-10T00:00:00Z","endsAt":"2026-09-11T00:00:00Z"}""")
        val coupon=storeCoupon(value)
        assertFalse(coupon.visible(Instant.parse("2026-09-09T23:59:59Z")))
        assertTrue(coupon.visible(Instant.parse("2026-09-10T23:59:59Z")))
        assertFalse(coupon.visible(Instant.parse("2026-09-11T00:00:00Z")))
        assertEquals("8.5 折",coupon.benefitText);assertEquals("EOS Lab旗舰店 · 全部商品",coupon.scope)
    }
    @Test fun playerLimitsShowConfiguredResetTimes() {
        val summaries=productLimitDescriptions(JSONObject("""{"lifetime":2,"daily":1,"weekly":3,"weeklyDay":7,"weeklyTime":"04:30","monthly":9,"monthlyDay":31,"monthlyTime":"08:00"}"""))
        assertEquals(4,summaries.size)
        assertTrue(summaries[1].contains("00:00"));assertTrue(summaries[2].contains("周日 04:30"));assertTrue(summaries[3].contains("31 日 08:00"))
    }
}
