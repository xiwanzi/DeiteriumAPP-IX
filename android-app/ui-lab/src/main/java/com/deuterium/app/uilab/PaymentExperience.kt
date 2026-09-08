package com.deuterium.app.uilab

import androidx.compose.animation.*
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.verticalScroll
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.ErrorOutline
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.*
import androidx.compose.ui.window.*
import kotlinx.coroutines.delay

@Composable
fun PaymentExperience(amount:Long,recipient:String,successTitle:String,onCommit:suspend ()->Boolean,autoCloseOnSuccess:Boolean=false,embedded:Boolean=false,errorMessage:String?=null,confirmed:Boolean?=null,onClose:()->Unit) {
    var result by rememberSaveable { mutableStateOf<Boolean?>(null) }
    var authenticated by rememberSaveable{mutableStateOf(false)}
    val scanUntil by rememberSaveable { mutableLongStateOf(android.os.SystemClock.elapsedRealtime()+kotlin.random.Random.nextLong(1000L,3001L)) }
    var completed by rememberSaveable { mutableStateOf(false) }
    var commitError by rememberSaveable{mutableStateOf<String?>(null)}
    // A recovered success is authoritative, but does not skip the remaining visual scan.
    LaunchedEffect(confirmed){if(confirmed==true)result=true}
    val commit by rememberUpdatedState(onCommit)
    LaunchedEffect(Unit){if(!authenticated){delay((scanUntil-android.os.SystemClock.elapsedRealtime()).coerceAtLeast(0));authenticated=true}}
    // The caller has already confirmed payment. Persist an early response immediately
    // so recreating the page during its scan cannot submit an already-finished payment again.
    LaunchedEffect(Unit){
        if(result==null&&confirmed!=true){
            try{val outcome=commit();if(result!=true)result=outcome}
            catch(cancelled:kotlinx.coroutines.CancellationException){throw cancelled}
            catch(error:Exception){if(result!=true){commitError=error.message ?: "结果待确认，请在钱包或订单中查看";result=false}}
        }
    }
    val visibleResult=if(authenticated)result else null
    val close by rememberUpdatedState(onClose)
    LaunchedEffect(completed,result) { if(completed&&result==true&&autoCloseOnSuccess){delay(850);close()} }
    val paymentContent:@Composable ()->Unit = {
        Box(Modifier.fillMaxSize().padding(22.dp),contentAlignment=Alignment.Center) {
            SoftGlassSurface(Modifier.fillMaxWidth().widthIn(max=440.dp),radius=34.dp) {
                Column(Modifier.fillMaxWidth().verticalScroll(rememberScrollState()).padding(horizontal=26.dp,vertical=28.dp),horizontalAlignment=Alignment.CenterHorizontally) {
                    AnimatedContent(if(!authenticated)"面容确认" else "付款结果",transitionSpec={fadeIn(tween(130,30)) togetherWith fadeOut(tween(100))},label="payment-label"){label->
                        Text(label,style=MaterialTheme.typography.labelLarge,color=MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                    Box(Modifier.fillMaxWidth().height(220.dp),contentAlignment=Alignment.Center) {
                        if(visibleResult!=false)PaymentConfirmation(Modifier.fillMaxSize(),play=!completed,waiting=visibleResult==null,authenticating=!authenticated){completed=true}
                        else { Icon(Icons.Outlined.ErrorOutline,null,Modifier.size(56.dp),tint=MaterialTheme.colorScheme.error);LaunchedEffect(Unit){completed=true} }
                    }
                    Text(if(!authenticated)"请看向屏幕" else if(visibleResult==null)"正在确认" else if(visibleResult==true)successTitle else "支付未完成",style=MaterialTheme.typography.headlineSmall)
                    Text(credit(amount),Modifier.padding(top=16.dp),fontSize=36.sp,fontWeight=FontWeight.SemiBold,letterSpacing=(-1).sp)
                    Text("信用点",style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
                    Text(if(visibleResult==false)commitError ?: errorMessage ?: "请查看返回信息；结果待确认时请勿重复付款" else recipient,Modifier.padding(vertical=22.dp),style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)
                    MotionButton(onClose,Modifier.fillMaxWidth().height(52.dp),enabled=completed){Text(if(completed&&autoCloseOnSuccess)"查看订单" else if(completed)"完成" else "请稍候…")}
                }
            }
        }
    }
    if(embedded)paymentContent() else IosOverlayHost(onDismissRequest={if(completed)onClose()}){paymentContent()}
}
