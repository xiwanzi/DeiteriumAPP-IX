package com.deuterium.app.uilab

import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test

class AccountErasureTest {
    @Test fun delayedOrderKeepsMoneyAndAnonymizesOnlyErasedParty(){
        val order=JSONObject("""{"amount":"19999.00","seller":{"playerRef":"old","displayName":"OldPlayer","contactQq":"12345","avatar":{"assetId":"avatar-old"}},"buyer":{"playerRef":"live","displayName":"KeepPlayer","contactQq":"67890"}}""")
        redactErasedAccounts(order,setOf("old"))
        assertEquals("19999.00",order.getString("amount"))
        assertEquals(ErasedAccountName,order.getJSONObject("seller").getString("displayName"))
        assertEquals("",order.getJSONObject("seller").getString("contactQq"))
        assertTrue(order.getJSONObject("seller").isNull("avatar"))
        assertEquals("KeepPlayer",order.getJSONObject("buyer").getString("displayName"))
        assertEquals("67890",order.getJSONObject("buyer").getString("contactQq"))
    }
    @Test fun delayedForwardCannotRestoreDeletedMessageText(){
        val message=JSONObject("""{"content":"private message","forwarded":{"messageId":"message-old","content":"private message","availability":"AVAILABLE","sender":{"playerRef":"old","gameId":"OldPlayer"}},"sender":{"playerRef":"live","gameId":"KeepPlayer"}}""")
        redactErasedAccounts(message,setOf("old"))
        assertFalse(message.toString().contains("private message"))
        assertEquals("UNAVAILABLE",message.getJSONObject("forwarded").getString("availability"))
        assertEquals("KeepPlayer",message.getJSONObject("sender").getString("gameId"))
    }
    @Test fun newRegistrationWithSameNameHasIndependentReference(){
        val fresh=JSONObject("""{"gameId":"OldPlayer","playerRef":"new","qq":"12345"}""")
        redactErasedAccounts(fresh,setOf("old"))
        assertFalse(fresh.has("deleted"));assertEquals("OldPlayer",fresh.getString("gameId"))
    }
}
