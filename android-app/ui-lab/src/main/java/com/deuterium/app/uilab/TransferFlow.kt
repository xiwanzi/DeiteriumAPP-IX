package com.deuterium.app.uilab

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Modifier
import androidx.compose.ui.Alignment
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.delay

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TransferSheet(state: LabState, initialRecipient: String? = null, onClose: () -> Unit) {
    var amount by rememberSaveable { mutableStateOf("") }
    var query by rememberSaveable { mutableStateOf(initialRecipient.orEmpty()) }
    var selectedName by rememberSaveable { mutableStateOf(initialRecipient) }
    var verifiedRecipient by remember { mutableStateOf<PlayerProfile?>(null) }
    val recipient = verifiedRecipient?.takeIf{it.name==selectedName}
    var results by remember { mutableStateOf<List<PlayerProfile>>(emptyList()) }
    var searching by remember { mutableStateOf(false) }
    var note by rememberSaveable { mutableStateOf("") }
    var error by remember { mutableStateOf<String?>(null) }
    var pending by rememberSaveable { mutableLongStateOf(0L) }
    var paymentKey by rememberSaveable { mutableStateOf<String?>(null) }
    var paymentRecipientRef by rememberSaveable { mutableStateOf<String?>(null) }
    LaunchedEffect(selectedName,verifiedRecipient?.playerRef){if(recipient!=null)state.loadProfile(recipient.name)}
    LaunchedEffect(query) {
        if(query.isBlank()){results=emptyList();return@LaunchedEffect}
        delay(300);searching=true
        runCatching{state.searchRecipients(query)}.onSuccess{results=it;verifiedRecipient=it.find{candidate->candidate.name==selectedName};error=null}.onFailure{results=emptyList();verifiedRecipient=null;error=it.message}
        searching=false
    }
    IosSheet(onDismissRequest = {if(pending==0L)onClose()}, containerColor = MaterialTheme.colorScheme.surface,
        replacementContent=if(pending>0) {{
            PaymentExperience(pending,"转给 ${selectedName.orEmpty()}","转账成功",onCommit={val result=selectedName?.let{state.transfer(it,pending,note,paymentKey,paymentRecipientRef)} ?: false;paymentKey=state.activeTransferKey;result},errorMessage=state.storageMessage,confirmed=if(paymentKey!=null&&paymentKey==state.completedTransferKey&&state.transferSucceeded)true else null,embedded=true,onClose=onClose)
        }} else null) {
        Column(Modifier.fillMaxWidth().verticalScroll(rememberScrollState()).padding(horizontal = 24.dp).padding(bottom = 24.dp)) {
            Text("转账", style = MaterialTheme.typography.headlineSmall)
            Text("先搜索并确认收款玩家。", Modifier.padding(top = 6.dp, bottom = 18.dp), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            RefinedField(query, { query = it.take(32); selectedName = null; error = null }, label = { Text("搜索玩家 ID 或 QQ") }, singleLine = true,
                leadingIcon = { Icon(Icons.Outlined.Search,null) }, modifier = Modifier.fillMaxWidth(), shape = RoundedCornerShape(18.dp))
            if(recipient == null && query.isNotBlank()) {
                if(results.isEmpty()) Text(if(searching)"正在查询玩家…" else error ?: "没有找到玩家，请换个 ID 或 QQ。", Modifier.padding(vertical = 10.dp), color = MaterialTheme.colorScheme.onSurfaceVariant, style = MaterialTheme.typography.bodySmall)
                results.forEach { candidate ->
                    Surface(modifier = Modifier.fillMaxWidth().padding(top = 8.dp), shape = RoundedCornerShape(16.dp), color = MaterialTheme.colorScheme.surfaceVariant.copy(alpha=.45f), onClick = { query = candidate.name; selectedName = candidate.name;verifiedRecipient=candidate }) {
                        Row(Modifier.padding(12.dp), verticalAlignment = Alignment.CenterVertically) {
                            PlayerAvatar(candidate.name,Modifier.size(34.dp))
                            Column(Modifier.weight(1f).padding(start=10.dp)) { Text(candidate.name,style=MaterialTheme.typography.titleMedium);Text(if(candidate.qq.isBlank()||candidate.qq=="null")"玩家身份已确认" else "QQ ${candidate.qq}",style=MaterialTheme.typography.bodySmall) }
                            Icon(Icons.Outlined.ChevronRight,null)
                        }
                    }
                }
            }
            if(recipient != null) {
                Row(Modifier.fillMaxWidth().padding(vertical = 14.dp),verticalAlignment=Alignment.CenterVertically){
                    PlayerAvatar(recipient.name);Column(Modifier.weight(1f).padding(start=12.dp)){Text(recipient.name,style=MaterialTheme.typography.titleMedium);Text(if(recipient.qq.isBlank()||recipient.qq=="null")"已确认玩家身份" else "已确认玩家 · QQ ${recipient.qq}",style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)}
                    Icon(Icons.Outlined.CheckCircle,null,tint=MaterialTheme.colorScheme.primary)
                }
            } else {
                Text("最近转账",Modifier.padding(top=14.dp),style=MaterialTheme.typography.labelMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)
                Row(Modifier.horizontalScroll(rememberScrollState()),horizontalArrangement=Arrangement.spacedBy(8.dp)){
                    state.recentTransfers.forEach { name -> AssistChoice(onClick={query=name;selectedName=name},label={Text(name)}) }
                }
            }
            Spacer(Modifier.height(16.dp))
            RefinedField(amount, { amount = it.take(10); error = null }, label = { Text("转账金额") }, suffix = { Text("信用点") },
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal), singleLine = true, isError = error != null,
                modifier = Modifier.fillMaxWidth(), shape = RoundedCornerShape(18.dp))
            Text(error ?: if(state.balanceKnown) "可用余额 ${credit(state.balance)}" else "余额尚未同步，请先刷新钱包", Modifier.padding(top = 8.dp), style = MaterialTheme.typography.bodySmall,
                color = if(error != null) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.onSurfaceVariant)
            Spacer(Modifier.height(16.dp))
            RefinedField(note, { note = it.take(40) }, label = { Text("转账备注（选填）") }, singleLine = true, modifier = Modifier.fillMaxWidth(), shape = RoundedCornerShape(18.dp))
            Spacer(Modifier.height(24.dp))
            MotionButton(onClick = {
                if(pending > 0) return@MotionButton
                val cents = runCatching { amount.toBigDecimal().movePointRight(2).longValueExact() }.getOrNull()
                when { recipient == null -> error = "请搜索并确认收款玩家"
                    !state.balanceKnown -> error = "余额尚未同步，请先刷新钱包"
                    cents == null || cents <= 0 -> error = "请输入有效金额，最多两位小数"
                    cents > state.balance -> error = "余额不足"
                    else -> {paymentKey=java.util.UUID.randomUUID().toString();paymentRecipientRef=recipient.playerRef;pending=cents} }
            }, modifier = Modifier.fillMaxWidth().height(54.dp),enabled=!state.transferPending) { Text(if(state.transferPending)"正在核对上一笔转账" else "确认转账") }
        }
    }
}
