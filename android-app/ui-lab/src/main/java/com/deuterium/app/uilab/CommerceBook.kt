package com.deuterium.app.uilab

import androidx.compose.runtime.*
import java.time.*
import java.time.format.DateTimeFormatter

enum class OrderChannel { Official, Market }
enum class OrderStage { AwaitingShipment, Shipped, Completed, Confirmed, AwaitingClaim, Claimed }
enum class RefundState { None, Requested, Rejected, Approved }
enum class DeliveryMethod(val label:String) { Door("送货上门"), Pickup("约定自取"), Mailbox("游戏内邮箱"), Worksite("工程现场") }
data class OrderLine(val productId:String,val title:String,val subtitle:String,val unitPrice:Long,val quantity:Int,
    val image:Int=0,val artKey:String="",val imageUri:String?=null,val imageUris:List<String> = emptyList()) { val total:Long get()=Math.multiplyExact(unitPrice,quantity.toLong()) }
data class MarketListing(val id:String,val title:String,val subtitle:String,val description:String,val category:String,val price:Long,val stock:Int,
    val seller:String,val qq:String,val methods:Set<DeliveryMethod>,val pickupLocation:String,val artKey:String="",val imageUri:String?=null,val active:Boolean=true,
    val imageUris:List<String> = emptyList(),val workHours:Int=168,val version:Long=1,val sellerRef:String="",val canHideRecord:Boolean=false,val createdAt:LocalDateTime=LocalDateTime.MIN) {
    val photos:List<String> get()=imageUris.ifEmpty{listOfNotNull(imageUri)}
    val construction:Boolean get()=category.startsWith("建筑服务")
    val confirmationHours:Int get()=if(construction)workHours else 72
}
data class TradeNotice(val id:String,val recipient:String,val title:String,val body:String,val orderId:String,val kind:String,val at:LocalDateTime,val read:Boolean=false,val route:String=if(kind.startsWith("commission_"))"commission:$orderId" else if(kind.startsWith("refund"))"refund:$orderId" else "order:$orderId")
data class CommerceOrder(val id:String,val key:String,val channel:OrderChannel,val buyer:String,val seller:String,val sellerQQ:String,
    val lines:List<OrderLine>,val method:DeliveryMethod,val location:String,val createdAt:LocalDateTime,
    val stage:OrderStage,val refund:RefundState=RefundState.None,val refundReason:String="",val refundMessage:String="",
    val shippedAt:LocalDateTime?=null,val finishedAt:LocalDateTime?=null,val refundedAt:LocalDateTime?=null,
    val construction:Boolean=false,val confirmationHours:Int=72,val projectName:String="",val deadlineMillis:Long?=null,
    val pausedMillis:Long?=null,val refundAttempts:Int=0,val refundRequestedAt:LocalDateTime?=null,val refundResolvedAt:LocalDateTime?=null,val automatic:Boolean=false,
    val completedAt:LocalDateTime?=null,val rejectionReason:String="",val intervention:InterventionCase?=null,
    val serverStatus:String?=null,val fundsStatus:String?=null,val serverActions:Set<String>?=null,val version:Long=1,val refundId:String?=null,val refundVersion:Long=1,val interventionCaseId:String?=null,val pendingOperationId:String?=null,val canHideRecord:Boolean=false,val isSaki:Boolean=false,val sellerAvatarUri:String?=null,val aiExpiresAt:LocalDateTime?=null) {
    val platformPending:Boolean get()=fundsStatus=="INTERVENTION_HOLD"||intervention?.pending==true
    val canIntervene:Boolean get()=pendingOperationId==null&&(serverActions?.contains("REQUEST_INTERVENTION") ?: (channel==OrderChannel.Market&&refund==RefundState.Rejected&&intervention==null))
    val amount:Long get()=lines.sumOf{it.total}
    val receiptLabel:String get()=if(construction)"确认验收" else "确认收货"
    val shipLabel:String get()=if(construction)"开始施工" else "确认发货"
    val status:String get()=when {
        isSaki&&fundsStatus=="SETTLED"->"已开通"
        fundsStatus=="UNPAID"->if(serverStatus=="CANCELLED")"已取消 · 未扣款" else "未付款"
        fundsStatus=="SETTLING"->"结算处理中";fundsStatus=="REFUNDING"->"退款处理中";fundsStatus=="UNKNOWN"->"资金结果待确认"
        pendingOperationId!=null->"操作处理中"
        serverStatus=="PAYMENT_PROCESSING"->"付款结果待确认";serverStatus=="CANCELLED"->"已取消"
        platformPending->"平台介入中";intervention?.decision!=null->"平台判决 · ${intervention.resultText}"
        refund==RefundState.Approved->"已退款";refund==RefundState.Requested->"退款处理中"
        stage==OrderStage.AwaitingShipment->if(construction)"待开工" else "未发货"
        stage==OrderStage.Shipped->if(construction)"施工中" else "已发货"
        stage==OrderStage.Completed->"已完成 · 待验收"
        stage==OrderStage.Confirmed->if(construction)"已验收" else "已确认"
        stage==OrderStage.AwaitingClaim->"未领取";else->"已领取"
    }
    val held:Boolean get()=fundsStatus?.let{channel==OrderChannel.Market&&it in setOf("HELD","INTERVENTION_HOLD","REFUNDING","SETTLING")} ?: (channel==OrderChannel.Market&&stage!=OrderStage.Confirmed&&refund!=RefundState.Approved)
    val canAutoConfirm:Boolean get()=!platformPending&&stage in listOf(OrderStage.Shipped,OrderStage.Completed)&&refund!=RefundState.Requested&&refund!=RefundState.Approved
    val canReceive:Boolean get()=pendingOperationId==null&&(serverActions?.any{it in setOf("CONFIRM_RECEIPT","CONFIRM_ACCEPTANCE")} ?: (canAutoConfirm&&(!construction||stage==OrderStage.Completed)))
    val canRefund:Boolean get()=pendingOperationId==null&&(serverActions?.contains("REQUEST_REFUND") ?: (!platformPending&&stage !in listOf(OrderStage.Confirmed,OrderStage.Claimed)&&refund!=RefundState.Requested&&refund!=RefundState.Approved&&(channel==OrderChannel.Official||stage==OrderStage.AwaitingShipment||refundAttempts==0)))
}

