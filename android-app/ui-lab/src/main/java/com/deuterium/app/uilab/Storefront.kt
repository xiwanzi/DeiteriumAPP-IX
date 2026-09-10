package com.deuterium.app.uilab

import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.*
import androidx.compose.foundation.shape.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.layout.onGloballyPositioned
import androidx.compose.ui.layout.positionInRoot
import java.util.UUID
import kotlinx.coroutines.launch
import kotlinx.coroutines.delay
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.isActive
import org.json.JSONObject
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.*

val ShopCatalog=mutableStateListOf<ShopProduct>()

@Composable
fun ShopPage(state:LabState,onProduct:(String)->Unit,topInset:Dp,query:String) {
    var category by rememberSaveable { mutableStateOf("全部") }
    LaunchedEffect(Unit){state.commerce.network?.refreshStore()}
    val products=ShopCatalog.filter{(category=="全部"||it.brand==category||it.category==category)&&(query.isBlank()||it.name.contains(query,true)||it.category.contains(query,true)||it.brand.contains(query,true))}
    LazyColumn(contentPadding=PaddingValues(top=topInset,bottom=115.dp),verticalArrangement=Arrangement.spacedBy(22.dp)) {
        item { Row(Modifier.horizontalScroll(rememberScrollState()).padding(horizontal=20.dp),horizontalArrangement=Arrangement.spacedBy(8.dp)) {
            (listOf("全部")+ShopCatalog.map{it.category}.distinct()).forEach{label->ChoiceChip(category==label,{category=label},{Text(label)})}
        } }
        state.commerce.network?.shopError?.let{error->item{Text(error,Modifier.padding(horizontal=24.dp),color=MaterialTheme.colorScheme.error)}}
        if(products.isEmpty())item{Text(if(query.isBlank())"暂无上架商品" else "未找到相关商品",Modifier.padding(28.dp),color=MaterialTheme.colorScheme.onSurfaceVariant)}
        if(products.isNotEmpty()) {
            item { Text(if(query.isBlank())"新品推荐" else "搜索结果",Modifier.padding(horizontal=24.dp),style=MaterialTheme.typography.titleLarge) }
            item { LazyRow(contentPadding=PaddingValues(horizontal=20.dp),horizontalArrangement=Arrangement.spacedBy(16.dp)){items(products,key={it.id}){ProductPoster(it,Modifier.width(308.dp).height(405.dp)){onProduct(it.id)}}} }
            item { Text("探索更多",Modifier.padding(horizontal=24.dp,vertical=6.dp),style=MaterialTheme.typography.titleLarge) }
            items(products.chunked(2)){row->Row(Modifier.padding(horizontal=20.dp),horizontalArrangement=Arrangement.spacedBy(12.dp)){
                row.forEach{product->Surface(onClick={onProduct(product.id)},modifier=Modifier.weight(1f),shape=RoundedCornerShape(24.dp),color=MaterialTheme.colorScheme.surface){Column{
                    ServerAssetImage(product.photos.firstOrNull(),product.name,Modifier.fillMaxWidth().height(170.dp),scale=ContentScale.Crop)
                    Column(Modifier.padding(16.dp)){Text(product.name,style=MaterialTheme.typography.titleMedium,minLines=2);ShopPriceLabel(product,modifier=Modifier.padding(top=6.dp))}
                }}}
                if(row.size==1)Spacer(Modifier.weight(1f))
            }}
        }
    }
}
@Composable
private fun ProductPoster(product:ShopProduct,modifier:Modifier,onClick:()->Unit) {
    val ink=if(product.darkArt)Color.White else Color(0xFF1D1D1F)
    Box(modifier.clip(RoundedCornerShape(28.dp)).background(if(product.darkArt)Color.Black else Color.White).clickable(onClick=onClick)){
        if(product.poster)ServerAssetImage(product.photos.firstOrNull(),product.name,Modifier.fillMaxSize(),scale=ContentScale.Crop) else ServerAssetImage(product.photos.firstOrNull(),product.name,Modifier.align(Alignment.BottomCenter).fillMaxWidth().height(220.dp),scale=ContentScale.Crop)
        Column(Modifier.padding(25.dp)){Text(product.category.uppercase(),fontSize=11.sp,fontWeight=FontWeight.SemiBold,letterSpacing=1.sp,color=ink.copy(alpha=.65f));Text(product.name,Modifier.padding(top=9.dp),fontSize=25.sp,lineHeight=31.sp,fontWeight=FontWeight.Bold,color=ink);Text(product.subtitle,Modifier.padding(top=7.dp),style=MaterialTheme.typography.bodyMedium,color=ink);Text("${credit(product.price)} 信用点",Modifier.padding(top=7.dp),style=MaterialTheme.typography.bodySmall,color=ink.copy(alpha=.75f));if(product.originalPrice>product.price)Text(credit(product.originalPrice),style=MaterialTheme.typography.bodySmall,textDecoration=TextDecoration.LineThrough,color=ink.copy(alpha=.5f))}
    }
}
@Composable
fun ProductPage(product:ShopProduct,state:LabState,topInset:Dp,onBag:()->Unit,onAdded:(ShopProduct,Offset)->Unit) {
    var quantity by rememberSaveable(product.id){mutableIntStateOf(1)};var added by rememberSaveable(product.id){mutableStateOf(false)}
    var origin by remember{mutableStateOf(Offset.Zero)}
    val scope=rememberCoroutineScope();var adding by remember{mutableStateOf(false)}
    LaunchedEffect(product.id){state.commerce.network?.refreshProduct(product.id)}
    LaunchedEffect(product.stock,product.limit){quantity=quantity.coerceAtMost(minOf(product.stock,product.limit).coerceAtLeast(1))}
    Box(Modifier.fillMaxSize()) {
        LazyColumn(contentPadding=PaddingValues(top=topInset,bottom=140.dp),verticalArrangement=Arrangement.spacedBy(28.dp)) {
            item{Column(Modifier.padding(horizontal=24.dp)){Eyebrow("${product.brand.uppercase()} · ${product.category}");Text("购买 ${product.name}",Modifier.padding(top=8.dp),style=MaterialTheme.typography.headlineLarge);Text(product.subtitle,Modifier.padding(top=9.dp),style=MaterialTheme.typography.bodyLarge,color=MaterialTheme.colorScheme.onSurfaceVariant)}}
            item{ProductGallery(product,Modifier.padding(horizontal=20.dp))}
            item{Column(Modifier.padding(horizontal=24.dp)){Text("商品详情",style=MaterialTheme.typography.headlineSmall);Text(product.description,Modifier.padding(top=13.dp),style=MaterialTheme.typography.bodyLarge,color=MaterialTheme.colorScheme.onSurfaceVariant)}}
            item{ServerAssetImage(product.photos.lastOrNull(),"${product.name} 细节",Modifier.padding(horizontal=20.dp).fillMaxWidth().height(260.dp).clip(RoundedCornerShape(24.dp)))}
            item{Column(Modifier.padding(horizontal=20.dp)){LabCard{Text("包装内容",style=MaterialTheme.typography.titleLarge);product.contents.forEach{Text(it,Modifier.padding(top=12.dp),style=MaterialTheme.typography.bodyLarge)};if(product.deliveryCredits>0)Text("附带 ${product.deliveryCredits} 信用点",Modifier.padding(top=12.dp),style=MaterialTheme.typography.bodyLarge);ShopPriceLabel(product,modifier=Modifier.padding(vertical=14.dp));Row(Modifier.fillMaxWidth(),verticalAlignment=Alignment.CenterVertically){Text("数量",Modifier.weight(1f));QuantityControl(quantity,{quantity=it;added=false},max=minOf(product.stock,product.limit).coerceAtLeast(1))}}}}
            if(product.purchaseLimits.isNotEmpty())item{Column(Modifier.padding(horizontal=20.dp)){LabCard{Text("购买说明",style=MaterialTheme.typography.titleLarge);product.purchaseLimits.forEach{Text(it,Modifier.padding(top=12.dp),style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)};Text("刷新时间为北京时间。",Modifier.padding(top=10.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)}}}
            item{Column(Modifier.padding(horizontal=20.dp)){LabCard{Icon(Icons.Outlined.LocalShipping,null,tint=MaterialTheme.colorScheme.primary);Text(product.estimatedDelivery,Modifier.padding(top=12.dp),style=MaterialTheme.typography.titleLarge);Text(product.deliverySummary,Modifier.padding(top=10.dp),style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant);DetailRow("领取账户",state.userName);DetailRow("退款规则","未领取可申请退款");DetailRow("交付方式","游戏内邮箱")}}}
        }
        Surface(Modifier.align(Alignment.BottomCenter).fillMaxWidth(),color=MaterialTheme.colorScheme.surface.copy(alpha=.97f)) {
            Row(Modifier.navigationBarsPadding().padding(18.dp),verticalAlignment=Alignment.CenterVertically,horizontalArrangement=Arrangement.spacedBy(14.dp)) {
                Column(Modifier.weight(1f)){Text(credit(product.price*quantity),style=MaterialTheme.typography.titleLarge);Text(if(added&&product.id in state.cart)"已加入购物袋" else "信用点",style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)}
                MotionButton({if(added&&product.id in state.cart)onBag() else if(!adding)scope.launch{adding=true;if(state.commerce.network?.setCart(product.id,((state.cart[product.id] ?: 0)+quantity).coerceAtMost(product.limit))==true){added=true;onAdded(product,origin)};adding=false}},Modifier.height(50.dp).onGloballyPositioned{origin=it.positionInRoot()+Offset(it.size.width/2f,it.size.height/2f)},enabled=!adding&&product.stock>0){Text(if(product.stock<=0)"已售罄" else if(adding)"正在加入…" else if(added&&product.id in state.cart)"查看购物袋" else "加入购物袋")}
            }
        }
    }
}
@Composable
fun QuantityControl(count:Int,onChange:(Int)->Unit,max:Int=9) {
    var editing by remember{mutableStateOf(false)};var value by remember{mutableStateOf(count.toString())}
    Row(Modifier.clip(CircleShape).background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha=.55f)),verticalAlignment=Alignment.CenterVertically){IconButton({onChange(count-1)},enabled=count>1){Icon(Icons.Outlined.Remove,"减少数量",Modifier.size(18.dp))};Text(count.toString(),Modifier.widthIn(min=24.dp).clickable{value=count.toString();editing=true}.padding(vertical=10.dp),textAlign=androidx.compose.ui.text.style.TextAlign.Center);IconButton({onChange(count+1)},enabled=count<max){Icon(Icons.Outlined.Add,"增加数量",Modifier.size(18.dp))}}
    if(editing)IosDialog({editing=false},{Text("购买数量")},{Column{RefinedField(value,{value=it.filter(Char::isDigit).take(3)},Modifier.fillMaxWidth(),singleLine=true,keyboardOptions=androidx.compose.foundation.text.KeyboardOptions(keyboardType=androidx.compose.ui.text.input.KeyboardType.Number));Text("可填写 1–$max",Modifier.padding(top=8.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)}},{PlainButton({onChange(value.toInt());editing=false},enabled=value.toIntOrNull() in 1..max){Text("确定")}}, {PlainButton({editing=false}){Text("取消")}})
}
@Composable
fun ShoppingBag(state:LabState,topInset:Dp,onClose:()->Unit,onOrder:(String)->Unit) {
    val scope=rememberCoroutineScope()
    LaunchedEffect(Unit){state.commerce.network?.refreshCart()}
    val items=ShopCatalog.mapNotNull{p->state.cart[p.id]?.takeIf{it>0}?.let{p to it}}
    val unavailable=state.cart.keys.filter{id->ShopCatalog.none{it.id==id}}
    var quoteKey by rememberSaveable{mutableStateOf(UUID.randomUUID().toString())}
    LaunchedEffect(state.cart.toMap()){quoteKey=UUID.randomUUID().toString()}
    var quoteText by rememberSaveable{mutableStateOf<String?>(null)};var quoting by remember{mutableStateOf(false)};var confirmQuote by rememberSaveable{mutableStateOf(false)}
    var pending by rememberSaveable{mutableStateOf(hashMapOf<String,Int>())};var orderId by rememberSaveable{mutableStateOf<String?>(null)};var paymentKey by rememberSaveable{mutableStateOf("")};var error by remember{mutableStateOf<String?>(null)}
    val cartFingerprint=items.map{Triple(it.first.id,it.second,it.first.version)}
    var quotedFingerprint by remember{mutableStateOf<List<Triple<String,Int,Long>>>(emptyList())}
    LaunchedEffect(quoteKey,cartFingerprint){
        if(pending.isNotEmpty()||confirmQuote)return@LaunchedEffect
        quoteText=null;error=null
        if(items.isEmpty())return@LaunchedEffect
        quoting=true;delay(200)
        val quote=state.commerce.network?.quoteStore(items,quoteKey)
        if(!currentCoroutineContext().isActive)return@LaunchedEffect
        quoteText=quote?.toString();quotedFingerprint=cartFingerprint;if(quote==null)error=state.commerce.network?.error;quoting=false
    }
    val summary=quoteText?.takeIf{quotedFingerprint==cartFingerprint}?.let{runCatching{checkoutSummary(JSONObject(it))}.getOrNull()}
    Box(Modifier.fillMaxSize()){
        LazyColumn(contentPadding=PaddingValues(start=20.dp,end=20.dp,top=topInset,bottom=150.dp),verticalArrangement=Arrangement.spacedBy(18.dp)){
            item{Text(if(items.isEmpty()&&unavailable.isEmpty())"购物袋还是空的" else "你的购物袋",style=MaterialTheme.typography.headlineLarge,modifier=Modifier.padding(vertical=8.dp))}
            if(items.isEmpty()&&unavailable.isEmpty())item{Text("添加喜欢的商品后，在这里完成付款。",color=MaterialTheme.colorScheme.onSurfaceVariant);PlainButton(onClose){Text("继续选购")}}
            items(unavailable,key={"unavailable-$it"}){id->LabCard{Text("该商品暂时无法购买",style=MaterialTheme.typography.titleMedium);Text("商品可能已下架或暂不可用。",Modifier.padding(top=8.dp),color=MaterialTheme.colorScheme.onSurfaceVariant);PlainButton({scope.launch{state.commerce.network?.setCart(id,0)}}){Text("移出购物袋")}}}
            items(items,key={it.first.id}){(product,count)->LabCard{
                Row(verticalAlignment=Alignment.CenterVertically){ServerAssetImage(product.photos.firstOrNull(),product.name,Modifier.size(80.dp,100.dp).clip(RoundedCornerShape(15.dp)),scale=ContentScale.Crop);Column(Modifier.weight(1f).padding(start=14.dp)){Text(product.name,style=MaterialTheme.typography.titleMedium);ShopPriceLabel(product,modifier=Modifier.padding(top=6.dp))};IconButton({scope.launch{state.commerce.network?.setCart(product.id,0);error=state.commerce.network?.error}}){Icon(Icons.Outlined.Close,"移除${product.name}",Modifier.size(18.dp))}}
                Row(Modifier.fillMaxWidth().padding(top=16.dp),verticalAlignment=Alignment.CenterVertically,horizontalArrangement=Arrangement.SpaceBetween){Text(credit(product.price*count),style=MaterialTheme.typography.titleMedium);QuantityControl(count,{number->scope.launch{state.commerce.network?.setCart(product.id,number);error=state.commerce.network?.error}},max=product.limit)}
            }}
            if(items.isNotEmpty())item{LabCard{DetailRow("交付方式","游戏内邮箱");DetailRow("领取账户",state.userName);DetailRow("付款方式","钱包余额");DetailRow("可用余额",if(state.balanceKnown)credit(state.balance) else "正在同步");
                if(summary!=null){if(summary.originalTotal>summary.total)DetailRow("商品原价","${credit(summary.originalTotal)}");if(summary.productDiscount>0)DetailRow("商品优惠","−${credit(summary.productDiscount)}");if(summary.couponDiscount>0){DetailRow("优惠券","−${credit(summary.couponDiscount)}");Text("${summary.couponName ?: "已为你选择优惠"} · 已自动择优",Modifier.padding(top=8.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.primary)};DetailRow("实付合计","${credit(summary.total)} 信用点")}
                else Text(if(quoting)"正在计算优惠…" else "重新确认价格后即可结算",Modifier.padding(top=12.dp),color=MaterialTheme.colorScheme.onSurfaceVariant)
                error?.let{Text(it,Modifier.padding(top=12.dp),color=MaterialTheme.colorScheme.error);PlainButton({scope.launch{state.commerce.network?.refreshStore();quoteKey=UUID.randomUUID().toString()}}){Text("刷新商品与优惠")}}
            }}
        }
        if(items.isNotEmpty())Surface(Modifier.align(Alignment.BottomCenter).fillMaxWidth(),color=MaterialTheme.colorScheme.surface){MotionButton({if(!quoting)scope.launch{
            quoting=true
            val fresh=state.commerce.network?.quoteStore(items,UUID.randomUUID().toString())
            val current=ShopCatalog.mapNotNull{p->state.cart[p.id]?.takeIf{it>0}?.let{Triple(p.id,it,p.version)}}
            if(current!=cartFingerprint){error="购物袋已变化，请重新确认"}
            else if(fresh==null){error=state.commerce.network?.error}
            else{val prices=checkoutSummary(fresh);quoteText=fresh.toString();quotedFingerprint=cartFingerprint
                if(!state.balanceKnown&&prices.total!=0L)error="余额尚未同步，请先刷新钱包"
                else if(prices.total>state.balance)error="余额不足，请调整商品数量"
                else{error=null;paymentKey=UUID.randomUUID().toString();confirmQuote=true}}
            quoting=false
        }},Modifier.navigationBarsPadding().padding(18.dp).fillMaxWidth().height(52.dp),enabled=!quoting&&summary!=null&&state.commerce.network?.cartBusy!=true){Text(if(quoting)"正在计算优惠…" else if(summary!=null)"去结算 ${credit(summary.total)}" else "等待确认价格")}}
    }
    if(confirmQuote&&quoteText!=null)CheckoutQuoteConfirmation(JSONObject(quoteText!!),{confirmQuote=false;quoteKey=UUID.randomUUID().toString()}){confirmQuote=false;val lines=JSONObject(quoteText!!).getJSONArray("items");pending=HashMap((0 until lines.length()).associate{val item=lines.getJSONObject(it);item.getString("productId") to item.getInt("quantity")})}
    if(pending.isNotEmpty()&&quoteText!=null) {
        val quote=JSONObject(quoteText!!)
        PaymentExperience(apiCents(quote.getString("totalAmount")),orderId?.let{state.commerce.order(it)?.seller} ?: quote.optString("storeName","官方商城"),"支付成功",onCommit={orderId=state.commerce.network?.createOrder(quote,paymentKey);orderId!=null},autoCloseOnSuccess=true,errorMessage=state.storageMessage){pending=hashMapOf();quoteKey=UUID.randomUUID().toString();orderId?.let(onOrder)}
    }
}
@Composable
fun CheckoutQuoteConfirmation(quote:JSONObject,onClose:()->Unit,onConfirm:()->Unit) {
    val summary=remember(quote.toString()){checkoutSummary(quote)}
    IosDialog(onClose,{Text("确认付款")},{Column{Text("是否支付 ${credit(summary.total)} 信用点？");if(summary.originalTotal>summary.total)Text("原价 ${credit(summary.originalTotal)}",Modifier.padding(top=8.dp),textDecoration=TextDecoration.LineThrough,color=MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha=.5f),style=MaterialTheme.typography.bodySmall);summary.couponName?.let{Text("已使用：$it",Modifier.padding(top=8.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)}}},
        {PlainButton(onConfirm){Text("确认付款")}}, {PlainButton(onClose){Text("取消")}})
}
