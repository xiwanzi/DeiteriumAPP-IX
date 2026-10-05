package com.deuterium.app.uilab

import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.rememberGraphicsLayer
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.semantics.contentDescription
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import androidx.compose.ui.platform.LocalContext
import org.json.JSONObject

@Composable
fun AuthPage(onSignedIn:(String)->Unit) {
    var register by rememberSaveable{mutableStateOf(false)};var account by rememberSaveable{mutableStateOf("")};var email by rememberSaveable{mutableStateOf("")}
    var password by remember{mutableStateOf("")};var code by remember{mutableStateOf("")};var codeSent by remember{mutableStateOf(false)};var busy by remember{mutableStateOf(false)}
    var error by remember{mutableStateOf<String?>(null)};var reset by remember{mutableStateOf(false)};var privacy by remember{mutableStateOf(false)}
    var verificationToken by remember{mutableStateOf("")};var resendSeconds by remember{mutableIntStateOf(0)}
    var codeBusy by remember{mutableStateOf(false)};var codeRequest by remember{mutableIntStateOf(0)}
    val codeCooldowns=remember{mutableStateMapOf<String,Pair<Long,Int>>()};val loginCooldowns=remember{mutableStateMapOf<String,Long>()}
    var loginRetrySeconds by remember{mutableIntStateOf(0)}
    fun recipient()=email.trim().lowercase()
    fun secondsUntil(deadline:Long)=maxOf(0,((deadline-android.os.SystemClock.elapsedRealtime()+999)/1000).toInt())
    fun codeDeadline()=maxOf(codeCooldowns[recipient()]?.first ?: 0,codeCooldowns["*"]?.first ?: 0)
    fun invalidateCode(){codeRequest++;codeBusy=false;codeSent=false;verificationToken="";resendSeconds=secondsUntil(codeDeadline());error=null}
    LaunchedEffect(email,register,codeCooldowns[recipient()],codeCooldowns["*"]){do{resendSeconds=secondsUntil(codeDeadline());if(resendSeconds>0)delay(250)}while(resendSeconds>0)}
    LaunchedEffect(account,register,loginCooldowns[account.trim().lowercase()]){do{loginRetrySeconds=secondsUntil(loginCooldowns[account.trim().lowercase()] ?: 0);if(loginRetrySeconds>0)delay(250)}while(loginRetrySeconds>0)}
    val api=BackendApi.get(LocalContext.current)
    val scope=rememberCoroutineScope();val colors=MaterialTheme.colorScheme
    val backdrop=rememberGraphicsLayer()
    CompositionLocalProvider(LocalOverlayBackdrop provides backdrop) {
    Column(Modifier.fillMaxSize().recordGlassBackdrop(backdrop).background(colors.background).statusBarsPadding().navigationBarsPadding().imePadding().verticalScroll(rememberScrollState()).padding(horizontal=26.dp)) {
        Spacer(Modifier.height(18.dp))
        LoginVersionBadge(Modifier.align(Alignment.CenterHorizontally).size(138.dp))
        Spacer(Modifier.height(18.dp))
        Text("Deuterium",Modifier.fillMaxWidth(),fontSize=32.sp,fontWeight=androidx.compose.ui.text.font.FontWeight.Bold,letterSpacing=(-.7).sp,textAlign=androidx.compose.ui.text.style.TextAlign.Center)
        Text(if(register)"创建账号，再连接你的游戏身份" else "钱包、消息与交易，随时相连",Modifier.fillMaxWidth().padding(top=8.dp),style=MaterialTheme.typography.bodyMedium,color=colors.onSurfaceVariant,textAlign=androidx.compose.ui.text.style.TextAlign.Center)
        Spacer(Modifier.height(27.dp))
        SegmentedControl(listOf("登录","创建账号"),if(register)1 else 0,{register=it==1;invalidateCode()})
        Spacer(Modifier.height(21.dp))
        Surface(shape=RoundedCornerShape(24.dp),color=colors.surface){Column {
            AuthInput(account,{val next=it.take(if(register)32 else 254);if(next!=account){account=next;invalidateCode()}},if(register)"游戏内 ID" else "邮箱、玩家 ID 或 QQ",Icons.Outlined.PersonOutline)
            AuthLine()
            if(register){AuthInput(email,{val next=it.take(254);if(next!=email){email=next;invalidateCode()}},"邮箱",Icons.Outlined.AlternateEmail,keyboard=KeyboardType.Email);AuthLine();AuthInput(code,{code=it.filter(Char::isDigit).take(6)},"验证码",Icons.Outlined.VerifiedUser,keyboard=KeyboardType.Number,trailing={PlainButton({
                if(!busy&&!codeBusy&&resendSeconds==0){
                    error=when{!account.trim().matches(Regex("[A-Za-z0-9_]{3,16}"))->"请填写 Minecraft Java 版玩家名：3–16 位字母、数字或下划线。";!android.util.Patterns.EMAIL_ADDRESS.matcher(email.trim()).matches()->"请输入有效的邮箱";else->null}
                    if(error==null){
                        val requestedAccount=account.trim();val requestedEmail=email.trim();val requestedRecipient=requestedEmail.lowercase();val requestId=++codeRequest
                        codeBusy=true;codeCooldowns[requestedRecipient]=(android.os.SystemClock.elapsedRealtime()+60000) to requestId;resendSeconds=60
                        scope.launch{
                            runCatching{api.request("POST","/account/registration-code",JSONObject().put("gameId",requestedAccount).put("email",requestedEmail),authenticated=false)}.onSuccess{
                                if(codeCooldowns[requestedRecipient]?.second==requestId)codeCooldowns[requestedRecipient]=(android.os.SystemClock.elapsedRealtime()+it.optInt("resendAfterSeconds",60)*1000L) to requestId
                                if(requestId==codeRequest){verificationToken=it.getString("verificationToken");codeSent=true;error=null}
                            }.onFailure{
                                val failure=it as? ApiFailure
                                val knownNoMail=failure?.code in setOf("MINECRAFT_PLAYER_NOT_FOUND","MINECRAFT_LOOKUP_UNAVAILABLE","MAIL_NOT_READY","EMAIL_INVALID","GAME_ID_INVALID","ACCOUNT_ALREADY_EXISTS","FORBIDDEN","INVALID_REQUEST")||failure?.status in setOf(400,403,409)
                                if(codeCooldowns[requestedRecipient]?.second==requestId){if(failure?.status==429){val entry=(android.os.SystemClock.elapsedRealtime()+maxOf(1,failure.retryAfterSeconds.takeIf{it>0} ?: 60)*1000L) to requestId;codeCooldowns[requestedRecipient]=entry;if(failure.retryScope=="SOURCE")codeCooldowns["*"]=entry} else if(knownNoMail)codeCooldowns.remove(requestedRecipient)}
                                if(requestId==codeRequest)error=if(knownNoMail||failure?.status==429)it.message else "本次发送结果暂未确认，请查看邮箱；倒计时结束后可重试。"
                            }
                            if(requestId==codeRequest)codeBusy=false
                            resendSeconds=secondsUntil(codeDeadline())
                        }
                    }
                }
            }){Text(if(resendSeconds>0)"${resendSeconds}s" else if(codeSent)"重发" else "获取",style=MaterialTheme.typography.bodyMedium)}});AuthLine()}
            AuthInput(password,{password=it.take(72)},if(register)"设置密码（至少8位）" else "密码",Icons.Outlined.Lock,secret=true)
        }}
        error?.let{Text(it,Modifier.padding(top=12.dp,start=5.dp),style=MaterialTheme.typography.bodySmall,color=colors.error)}
        if(!register)PlainButton({reset=true},Modifier.align(Alignment.End).padding(top=4.dp)){Text("忘记密码？",style=MaterialTheme.typography.bodyMedium)} else Spacer(Modifier.height(18.dp))
        MotionButton({
            error=when{account.isBlank()->"请输入玩家 ID 或 QQ";register&&!android.util.Patterns.EMAIL_ADDRESS.matcher(email.trim()).matches()->"请输入有效的邮箱";register&&(!codeSent||code.length!=6)->"请获取验证码";password.codePointCount(0,password.length) !in 8..64->"密码需要 8–64 位";else->null}
            if(error==null&&!busy&&!codeBusy&&(register||loginRetrySeconds==0)){
                val requestedRegister=register;val requestedAccount=account.trim();val requestedEmail=email.trim()
                val requestedPassword=password;val requestedToken=verificationToken;val requestedCode=code
                busy=true
                scope.launch{
                    runCatching{if(requestedRegister)api.register(requestedAccount,requestedEmail,requestedPassword,requestedToken,requestedCode) else api.login(requestedAccount,requestedPassword)}
                        .onSuccess{name->password="";onSignedIn(name)}
                        .onFailure{
                            val failure=it as? ApiFailure
                            if(!requestedRegister&&failure?.status==429){
                                val retry=maxOf(1,failure.retryAfterSeconds.takeIf{it>0} ?: 900)
                                loginCooldowns[requestedAccount.lowercase()]=android.os.SystemClock.elapsedRealtime()+retry*1000L
                            }
                            if(register==requestedRegister&&account.trim()==requestedAccount){
                                error=if(requestedRegister&&failure?.code in setOf("NETWORK_UNAVAILABLE","INVALID_RESPONSE","SERVICE_UNAVAILABLE"))"注册结果暂未确认，请尝试使用刚才的邮箱和密码登录。" else it.message ?: "登录失败，请稍后重试"
                            }
                        }
                    busy=false
                }
            }
        },Modifier.fillMaxWidth().height(54.dp),enabled=!busy&&!codeBusy&&(register||loginRetrySeconds==0)){Text(if(busy)"正在进入…" else if(register)"创建账号并继续" else if(loginRetrySeconds>0)"${loginRetrySeconds}s 后重试" else "登录")}
        Spacer(Modifier.height(22.dp))
        Text("Deuterium ${BuildConfig.VERSION_NAME} · 与游戏世界保持连接",Modifier.fillMaxWidth(),style=MaterialTheme.typography.bodySmall,color=colors.onSurfaceVariant,textAlign=androidx.compose.ui.text.style.TextAlign.Center)
        PlainButton({privacy=true},Modifier.align(Alignment.CenterHorizontally).padding(bottom=12.dp)){Text("账号与隐私说明",style=MaterialTheme.typography.bodySmall)}
    }
    if(reset)ResetPasswordSheet(account){reset=false}
    if(privacy)IosDialog({privacy=false},{Text("账号与隐私")},{Text("账号通过服务器验证，密码仅用于本次身份校验，不会保存在设备中。登录凭据加密保存在本机；交易与消息由后端处理。注册通过邮箱验证码创建账号，游戏身份在登录后单独验证。支持通过绑定邮箱找回密码。")},{PlainButton({privacy=false}){Text("知道了")}})
    }
}
@Composable
private fun AuthLine(){HorizontalDivider(Modifier.padding(start=55.dp),thickness=.5.dp,color=MaterialTheme.colorScheme.outlineVariant.copy(alpha=.55f))}
@Composable
private fun AuthInput(value:String,onValue:(String)->Unit,hint:String,icon:androidx.compose.ui.graphics.vector.ImageVector,secret:Boolean=false,keyboard:KeyboardType=KeyboardType.Text,trailing:(@Composable ()->Unit)?=null) {
    var visible by remember{mutableStateOf(false)}
    Row(Modifier.fillMaxWidth().heightIn(min=60.dp).padding(start=18.dp,end=10.dp),verticalAlignment=Alignment.CenterVertically){
        Icon(icon,null,Modifier.size(22.dp),tint=MaterialTheme.colorScheme.onSurfaceVariant)
        androidx.compose.foundation.text.BasicTextField(value,onValue,Modifier.weight(1f).padding(horizontal=12.dp,vertical=16.dp).semantics{contentDescription=hint},singleLine=true,
            textStyle=MaterialTheme.typography.bodyLarge.copy(color=MaterialTheme.colorScheme.onSurface),cursorBrush=androidx.compose.ui.graphics.SolidColor(MaterialTheme.colorScheme.primary),
            visualTransformation=if(secret&&!visible)PasswordVisualTransformation() else VisualTransformation.None,
            keyboardOptions=KeyboardOptions(keyboardType=if(secret)KeyboardType.Password else keyboard),decorationBox={inner->Box{if(value.isEmpty())Text(hint,style=MaterialTheme.typography.bodyLarge,color=MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha=.8f));inner()}})
        if(secret)IconButton({visible=!visible}){Icon(if(visible)Icons.Outlined.VisibilityOff else Icons.Outlined.Visibility,if(visible)"隐藏密码" else "显示密码",Modifier.size(21.dp),tint=MaterialTheme.colorScheme.onSurfaceVariant)}
        trailing?.invoke()
    }
}

