package com.deuterium.app.uilab

import androidx.compose.animation.*
import androidx.compose.animation.core.tween
import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
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
import org.json.JSONObject
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.util.UUID

@Composable
internal fun SakiPurchaseCheckout(state:LabState,selected:JSONObject,onClose:()->Unit){
    val ai=state.ai ?: return
    var quoteRaw by rememberSaveable{mutableStateOf<String?>(null)}
    var paying by rememberSaveable{mutableStateOf(false)}
    var requestId by rememberSaveable{mutableStateOf("")}
    var attempt by remember{mutableIntStateOf(0)}
    var loading by remember{mutableStateOf(false)}
    var failure by remember{mutableStateOf<String?>(null)}
    var expired by remember{mutableStateOf(false)}
    val motion=LocalMotion.current
    LaunchedEffect(attempt){
        if(quoteRaw==null&&!paying){
            loading=true;failure=null
            try{quoteRaw=ai.purchaseQuote(selected).toString()}
            catch(error:Exception){if(error is kotlinx.coroutines.CancellationException)throw error;failure=error.message ?: "暂时无法读取价格，请重试"}
            finally{loading=false}
        }
    }
    val quote=remember(quoteRaw){quoteRaw?.let(::JSONObject)}
    LaunchedEffect(quoteRaw,paying){
        if(quote!=null&&!paying){
            val expiry=Instant.parse(quote.getString("expiresAt"))
            while(!paying){expired=!Instant.now().isBefore(expiry);if(expired)break;delay(1000)}
        }
    }
    val plan=quote?.getJSONObject("plan") ?: selected
    val upgrade=quote?.optString("kind")=="UPGRADE"
    val renewal=quote?.optString("kind")=="RENEWAL"
    fun reload(){quoteRaw=null;expired=false;attempt++}
    IosOverlayHost(onDismissRequest={if(!paying)onClose()}){
        AnimatedContent(paying,Modifier.fillMaxSize(),contentAlignment=Alignment.BottomCenter,
            transitionSpec={
                (fadeIn(tween(if(motion)240 else 0,if(motion)60 else 0))+slideInVertically(tween(if(motion)320 else 0)){it/18}) togetherWith
                    (fadeOut(tween(if(motion)150 else 0))+slideOutVertically(tween(if(motion)220 else 0)){it/24})
            },label="saki-checkout") { inPayment ->
            if(inPayment&&quote!=null){
                PaymentExperience(apiCents(quote.getString("totalAmount")),"Saki AI · ${plan.getString("name")}",if(upgrade)"套餐已升级" else if(renewal)"套餐已续购" else "套餐已开通",
                    {ai.purchase(plan,requestId,quote.getString("quoteId"))},embedded=true,errorMessage=ai.purchaseError,onClose=onClose)
            }else{
                BoxWithConstraints(Modifier.fillMaxSize().statusBarsPadding()){
                    Box(Modifier.matchParentSize().clickable(indication=null,interactionSource=remember{androidx.compose.foundation.interaction.MutableInteractionSource()}){if(!paying)onClose()})
                    SoftGlassSurface(Modifier.align(Alignment.BottomCenter).fillMaxWidth().heightIn(max=maxHeight*.96f),radius=30.dp,tint=MaterialTheme.colorScheme.background){
                        Column(Modifier.navigationBarsPadding().verticalScroll(rememberScrollState()).padding(horizontal=20.dp).padding(top=14.dp,bottom=22.dp),verticalArrangement=Arrangement.spacedBy(16.dp)){
                            Row(Modifier.fillMaxWidth(),verticalAlignment=Alignment.CenterVertically){
                                Text("Saki AI",Modifier.weight(1f),fontSize=20.sp,fontWeight=FontWeight.SemiBold)
                                PlainButton({if(!paying)onClose()},Modifier.size(44.dp),enabled=!paying){Icon(Icons.Outlined.Close,"取消",Modifier.size(19.dp),tint=MaterialTheme.colorScheme.onSurfaceVariant)}
                            }
                            Surface(shape=RoundedCornerShape(19.dp),color=MaterialTheme.colorScheme.surface){
                                Column{
                                    Row(Modifier.padding(17.dp),verticalAlignment=Alignment.CenterVertically,horizontalArrangement=Arrangement.spacedBy(14.dp)){
                                        Image(painterResource(R.drawable.xiaoxiang_avatar),null,Modifier.size(54.dp).clip(RoundedCornerShape(14.dp)))
                                        Column(Modifier.weight(1f),verticalArrangement=Arrangement.spacedBy(4.dp)){
                                            Text(plan.getString("name"),style=MaterialTheme.typography.titleMedium)
                                            Text("${plan.getInt("quotaPerWindow")} 次 / ${plan.getInt("windowHours")} 小时",style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
                                            Text(if(upgrade)"套餐升级" else if(renewal)"套餐续购" else "Saki AI 服务",style=MaterialTheme.typography.labelSmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
                                        }
                                    }
                                    HorizontalDivider(Modifier.padding(start=17.dp),color=MaterialTheme.colorScheme.outlineVariant.copy(alpha=.4f))
                                    Column(Modifier.padding(17.dp),verticalArrangement=Arrangement.spacedBy(7.dp)){
                                        Text(if(upgrade)"本次补差价" else "本次付款",style=MaterialTheme.typography.labelMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)
                                        if(quote==null)Text(if(loading)"正在读取价格…" else "价格暂不可用",fontSize=22.sp,fontWeight=FontWeight.SemiBold)
                                        else Row(verticalAlignment=Alignment.Bottom,horizontalArrangement=Arrangement.spacedBy(7.dp)){
                                            Box(Modifier.weight(1f,false)){AdaptiveMoney(credit(apiCents(quote.getString("totalAmount"))),size=29.sp)}
                                            Text("信用点",Modifier.padding(bottom=4.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
                                        }
                                        if(!upgrade)Text("${plan.getInt("durationDays")} 天使用权益",style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
                                    }
                                    HorizontalDivider(Modifier.padding(start=17.dp),color=MaterialTheme.colorScheme.outlineVariant.copy(alpha=.4f))
                                    Column(Modifier.padding(17.dp),verticalArrangement=Arrangement.spacedBy(8.dp)){
                                        if(upgrade&&quote!=null){
                                            val date=Instant.parse(quote.getString("entitlementExpiresAt")).atZone(ZoneId.of("Asia/Shanghai")).format(DateTimeFormatter.ofPattern("yyyy年M月d日 HH:mm"))
                                            Text("有效期至 $date",style=MaterialTheme.typography.bodyMedium,fontWeight=FontWeight.Medium)
                                            Text("升级立即生效，到期时间不变。",style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
                                        }else Text(if(renewal)"付款后延长当前套餐有效期。" else "付款后立即生效，无需领取。",style=MaterialTheme.typography.bodySmall)
                                        Text("不会自动续费，不支持降级与退款。",style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
                                    }
                                    HorizontalDivider(Modifier.padding(start=17.dp),color=MaterialTheme.colorScheme.outlineVariant.copy(alpha=.4f))
                                    Text("购买账号：${state.userName}",Modifier.padding(17.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
                                }
                            }
                            if(expired||failure!=null)Text(if(expired)"价格确认已过期，请刷新后再次确认。" else failure.orEmpty(),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.error)
                            MotionButton({
                                if(expired||failure!=null)reload()
                                else if(quote!=null&&!paying){
                                    if(!Instant.now().isBefore(Instant.parse(quote.getString("expiresAt"))))expired=true
                                    else{requestId=UUID.randomUUID().toString();paying=true}
                                }
                            },Modifier.fillMaxWidth().heightIn(min=54.dp),enabled=!paying&&!loading&&(quote!=null||failure!=null)){
                                Text(if(loading)"正在读取…" else if(expired||failure!=null)"重新读取价格" else "确认购买",fontSize=17.sp)
                            }
                            Text("使用信用点余额付款",Modifier.fillMaxWidth(),textAlign=TextAlign.Center,style=MaterialTheme.typography.labelSmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
                        }
                    }
                }
            }
        }
    }
}
