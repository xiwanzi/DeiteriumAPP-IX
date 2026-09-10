package com.deuterium.app.uilab

import org.json.JSONObject

data class CheckoutItem(val title: String, val unitPrice: Long, val quantity: Int, val subtotal: Long)
data class CheckoutSummary(val items: List<CheckoutItem>, val total: Long, val originalTotal:Long=total, val productDiscount:Long=0, val couponDiscount:Long=0, val couponName:String?=null)
data class CheckoutAmounts(val total:Long,val originalTotal:Long,val productDiscount:Long,val couponDiscount:Long)

fun checkoutAmounts(value:JSONObject, subtotal:Long, amountField:String="totalAmount"):CheckoutAmounts {
    fun optional(key:String,default:Long)=if(value.isNull(key))default else apiCents(value.getString(key))
    val total=apiCents(value.getString(amountField))
    val coupon=optional("couponDiscount",0)
    val product=optional("productDiscount",0)
    val original=optional("originalTotal",subtotal)
    require(total>=0&&coupon>=0&&product>=0&&Math.addExact(total,coupon)==subtotal&&Math.addExact(subtotal,product)==original){"订单优惠与金额不一致，请重新确认"}
    require(optional("discountTotal",Math.addExact(coupon,product))==Math.addExact(coupon,product)){"订单优惠与金额不一致，请重新确认"}
    return CheckoutAmounts(total,original,product,coupon)
}

/** Display exactly the immutable server snapshot that the confirmed payment will consume. */
fun checkoutSummary(quote: JSONObject): CheckoutSummary {
    val values = quote.getJSONArray("items")
    require(values.length() in 1..100) { "订单商品信息不完整，请重新确认" }
    val items = (0 until values.length()).map { index ->
        val item = values.getJSONObject(index)
        val price = apiCents(item.getString("unitPrice"))
        val count = item.getInt("quantity")
        val subtotal = apiCents(item.getString("subtotal"))
        require(price >= 0 && (price > 0 || quote.optString("channel")=="OFFICIAL_STORE") && count in 1..999 && Math.multiplyExact(price, count.toLong()) == subtotal) { "订单金额已变化，请重新确认" }
        CheckoutItem(item.getString("title"), price, count, subtotal)
    }
    val amounts=checkoutAmounts(quote,items.fold(0L) { sum, item -> Math.addExact(sum, item.subtotal) })
    return CheckoutSummary(items,amounts.total,amounts.originalTotal,amounts.productDiscount,amounts.couponDiscount,quote.optJSONObject("coupon")?.optString("name"))
}

fun shopSearchSuggestions(products: List<ShopProduct>): List<String> =
    (products.map { it.name } + products.map { it.category }).filter { it.isNotBlank() }.distinct().take(6)
