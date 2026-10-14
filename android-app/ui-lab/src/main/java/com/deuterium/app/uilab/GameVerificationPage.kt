package com.deuterium.app.uilab

import androidx.activity.compose.BackHandler
import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.togetherWith
import androidx.compose.animation.core.tween
import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import org.json.JSONObject
import java.util.UUID

@Composable
fun GameVerificationPage(motion:Boolean,onVerified:()->Unit,onLogout:()->Unit) {
    val api=BackendApi.get(LocalContext.current);val scope=rememberCoroutineScope();val colors=MaterialTheme.colorScheme
    var qq by rememberSaveable{mutableStateOf("")};var code by remember{mutableStateOf("")};var token by remember{mutableStateOf("")}
    var busy by remember{mutableStateOf(false)};var error by remember{mutableStateOf<String?>(null)};var wait by remember{mutableIntStateOf(0)}
    var done by remember{mutableStateOf(false)};var support by rememberSaveable{mutableStateOf(false)}
    var devices by rememberSaveable{mutableStateOf(false)}
    LaunchedEffect(wait){if(wait>0){delay(1000);wait--}}
    LaunchedEffect(done){if(done){if(motion)delay(550);runCatching{api.verifySession()}.onSuccess{onVerified()}.onFailure{error=it.message;done=false}}}
    BackHandler(support){support=false}
    BackHandler(devices){devices=false}
    if(devices){Column(Modifier.fillMaxSize().background(colors.background).statusBarsPadding()){PlainButton({devices=false}){Text("返回身份验证")};DeviceSessionsPage(0.dp)};return}
    if(support){Column(Modifier.fillMaxSize().background(colors.background).statusBarsPadding()){PlainButton({support=false}){Text("返回身份验证")};SupportContactPage(0.dp)};return}
    Column(Modifier.fillMaxSize().background(colors.background).statusBarsPadding().navigationBarsPadding().imePadding().verticalScroll(rememberScrollState()).padding(horizontal=26.dp),horizontalAlignment=Alignment.CenterHorizontally){
        Row(Modifier.fillMaxWidth(),horizontalArrangement=Arrangement.End){PlainButton({devices=true}){Text("登录设备")};PlainButton(onLogout){Text("退出账号")}}
        Spacer(Modifier.height(20.dp));LoginVersionBadge(Modifier.size(112.dp));Spacer(Modifier.height(24.dp))
        AnimatedContent(done,transitionSpec={fadeIn(tween(if(motion)220 else 0)) togetherWith fadeOut(tween(if(motion)150 else 0))},label="game-verification"){completed->
            Column(horizontalAlignment=Alignment.CenterHorizontally){
                Text(if(completed)"验证完成" else "连接你的游戏身份",style=MaterialTheme.typography.headlineSmall)
                Text(if(completed)"正在进入你的社区…" else "账号已创建。完成游戏验证，即可使用 App 和玩家社区。",Modifier.padding(top=10.dp,bottom=24.dp),style=MaterialTheme.typography.bodyMedium,color=colors.onSurfaceVariant)
                if(completed)Icon(Icons.Outlined.CheckCircle,null,Modifier.size(64.dp),tint=colors.primary)
                else{
                    Surface(shape=RoundedCornerShape(22.dp),color=colors.surface,modifier=Modifier.fillMaxWidth()){Column(Modifier.padding(20.dp)){Text(api.userName.ifBlank{"待登记游戏身份"},style=MaterialTheme.typography.titleLarge);Text(api.user?.optString("email").orEmpty(),Modifier.padding(top=6.dp),style=MaterialTheme.typography.bodySmall,color=colors.onSurfaceVariant)}}
                    Spacer(Modifier.height(20.dp))
                    if(api.user?.optString("gameId").isNullOrBlank()){
                        RegisterReleasedProfileForm(api)
                    }else{
                    RefinedField(qq,{qq=it.filter(Char::isDigit).take(20);token=""},Modifier.fillMaxWidth(),label={Text("QQ 号")},singleLine=true,keyboardOptions=KeyboardOptions(keyboardType=KeyboardType.Number),shape=RoundedCornerShape(18.dp))
                    Text("请使用 ${api.userName} 进入游戏，验证码会通过游戏内私聊发送。",Modifier.padding(vertical=14.dp),style=MaterialTheme.typography.bodySmall,color=colors.onSurfaceVariant)
                    PlainButton({if(!busy&&wait==0&&qq.length in 5..20)scope.launch{busy=true;error=null;runCatching{api.request("POST","/account/game-verification-code",JSONObject().put("qq",qq))}.onSuccess{token=it.getString("verificationToken");wait=it.optInt("resendAfterSeconds",60);code=""}.onFailure{error=it.message};busy=false}}){Text(if(wait>0)"${wait} 秒后可重新发送" else if(token.isBlank())"发送游戏验证码" else "重新发送游戏验证码")}
                    if(token.isNotBlank()){Spacer(Modifier.height(10.dp));RefinedField(code,{code=it.filter(Char::isDigit).take(6)},Modifier.fillMaxWidth(),label={Text("游戏内验证码")},singleLine=true,keyboardOptions=KeyboardOptions(keyboardType=KeyboardType.Number),shape=RoundedCornerShape(18.dp))}
                    error?.let{Text(it,Modifier.padding(top=14.dp),color=colors.error,style=MaterialTheme.typography.bodySmall)}
                    MotionButton({if(!busy)scope.launch{busy=true;error=null;runCatching{api.request("POST","/account/game-verification",JSONObject().put("verificationToken",token).put("code",code))}.onSuccess{done=true}.onFailure{error=it.message};busy=false}},Modifier.fillMaxWidth().padding(top=24.dp).height(54.dp),enabled=!busy&&token.isNotBlank()&&code.length==6){Text(if(busy)"正在验证…" else "完成验证")}
                    }
                    PlainButton({support=true},Modifier.padding(top=14.dp,bottom=24.dp)){Text("暂时无法验证？通过邮件联系我们")}
                }
            }
        }
    }
}