/** Local transaction model; production authorization and durable settlement belong to the server. */
class CommerceBook(val userName:String,initialListings:List<MarketListing>,private val balance:()->Long,private val bookCash:(Long,String,String)->Unit,
    private val clock:Clock=Clock.systemDefaultZone(),val remoteOnly:Boolean=false,private val onNotice:(TradeNotice)->Unit={}) {
    val listings=mutableStateListOf<MarketListing>().apply{addAll(initialListings)}
    private val seedListings=initialListings.toList()
    var network:BackendCatalog?=null
    var interventions:BackendInterventions?=null
    val orders=mutableStateListOf<CommerceOrder>();val notices=mutableStateListOf<TradeNotice>();val sellerPayments=mutableStateMapOf<String,Long>()
    var error by mutableStateOf<String?>(null);private set
    var nowMillis by mutableLongStateOf(clock.millis());private set
    var nextSequence=1L;private set
    val heldForUser:Long get()=orders.filter{it.buyer==userName&&it.held}.sumOf{it.amount}
    val heldTotal:Long get()=orders.filter{it.held}.sumOf{it.amount}
    val availableBalance:Long get()=balance()
    private fun now()=LocalDateTime.now(clock)
    fun order(id:String)=orders.find{it.id==id}
    private fun reject(message:String):Nothing?{error=message;return null}
    private fun invalid(message:String):Boolean{error=message;return false}
    private fun nextId()="DT"+now().format(DateTimeFormatter.ofPattern("yyyyMMddHHmmss"))+(nextSequence++).toString().padStart(4,'0')
    private fun replace(order:CommerceOrder){orders[orders.indexOfFirst{it.id==order.id}]=order;error=null}
    private fun emit(recipient:String,title:String,body:String,order:CommerceOrder,kind:String) {
        val notice=TradeNotice("N${nextSequence++}",recipient,title,body,order.id,kind,now());notices.add(0,notice);onNotice(notice)
    }
    fun markNoticeRead(id:String){val index=notices.indexOfFirst{it.id==id};if(index>=0)notices[index]=notices[index].copy(read=true)}
    fun reset(){orders.clear();notices.clear();sellerPayments.clear();listings.clear();listings.addAll(seedListings);error=null}
    fun restore(savedListings:List<MarketListing>,savedOrders:List<CommerceOrder>,savedNotices:List<TradeNotice>,payments:Map<String,Long>,sequence:Long){listings.clear();listings.addAll(savedListings);orders.clear();orders.addAll(savedOrders);notices.clear();notices.addAll(savedNotices);sellerPayments.clear();sellerPayments.putAll(payments);nextSequence=sequence.coerceAtLeast(1)}
    private fun validListing(listing:MarketListing):Boolean {
        if(listing.seller!=userName||listing.title.isBlank()||listing.subtitle.isBlank()||listing.description.isBlank()||listing.category.isBlank()||listing.qq.length !in 5..12||listing.qq.any{!it.isDigit()}||listing.price !in 1L..999999999L||listing.stock !in 1..999||listing.photos.size !in 1..5)return invalid("请填写完整信息，上传 1–5 张图片，库存填写 1–999")
        val reserved=orders.filter{it.held}.flatMap{it.lines}.filter{it.productId==listing.id}.sumOf{it.quantity}
        if(listing.stock+reserved>999)return invalid("已有 $reserved 件订单待履约，可用库存最多 ${999-reserved}")
        if(listing.construction){if(listing.workHours !in 1..8760)return invalid("工期需为 1 小时至 365 天，并包含验收预留时间")}
        else if(listing.methods.isEmpty()||listing.methods.any{it !in setOf(DeliveryMethod.Door,DeliveryMethod.Pickup)})return invalid("请选择交付方式")
        if(!listing.construction&&DeliveryMethod.Pickup in listing.methods&&listing.pickupLocation.isBlank())return invalid("请填写自取地点")
        return true
    }
    fun publish(listing:MarketListing):Boolean {
        if(remoteOnly)return invalid("市场发布服务暂不可用，请稍后重试")
        if(!validListing(listing))return false
        if(listings.any{it.id==listing.id})return invalid("该商品已经发布")
        listings.add(0,listing.copy(imageUri=listing.photos.first(),imageUris=listing.photos.toList()));error=null;return true
    }
    fun republish(listing:MarketListing):Boolean {if(remoteOnly)return invalid("交易服务暂不可用，请稍后重试");
        val index=listings.indexOfFirst{it.id==listing.id&&it.seller==userName};if(index<0)return invalid("找不到可编辑的商品")
        if(!validListing(listing))return false
        listings[index]=listing.copy(active=true,imageUri=listing.photos.first(),imageUris=listing.photos.toList());error=null;return true
    }
    fun setListingActive(id:String,active:Boolean):Boolean {if(remoteOnly)return invalid("交易服务暂不可用，请稍后重试");
        val index=listings.indexOfFirst{it.id==id&&it.seller==userName};if(index<0)return false
        listings[index]=listings[index].copy(active=active);return true
    }
    @Synchronized fun buyOfficial(key:String,lines:List<OrderLine>):String? {if(remoteOnly)return reject("交易服务暂不可用，请稍后重试");
        advanceTime();orders.find{it.key==key}?.let{return it.id}
        if(lines.isEmpty()||lines.any{it.quantity !in 1..9||it.unitPrice<=0})return reject("购物袋商品无效")
        val total=runCatching{lines.fold(0L){sum,line->Math.addExact(sum,line.total)}}.getOrNull() ?: return reject("金额超出范围")
        if(total>balance())return reject("余额不足，请调整商品数量")
        val at=now();val order=CommerceOrder(nextId(),key,OrderChannel.Official,userName,"Deuterium 官方商城","1000000",lines.toList(),DeliveryMethod.Mailbox,userName,at,OrderStage.AwaitingClaim,shippedAt=at)
        bookCash(-total,order.seller,"商城订单 ${order.id.takeLast(8)}");orders.add(0,order);error=null
        emit(order.buyer,"商城物品已发送","请到游戏内邮箱领取。未领取前可退款。",order,"delivery");return order.id
    }
    @Synchronized fun buyMarket(key:String,listingId:String,quantity:Int,method:DeliveryMethod?,location:String,buyer:String=userName,projectName:String=""):String? {if(remoteOnly)return reject("交易服务暂不可用，请稍后重试");
        advanceTime();orders.find{it.key==key}?.let{return it.id}
        val listing=listings.find{it.id==listingId&&it.active} ?: return reject("商品已下架")
        if(listing.price<=0)return reject("商品金额无效")
        if(buyer==listing.seller)return reject("不能购买自己发布的商品")
        if(quantity !in 1..999||quantity>listing.stock)return reject("库存不足")
        if(listing.construction){if(method!=DeliveryMethod.Worksite||projectName.isBlank())return reject("请填写建筑项目名称")}
        else if(method==null||method !in listing.methods)return reject("请选择交付方式")
        if(location.isBlank())return reject(if(listing.construction)"请填写工程地点" else "请填写交付地点")
        val total=runCatching{Math.multiplyExact(listing.price,quantity.toLong())}.getOrNull() ?: return reject("金额超出范围")
        if(buyer==userName&&total>balance())return reject("余额不足")
        val line=OrderLine(listing.id,listing.title,listing.subtitle,listing.price,quantity,artKey=listing.artKey,imageUri=listing.photos.firstOrNull(),imageUris=listing.photos.toList())
        val order=CommerceOrder(nextId(),key,OrderChannel.Market,buyer,listing.seller,listing.qq,listOf(line),method!!,location.trim(),now(),OrderStage.AwaitingShipment,construction=listing.construction,confirmationHours=listing.confirmationHours,projectName=projectName.trim())
        if(buyer==userName)bookCash(-total,"平台担保","市场订单 ${order.id.takeLast(8)} · 待确认结算")
        listings[listings.indexOf(listing)]=listing.copy(stock=listing.stock-quantity);orders.add(0,order);error=null
        emit(order.seller,"收到新订单","${order.buyer} 已付款 ${creditForNotice(total)}，请及时${if(order.construction)"安排施工" else "发货"}。",order,"new_order")
        return order.id
    }
    @Synchronized fun ship(id:String,actor:String):Boolean {if(remoteOnly)return invalid("交易服务暂不可用，请稍后重试");
        advanceTime();val order=order(id) ?: return false
        if(order.platformPending||order.seller!=actor||order.channel!=OrderChannel.Market||order.stage!=OrderStage.AwaitingShipment||order.refund in listOf(RefundState.Requested,RefundState.Approved))return false
        val updated=order.copy(stage=OrderStage.Shipped,shippedAt=now(),deadlineMillis=clock.millis()+order.confirmationHours*3600000L)
        replace(updated);emit(order.buyer,if(order.construction)"项目已开始施工" else "卖家已发货",if(order.construction)"工期与验收倒计时已开始，请留意项目进度。" else "72 小时后自动确认，请及时检查物品。",updated,"shipped");return true
    }
    @Synchronized fun completeWork(id:String,actor:String):Boolean {if(remoteOnly)return invalid("交易服务暂不可用，请稍后重试");
        advanceTime();val order=order(id) ?: return false
        if(order.platformPending||!order.construction||order.seller!=actor||order.stage!=OrderStage.Shipped||order.refund in listOf(RefundState.Requested,RefundState.Approved))return false
        val updated=order.copy(stage=OrderStage.Completed,completedAt=now())
        replace(updated);emit(order.buyer,"建筑项目已完成","卖家已提交完成，请在剩余验收期限内检查项目。",updated,"work_completed");return true
    }
    private fun settle(order:CommerceOrder,automatic:Boolean) {
        if(!order.canAutoConfirm)return
        sellerPayments[order.seller]=(sellerPayments[order.seller] ?: 0)+order.amount
        if(order.seller==userName)bookCash(order.amount,order.buyer,"市场订单结算 ${order.id.takeLast(8)}")
        val updated=order.copy(stage=OrderStage.Confirmed,finishedAt=now(),deadlineMillis=null,pausedMillis=null,automatic=automatic)
        replace(updated);emit(order.seller,"订单款项已到账","${creditForNotice(order.amount)} 已结算到钱包。",updated,"settled")
        if(automatic)emit(order.buyer,if(order.construction)"项目已自动验收" else "订单已自动确认","期限已到，平台已完成结算。",updated,"auto_confirmed")
    }
    @Synchronized fun advanceTime() {nowMillis=clock.millis();if(remoteOnly)return;orders.filter{it.canAutoConfirm&&it.deadlineMillis!=null&&it.deadlineMillis<=nowMillis}.toList().forEach{settle(it,true)} }
    @Synchronized fun confirmReceipt(id:String,actor:String):Boolean {if(remoteOnly)return invalid("交易服务暂不可用，请稍后重试");advanceTime();val order=order(id) ?: return false;if(order.buyer!=actor||!order.canReceive)return false;settle(order,false);return true}
    @Synchronized fun claim(id:String,actor:String):Boolean {if(remoteOnly)return invalid("交易服务暂不可用，请稍后重试");
        val order=order(id) ?: return false
        if(order.buyer!=actor||order.channel!=OrderChannel.Official||order.stage!=OrderStage.AwaitingClaim||order.refund in listOf(RefundState.Requested,RefundState.Approved))return false
        replace(order.copy(stage=OrderStage.Claimed,finishedAt=now()));return true
    }
    @Synchronized fun requestRefund(id:String,actor:String,reason:String):Boolean {if(remoteOnly)return invalid("交易服务暂不可用，请稍后重试");
        advanceTime();val order=order(id) ?: return false
        if(order.buyer!=actor||!order.canRefund||reason.isBlank())return false
        val requested=order.copy(refundReason=reason.trim(),refundRequestedAt=now(),refundAttempts=order.refundAttempts+1)
        if(order.channel==OrderChannel.Official||order.stage==OrderStage.AwaitingShipment)refund(requested,reason)
        else {
            val updated=requested.copy(refund=RefundState.Requested,refundMessage="已通知卖家，等待处理",pausedMillis=((order.deadlineMillis ?: clock.millis())-clock.millis()).coerceAtLeast(0),deadlineMillis=null)
            replace(updated);emit(order.seller,"收到退款申请","${order.buyer} 申请退回 ${creditForNotice(order.amount)}。请查看原因并处理。",updated,"refund_request")
            emit(order.buyer,"退款申请已提交","已通知 ${order.seller}，自动确认计时已暂停。",updated,"refund_submitted")
        }
        return true
    }
    @Synchronized fun resolveRefund(id:String,actor:String,approve:Boolean,reason:String=""):Boolean {if(remoteOnly)return invalid("交易服务暂不可用，请稍后重试");
        val order=order(id) ?: return false;if(order.platformPending||order.seller!=actor||order.refund!=RefundState.Requested)return false
        if(!approve&&reason.trim().length !in 2..500)return invalid("请填写 2–500 字的拒绝理由")
        if(approve)refund(order,order.refundReason) else {
            val updated=order.copy(refund=RefundState.Rejected,refundMessage="卖家未同意退款：${reason.trim()}",rejectionReason=reason.trim(),refundResolvedAt=now(),deadlineMillis=clock.millis()+(order.pausedMillis ?: 0),pausedMillis=null)
            replace(updated);emit(order.buyer,"卖家已回复退款申请",updated.refundMessage,updated,"refund_rejected")
        };return true
    }
    @Synchronized fun cancelRefund(id:String,actor:String):Boolean {if(remoteOnly)return invalid("交易服务暂不可用，请稍后重试");
        val order=order(id) ?: return false;if(order.buyer!=actor||order.refund!=RefundState.Requested)return false
        val updated=order.copy(refund=RefundState.None,refundMessage="申请已撤回，唯一退款机会已使用",refundResolvedAt=now(),deadlineMillis=clock.millis()+(order.pausedMillis ?: 0),pausedMillis=null)
        replace(updated);emit(order.seller,"买家撤回退款申请","订单继续履约。",updated,"refund_withdrawn");return true
    }
    private fun refund(order:CommerceOrder,reason:String,restock:Boolean=true) {
        if(order.refund==RefundState.Approved)return
        if(order.buyer==userName)bookCash(order.amount,order.seller,"订单退款 ${order.id.takeLast(8)}")
        if(restock&&order.channel==OrderChannel.Market)order.lines.forEach{line->val index=listings.indexOfFirst{it.id==line.productId};if(index>=0)listings[index]=listings[index].copy(stock=listings[index].stock+line.quantity)}
        val updated=order.copy(refund=RefundState.Approved,refundReason=reason,refundMessage="已原路退回钱包",refundedAt=now(),refundResolvedAt=now(),deadlineMillis=null,pausedMillis=null)
        replace(updated);emit(order.buyer,"退款已到账","${creditForNotice(order.amount)} 已退回钱包。",updated,"refund_approved")
        emit(order.seller,"订单已退款","该订单已取消，货款退回买家。",updated,"refund_closed")
    }
    @Synchronized fun previewDeadline(id:String,actor:String):Boolean {if(remoteOnly)return invalid("交易服务暂不可用，请稍后重试");val order=order(id) ?: return false;if(actor!=userName||!order.canAutoConfirm)return false;replace(order.copy(deadlineMillis=clock.millis()+10000));return true}
    private fun creditForNotice(amount:Long)=java.lang.String.format(java.util.Locale.US,"%,.2f 信用点",amount/100.0)

    @Synchronized fun requestIntervention(id:String,actor:String,key:String,form:InterventionForm):Boolean {if(remoteOnly)return invalid("交易服务暂不可用，请稍后重试");
        advanceTime();val order=order(id) ?: return false
        if(actor!=order.buyer)return false
        val clean=form.normalized()
        order.intervention?.let{return if(it.requestKey==key&&it.form==clean)true else invalid("该交易已提交平台介入，请查看现有案件")}
        if(!order.canIntervene||key.isBlank()||!clean.valid(order.amount))return invalid("请填写完整的介入原因与诉求")
        val snapshot=InterventionSnapshot(id,order.lines.joinToString("、"){it.title},order.buyer,order.seller,order.amount,order.location,
            "${order.method.label} · ${order.projectName} · ${order.confirmationHours} 小时；"+order.lines.joinToString("；"){"${it.title} × ${it.quantity}：${it.subtitle}"},order.createdAt,order.refundReason,order.rejectionReason,order.status,now())
        val case=InterventionCase("CASE-${nextSequence++}",key,clean,snapshot,order.held)
        val updated=order.copy(intervention=case,pausedMillis=if(order.held)order.deadlineMillis?.let{(it-clock.millis()).coerceAtLeast(0)} ?: order.pausedMillis else null,deadlineMillis=null)
        replace(updated)
        listOf(order.buyer,order.seller).forEach{emit(it,"平台介入申请已提交",if(order.held)"担保资金已冻结，确认计时暂停，等待平台处理。" else "交易已结算，平台将人工审核争议。",updated,"intervention_submitted")}
        return true
    }
    @Synchronized fun simulateInterventionReview(id:String):Boolean {if(remoteOnly)return invalid("交易服务暂不可用，请稍后重试");
        val order=order(id) ?: return false;val case=order.intervention ?: return false
        if(case.status!=InterventionStatus.Submitted)return false
        replace(order.copy(intervention=case.copy(status=InterventionStatus.InReview,updatedAt=now())));return true
    }
    /** Local experience tool only. Production verdicts require an authorized platform API. */
    @Synchronized fun simulatePlatformDecision(id:String,decision:PlatformDecision,refundAmount:Long,reason:String):Boolean {if(remoteOnly)return invalid("交易服务暂不可用，请稍后重试");
        val order=order(id) ?: return false;val case=order.intervention ?: return false
        val result=interventionOutcome(case,decision,refundAmount,reason) ?: return invalid("判决无效、金额超出担保范围，或案件已处理")
        if(case.fundsHeldForReview&&!order.held)return invalid("担保状态已变化，请重新查询")
        val resolved=case.copy(status=InterventionStatus.Resolved,updatedAt=now(),decision=decision,decisionReason=reason.trim(),refunded=result.refund,paidToPayee=result.payout)
        var updated=order.copy(intervention=resolved)
        if(decision==PlatformDecision.FullRefund)refund(updated,order.refundReason,restock=false)
        else if(result.refund>0||result.payout>0){
            if(result.refund>0&&order.buyer==userName)bookCash(result.refund,"平台判决退款",order.id)
            if(result.payout>0){sellerPayments[order.seller]=(sellerPayments[order.seller] ?: 0)+result.payout;if(order.seller==userName)bookCash(result.payout,order.buyer,"平台判决结算 ${order.id}")}
            updated=updated.copy(stage=OrderStage.Confirmed,finishedAt=now(),deadlineMillis=null,pausedMillis=null);replace(updated)
        } else {if(result.resume)updated=updated.copy(deadlineMillis=order.pausedMillis?.let{clock.millis()+it},pausedMillis=null);replace(updated)}
        val final=order(id)!!
        listOf(order.buyer,order.seller).forEach{emit(it,"平台判决：${decision.label}","${reason.trim()}；退款 ${creditForNotice(result.refund)}，结算 ${creditForNotice(result.payout)}。",final,"intervention_resolved")}
        return true
    }
}
