package com.deuterium.app.uilab

import androidx.compose.animation.*
import androidx.compose.animation.core.*
import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.ArrowForward
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlin.math.abs
import androidx.compose.ui.unit.Dp

@Composable
fun WalletPage(state: LabState, onTransfer: () -> Unit, onRecord: (LedgerEntry) -> Unit, topInset: Dp = 12.dp, onBills: (String) -> Unit = {}) {
    LaunchedEffect(state){state.refresh()}
    val motion = LocalMotion.current
    LazyColumn(contentPadding = PaddingValues(start = 20.dp, end = 20.dp, top = topInset, bottom = 118.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp)) {
        item(key = "balance") { BalanceCard(state, onTransfer) }
        state.walletError?.let{message->item{Text(message,color=MaterialTheme.colorScheme.error,style=MaterialTheme.typography.bodySmall)}}
        item(key = "summary") {
            Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                SummaryCard("本日收入", state.todayIncome?.let{"+${credit(it)}"} ?: "—", true, Modifier.weight(1f), { onBills("income") })
                SummaryCard("本日支出", state.todayExpense?.let{"−${credit(it)}"} ?: "—", false, Modifier.weight(1f), { onBills("expense") })
            }
        }
        item(key = "label") {
            Row(Modifier.fillMaxWidth().padding(top = 8.dp, bottom = 2.dp), verticalAlignment = Alignment.CenterVertically) {
                Text("最近流水", style = MaterialTheme.typography.titleLarge, modifier = Modifier.weight(1f))
                PlainButton({ onBills("all") }) { Text("历史账单") }
            }
        }
        items(state.ledger.sortedWith(compareByDescending<LedgerEntry>{it.at}.thenByDescending{it.id}).take(8), key = { it.id }) { record ->
            LedgerRow(record, onClick = { onRecord(record) }, modifier = if(motion) Modifier.animateItem() else Modifier)
        }
        item { Text("所有收支均可在历史账单中查看。", Modifier.fillMaxWidth().padding(vertical = 8.dp),
            style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant) }
    }
}

@Composable
private fun BalanceCard(state: LabState, onTransfer: () -> Unit) {
    val motion = LocalMotion.current
    val rotation = if (state.refreshing && motion) {
        val rotating = rememberInfiniteTransition(label = "refresh")
        val angle = rotating.animateFloat(0f, 360f, infiniteRepeatable(tween(850, easing = LinearEasing)), label = "refresh-angle")
        angle
    } else null
    Box(Modifier.fillMaxWidth().clip(RoundedCornerShape(30.dp)).background(MaterialTheme.colorScheme.surface)) {
        Canvas(Modifier.matchParentSize()) {
            val center = Offset(size.width * .95f, size.height * .20f)
            repeat(3) { i -> drawCircle(Color(0xFF7BAF9A).copy(alpha = .14f), radius = size.width * (.24f + i * .1f),
                center = center, style = Stroke(1.1.dp.toPx())) }
            drawCircle(Color(0xFFB1DCC3).copy(alpha = .07f), size.width * .27f, Offset(size.width, size.height))
        }
        Column(Modifier.padding(24.dp)) {
            Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                Text("可用余额", color = MaterialTheme.colorScheme.onSurfaceVariant, style = MaterialTheme.typography.labelLarge, modifier = Modifier.weight(1f))
                IconButton(onClick = { state.privacy = !state.privacy }, modifier = Modifier.size(48.dp)) {
                    Icon(if(state.privacy) Icons.Outlined.VisibilityOff else Icons.Outlined.Visibility,
                        if(state.privacy) "显示余额" else "隐藏余额", tint = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.size(21.dp))
                }
            }
            AnimatedContent(if(state.privacy) "••••••" else if(!state.balanceKnown) "—" else credit(state.balance), label = "balance",
                transitionSpec = { fadeIn(tween(if(motion) 200 else 0)) togetherWith fadeOut(tween(if(motion) 100 else 0)) }) { amount ->
                Text(amount, color = MaterialTheme.colorScheme.onSurface, fontSize = 39.sp, lineHeight = 49.sp,
                    fontWeight = FontWeight.SemiBold, letterSpacing = (-1.2).sp, maxLines = 1)
            }
            Spacer(Modifier.height(4.dp))
            Row(verticalAlignment = Alignment.CenterVertically) {
                Box(Modifier.size(5.dp).background(MaterialTheme.colorScheme.primary, CircleShape))
                Spacer(Modifier.width(7.dp))
                Text(if(state.refreshing) "正在更新…" else state.refreshed,
                    color = MaterialTheme.colorScheme.onSurfaceVariant, style = MaterialTheme.typography.bodySmall)
            }
            if(state.heldBalance>0)Text("担保冻结 ${credit(state.heldBalance)} 信用点",Modifier.padding(top=12.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.primary)
            Spacer(Modifier.height(25.dp))
            Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                MotionButton(onClick = onTransfer, shape = RoundedCornerShape(16.dp), modifier = Modifier.weight(1f).height(51.dp),
                    colors = ButtonDefaults.buttonColors()) {
                    Text("发起转账", fontWeight = FontWeight.SemiBold)
                    Spacer(Modifier.width(9.dp))
                    Icon(Icons.AutoMirrored.Outlined.ArrowForward, null, Modifier.size(19.dp))
                }
                OutlinedIconButton(onClick = state::refresh, enabled = !state.refreshing,
                    modifier = Modifier.size(51.dp), shape = RoundedCornerShape(16.dp),
                    border = BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant)) {
                    Icon(Icons.Outlined.Refresh, "刷新余额", tint = MaterialTheme.colorScheme.primary, modifier = Modifier.size(23.dp)
                        .graphicsLayer { rotationZ = rotation?.value ?: 0f })
                }
            }
        }
    }
}

