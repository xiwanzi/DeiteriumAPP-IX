package com.deuterium.app.uilab

import org.json.JSONObject

data class CheckoutItem(val title: String, val unitPrice: Long, val quantity: Int, val subtotal: Long)
data class CheckoutSummary(val items: List<CheckoutItem>, val total: Long)

/** Display exactly the immutable server snapshot that the confirmed payment will consume. */
fun checkoutSummary(quote: JSONObject): CheckoutSummary {
    val values = quote.getJSONArray("items")
    require(values.length() in 1..100) { "订单商品信息不完整，请重新确认" }
    val items = (0 until values.length()).map { index ->
        val item = values.getJSONObject(index)
        val price = apiCents(item.getString("unitPrice"))
        val count = item.getInt("quantity")
        val subtotal = apiCents(item.getString("subtotal"))
        require(price > 0 && count in 1..999 && Math.multiplyExact(price, count.toLong()) == subtotal) { "订单金额已变化，请重新确认" }
        CheckoutItem(item.getString("title"), price, count, subtotal)
    }
    val total = apiCents(quote.getString("totalAmount"))
    require(items.fold(0L) { sum, item -> Math.addExact(sum, item.subtotal) } == total) { "订单金额已变化，请重新确认" }
    return CheckoutSummary(items, total)
}

fun shopSearchSuggestions(products: List<ShopProduct>): List<String> =
    (products.map { it.name } + products.map { it.category }).filter { it.isNotBlank() }.distinct().take(6)
