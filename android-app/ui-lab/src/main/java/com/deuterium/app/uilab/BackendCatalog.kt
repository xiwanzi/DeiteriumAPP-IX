package com.deuterium.app.uilab

import androidx.compose.runtime.*
import androidx.compose.ui.graphics.Color
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import org.json.JSONArray
import org.json.JSONObject
import java.util.UUID
import java.time.Instant
import java.time.LocalDateTime
import java.time.ZoneId

private fun JSONArray.strings()=(0 until length()).map{getString(it)}
private fun JSONObject.array(name:String)=optJSONArray(name) ?: JSONArray()
private fun amount(cents:Long)=java.math.BigDecimal(cents).movePointLeft(2).toPlainString()
private val newestOrderFirst=compareByDescending<CommerceOrder>{it.createdAt}.thenByDescending{it.id}
fun categoryCode(label:String)=when(marketCategoryLabel(label)){"建材"->"MATERIALS";"装备"->"EQUIPMENT";"补给"->"SUPPLIES";"装饰"->"DECORATION";"建筑服务"->"CONSTRUCTION";else->"OTHER"}
fun categoryName(code:String)=when(code){"MATERIALS"->"建材";"EQUIPMENT"->"装备";"SUPPLIES"->"补给";"DECORATION"->"装饰";"CONSTRUCTION"->"建筑服务";else->"其他"}
fun methodCode(method:DeliveryMethod)=when(method){DeliveryMethod.Door->"DOOR";DeliveryMethod.Pickup->"PICKUP";DeliveryMethod.Worksite->"WORKSITE";DeliveryMethod.Mailbox->"MAILBOX"}
fun deliveryMethod(code:String)=when(code){"DOOR"->DeliveryMethod.Door;"PICKUP"->DeliveryMethod.Pickup;"WORKSITE"->DeliveryMethod.Worksite;else->DeliveryMethod.Mailbox}