@Composable
private fun SecretField(value: String, onValue: (String) -> Unit, label: String) {
    var visible by remember { mutableStateOf(false) }
    RefinedField(value, { onValue(it.take(72)) }, label = { Text(label) }, singleLine = true,
        visualTransformation = if(visible) VisualTransformation.None else PasswordVisualTransformation(), keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password),
        trailingIcon = { IconButton({ visible = !visible }) { Icon(if(visible) Icons.Outlined.VisibilityOff else Icons.Outlined.Visibility, if(visible) "隐藏密码" else "显示密码") } },
        modifier = Modifier.fillMaxWidth(), shape = RoundedCornerShape(18.dp))
}

@Composable
private fun VerificationCodeField(code: String, onCode: (String) -> Unit, sent: Boolean, seconds:Int=0, enabled:Boolean=true, send: () -> Unit) {
    Spacer(Modifier.height(14.dp))
    RefinedField(code, { onCode(it.filter(Char::isDigit).take(6)) }, label = { Text("验证码") }, singleLine = true,
        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number), modifier = Modifier.fillMaxWidth(), shape = RoundedCornerShape(18.dp))
    Row(verticalAlignment = Alignment.CenterVertically) {
        PlainButton(onClick = {if(enabled&&seconds==0)send()}) { Text(if(seconds>0)"${seconds}s 后重发" else if(sent) "重新获取验证码" else "获取验证码") }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ResetPasswordSheet(initialAccount: String, changing:Boolean=false, onClose: () -> Unit) {
    var account by rememberSaveable { mutableStateOf(initialAccount) }
    var code by remember { mutableStateOf("") }
    var codeSent by remember { mutableStateOf(false) }
    var password by remember { mutableStateOf("") }
    var confirm by remember { mutableStateOf("") }
    var error by remember { mutableStateOf<String?>(null) }
    var done by remember { mutableStateOf(false) }
    var busy by remember{mutableStateOf(false)}
    var verificationToken by remember{mutableStateOf("")}
    var codeBusy by remember{mutableStateOf(false)};var codeRequest by remember{mutableIntStateOf(0)};var resendSeconds by remember{mutableIntStateOf(0)}
    val api=BackendApi.get(LocalContext.current);val scope=rememberCoroutineScope()
    val deadlines=api.passwordCodeCooldowns
    fun key()=account.trim().lowercase()
    fun remaining()=maxOf(0,((maxOf(deadlines[key()]?.first ?: 0,deadlines["*"]?.first ?: 0)-android.os.SystemClock.elapsedRealtime()+999)/1000).toInt())
    LaunchedEffect(account,deadlines[key()],deadlines["*"]){do{resendSeconds=remaining();if(resendSeconds>0)delay(250)}while(resendSeconds>0)}
    IosSheet(onDismissRequest = onClose, containerColor = MaterialTheme.colorScheme.surface) {
        Column(Modifier.fillMaxWidth().verticalScroll(rememberScrollState()).padding(24.dp).navigationBarsPadding()) {
            Text(if(done) "密码修改完成" else if(changing)"更改密码" else "找回密码", style = MaterialTheme.typography.headlineSmall)
            if(done) {
                Text("密码已更新，请使用新密码登录。", Modifier.padding(vertical = 22.dp), style = MaterialTheme.typography.bodyMedium)
                MotionButton(onClose, Modifier.fillMaxWidth().height(52.dp)) { Text("完成") }
            } else {
                Spacer(Modifier.height(18.dp))
                if(changing)Text("账号：$account",Modifier.fillMaxWidth().padding(bottom=12.dp),style=MaterialTheme.typography.titleMedium) else RefinedField(account, {val next=it.take(254);if(next!=account){account=next;codeRequest++;codeBusy=false;codeSent=false;verificationToken="";resendSeconds=remaining();error=null}}, label = { Text("邮箱、玩家 ID 或 QQ") }, singleLine = true, modifier = Modifier.fillMaxWidth(), shape = RoundedCornerShape(18.dp))
                VerificationCodeField(code, { code = it }, codeSent,resendSeconds,!busy&&!codeBusy) {
                    if(!busy&&!codeBusy&&remaining()==0){
                        if(account.isBlank()){error="请输入账号"}else{
                            val requestedKey=key();val requestedAccount=account.trim();val requestId=++api.passwordCodeRequestId;codeRequest=requestId;codeBusy=true;resendSeconds=60;deadlines[requestedKey]=(android.os.SystemClock.elapsedRealtime()+60000) to requestId
                            scope.launch{
                                runCatching{api.request("POST",if(changing)"/account/password-change-code" else "/account/password-reset-code",if(changing)JSONObject() else JSONObject().put("account",requestedAccount),authenticated=changing)}.onSuccess{
                                    if(deadlines[requestedKey]?.second==requestId)deadlines[requestedKey]=(android.os.SystemClock.elapsedRealtime()+it.optInt("resendAfterSeconds",60)*1000L) to requestId
                                    if(requestId==codeRequest){verificationToken=it.getString("verificationToken");codeSent=true;error=null}
                                }.onFailure{
                                    val failure=it as? ApiFailure;val known=failure?.status in setOf(400,401,403,404,409)||failure?.code in setOf("MAIL_NOT_READY","PLAYER_OFFLINE","PLUGIN_BRIDGE_UNAVAILABLE")
                                    if(deadlines[requestedKey]?.second==requestId){if(failure?.status==429){val entry=(android.os.SystemClock.elapsedRealtime()+maxOf(1,failure.retryAfterSeconds.takeIf{it>0} ?: 60)*1000L) to requestId;deadlines[requestedKey]=entry;if(failure.retryScope=="SOURCE")deadlines["*"]=entry}else if(known)deadlines.remove(requestedKey)}
                                    if(requestId==codeRequest)error=if(known||failure?.status==429)it.message else "本次验证码发送结果暂未确认，请等待倒计时结束后重试。"
                                }
                                if(requestId==codeRequest)codeBusy=false;resendSeconds=remaining()
                            }
                        }
                    }
                }
                Spacer(Modifier.height(12.dp)); SecretField(password, { password = it }, "新密码（至少 8 位）")
                Spacer(Modifier.height(12.dp)); SecretField(confirm, { confirm = it }, "再次确认新密码")
                error?.let { Text(it, Modifier.padding(top = 12.dp), color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall) }
                MotionButton(onClick = {
                    error = when { account.isBlank() -> "请输入账号"; !codeSent || code.length != 6 -> "请输入验证码"; password.codePointCount(0,password.length) !in 8..64 -> "密码需要 8–64 位"; password != confirm -> "两次输入的密码不一致"; else -> null }
                    if(error == null && !busy&&!codeBusy)scope.launch{busy=true;runCatching{api.request("POST",if(changing)"/account/password-change" else "/account/password-reset",JSONObject().put("verificationToken",verificationToken).put("code",code).put("newPassword",password),authenticated=changing)}.onSuccess{password="";confirm="";done=true;api.forgetSession()}.onFailure{error=it.message};busy=false}
                }, modifier = Modifier.fillMaxWidth().padding(top = 22.dp).height(54.dp)) { Text("确认修改") }
            }
            Spacer(Modifier.height(18.dp))
        }
    }
}
