package com.deuterium.app.uilab

import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test

class CheckoutSummaryTest {
    private fun grouped():JSONObject {
        val first=JSONObject("""{"storeId":"a","storeName":"甲店","channel":"OFFICIAL_STORE","totalAmount":"7.00","originalTotal":"10.00","productDiscount":"0.00","couponDiscount":"3.00","discountTotal":"3.00","coupon":{"couponId":"coupon-a","name":"甲店优惠"},"items":[{"productId":"a-product","title":"甲商品","unitPrice":"10.00","quantity":1,"subtotal":"10.00"}]}""")
        val second=JSONObject(first.toString()).put("storeId","b").put("storeName","乙店");second.getJSONArray("items").getJSONObject(0).put("productId","b-product");second.getJSONObject("coupon").put("couponId","coupon-b").put("name","乙店优惠")
        return JSONObject().put("channel","OFFICIAL_STORE").put("totalAmount","14.00").put("originalTotal","20.00").put("productDiscount","0.00").put("couponDiscount","6.00").put("discountTotal","6.00").put("items",org.json.JSONArray().put(first.getJSONArray("items").getJSONObject(0)).put(second.getJSONArray("items").getJSONObject(0))).put("storeOrders",org.json.JSONArray().put(first).put(second))
    }
    @Test fun groupedQuoteRetainsEachStoreAndExactCombinedAmount() {
        val summary=checkoutSummary(grouped())
        assertEquals(1400L,summary.total);assertEquals(listOf("甲店","乙店"),summary.stores.map{it.name});assertEquals("甲店优惠、乙店优惠",summary.couponName)
    }
    @Test fun groupedQuoteRejectsRepeatedCouponAndWrongItemMembership() {
        val quote=grouped();quote.getJSONArray("storeOrders").getJSONObject(1).getJSONObject("coupon").put("couponId","coupon-a")
        assertThrows(IllegalArgumentException::class.java){checkoutSummary(quote)}
        val wrong=grouped();val second=wrong.getJSONArray("storeOrders").getJSONObject(1);val lines=org.json.JSONArray(second.getJSONArray("items").toString());lines.getJSONObject(0).put("productId","unknown");second.put("items",lines)
        assertThrows(IllegalArgumentException::class.java){checkoutSummary(wrong)}
    }
    @Test fun groupedQuoteRejectsWrongSplitTotals() {
        val quote=grouped();quote.getJSONArray("storeOrders").getJSONObject(0).put("totalAmount","6.00").put("couponDiscount","4.00").put("discountTotal","4.00")
        assertThrows(IllegalArgumentException::class.java){checkoutSummary(quote)}
    }
    @Test fun operationRecoveryIncludesAllDistinctOrdersAndKeepsSingleCompatibility() {
        assertEquals(listOf("a","b"),operationOrderIds(JSONObject("""{"resourceId":"a","resourceIds":["a","b"]}""")))
        assertEquals(listOf("a"),operationOrderIds(JSONObject("""{"resourceId":"a"}""")))
        assertThrows(IllegalArgumentException::class.java){operationOrderIds(JSONObject("""{"resourceId":"a","resourceIds":["b","b"]}"""))}
    }
    private fun quote()=JSONObject("""{"totalAmount":"24.60","items":[{"title":"玩家新商品","quantity":2,"unitPrice":"12.30","subtotal":"24.60"}],"warnings":["报价不预留库存"]}""")
    @Test fun confirmationUsesServerItemsAndExactDecimalAmounts() {
        val summary=checkoutSummary(quote())
        assertEquals("玩家新商品",summary.items.single().title)
        assertEquals(2,summary.items.single().quantity)
        assertEquals(1230L,summary.items.single().unitPrice)
        assertEquals(2460L,summary.total)
    }
    @Test fun mismatchedTotalNeverReachesPaymentConfirmation() {
        assertThrows(IllegalArgumentException::class.java){checkoutSummary(quote().put("totalAmount","1.00"))}
    }
    @Test fun corruptQuantityAndMissingProductsAreRejected() {
        val quote=quote();quote.getJSONArray("items").getJSONObject(0).put("quantity",0)
        assertThrows(IllegalArgumentException::class.java){checkoutSummary(quote)}
        assertThrows(IllegalArgumentException::class.java){checkoutSummary(JSONObject("""{"totalAmount":"0.00","items":[]}"""))}
    }
    @Test fun recommendationsComeOnlyFromCurrentProducts() {
        val product=ShopProduct("new","石材礼包","建材","",1230,androidx.compose.ui.graphics.Color.White)
        assertEquals(listOf("石材礼包","建材"),shopSearchSuggestions(listOf(product,product)))
        assertTrue(shopSearchSuggestions(emptyList()).isEmpty())
    }
}
