package com.deuterium.app.uilab

import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test

class ShopOfferPreviewTest {
    private fun valid()=JSONObject().put("couponId","coupon").put("couponName","家居券").put("condition","满100减20").put("finalPrice","80.00").put("saving","20.00").put("stack",true)
    @Test fun optionalPreviewDoesNotInventCouponEligibility() {
        assertNull(shopOfferPreview(null,10000))
        val offer=shopOfferPreview(valid(),10000)!!
        assertEquals(8000L,offer.finalPrice);assertEquals(2000L,offer.saving)
    }
    @Test fun inconsistentOrStalePreviewsAreDiscarded() {
        assertNull(shopOfferPreview(valid(),9000))
        assertNull(shopOfferPreview(valid().put("finalPrice","-1.00"),10000))
        assertNull(shopOfferPreview(valid().put("saving","1.00"),10000))
        assertNull(shopOfferPreview(valid().put("couponId",""),10000))
        assertNull(shopOfferPreview(valid().put("finalPrice","bad data"),10000))
    }
}