@Composable
private fun SummaryCard(label: String, value: String, income: Boolean, modifier: Modifier, onClick: () -> Unit) {
    Surface(onClick = onClick, modifier = modifier, color = MaterialTheme.colorScheme.surface, shape = RoundedCornerShape(22.dp)) {
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Icon(if(income) Icons.Outlined.SouthWest else Icons.Outlined.NorthEast, null, Modifier.size(15.dp), tint = MaterialTheme.colorScheme.onSurfaceVariant)
                Spacer(Modifier.width(6.dp))
                Text(label, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            Text(value, fontSize = 19.sp, fontWeight = FontWeight.SemiBold,
                color = if(income) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurface, maxLines = 1)
        }
    }
}

@Composable
fun LedgerRow(record: LedgerEntry, onClick: () -> Unit, modifier: Modifier = Modifier) {
    val incoming=record.amount>0
    val money=(if(incoming) "+" else "−")+credit(abs(record.amount))
    val measurer=androidx.compose.ui.text.rememberTextMeasurer()
    val density=androidx.compose.ui.platform.LocalDensity.current
    val amountStyle=MaterialTheme.typography.titleMedium.copy(fontSize=16.sp,fontWeight=FontWeight.SemiBold)
    Surface(onClick=onClick,modifier=modifier.fillMaxWidth(),shape=RoundedCornerShape(22.dp),color=MaterialTheme.colorScheme.surface){
        BoxWithConstraints(Modifier.padding(15.dp)){
            val stacked=measurer.measure(androidx.compose.ui.text.AnnotatedString(money),amountStyle,maxLines=1,softWrap=false).size.width>with(density){(maxWidth-68.dp).toPx()}*.55f
            Row(verticalAlignment=Alignment.CenterVertically){
                Box(Modifier.size(44.dp).background(if(incoming)MaterialTheme.colorScheme.primaryContainer else MaterialTheme.colorScheme.surfaceVariant,RoundedCornerShape(15.dp)),contentAlignment=Alignment.Center){Icon(if(incoming)Icons.Outlined.SouthWest else Icons.Outlined.NorthEast,null,Modifier.size(20.dp),tint=if(incoming)MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant)}
                Column(Modifier.weight(1f).padding(start=12.dp)){
                    Row(verticalAlignment=Alignment.CenterVertically){
                        Column(Modifier.weight(1f).padding(end=10.dp)){
                            Text(record.name,style=MaterialTheme.typography.titleMedium,maxLines=1,overflow=TextOverflow.Ellipsis)
                            if(record.detail!=record.name)Text(record.detail,style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant,maxLines=1,overflow=TextOverflow.Ellipsis)
                        }
                        Column(horizontalAlignment=Alignment.End){
                            if(!stacked)Text(money,style=amountStyle,color=if(incoming)MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurface,maxLines=1,softWrap=false)
                            Text(record.time,style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
                        }
                    }
                    if(stacked)AdaptiveMoney(money,Modifier.padding(top=8.dp),if(incoming)MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurface,20.sp)
                }
            }
        }
    }
}
