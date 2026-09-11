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
    var register by rememberSaveable{mutableStateOf(false)};var account by rememberSaveable{mutableStateOf("")};var qq by rememberSaveable{mutableStateOf("")}
    var password by remember{mutableStateOf("")};var code by remember{mutableStateOf("")};var codeSent by remember{mutableStateOf(false)};var busy by remember{mutableStateOf(false)}
    var error by remember{mutableStateOf<String?>(null)};var reset by remember{mutableStateOf(false)};var privacy by remember{mutableStateOf(false)}
    var verificationToken by remember{mutableStateOf("")}
    val api=BackendApi.get(LocalContext.current)
    val scope=rememberCoroutineScope();val colors=MaterialTheme.colorScheme
    val backdrop=rememberGraphicsLayer()
    CompositionLocalProvider(LocalOverlayBackdrop provides backdrop) {
    Column(Modifier.fillMaxSize().recordGlassBackdrop(backdrop).background(colors.background).statusBarsPadding().navigationBarsPadding().imePadding().verticalScroll(rememberScrollState()).padding(horizontal=26.dp)) {
        Spacer(Modifier.height(18.dp))
        LoginVersionBadge(Modifier.align(Alignment.CenterHorizontally).size(138.dp))
        Spacer(Modifier.height(18.dp))
        Text("Deuterium",Modifier.fillMaxWidth(),fontSize=32.sp,fontWeight=androidx.compose.ui.text.font.FontWeight.Bold,letterSpacing=(-.7).sp,textAlign=androidx.compose.ui.text.style.TextAlign.Center)
        Text(if(register)"从你的游戏身份开始" else "钱包、消息与交易，随时相连",Modifier.fillMaxWidth().padding(top=8.dp),style=MaterialTheme.typography.bodyMedium,color=colors.onSurfaceVariant,textAlign=androidx.compose.ui.text.style.TextAlign.Center)
        Spacer(Modifier.height(27.dp))
        SegmentedControl(listOf("登录","创建账号"),if(register)1 else 0,{register=it==1;error=null})
        Spacer(Modifier.height(21.dp))
        Surface(shape=RoundedCornerShape(24.dp),color=colors.surface){Column {
            AuthInput(account,{account=it.take(32);codeSent=false;verificationToken=""},if(register)"游戏内 ID" else "玩家 ID 或 QQ",Icons.Outlined.PersonOutline)
            AuthLine()
            if(register){AuthInput(qq,{qq=it.filter{char->char in '0'..'9'}.take(20);codeSent=false;verificationToken=""},"QQ 号",Icons.Outlined.AlternateEmail,keyboard=KeyboardType.Number);AuthLine();AuthInput(code,{code=it.filter(Char::isDigit).take(6)},"游戏内验证码",Icons.Outlined.VerifiedUser,keyboard=KeyboardType.Number,trailing={PlainButton({
                if(!busy){
                    error=when{!account.trim().matches(Regex("[A-Za-z0-9_]{1,32}"))->"请输入有效的游戏 ID（1–32 位字母、数字或下划线）";qq.length !in 5..20->"请输入有效的 QQ 号（5–20 位数字）";else->null}
                    if(error==null)scope.launch{busy=true;runCatching{api.request("POST","/account/registration-code",JSONObject().put("gameId",account.trim()).put("qq",qq),authenticated=false)}.onSuccess{verificationToken=it.getString("verificationToken");codeSent=true;error=null}.onFailure{error=it.message};busy=false}
                }
            }){Text(if(codeSent)"重发" else "获取",style=MaterialTheme.typography.bodyMedium)}});AuthLine()}
            AuthInput(password,{password=it.take(72)},if(register)"设置密码（至少8位）" else "密码",Icons.Outlined.Lock,secret=true)
        }}
        if(codeSent&&register)Text("验证码已发送至游戏内，请在服务器中查看",Modifier.padding(start=5.dp,top=10.dp),style=MaterialTheme.typography.bodySmall,color=colors.onSurfaceVariant)
        error?.let{Text(it,Modifier.padding(top=12.dp,start=5.dp),style=MaterialTheme.typography.bodySmall,color=colors.error)}
        if(!register)PlainButton({reset=true},Modifier.align(Alignment.End).padding(top=4.dp)){Text("忘记密码？",style=MaterialTheme.typography.bodyMedium)} else Spacer(Modifier.height(18.dp))
        MotionButton({
            error=when{account.isBlank()->"请输入玩家 ID 或 QQ";register&&qq.length !in 5..20->"请输入有效的 QQ 号";register&&(!codeSent||code.length!=6)->"请获取游戏内验证码";password.codePointCount(0,password.length) !in 8..64->"密码需要 8–64 位";else->null}
            if(error==null&&!busy)scope.launch{busy=true;runCatching{if(register)api.register(account.trim(),qq,password,verificationToken,code) else api.login(account.trim(),password)}.onSuccess{name->password="";onSignedIn(name)}.onFailure{error=it.message ?: "登录失败，请稍后重试"};busy=false}
        },Modifier.fillMaxWidth().height(54.dp),enabled=!busy){Text(if(busy)"正在进入…" else if(register)"创建账号并继续" else "登录")}
        Spacer(Modifier.height(22.dp))
        Text("Deuterium ${BuildConfig.VERSION_NAME} · 与游戏世界保持连接",Modifier.fillMaxWidth(),style=MaterialTheme.typography.bodySmall,color=colors.onSurfaceVariant,textAlign=androidx.compose.ui.text.style.TextAlign.Center)
        PlainButton({privacy=true},Modifier.align(Alignment.CenterHorizontally).padding(bottom=12.dp)){Text("账号与隐私说明",style=MaterialTheme.typography.bodySmall)}
    }
    if(reset)ResetPasswordSheet(account){reset=false}
    if(privacy)IosDialog({privacy=false},{Text("账号与隐私")},{Text("账号通过服务器验证，密码仅用于本次身份校验，不会保存在设备中。登录凭据加密保存在本机；交易与消息由后端处理。注册和改密通过游戏内验证码确认身份。")},{PlainButton({privacy=false}){Text("知道了")}})
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
private fun VerificationCodeField(code: String, onCode: (String) -> Unit, sent: Boolean, send: () -> Unit) {
    Spacer(Modifier.height(14.dp))
    RefinedField(code, { onCode(it.filter(Char::isDigit).take(6)) }, label = { Text("游戏内验证码") }, singleLine = true,
        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number), modifier = Modifier.fillMaxWidth(), shape = RoundedCornerShape(18.dp))
    Row(verticalAlignment = Alignment.CenterVertically) {
        PlainButton(onClick = send) { Text(if(sent) "重新获取验证码" else "获取验证码") }
    }
    if(sent) Text("验证码已发送至游戏内", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.primary)
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ResetPasswordSheet(initialAccount: String, onClose: () -> Unit) {
    var account by rememberSaveable { mutableStateOf(initialAccount) }
    var code by remember { mutableStateOf("") }
    var codeSent by remember { mutableStateOf(false) }
    var password by remember { mutableStateOf("") }
    var confirm by remember { mutableStateOf("") }
    var error by remember { mutableStateOf<String?>(null) }
    var done by remember { mutableStateOf(false) }
    var busy by remember{mutableStateOf(false)}
    var verificationToken by remember{mutableStateOf("")}
    val api=BackendApi.get(LocalContext.current);val scope=rememberCoroutineScope()
    IosSheet(onDismissRequest = onClose, containerColor = MaterialTheme.colorScheme.surface) {
        Column(Modifier.fillMaxWidth().verticalScroll(rememberScrollState()).padding(24.dp).navigationBarsPadding()) {
            Text(if(done) "密码修改完成" else "重置密码", style = MaterialTheme.typography.headlineSmall)
            if(done) {
                Text("密码已更新，请使用新密码登录。", Modifier.padding(vertical = 22.dp), style = MaterialTheme.typography.bodyMedium)
                MotionButton(onClose, Modifier.fillMaxWidth().height(52.dp)) { Text("完成") }
            } else {
                Text("通过游戏内验证码验证身份。", Modifier.padding(top = 8.dp, bottom = 18.dp), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                RefinedField(account, { account = it.take(32) }, label = { Text("玩家 ID 或 QQ") }, singleLine = true, modifier = Modifier.fillMaxWidth(), shape = RoundedCornerShape(18.dp))
                VerificationCodeField(code, { code = it }, codeSent) { if(!busy)scope.launch{busy=true;runCatching{api.request("POST","/account/password-reset-code",JSONObject().put("account",account.trim()),authenticated=false)}.onSuccess{verificationToken=it.getString("verificationToken");codeSent=true;error=null}.onFailure{error=it.message};busy=false} }
                Spacer(Modifier.height(12.dp)); SecretField(password, { password = it }, "新密码（至少 8 位）")
                Spacer(Modifier.height(12.dp)); SecretField(confirm, { confirm = it }, "再次确认新密码")
                error?.let { Text(it, Modifier.padding(top = 12.dp), color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall) }
                MotionButton(onClick = {
                    error = when { account.isBlank() -> "请输入账号"; !codeSent || code.length != 6 -> "请输入游戏内验证码"; password.length < 8 -> "新密码至少 8 位"; password != confirm -> "两次输入的密码不一致"; else -> null }
                    if(error == null && !busy)scope.launch{busy=true;runCatching{api.request("POST","/account/password-reset",JSONObject().put("verificationToken",verificationToken).put("code",code).put("newPassword",password),authenticated=false)}.onSuccess{password="";confirm="";done=true;api.forgetSession()}.onFailure{error=it.message};busy=false}
                }, modifier = Modifier.fillMaxWidth().padding(top = 22.dp).height(54.dp)) { Text("确认修改") }
            }
            Spacer(Modifier.height(18.dp))
        }
    }
}