class BackendCatalog(private val api:BackendApi,private val state:LabState) {
    val couponAttention=CouponAttention(api)
    var shopError by mutableStateOf<String?>(null);private set
    var marketError by mutableStateOf<String?>(null);private set
    var error by mutableStateOf<String?>(null);private set
    var cartVersion by mutableLongStateOf(0);private set
    var cartBusy by mutableStateOf(false);private set
    val coupons=mutableStateListOf<StoreCoupon>()
    var couponError by mutableStateOf<String?>(null);private set
    var couponsLoading by mutableStateOf(false);private set
    var hasMoreCoupons by mutableStateOf(false);private set
    private var couponCursor:String?=null
    private var couponRevision=0L
    private var couponClock=Instant.now()
    private var couponClockNanos=System.nanoTime()
    fun couponNow():Instant=couponClock.plusNanos((System.nanoTime()-couponClockNanos).coerceAtLeast(0))
    suspend fun refreshCoupons(more:Boolean=false) {
        val revision=++couponRevision;couponsLoading=true
        val scope=api.financialScope()
        runCatching{
                val value=api.request("GET","/store/coupons?limit=50"+(if(more&&couponCursor!=null)"&cursor=${java.net.URLEncoder.encode(couponCursor,"UTF-8")}" else ""))
                scope.verifyCurrent(api.financialScope())
                val items=value.array("items");val next=(0 until items.length()).map{storeCoupon(items.getJSONObject(it))}
                value to next
            }.onSuccess{(value,next)->
                if(revision!=couponRevision)return@onSuccess
                if(!more)coupons.clear();coupons.addAll(next.filter{item->coupons.none{it.id==item.id}})
                couponCursor=value.optJSONObject("_page")?.optString("nextCursor")?.takeUnless{it.isBlank()||it=="null"};hasMoreCoupons=couponCursor!=null
                couponClock=runCatching{Instant.parse(value.getString("_serverTime"))}.getOrDefault(Instant.now());couponClockNanos=System.nanoTime();couponError=null
            }.onFailure{if(revision==couponRevision)couponError=it.message}
        if(revision==couponRevision)couponsLoading=false
    }
    private val cartLock=Mutex()
    private val orderSubmission=Mutex()
    private val storeRefresh=ConcurrentRefresh()
    private val categories=mutableMapOf<String,String>()
    private val brands=mutableMapOf<String,String>()
    private var visibilityRevision=0L
    private val serverOrders=mutableMapOf<String,JSONObject>()
    private fun report(failure:Throwable){error=failure.message ?: "服务暂不可用";state.storageMessage=error}
    suspend fun refreshStore(){
        val requestScope=api.financialScope()
        runCatching{
            storeRefresh.run {
                runCatching{api.listAll("/store/categories").forEach{categories[it.getString("categoryId")]=it.getString("name")}}
                runCatching{api.listAll("/store/brands").forEach{brands[it.getString("brandId")]=it.getString("name")}}
                val products=api.listAll("/store/products").map(::product)
                requestScope.verifyCurrent(api.financialScope())
                ShopCatalog.clear();ShopCatalog.addAll(products);shopError=null
            }
        }.onFailure{shopError=it.message}
        // Cart refresh remains per caller; a concurrent cart edit must still be observed.
        refreshCart()
    }
    private fun product(value:JSONObject):ShopProduct {
        val c=value.getJSONObject("content")
        return ShopProduct(value.getString("productId"),c.getString("title"),categories[c.optString("categoryId")] ?: "商城商品",c.getString("subtitle"),apiCents(value.optString("effectivePrice",c.getString("price"))),
            runCatching{Color(android.graphics.Color.parseColor(c.optString("accentColor")))}.getOrDefault(Color(0xFFEDF1F4)),darkArt=c.optString("posterTone")=="DARK",
            contents=c.array("includedItems").strings(),brand=brands[c.optString("brandId")] ?: "Deuterium",photos=c.array("galleryAssetIds").strings().map{"asset:$it"},description=c.getString("description"),
            deliverySummary=c.getString("deliverySummary"),estimatedDelivery=c.getString("estimatedDelivery"),version=value.getLong("version"),stock=if(value.isNull("availableStock"))999 else value.getInt("availableStock"),limit=c.optInt("limitPerOrder",9),
            originalPrice=apiCents(c.getString("price")),deliveryCredits=c.optLong("deliveryCredits"),purchaseLimits=productLimitDescriptions(c.optJSONObject("purchaseLimits")),storeId=value.optString("storeId"))
    }
    suspend fun refreshProduct(id:String){runCatching{api.request("GET","/store/products/$id")}.onSuccess{value->val p=product(value);val index=ShopCatalog.indexOfFirst{it.id==id};if(index>=0)ShopCatalog[index]=p else ShopCatalog.add(p)}.onFailure{report(it)}}
    suspend fun refreshMarket(){
        val revision=visibilityRevision
        runCatching{val visible=api.listAll("/market/listings");val own=api.listAll("/market/me/listings");(visible+own).associateBy{it.getString("listingId")}.values.filterNot{it.optBoolean("hiddenFromHistory")}.map(::listing)}
            .onSuccess{if(revision!=visibilityRevision)return@onSuccess;state.commerce.listings.clear();state.commerce.listings.addAll(it.filterNot{entry->state.isErasedPlayer(entry.sellerRef)}.sortedWith(compareByDescending<MarketListing>{entry->entry.createdAt}.thenByDescending{entry->entry.id}));marketError=null}.onFailure{marketError=it.message}
    }
    private fun listing(value:JSONObject):MarketListing {
        val seller=value.getJSONObject("seller");val name=seller.getString("gameId")
        if(name!=state.userName&&!state.isErasedPlayer(seller.optString("playerRef"))){Players.removeAll{it.name==name};Players.add(PlayerProfile(name,seller.optString("qq").takeUnless{it=="null"}.orEmpty(),seller.optString("bio"),seller.optBoolean("online"),seller.optString("lastSeenAt"),seller.getString("playerRef"),seller.optJSONObject("avatar")?.optString("assetId")?.takeIf{it.isNotBlank()}?.let{"asset:$it"}))}
        val photos=value.array("photoAssetIds").strings().map{"asset:$it"}
        return MarketListing(value.getString("listingId"),value.getString("title"),value.getString("subtitle"),value.getString("description"),categoryName(value.getString("categoryCode")),apiCents(value.getString("price")),value.getInt("stock"),
            name,value.getString("contactQq"),value.array("deliveryMethods").strings().map(::deliveryMethod).toSet(),value.optString("pickupLocation"),imageUri=photos.firstOrNull(),active=value.getBoolean("active"),imageUris=photos,workHours=value.optInt("workHours",168),version=value.getLong("version"),sellerRef=seller.getString("playerRef"),canHideRecord=value.optBoolean("canHideRecord"),createdAt=date(value,"createdAt") ?: LocalDateTime.MIN)
    }
    private fun putListing(value:JSONObject):String {val p=listing(value);if(value.optBoolean("hiddenFromHistory")||state.isErasedPlayer(p.sellerRef)){state.commerce.listings.removeAll{it.id==p.id};return p.id};val index=state.commerce.listings.indexOfFirst{it.id==p.id};if(index>=0)state.commerce.listings[index]=p else state.commerce.listings.add(0,p);state.commerce.listings.sortWith(compareByDescending<MarketListing>{it.createdAt}.thenByDescending{it.id});return p.id}
    suspend fun refreshListing(id:String){val revision=visibilityRevision;runCatching{api.request("GET","/market/listings/$id")}.onSuccess{if(revision==visibilityRevision)putListing(it)}.onFailure{report(it)}}
    suspend fun publish(draft:MarketListing,original:MarketListing?):String? = runCatching{
        require(draft.photos.isNotEmpty()&&draft.photos.all{it.startsWith("asset:")}){"请先完成图片上传"}
        val content=JSONObject().put("title",draft.title).put("subtitle",draft.subtitle).put("description",draft.description).put("categoryCode",categoryCode(draft.category))
            .put("price",amount(draft.price)).put("stock",draft.stock).put("photoAssetIds",JSONArray(draft.photos.map{it.removePrefix("asset:")})).put("contactQq",draft.qq)
            .put("deliveryMethods",JSONArray(draft.methods.map(::methodCode))).put("pickupLocation",draft.pickupLocation).put("workHours",draft.workHours)
        val request=JSONObject().put("clientRequestId",draft.id).put("content",content)
        val response=if(original==null)api.request("POST","/market/listings",request) else api.request("POST","/market/listings/${original.id}/republish",request.put("expectedVersion",original.version))
        error=null;putListing(response)
    }.getOrElse{report(it);null}
    suspend fun unlist(id:String):Boolean=runCatching{
        val original=state.commerce.listings.first{it.id==id}
        putListing(api.request("POST","/market/listings/$id/unlist",JSONObject().put("clientRequestId",UUID.randomUUID().toString()).put("expectedVersion",original.version)));true
    }.getOrElse{report(it);false}
    suspend fun hideListing(id:String):Boolean {val version=state.commerce.listings.find{it.id==id}?.version ?: return false;return hideRecord("market/listings",id,version){state.commerce.listings.removeAll{it.id==id}}}
    suspend fun hideOrder(id:String):Boolean {val version=state.commerce.order(id)?.version ?: return false;return hideRecord("orders",id,version){state.commerce.orders.removeAll{it.id==id};serverOrders.remove(id)}}
    private suspend fun hideRecord(path:String,id:String,version:Long,remove:()->Unit):Boolean=runCatching{
        val response=api.request("POST","/$path/$id/hide",JSONObject().put("clientRequestId",UUID.nameUUIDFromBytes("${api.playerRef}:hide:$path:$id:$version".toByteArray()).toString()).put("expectedVersion",version))
        require(response.getBoolean("hidden")){"删除未完成"};visibilityRevision++;remove();error=null;true
    }.getOrElse{report(it);false}
    private fun applyCart(value:JSONObject){cartVersion=value.getLong("version");state.cart.clear();val items=value.getJSONArray("items");for(index in 0 until items.length()){val item=items.getJSONObject(index);state.cart[item.getString("productId")]=item.getInt("quantity")}}
    suspend fun refreshCart(){runCatching{api.request("GET","/store/cart")}.onSuccess(::applyCart).onFailure{error=it.message}}
    suspend fun setCart(productId:String,quantity:Int):Boolean=cartLock.withLock{
        cartBusy=true
        try{runCatching{
            applyCart(api.request("GET","/store/cart"))
            val request=JSONObject().put("clientRequestId",UUID.randomUUID().toString()).put("expectedVersion",cartVersion)
            val result=if(quantity<=0)api.request("POST","/store/cart/items/$productId/remove",request) else api.request("PUT","/store/cart/items/$productId",request.put("quantity",quantity))
            applyCart(result);error=null;true
        }.getOrElse{report(it);false}}finally{cartBusy=false}
    }
    suspend fun quoteStore(items:List<Pair<ShopProduct,Int>>,quoteKey:String):JSONObject?=runCatching{
        api.request("POST","/checkout/quotes",JSONObject().put("channel","OFFICIAL_STORE").put("source","CART").put("items",JSONArray(items.map{(p,count)->JSONObject().put("productId",p.id).put("quantity",count).put("expectedProductVersion",p.version)}))
            .put("delivery",JSONObject().put("method","MAILBOX").put("location","").put("projectName","")),idempotencyKey=quoteKey).also{checkoutSummary(it)}
    }.getOrElse{report(it);null}
    suspend fun quoteMarket(listing:MarketListing,count:Int,method:DeliveryMethod,location:String,project:String,quoteKey:String):JSONObject?=runCatching{
        api.request("POST","/checkout/quotes",JSONObject().put("channel","PLAYER_MARKET").put("items",JSONArray(listOf(JSONObject().put("productId",listing.id).put("quantity",count).put("expectedProductVersion",listing.version))))
            .put("delivery",JSONObject().put("method",methodCode(method)).put("location",location).put("projectName",project)),idempotencyKey=quoteKey).also{checkoutSummary(it)}
    }.getOrElse{report(it);null}
    private fun date(value:JSONObject,key:String):LocalDateTime?=value.optString(key).takeUnless{it.isBlank()||it=="null"}?.let{Instant.parse(it).atZone(ZoneId.systemDefault()).toLocalDateTime()}
    private fun putOrder(value:JSONObject):String {
        val id=value.getString("orderId");if(value.optBoolean("hiddenFromHistory")){state.commerce.orders.removeAll{it.id==id};return id};val refund=value.optJSONObject("refund");val delivery=value.getJSONObject("delivery");val seller=value.getJSONObject("seller");val buyer=value.getJSONObject("buyer")
        val items=value.getJSONArray("items");val lines=(0 until items.length()).map{index->val item=items.getJSONObject(index);val photos=item.array("photoAssetIds").strings().map{"asset:$it"};OrderLine(item.getString("productId"),item.getString("title"),item.optString("subtitle"),apiCents(item.getString("unitPrice")),item.getInt("quantity"),image=if(value.optString("orderType")=="AI_SUBSCRIPTION")R.drawable.xiaoxiang_avatar else 0,imageUri=photos.firstOrNull(),imageUris=photos)}
        val amounts=checkoutAmounts(value,lines.fold(0L){sum,line->Math.addExact(sum,line.total)},"amount")
        val stage=when(value.getString("status")){"SHIPPED"->OrderStage.Shipped;"WORK_COMPLETED"->OrderStage.Completed;"CONFIRMED"->OrderStage.Confirmed;"AWAITING_CLAIM"->OrderStage.AwaitingClaim;"CLAIMED"->OrderStage.Claimed;else->OrderStage.AwaitingShipment}
        val order=CommerceOrder(id,id,if(value.getString("channel")=="OFFICIAL_STORE")OrderChannel.Official else OrderChannel.Market,buyer.getString("displayName"),seller.getString("displayName"),seller.optString("contactQq").takeUnless{it=="null"}.orEmpty(),lines,deliveryMethod(delivery.getString("method")),delivery.optString("location"),date(value,"createdAt")!!,stage,
            refund=when{value.optString("status")=="REFUNDED"->RefundState.Approved;refund?.optString("status") in listOf("REQUESTED","PROCESSING")->RefundState.Requested;refund?.optString("status")=="APPROVED"->RefundState.Approved;refund?.optString("status")=="REJECTED"->RefundState.Rejected;else->RefundState.None},refundReason=refund?.optString("reason").orEmpty(),
            shippedAt=date(value,"shippedAt"),finishedAt=date(value,"confirmedAt"),construction=value.optBoolean("construction"),confirmationHours=value.optInt("confirmationHours",72),projectName=delivery.optString("projectName"),
            deadlineMillis=value.optString("autoConfirmAt").takeUnless{it.isBlank()||it=="null"}?.let{Instant.parse(it).toEpochMilli()},pausedMillis=if(value.isNull("pausedRemainingSeconds"))null else value.optLong("pausedRemainingSeconds")*1000,
            refundAttempts=value.optInt("refundAttemptsUsed"),refundRequestedAt=refund?.let{date(it,"requestedAt")},refundResolvedAt=refund?.let{date(it,"resolvedAt")},automatic=value.optBoolean("automatic"),completedAt=date(value,"workCompletedAt"),rejectionReason=refund?.optString("rejectionReason").orEmpty(),
            serverStatus=value.getString("status"),fundsStatus=value.getString("fundsStatus"),serverActions=value.array("availableActions").strings().toSet(),version=value.getLong("version"),refundId=refund?.optString("refundId"),refundVersion=refund?.optLong("version",1) ?: 1,interventionCaseId=value.optString("interventionCaseId").takeUnless{it.isBlank()||it=="null"},intervention=state.interventions?.cached(value.optString("interventionCaseId")),pendingOperationId=value.optString("pendingOperationId").takeUnless{it.isBlank()||it=="null"},canHideRecord=value.optBoolean("canHideRecord"),isSaki=value.optString("orderType")=="AI_SUBSCRIPTION",sellerAvatarUri=seller.optJSONObject("avatar")?.optString("url"),aiExpiresAt=date(value,"aiExpiresAt"),
            paidAmount=amounts.total,originalAmount=amounts.originalTotal,productDiscount=amounts.productDiscount,couponDiscount=amounts.couponDiscount,couponName=value.optJSONObject("coupon")?.optString("name"))
        val index=state.commerce.orders.indexOfFirst{it.id==id}
        state.commerce.orders.replaceInOrder(index,order,newestOrderFirst)
        serverOrders[id]=value;return id
    }
    suspend fun refreshOrders(){val revision=visibilityRevision;runCatching{api.listAll("/orders")}.onSuccess{values->if(revision!=visibilityRevision)return@onSuccess;state.commerce.orders.clear();values.forEach{putOrder(it)}}.onFailure{report(it)}}
    suspend fun refreshOrder(id:String){val revision=visibilityRevision;runCatching{var value=api.request("GET","/orders/$id");value.optString("pendingOperationId").takeUnless{it.isBlank()||it=="null"}?.let{operationId->runCatching{api.request("GET","/operations/$operationId")}.onSuccess{op->if(op.optString("status") in setOf("COMPLETED","FAILED"))value=api.request("GET","/orders/$id")}};value}.onSuccess{value->if(revision==visibilityRevision)putOrder(value);value.optString("interventionCaseId").takeUnless{it.isBlank()||it=="null"}?.let{state.interventions?.refresh(it)}}.onFailure{report(it)}}
    suspend fun createOrder(quote:JSONObject,key:String):String? {
        if(!orderSubmission.tryLock())return null
        return try { createOrderLocked(quote,key) } finally { orderSubmission.unlock() }
    }
    private suspend fun createOrderLocked(quote:JSONObject,key:String):String? {
        val requestScope=api.financialScope()
        val official=quote.getString("channel")=="OFFICIAL_STORE";val kind=if(official)"STORE_PURCHASE" else "MARKET_PURCHASE"
        if(api.pendingOperation(kind)!=null){report(ApiFailure("RESULT_UNKNOWN","上一笔交易结果待确认，请先查看订单"));recoverOrder(kind);return null}
        val body=JSONObject().put("clientRequestId",key).put("quoteId",quote.getString("quoteId")).put("expectedQuoteVersion",quote.getLong("version"))
        api.saveOperation(kind,JSONObject().put("request",body),requestScope)
        return runCatching {
            val response=api.request("POST",if(official)"/store/orders" else "/market/orders",body)
            val op=response.getJSONObject("operation");api.saveOperation(kind,JSONObject().put("request",body).put("operationId",op.getString("operationId")),requestScope)
            requestScope.verifyCurrent(api.financialScope())
            if(op.getString("status")=="COMPLETED"){
                val order=response.getJSONObject("order");require(order.getString("fundsStatus") in setOf("PAID","HELD","SETTLED","INTERVENTION_HOLD")){"付款结果仍待核对"}
                val id=putOrder(order);api.saveOperation(kind,null,requestScope);state.refresh();refreshCart();error=null;id
            }else if(op.getString("status")=="FAILED"){api.saveOperation(kind,null,requestScope);error("付款未完成，请检查余额与服务器状态")}
            else error("交易结果待确认，请勿重复付款；重新打开应用会继续核对")
        }.getOrElse{failure->if(failure is ApiFailure&&(failure.status in listOf(400,401,403,404,409,422,501)||failure.code=="CAPABILITY_UNAVAILABLE"))api.saveOperation(kind,null,requestScope);report(failure);null}
    }
    suspend fun recoverOrder(kind:String){
        val pending=api.pendingOperation(kind) ?: return
        val requestScope=api.financialScope()
        runCatching{
            val op=recoverPendingOperation(kind,pending,requestScope,{api.financialScope()},
                {method,path,body->api.request(method,path,body)},{value->api.saveOperation(kind,value,requestScope)})
            when(op.getString("status")){"COMPLETED"->{val order=api.request("GET","/orders/${op.getString("resourceId")}");requestScope.verifyCurrent(api.financialScope());putOrder(order);api.saveOperation(kind,null,requestScope);state.refresh();refreshCart()};"FAILED"->api.saveOperation(kind,null,requestScope);else->report(ApiFailure("RESULT_UNKNOWN","仍有交易结果待确认，请勿重复付款"))}
        }.onFailure{report(it)}
    }
    suspend fun orderAction(id:String,action:String,description:String="",approve:Boolean?=null):Boolean {
        val original=state.commerce.order(id) ?: return false
        return runCatching {
            val nested=action in setOf("withdraw","resolve")
            val version=if(nested)original.refundVersion else original.version
            val request=JSONObject().put("expectedVersion",version)
            val suffix=when(action){
                "request-refund"->{request.put("reasonCode","OTHER").put("description",description).put("evidenceAssetIds",JSONArray());"refunds"}
                "withdraw"->"refunds/${original.refundId ?: error("退款记录尚未同步")}/withdraw"
                "resolve"->{request.put("decision",if(approve==true)"APPROVE" else "REJECT").put("reason",description);"refunds/${original.refundId ?: error("退款记录尚未同步")}/resolve"}
                "complete-work"->{request.put("description",description.ifBlank{"已按约定完成工程，请验收。"}).put("evidenceAssetIds",JSONArray());action}
                else->action
            }
            request.put("clientRequestId",UUID.nameUUIDFromBytes("${api.playerRef}:$id:$suffix:$request".toByteArray()).toString())
            val response=api.request("POST","/orders/$id/$suffix",request)
            val value=if(action=="confirm")response.optJSONObject("order") ?: api.request("GET","/orders/$id") else response
            putOrder(value)
            if(!value.isNull("pendingOperationId")&&value.optString("pendingOperationId").isNotBlank())error("请求正在处理，请勿重复操作；状态将自动更新")
            if(action=="confirm"&&response.getJSONObject("operation").getString("status")!="COMPLETED")error("结算结果待确认，请勿重复操作")
            if((action=="confirm"&&value.getString("fundsStatus")!="SETTLED")||(action=="resolve"&&approve==true&&value.getString("fundsStatus")!="REFUNDED"))error("资金处理已提交，结果仍待确认，请勿重复操作")
            state.refresh();error=null;true
        }.getOrElse{report(it);runCatching{putOrder(api.request("GET","/orders/$id"))};false}
    }
    suspend fun refreshMailbox(id:String){runCatching{putOrder(api.request("GET","/orders/$id/mailbox"))}.onFailure{report(it)}}
}
