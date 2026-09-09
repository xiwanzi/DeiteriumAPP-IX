package com.deuterium.app.uilab

import android.net.Uri
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.PickVisualMediaRequest
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.*
import androidx.compose.foundation.lazy.grid.*
import androidx.compose.foundation.pager.*
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.foundation.shape.*
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.*
import kotlinx.coroutines.*
import java.util.UUID
import org.json.JSONObject

@Composable
fun MarketPage(book:CommerceBook,query:String,topInset:Dp,onProduct:(String)->Unit,ownOnly:Boolean=false,onPublish:()->Unit={}) {
    var deleting by remember{mutableStateOf<MarketListing?>(null)}
    var category by rememberSaveable{mutableStateOf("全部")}
    LaunchedEffect(Unit){book.network?.refreshMarket()}
    val listings=book.listings.filter{(if(ownOnly)it.seller==book.userName else it.active&&it.stock>0)&&marketCategoryMatches(it.category,category)&&(query.isBlank()||it.title.contains(query,true)||it.category.contains(query,true)||it.seller.contains(query,true))}
    LazyVerticalGrid(columns=GridCells.Fixed(2),contentPadding=PaddingValues(start=14.dp,end=14.dp,top=topInset,bottom=115.dp),horizontalArrangement=Arrangement.spacedBy(10.dp),verticalArrangement=Arrangement.spacedBy(13.dp)) {
        item(span={GridItemSpan(2)}){MarketCategoryBar(category,{category=it})}
        if(category.contains(" · "))item(span={GridItemSpan(2)}){Text(category,style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)}
        book.network?.marketError?.let{error->item(span={GridItemSpan(2)}){Text(error,color=MaterialTheme.colorScheme.error)}}
        if(listings.isEmpty())item(span={GridItemSpan(2)}){Column(Modifier.fillMaxWidth().padding(vertical=48.dp),horizontalAlignment=Alignment.CenterHorizontally){Text(if(ownOnly)"还没有发布商品" else "这个分类暂时没有商品",style=MaterialTheme.typography.titleMedium);if(ownOnly)PlainButton(onPublish){Text("发布商品")}}}
        items(listings,key={it.id}){listing->Surface(onClick={onProduct(listing.id)},shape=RoundedCornerShape(18.dp),color=MaterialTheme.colorScheme.surface){Column {
            Box{ListingImage(listing,Modifier.fillMaxWidth().aspectRatio(1.02f));if(listing.construction)Box(Modifier.align(Alignment.TopStart).padding(8.dp).background(MaterialTheme.colorScheme.primary,RoundedCornerShape(6.dp)).padding(horizontal=7.dp,vertical=3.dp)){Text("建筑服务",color=MaterialTheme.colorScheme.onPrimary,style=MaterialTheme.typography.labelSmall)}}
            Column(Modifier.padding(11.dp)){
                Text(listing.title,style=MaterialTheme.typography.bodyMedium,maxLines=2,overflow=TextOverflow.Ellipsis)
                Text(credit(listing.price),Modifier.padding(top=5.dp),fontSize=18.sp,lineHeight=25.sp,fontWeight=androidx.compose.ui.text.font.FontWeight.SemiBold,color=MaterialTheme.colorScheme.primary)
                Row(Modifier.padding(top=8.dp),verticalAlignment=Alignment.CenterVertically){PlayerAvatar(listing.seller,Modifier.size(21.dp));Text(listing.seller,Modifier.weight(1f).padding(start=6.dp),style=MaterialTheme.typography.bodySmall,maxLines=1);Text(if(!listing.active)"已下架" else if(listing.stock==0)"售罄" else "余${listing.stock}",style=MaterialTheme.typography.labelSmall,color=MaterialTheme.colorScheme.onSurfaceVariant)}
                if(ownOnly&&listing.canHideRecord)DeleteRecordButton({deleting=listing},Modifier.align(Alignment.End))
            }
        }}}
    }
    deleting?.let{listing->DeleteRecordDialog(listing.title,{deleting=null}){book.network?.hideListing(listing.id)==true}}
}

