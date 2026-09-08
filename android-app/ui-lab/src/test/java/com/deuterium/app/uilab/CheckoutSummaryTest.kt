package com.deuterium.app.uilab

import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test

class CheckoutSummaryTest {
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
