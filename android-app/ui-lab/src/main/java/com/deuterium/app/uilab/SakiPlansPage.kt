package com.deuterium.app.uilab

import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.*
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import org.json.JSONObject
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter

@Composable
fun SakiPlansPage(state:LabState,topInset:Dp){
    val ai=state.ai ?: return
    val scope=rememberCoroutineScope()
    var selected by rememberSaveable{mutableStateOf("")}
    var checkout by rememberSaveable{mutableStateOf<String?>(null)}
    var checking by remember{mutableStateOf(false)}
    LaunchedEffect(Unit){ai.refresh();ai.loadPlans();if(ai.purchasePending)ai.recoverPurchase()}
    LaunchedEffect(ai.purchasePending){while(ai.purchasePending){delay(5000);ai.recoverPurchase()}}
    val plan=ai.plans.find{it.optString("planId")==selected}
    val switching=ai.expiresAt!=null&&plan!=null&&ai.currentPlan?.optString("planId")!=plan.getString("planId")
    val downgrade=switching&&ai.plans.indexOfFirst{it.optString("planId")==selected}<=ai.plans.indexOfFirst{it.optString("planId")==ai.currentPlan?.optString("planId")}
    val available=plan?.optBoolean("purchasable")==true&&!downgrade&&!ai.purchasePending
    Box(Modifier.fillMaxSize()){
        LazyColumn(contentPadding=PaddingValues(start=22.dp,end=22.dp,top=topInset,bottom=245.dp),verticalArrangement=Arrangement.spacedBy(18.dp)){
            item{Column(Modifier.fillMaxWidth().padding(top=10.dp,bottom=8.dp),horizontalAlignment=Alignment.CenterHorizontally){
                Image(painterResource(R.drawable.xiaoxiang_avatar),"小祥",Modifier.size(86.dp).clip(RoundedCornerShape(25.dp)))
                Text("Saki AI",Modifier.padding(top=18.dp),fontSize=32.sp,fontWeight=FontWeight.Bold)
                Text("让小祥，陪你多聊一点。",Modifier.padding(top=9.dp),style=MaterialTheme.typography.titleMedium)
                Text("选择适合你的额度，探索更多想法。",Modifier.padding(top=8.dp),style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)
            }}
            item{LabCard{
                Text("当前套餐 · ${ai.currentPlan?.optString("name") ?: "正在读取"}",style=MaterialTheme.typography.titleMedium)
                Text(ai.quotaText,Modifier.padding(top=8.dp),style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)
                ai.expiresAt?.let{value->Text("有效期至 "+Instant.parse(value).atZone(ZoneId.of("Asia/Shanghai")).format(DateTimeFormatter.ofPattern("yyyy年MM月dd日 HH:mm")),Modifier.padding(top=8.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)}
            }}
            if(ai.plans.isNotEmpty())item{Text("选择套餐",style=MaterialTheme.typography.titleLarge)}
            ai.plans.filter{it.optString("code")!="free"}.forEach{p->item(key=p.getString("planId")){
                val chosen=selected==p.getString("planId");val purchasable=p.optBoolean("purchasable")
                Surface(onClick={if(checkout==null&&!ai.purchasePending)selected=p.getString("planId")},shape=RoundedCornerShape(23.dp),color=if(chosen)MaterialTheme.colorScheme.primaryContainer.copy(alpha=.45f)else MaterialTheme.colorScheme.surface,border=BorderStroke(if(chosen)2.dp else 1.dp,if(chosen)MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.outlineVariant.copy(alpha=.45f))){
                    Column(Modifier.fillMaxWidth().padding(20.dp),verticalArrangement=Arrangement.spacedBy(12.dp)){
                        Row(verticalAlignment=Alignment.CenterVertically){Column(Modifier.weight(1f)){Text(p.getString("name"),style=MaterialTheme.typography.titleLarge);Text("${p.getInt("quotaPerWindow")} 次 / ${p.getInt("windowHours")} 小时",Modifier.padding(top=5.dp),color=MaterialTheme.colorScheme.onSurfaceVariant,style=MaterialTheme.typography.bodyMedium)};Icon(if(chosen)Icons.Outlined.CheckCircle else Icons.Outlined.RadioButtonUnchecked,null,tint=if(chosen)MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.outlineVariant)}
                        if(p.optString("description").isNotBlank())Text(p.getString("description"),style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)
                        AdaptiveMoney(credit(apiCents(p.getString("price"))),size=25.sp)
                        Text("信用点 / ${p.getInt("durationDays")} 天"+(if(!purchasable)" · 暂未开放" else ""),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                }
            }}
            item{Text("付款后立即生效，不会自动续费。支持补差价升级，到期时间不变；同套餐续购可延长有效期。不支持降级与退款。",style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant,lineHeight=20.sp)}
            if(ai.plans.isEmpty())item{SecondaryButton({scope.launch{ai.loadPlans()}}){Text("重新读取套餐")}}
        }
        Surface(Modifier.align(Alignment.BottomCenter).fillMaxWidth(),color=MaterialTheme.colorScheme.surface,shadowElevation=5.dp){Column(Modifier.navigationBarsPadding().padding(horizontal=22.dp,vertical=18.dp),verticalArrangement=Arrangement.spacedBy(10.dp)){
            ai.purchaseError?.let{Text(it,color=MaterialTheme.colorScheme.error,style=MaterialTheme.typography.bodySmall)}
            if(ai.purchasePending)MotionButton({scope.launch{checking=true;ai.recoverPurchase();checking=false}},Modifier.fillMaxWidth().heightIn(min=52.dp),enabled=!checking){Text(if(checking)"正在查看…" else "查看开通进度")}
            else MotionButton({checkout=plan?.toString()},Modifier.fillMaxWidth().heightIn(min=52.dp),enabled=available){Text(if(available)"${if(switching)"升级至" else if(ai.expiresAt!=null)"续购" else "购买"} ${plan!!.getString("name")}" else if(downgrade)"不支持降级" else if(plan==null)"选择一个套餐" else "暂未开放购买")}
            Text("使用信用点付款 · 一次购买，按期使用",Modifier.fillMaxWidth(),textAlign=TextAlign.Center,style=MaterialTheme.typography.labelSmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
        }}
    }
    checkout?.let{raw->SakiPurchaseCheckout(state,JSONObject(raw),onClose={checkout=null;scope.launch{ai.refresh();ai.loadPlans()}})}
}

@Composable
fun SakiOrderDetail(book:CommerceBook,order:CommerceOrder,topInset:Dp,onChat:()->Unit,onDeleted:()->Unit){
    var deleting by remember{mutableStateOf(false)}
    val format=DateTimeFormatter.ofPattern("yyyy年MM月dd日 HH:mm")
    Box(Modifier.fillMaxSize()){
    LazyColumn(contentPadding=PaddingValues(start=22.dp,end=22.dp,top=topInset,bottom=145.dp),verticalArrangement=Arrangement.spacedBy(22.dp)){
        item{Column(Modifier.fillMaxWidth().padding(vertical=16.dp),horizontalAlignment=Alignment.CenterHorizontally){
            Image(painterResource(R.drawable.xiaoxiang_avatar),"Saki AI",Modifier.size(76.dp).clip(RoundedCornerShape(23.dp)))
            Text(order.status,Modifier.padding(top=18.dp),style=MaterialTheme.typography.headlineSmall)
            Text(if(order.fundsStatus=="SETTLED")"套餐已生效，和小祥继续聊天吧。" else if(order.fundsStatus=="REFUNDED")"套餐未能开通，信用点已退回。" else "购买进度会在这里更新。",Modifier.padding(top=9.dp),style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)
        }}
        item{LabCard{Text("Saki AI",style=MaterialTheme.typography.titleMedium);order.lines.forEach{Text(it.title,Modifier.padding(top=14.dp),style=MaterialTheme.typography.titleLarge);Text(it.subtitle,Modifier.padding(top=8.dp),color=MaterialTheme.colorScheme.onSurfaceVariant)};DetailRow("实付金额","${credit(order.amount)} 信用点");order.aiExpiresAt?.let{DetailRow("有效期至",it.format(format))}}}
        item{LabCard{Text("订单信息",style=MaterialTheme.typography.titleMedium);DetailRow("订单编号",order.id);DetailRow("购买时间",order.createdAt.format(format));DetailRow("付款方式","信用点余额");DetailRow("交付方式","自动开通");DetailRow("续费方式","手动续购");Text("即时开通服务，不支持退款。",Modifier.padding(top=16.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)}}
        if(order.canHideRecord)item{DeleteRecordButton({deleting=true},Modifier.fillMaxWidth())}
    }
    Surface(Modifier.align(Alignment.BottomCenter).fillMaxWidth(),color=MaterialTheme.colorScheme.surface){MotionButton(onChat,Modifier.navigationBarsPadding().padding(22.dp).fillMaxWidth().heightIn(min=52.dp)){Text("联系卖家 · 小祥")}}
    }
    if(deleting)DeleteRecordDialog(order.lines.firstOrNull()?.title ?: "Saki AI 订单",{deleting=false}){(book.network?.hideOrder(order.id)==true).also{if(it)onDeleted()}}
}
