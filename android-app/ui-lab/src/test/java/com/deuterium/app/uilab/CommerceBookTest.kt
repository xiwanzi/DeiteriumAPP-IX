package com.deuterium.app.uilab

import org.junit.Assert.*
import org.junit.Test

class CommerceBookTest {
    private var cash=100000L
    private fun listing(seller:String="Mori")=MarketListing("stone","石材","两组石材","石砖 × 128","建筑",10000,3,seller,"1000101",setOf(DeliveryMethod.Door,DeliveryMethod.Pickup),"出生点",imageUri="file:///sample.jpg")
    private fun book(user:String="Buyer")=CommerceBook(user,listOf(listing()),{cash},{delta,_,_->cash+=delta})
    private fun buy(book:CommerceBook,key:String="one")=book.buyMarket(key,"stone",1,DeliveryMethod.Pickup,"出生点")!!

    @Test fun invalidCheckoutDoesNotReserveStockOrCash() {
        val book=book()
        assertNull(book.buyMarket("a","stone",1,null,"出生点"))
        assertNull(book.buyMarket("b","stone",1,DeliveryMethod.Door,""))
        assertNull(book.buyMarket("c","stone",4,DeliveryMethod.Pickup,"出生点"))
        assertNull(book.buyMarket("d","stone",0,DeliveryMethod.Pickup,"出生点"))
        assertEquals(100000L,cash);assertEquals(3,book.listings.single().stock);assertEquals(0,book.orders.size)
    }
    @Test fun insufficientBalanceHasNoSideEffects(){val book=book();cash=9000;assertNull(book.buyMarket("a","stone",1,DeliveryMethod.Pickup,"出生点"));assertEquals(9000L,cash);assertEquals(3,book.listings.single().stock)}
    @Test fun cannotBuyOwnListing(){val book=book("Mori");assertNull(book.buyMarket("a","stone",1,DeliveryMethod.Pickup,"出生点"));assertEquals(100000L,cash)}
    @Test fun retriesReturnSameOrderAndReserveOnlyOnce(){val book=book();val id=buy(book);assertEquals(id,buy(book));assertEquals(90000L,cash);assertEquals(10000L,book.heldForUser);assertEquals(2,book.listings.single().stock);assertEquals(1,book.orders.size)}
    @Test fun settlementRequiresShipmentAndBuyerConfirmation(){val book=book();val id=buy(book);assertFalse(book.confirmReceipt(id,"Buyer"));assertFalse(book.ship(id,"Buyer"));assertTrue(book.ship(id,"Mori"));assertFalse(book.confirmReceipt(id,"Mori"));assertTrue(book.confirmReceipt(id,"Buyer"));assertFalse(book.confirmReceipt(id,"Buyer"));assertEquals(10000L,book.sellerPayments["Mori"]);assertEquals(0L,book.heldTotal);assertEquals(90000L,cash);assertFalse(book.requestRefund(id,"Buyer","不需要"))}
    @Test fun refundBeforeShipmentReturnsCashAndStockExactlyOnce(){val book=book();val id=buy(book);assertTrue(book.requestRefund(id,"Buyer","不再需要"));assertEquals(100000L,cash);assertEquals(3,book.listings.single().stock);assertEquals(0L,book.heldTotal);assertFalse(book.requestRefund(id,"Buyer","再次退款"));assertFalse(book.ship(id,"Mori"));assertFalse(book.confirmReceipt(id,"Buyer"))}
    @Test fun shippedRefundFreezesUntilSellerApproves(){val book=book();val id=buy(book);book.ship(id,"Mori");assertTrue(book.requestRefund(id,"Buyer","商品不符"));assertEquals(90000L,cash);assertEquals(10000L,book.heldTotal);assertFalse(book.confirmReceipt(id,"Buyer"));assertFalse(book.resolveRefund(id,"Buyer",true));assertTrue(book.resolveRefund(id,"Mori",true));assertEquals(100000L,cash);assertEquals(0L,book.heldTotal);assertFalse(book.resolveRefund(id,"Mori",true));assertEquals(3,book.listings.single().stock)}
    @Test fun rejectedRefundCanContinueToReceipt(){val book=book();val id=buy(book);book.ship(id,"Mori");book.requestRefund(id,"Buyer","不需要");assertTrue(book.resolveRefund(id,"Mori",false,"已按约定交付，请核对物品"));assertEquals(90000L,cash);assertTrue(book.confirmReceipt(id,"Buyer"));assertEquals(10000L,book.sellerPayments["Mori"])}
    @Test fun cancellingRefundDoesNotReleaseFunds(){val book=book();val id=buy(book);book.ship(id,"Mori");book.requestRefund(id,"Buyer","不需要");assertTrue(book.cancelRefund(id,"Buyer"));assertEquals(10000L,book.heldTotal);assertEquals(90000L,cash);assertTrue(book.confirmReceipt(id,"Buyer"))}
    @Test fun officialClaimPreventsLaterRefund(){val book=book();val id=book.buyOfficial("o",listOf(OrderLine("gift","礼盒","标准款",1000,2)))!!;assertEquals(98000L,cash);assertTrue(book.claim(id,"Buyer"));assertFalse(book.claim(id,"Buyer"));assertFalse(book.requestRefund(id,"Buyer","不需要"));assertEquals(98000L,cash)}
    @Test fun officialRefundRevokesClaim(){val book=book();val id=book.buyOfficial("o",listOf(OrderLine("gift","礼盒","标准款",1000,2)))!!;assertTrue(book.requestRefund(id,"Buyer","不需要"));assertFalse(book.claim(id,"Buyer"));assertFalse(book.requestRefund(id,"Buyer","再次退款"));assertEquals(100000L,cash)}
    @Test fun sellerGetsPaidOnlyAfterExternalBuyerReceives(){val book=book("Mori");val id=book.buyMarket("x","stone",1,DeliveryMethod.Pickup,"出生点","Luna")!!;assertEquals(100000L,cash);assertEquals(0L,book.heldForUser);book.ship(id,"Mori");book.confirmReceipt(id,"Luna");assertEquals(110000L,cash);assertFalse(book.confirmReceipt(id,"Luna"));assertEquals(110000L,cash)}
    @Test fun historicalItemIsASnapshot(){val book=book();val id=buy(book);book.listings[0]=book.listings[0].copy(title="改过的商品",price=20000);assertEquals("石材",book.order(id)!!.lines.single().title);assertEquals(10000L,book.order(id)!!.amount)}
    @Test fun publishingRequiresPhotoAndFulfilment(){val book=book();val product=listing("Buyer").copy(id="new");assertFalse(book.publish(product.copy(imageUri=null)));assertFalse(book.publish(product.copy(methods=emptySet())));assertTrue(book.publish(product));assertFalse(book.publish(product));assertTrue(book.setListingActive("new",false));assertNull(book.buyMarket("x","new",1,DeliveryMethod.Pickup,"出生点","Luna"))}
    @Test fun invalidOfficialAmountCannotDebit(){val book=book();assertNull(book.buyOfficial("x",listOf(OrderLine("x","物品","",Long.MAX_VALUE,2))));assertNull(book.buyOfficial("y",listOf(OrderLine("x","物品","",-1,1))));assertEquals(100000L,cash)}
}
