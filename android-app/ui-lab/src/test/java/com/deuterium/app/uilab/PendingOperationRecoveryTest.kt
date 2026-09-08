package com.deuterium.app.uilab

import kotlinx.coroutines.runBlocking
import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test

class PendingOperationRecoveryTest {
    private val scope=FinancialScope("original-player","https://server.example")
    private fun pending()=JSONObject().put("owner",scope.owner).put("origin",scope.origin)
        .put("request",JSONObject().put("clientRequestId","original-request").put("quoteId","original-quote").put("expectedQuoteVersion",7))
    private fun operation(status:String="PROCESSING")=JSONObject().put("operationId","original-operation").put("status",status)

    @Test fun absentLookupRepeatsOnlyTheOriginalCommandForEachSupportedKind()=runBlocking {
        for((kind,path) in mapOf("STORE_PURCHASE" to "/store/orders","MARKET_PURCHASE" to "/market/orders","COMMISSION_PUBLISH" to "/commissions")) {
            val original=pending();val body=original.getJSONObject("request").toString();val calls=mutableListOf<Pair<String,String>>()
            var persisted:JSONObject?=null
            val recovered=recoverPendingOperation(kind,original,scope,{scope},{method,url,input->
                calls+=method to url
                if(method=="GET")throw ApiFailure("NOT_FOUND","missing",404)
                assertEquals(path,url);assertEquals(body,input!!.toString())
                JSONObject().put("operation",operation())
            },{persisted=it})
            assertEquals(listOf("GET","POST"),calls.map{it.first})
            assertTrue(calls.first().second.contains("clientRequestId=original-request"))
            assertEquals("original-operation",recovered.getString("operationId"))
            assertEquals(body,persisted!!.getJSONObject("request").toString())
            assertEquals(scope.owner,persisted!!.getString("owner"));assertEquals(scope.origin,persisted!!.getString("origin"))
        }
    }

    @Test fun existingRequestOnlyGetsItsOperationAndNeverPosts()=runBlocking {
        var requests=0;var saved:JSONObject?=null
        val result=recoverPendingOperation("STORE_PURCHASE",pending(),scope,{scope},{method,_,_->
            requests++;assertEquals("GET",method);operation("UNKNOWN")
        },{saved=it})
        assertEquals(1,requests);assertEquals("UNKNOWN",result.getString("status"))
        assertEquals("original-operation",saved!!.getString("operationId"))
    }

    @Test fun knownOperationIdIsNeverResubmittedEvenIfItsLookupIs404()=runBlocking {
        var requests=0;var writes=0
        val failure=runCatching { recoverPendingOperation("STORE_PURCHASE",pending().put("operationId","registered"),scope,{scope},
            {method,path,_->requests++;assertEquals("GET",method);assertEquals("/operations/registered",path);throw ApiFailure("NOT_FOUND","missing",404)},
            {writes++}) }.exceptionOrNull()
        assertTrue(failure is ApiFailure);assertEquals(1,requests);assertEquals(0,writes)
    }

    @Test fun unstructured404NetworkErrorsAndUnknownStatesNeverPermitPost()=runBlocking {
        for(failure in listOf(ApiFailure("HTTP_404","proxy",404),ApiFailure("NETWORK_UNAVAILABLE","offline"),
            ApiFailure("UNKNOWN","unknown",409),ApiFailure("NOT_FOUND","unavailable",503))) {
            var requests=0;var writes=0
            val actual=runCatching { recoverPendingOperation("STORE_PURCHASE",pending(),scope,{scope},
                {method,_,_->requests++;assertEquals("GET",method);throw failure},{writes++}) }.exceptionOrNull()
            assertSame(failure,actual);assertEquals(1,requests);assertEquals(0,writes)
        }
    }

    @Test fun expiredOriginalQuoteStopsAndRequiresAUserToConfirmAnotherQuote()=runBlocking {
        val bodies=mutableListOf<String>();var clears=0
        val failure=runCatching { recoverPendingOperation("STORE_PURCHASE",pending(),scope,{scope},{method,path,body->
            if(method=="GET")throw ApiFailure("NOT_FOUND","missing",404)
            assertEquals("/store/orders",path);bodies+=body!!.toString();throw ApiFailure("QUOTE_EXPIRED","请重新获取报价",409)
        },{assertNull(it);clears++}) }.exceptionOrNull() as ApiFailure
        assertEquals("QUOTE_EXPIRED",failure.code);assertEquals(1,clears)
        assertEquals(listOf(pending().getJSONObject("request").toString()),bodies)
    }

    @Test fun uncertainReplayFailureKeepsOriginalPendingRequest()=runBlocking {
        var calls=0;var writes=0
        val failure=runCatching { recoverPendingOperation("COMMISSION_PUBLISH",pending(),scope,{scope},{method,_,_->
            calls++;if(method=="GET")throw ApiFailure("NOT_FOUND","missing",404)
            throw ApiFailure("NETWORK_UNAVAILABLE","response lost")
        },{writes++}) }.exceptionOrNull() as ApiFailure
        assertEquals("NETWORK_UNAVAILABLE",failure.code);assertEquals(2,calls);assertEquals(0,writes)
    }

    @Test fun mismatchedMissingOrLegacyScopesCannotMakeAnyRequest()=runBlocking {
        val old=pending().apply{remove("origin")}
        for(value in listOf(pending().put("owner","other-player"),pending().put("origin","https://other.example"),old)) {
            var calls=0
            val failure=runCatching { recoverPendingOperation("STORE_PURCHASE",value,scope,{scope},
                {_,_,_->calls++;operation()},{fail("A mismatched scope must not write")}) }.exceptionOrNull() as ApiFailure
            assertEquals("REQUEST_SCOPE_MISMATCH",failure.code);assertEquals(0,calls)
        }
    }

    @Test fun accountOrOriginChangeWhileLookupRunsPreventsReplay()=runBlocking {
        for(changed in listOf(FinancialScope("other-player",scope.origin),FinancialScope(scope.owner,"https://other.example"),FinancialScope("",scope.origin))) {
            var current=scope;var requests=0;var writes=0
            val failure=runCatching { recoverPendingOperation("STORE_PURCHASE",pending(),scope,{current},
                {method,_,_->requests++;assertEquals("GET",method);current=changed;throw ApiFailure("NOT_FOUND","missing",404)},
                {writes++}) }.exceptionOrNull() as ApiFailure
            assertEquals("SESSION_CHANGED",failure.code);assertEquals(1,requests);assertEquals(0,writes)
        }
    }

    @Test fun accountChangeAfterPostDoesNotAttachTheOldReceiptToAnotherAccount()=runBlocking {
        var current=scope;var requests=0;var writes=0
        val failure=runCatching { recoverPendingOperation("STORE_PURCHASE",pending(),scope,{current},{method,_,_->
            requests++;if(method=="GET")throw ApiFailure("NOT_FOUND","missing",404)
            current=FinancialScope("other-player",scope.origin)
            JSONObject().put("operation",operation("COMPLETED"))
        },{writes++}) }.exceptionOrNull() as ApiFailure
        assertEquals("SESSION_CHANGED",failure.code);assertEquals(2,requests);assertEquals(0,writes)
    }
}