@Composable
private fun RegisterReleasedProfileForm(api:BackendApi){
    var gameId by rememberSaveable{mutableStateOf("")}
    var busy by remember{mutableStateOf(false)}
    var error by remember{mutableStateOf<String?>(null)}
    var pending by remember{mutableStateOf<JSONObject?>(null)}
    var uncertain by remember{mutableStateOf(false)}
    val scope=rememberCoroutineScope()
    Column(verticalArrangement=Arrangement.spacedBy(16.dp)){
        Text("原游戏身份登记已由支持人员解除，你的平台账号和资料仍然保留。请登记自己的正版游戏账号。",style=MaterialTheme.typography.bodyMedium)
        RefinedField(gameId,{if(!busy&&!uncertain)gameId=it.take(16)},Modifier.fillMaxWidth(),label={Text("Minecraft Java 版游戏 ID")},singleLine=true,shape=RoundedCornerShape(18.dp))
        Text("登记后可等服务器开放，再进入游戏完成身份验证。",style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
        error?.let{Text(it,color=MaterialTheme.colorScheme.error,style=MaterialTheme.typography.bodySmall)}
        MotionButton({if(!busy)scope.launch{
            busy=true;error=null
            if(pending==null||!uncertain&&pending?.optString("gameId")!=gameId.trim())pending=JSONObject().put("clientRequestId",UUID.randomUUID().toString()).put("gameId",gameId.trim())
            try{api.request("POST","/account/game-profile",pending);api.verifySession()}
            catch(cancelled:kotlinx.coroutines.CancellationException){throw cancelled}
            catch(e:Exception){error=e.message;uncertain=e !is ApiFailure||e.status>=500||e.status==0}
            finally{busy=false}
        }},Modifier.fillMaxWidth().height(54.dp),enabled=!busy&&Regex("[A-Za-z0-9_]{3,16}").matches(gameId.trim())){Text(if(busy)"正在登记…" else if(uncertain)"确认登记结果" else "登记游戏身份")}
    }
}

@Composable
fun SupportContactPage(topInset:androidx.compose.ui.unit.Dp) {
    val context=LocalContext.current;val api=BackendApi.get(context)
    var address by remember{mutableStateOf("")};var error by remember{mutableStateOf<String?>(null)}
    LaunchedEffect(Unit){runCatching{api.request("GET","/support/contact",authenticated=false)}.onSuccess{address=it.optString("email")}.onFailure{error=it.message}}
    Column(Modifier.fillMaxSize().background(MaterialTheme.colorScheme.background).padding(horizontal=24.dp).padding(top=topInset,bottom=30.dp),verticalArrangement=Arrangement.spacedBy(18.dp)){
        Spacer(Modifier.height(18.dp));Icon(Icons.Outlined.Email,null,Modifier.size(42.dp),tint=MaterialTheme.colorScheme.primary)
        Text("邮件联系支持",style=MaterialTheme.typography.headlineSmall)
        Text("请通过邮件说明问题类型、游戏 ID 和详细情况。收到后，我们会向你回复工单编号，后续进展也会通过邮件联系。",color=MaterialTheme.colorScheme.onSurfaceVariant)
        if(address.isNotBlank()){
            Text(address,style=MaterialTheme.typography.titleMedium)
            MotionButton({val uri=android.net.Uri.parse("mailto:"+address+"?subject="+android.net.Uri.encode("Deuterium 支持请求")+"&body="+android.net.Uri.encode("游戏 ID：${api.userName}\n问题类型：\n详细情况：\n"));runCatching{context.startActivity(android.content.Intent(android.content.Intent.ACTION_SENDTO,uri))}.onFailure{error="未找到邮件应用，请复制客服邮箱后发送。"}},Modifier.fillMaxWidth().height(52.dp)){Text("打开邮件应用")}
            val clipboard=androidx.compose.ui.platform.LocalClipboardManager.current
            PlainButton({clipboard.setText(androidx.compose.ui.text.AnnotatedString(address))}){Text("复制客服邮箱")}
        }else Text(error ?: "客服邮箱尚未配置",color=MaterialTheme.colorScheme.onSurfaceVariant)
        error?.let{Text(it,color=MaterialTheme.colorScheme.error)}
    }
}