@Composable
fun MarketProductPage(book:CommerceBook,listingId:String,topInset:Dp,onChat:(String)->Unit,onOrder:(String)->Unit,onEdit:(String)->Unit={}) {
    LaunchedEffect(listingId){book.network?.refreshListing(listingId)}
    val listing=book.listings.find{it.id==listingId} ?: return
    var quantity by rememberSaveable{mutableIntStateOf(1)};var methodName by rememberSaveable{mutableStateOf<String?>(null)}
    val method=if(listing.construction)DeliveryMethod.Worksite else methodName?.let{DeliveryMethod.valueOf(it)}
    var location by rememberSaveable{mutableStateOf("")};var project by rememberSaveable{mutableStateOf("")};var error by remember{mutableStateOf<String?>(null)}
    var quoteKey by rememberSaveable{mutableStateOf(UUID.randomUUID().toString())}
    LaunchedEffect(listingId,quantity,methodName,location,project){quoteKey=UUID.randomUUID().toString()}
    var quoteText by rememberSaveable{mutableStateOf<String?>(null)};var quoting by remember{mutableStateOf(false)};var confirmQuote by rememberSaveable{mutableStateOf(false)}
    var pending by rememberSaveable{mutableStateOf(false)};var key by rememberSaveable{mutableStateOf("")};var orderId by rememberSaveable{mutableStateOf<String?>(null)}
    var tools by remember{mutableStateOf(false)};var unlist by remember{mutableStateOf(false)}
    val list=rememberLazyListState();val scope=rememberCoroutineScope();val own=listing.seller==book.userName
    Box(Modifier.fillMaxSize().imePadding()) {
        LazyColumn(state=list,contentPadding=PaddingValues(start=20.dp,end=20.dp,top=topInset,bottom=140.dp),verticalArrangement=Arrangement.spacedBy(22.dp)) {
            item{Column{Eyebrow(listing.category);Text(listing.title,Modifier.padding(top=8.dp),style=MaterialTheme.typography.headlineLarge);Text(listing.subtitle,Modifier.padding(top=9.dp),style=MaterialTheme.typography.bodyLarge,color=MaterialTheme.colorScheme.onSurfaceVariant)}}
            item{ListingGallery(listing)}
            item{Column{Text(if(listing.construction)"服务详情" else "商品详情",style=MaterialTheme.typography.titleLarge);Text(listing.description,Modifier.padding(top=12.dp),style=MaterialTheme.typography.bodyLarge,color=MaterialTheme.colorScheme.onSurfaceVariant)}}
            item{LabCard{
                Row(verticalAlignment=Alignment.CenterVertically){PlayerAvatar(listing.seller);Column(Modifier.weight(1f).padding(start=12.dp)){Text(listing.seller,style=MaterialTheme.typography.titleMedium);Text("QQ ${listing.qq}",style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)};if(!own)PlainButton({onChat(listing.seller)}){Text("私聊")}}
                DetailRow(if(listing.construction)"可接单数" else "库存","${listing.stock}");DetailRow("价格","${credit(listing.price)} 信用点")
                DetailRow(if(listing.construction)"工期（含验收）" else "自动确认期限",if(listing.construction)durationHours(listing.workHours) else "发货后 72 小时")
            }}
            item{LabCard{
                if(own){Text("商品管理",style=MaterialTheme.typography.titleLarge);DetailRow("当前状态",if(listing.active)"上架中" else "已下架");SecondaryButton({if(listing.active)unlist=true else onEdit(listing.id)},Modifier.fillMaxWidth().padding(top=12.dp)){Text(if(listing.active)"下架商品" else "编辑并重新上架")}}
                else {
                    Row(Modifier.fillMaxWidth(),verticalAlignment=Alignment.CenterVertically){Text("购买数量",Modifier.weight(1f),style=MaterialTheme.typography.titleMedium);QuantityControl(quantity,{quantity=it},listing.stock.coerceIn(1,999))}
                    if(listing.construction){
                        RefinedField(project,{project=it.take(80);error=null},Modifier.fillMaxWidth().padding(top=18.dp),label={Text("建筑项目（必填）")},placeholder={Text("项目名称、用途或施工目标")},maxLines=2)
                        RefinedField(location,{location=it.take(120);error=null},Modifier.fillMaxWidth().padding(top=18.dp),label={Text("工程地点（必填）")},placeholder={Text("世界、坐标与施工范围")},maxLines=3)
                        Text("从卖家开始施工起 ${durationHours(listing.workHours)} 后自动验收。工期包含验收预留时间，请在付款前与卖家确认方案。",Modifier.padding(top=16.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
                    } else {
                        Text("交付方式（必选）",Modifier.padding(top=22.dp,bottom=12.dp),style=MaterialTheme.typography.titleMedium)
                        Row(horizontalArrangement=Arrangement.spacedBy(8.dp)){listing.methods.forEach{option->ChoiceChip(method==option,{methodName=option.name;location=if(option==DeliveryMethod.Pickup)listing.pickupLocation else "";error=null},{Text(option.label)})}}
                        if(method!=null)RefinedField(location,{location=it.take(120);error=null},Modifier.fillMaxWidth().padding(top=18.dp),label={Text(if(method==DeliveryMethod.Door)"收货位置（必填）" else "自取地点（必填）")},placeholder={Text("世界、坐标或建筑名称")},maxLines=3)
                        Text("货款由平台担保。发货后 72 小时自动确认收货；发货后只能申请一次退款，请先与卖家沟通。",Modifier.padding(top=16.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                    error?.let{Text(it,Modifier.padding(top=12.dp),color=MaterialTheme.colorScheme.error)}
                }
            }}
        }
        if(!own)Surface(Modifier.align(Alignment.BottomCenter).fillMaxWidth(),color=MaterialTheme.colorScheme.surface){Row(Modifier.navigationBarsPadding().padding(18.dp),verticalAlignment=Alignment.CenterVertically,horizontalArrangement=Arrangement.spacedBy(12.dp)){
            Column(Modifier.weight(1f)){Text(credit(listing.price*quantity),style=MaterialTheme.typography.titleLarge);Text("信用点",style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)}
            MotionButton({error=when{method==null->"请选择交付方式";listing.construction&&project.isBlank()->"请填写建筑项目";location.isBlank()->"请填写交付地点";listing.price*quantity>book.availableBalance->"余额不足";else->null};if(error!=null)scope.launch{list.animateScrollToItem(4)}else if(!quoting)scope.launch{quoting=true;val quote=book.network?.quoteMarket(listing,quantity,method!!,location,project,quoteKey);if(quote!=null){quoteText=quote.toString();key=UUID.randomUUID().toString();confirmQuote=true}else error=book.network?.error;quoting=false}},Modifier.height(50.dp),enabled=listing.active&&listing.stock>0&&!quoting){Text(if(listing.stock<=0)"已售罄" else if(quoting)"正在准备订单…" else if(listing.construction)"预约服务" else "立即购买")}
        }}
    }
    if(confirmQuote&&quoteText!=null)CheckoutQuoteConfirmation(JSONObject(quoteText!!),{confirmQuote=false}){confirmQuote=false;pending=true}
    if(pending&&quoteText!=null)PaymentExperience(apiCents(JSONObject(quoteText!!).getString("totalAmount")),listing.seller,"支付成功",onCommit={orderId=book.network?.createOrder(JSONObject(quoteText!!),key);orderId!=null},autoCloseOnSuccess=true,errorMessage=book.network?.error){pending=false;quoteKey=UUID.randomUUID().toString();orderId?.let(onOrder)}
    if(unlist)IosDialog({unlist=false},{Text("下架这件商品？")},{Text("下架后其他玩家将无法购买，已经成交的订单仍需正常履约。重新上架时需要检查发布信息。")},{PlainButton({scope.launch{if(book.network?.unlist(listing.id)==true)unlist=false else error=book.network?.error}}){Text("确认下架",color=MaterialTheme.colorScheme.error)}},{PlainButton({unlist=false}){Text("取消")}})

}
fun durationHours(hours:Int)=if(hours%24==0)"${hours/24} 天" else "$hours 小时"

@Composable
fun PublishListingPage(book:CommerceBook,topInset:Dp,onPublished:(String)->Unit,initial:MarketListing?=null) {
    val scope=rememberCoroutineScope();var saving by remember{mutableStateOf(false)}
    var title by rememberSaveable{mutableStateOf(initial?.title.orEmpty())};var subtitle by rememberSaveable{mutableStateOf(initial?.subtitle.orEmpty())};var description by rememberSaveable{mutableStateOf(initial?.description.orEmpty())}
    var category by rememberSaveable{mutableStateOf(initial?.category ?: "建材 · 石材")};var amount by rememberSaveable{mutableStateOf(initial?.let{java.math.BigDecimal(it.price).movePointLeft(2).toPlainString()}.orEmpty())};var stock by rememberSaveable{mutableStateOf(initial?.stock?.toString() ?: "1")}
    var qq by rememberSaveable{mutableStateOf(initial?.qq.orEmpty())};var door by rememberSaveable{mutableStateOf(initial?.methods?.contains(DeliveryMethod.Door) ?: false)};var pickup by rememberSaveable{mutableStateOf(initial?.methods?.contains(DeliveryMethod.Pickup) ?: false)};var location by rememberSaveable{mutableStateOf(initial?.pickupLocation.orEmpty())}
    var photos by rememberSaveable{mutableStateOf(initial?.photos ?: emptyList<String>())};var error by remember{mutableStateOf<String?>(null)};var categories by remember{mutableStateOf(false)}
    var durationUnit by rememberSaveable{mutableIntStateOf(if(initial!=null&&initial.workHours%24!=0)0 else 1)}
    var duration by rememberSaveable{mutableStateOf(initial?.let{if(it.workHours%24==0)(it.workHours/24).toString() else it.workHours.toString()} ?: "7")}
    var uploading by remember{mutableStateOf(false)}
    val listingId=rememberSaveable{UUID.randomUUID().toString()};val construction=category.startsWith("建筑服务")
    Box(Modifier.fillMaxSize().imePadding()){
        LazyColumn(contentPadding=PaddingValues(start=16.dp,end=16.dp,top=topInset,bottom=130.dp),verticalArrangement=Arrangement.spacedBy(18.dp)) {
            item{LabCard{ListingPhotoEditor(photos,{photos=it},{uploading=it})}}
            item{LabCard{
                RefinedField(title,{title=it.take(40)},Modifier.fillMaxWidth(),label={Text("标题（必填）")},placeholder={Text("清楚说明你要出售什么")},singleLine=true)
                RefinedField(subtitle,{subtitle=it.take(60)},Modifier.fillMaxWidth().padding(top=16.dp),label={Text("简介（必填）")},placeholder={Text("一句话概括特色与用途")},maxLines=2)
                RefinedField(description,{description=it.take(1200)},Modifier.fillMaxWidth().padding(top=16.dp),label={Text("详情（必填）")},placeholder={Text("包含内容、品相、交易约定或施工方案")},minLines=4,maxLines=8)
                SettingsRow("分类",Icons.Outlined.Category,detail=category.substringAfter(" · ")){categories=true}
            }}
            item{LabCard{
                RefinedField(amount,{amount=it.take(12)},Modifier.fillMaxWidth(),label={Text("单价（信用点）")},keyboardOptions=KeyboardOptions(keyboardType=KeyboardType.Decimal),singleLine=true)
                RefinedField(stock,{stock=it.filter(Char::isDigit).take(3)},Modifier.fillMaxWidth().padding(top=16.dp),label={Text(if(construction)"可接单数（1–999）" else "库存（1–999）")},keyboardOptions=KeyboardOptions(keyboardType=KeyboardType.Number),singleLine=true)
                RefinedField(qq,{qq=it.filter(Char::isDigit).take(12)},Modifier.fillMaxWidth().padding(top=16.dp),label={Text("卖家 QQ（必填）")},keyboardOptions=KeyboardOptions(keyboardType=KeyboardType.Number),singleLine=true)
            }}
            item{LabCard{
                if(construction){
                    Text("工期与验收",style=MaterialTheme.typography.titleLarge)
                    RefinedField(duration,{duration=it.filter(Char::isDigit).take(4)},Modifier.fillMaxWidth().padding(top=16.dp),label={Text("总工期（必填）")},keyboardOptions=KeyboardOptions(keyboardType=KeyboardType.Number),singleLine=true)
                    SegmentedControl(listOf("小时","天"),durationUnit,{durationUnit=it},Modifier.padding(top=12.dp))
                    Text("请记得预留验收天数",Modifier.padding(top=18.dp),style=MaterialTheme.typography.titleMedium,color=MaterialTheme.colorScheme.primary)
                    Text("计时从你点击“开始施工”起，到期自动验收并结算。请将验收时间包含在工期内，例如施工 5 天、验收 2 天，应填写 7 天。",Modifier.padding(top=8.dp),style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)
                } else {
                    Text("交付方式",style=MaterialTheme.typography.titleLarge)
                    ToggleRow("送货上门","按买家填写的位置交付",door,{door=it});ToggleRow("约定自取","买家到约定地点取货",pickup,{pickup=it})
                    if(pickup)RefinedField(location,{location=it.take(120)},Modifier.fillMaxWidth(),label={Text("自取地点（必填）")},placeholder={Text("世界、坐标或建筑名称")},maxLines=3)
                    Text("发货后 72 小时自动确认收货。退款处理中会暂停计时。",Modifier.padding(top=16.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
                }
                error?.let{Text(it,Modifier.padding(top=14.dp),color=MaterialTheme.colorScheme.error)}
            }}
        }
        Surface(Modifier.align(Alignment.BottomCenter).fillMaxWidth(),color=MaterialTheme.colorScheme.surface){MotionButton({
            val cents=runCatching{amount.toBigDecimal().movePointRight(2).longValueExact()}.getOrNull() ?: 0L
            val methods=if(construction)setOf(DeliveryMethod.Worksite) else buildSet{if(door)add(DeliveryMethod.Door);if(pickup)add(DeliveryMethod.Pickup)}
            val hours=(duration.toIntOrNull() ?: 0)*(if(durationUnit==1)24 else 1)
            val listing=MarketListing(listingId,title.trim(),subtitle.trim(),description.trim(),category,cents,stock.toIntOrNull() ?: 0,book.userName,qq,methods,location.trim(),imageUri=photos.firstOrNull(),imageUris=photos,workHours=hours)
            if(!saving)scope.launch{saving=true;val id=book.network?.publish(listing,initial);if(id!=null)onPublished(id)else error=book.network?.error ?: "发布服务暂不可用";saving=false}
        },Modifier.navigationBarsPadding().padding(18.dp).fillMaxWidth().height(52.dp),enabled=!uploading&&!saving){Text(if(saving)"正在发布…" else if(uploading)"正在上传图片…" else if(initial==null)"发布商品" else "确认重新上架")}}
    }
    if(categories)CategorySheet(category,{categories=false}){category=it;categories=false}
}
