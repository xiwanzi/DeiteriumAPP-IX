package com.deuterium.app.uilab

import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.*
import java.time.*
import java.time.format.DateTimeFormatter
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun BillHistoryPage(state:LabState,initial:String,topInset:Dp,onRecord:(LedgerEntry)->Unit) {
    val today=LocalDate.now(ZoneId.of("Asia/Shanghai"))
    var type by rememberSaveable { mutableStateOf(initial) }
    var start by rememberSaveable { mutableLongStateOf(if(initial=="all")today.minusDays(29).toEpochDay() else today.toEpochDay()) }
    var end by rememberSaveable { mutableLongStateOf(today.toEpochDay()) }
    var picker by remember { mutableStateOf(false) }
    val scope=rememberCoroutineScope()
    var remoteRows by remember { mutableStateOf<List<LedgerEntry>>(emptyList()) }
    var cursor by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var refresh by remember { mutableIntStateOf(0) }
    var generation by remember { mutableIntStateOf(0) }
    suspend fun load(more:Boolean=false) {
        val api=state.api ?: return
        val current=if(more)generation else ++generation
        loading=true;error=null
        try {
            val result=api.request("GET",billHistoryPath(start,end,type,today,if(more)cursor else null))
            val array=result.getJSONArray("records")
            val records=ledgerEntries((0 until array.length()).map{array.getJSONObject(it)})
            if(current==generation){
                remoteRows=(if(more)remoteRows+records else records).distinctBy{it.id}
                cursor=result.optJSONObject("_page")?.optString("nextCursor")?.takeUnless{it.isBlank()||it=="null"}
            }
        }catch(cancelled:CancellationException){throw cancelled}
        catch(failure:Exception){if(current==generation)error=failure.message ?: "账单读取失败，请重试"}
        finally{if(current==generation)loading=false}
    }
    LaunchedEffect(start,end,type,refresh){remoteRows=emptyList();cursor=null;load()}
    val rows by remember(state,start,end,type){derivedStateOf{(if(state.api==null)state.ledger else remoteRows).filter { it.at.toLocalDate().toEpochDay() in start..end && (type=="all" || if(type=="income")it.amount>0 else it.amount<0) }.sortedWith(compareByDescending<LedgerEntry>{it.at}.thenByDescending{it.id})}}
    val days=remember(rows){rows.groupBy{it.at.toLocalDate()}}
    val totals=remember(rows){rows.filter{it.amount>0}.sumOf{it.amount} to rows.filter{it.amount<0}.sumOf{-it.amount}}
    val format=DateTimeFormatter.ofPattern("MM月dd日")
    LazyColumn(contentPadding=PaddingValues(start=20.dp,end=20.dp,top=topInset,bottom=40.dp),verticalArrangement=Arrangement.spacedBy(10.dp)) {
        item { Row(horizontalArrangement=Arrangement.spacedBy(8.dp)) { listOf("all" to "全部","income" to "收入","expense" to "支出").forEach { (id,title)->ChoiceChip(type==id,{type=id},label={Text(title)}) } } }
        item { Row(Modifier.horizontalScroll(rememberScrollState()),horizontalArrangement=Arrangement.spacedBy(8.dp)) {
            listOf(1L to "今天",7L to "近7天",30L to "近30天",90L to "近90天").forEach { (days,title)->
                ChoiceChip(start==today.minusDays(days-1).toEpochDay() && end==today.toEpochDay(),{start=today.minusDays(days-1).toEpochDay();end=today.toEpochDay()},label={Text(title)})
            }
        } }
        item { Surface(onClick={picker=true},shape=RoundedCornerShape(16.dp),color=MaterialTheme.colorScheme.surface) {
            Row(Modifier.fillMaxWidth().padding(16.dp),horizontalArrangement=Arrangement.SpaceBetween){Text("${LocalDate.ofEpochDay(start).format(format)} — ${LocalDate.ofEpochDay(end).format(format)}");Text("选择范围",color=MaterialTheme.colorScheme.primary)}
        } }
        if(rows.isNotEmpty() || (!loading && error==null))item { LabCard {
            Text("${if(cursor!=null)"已加载 " else ""}${rows.size} 笔交易 · 北京时间",style=MaterialTheme.typography.labelMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)
            LedgerTotals(totals.first,totals.second)
        } }
        error?.let{message->item{Text(message,color=MaterialTheme.colorScheme.error);PlainButton({if(rows.isEmpty())refresh++ else scope.launch{load(true)}},enabled=!loading){Text("重新读取")}}}
        if(rows.isEmpty()&&!loading&&error==null)item { Text("所选范围内暂无账单",Modifier.fillMaxWidth().padding(vertical=50.dp),textAlign=androidx.compose.ui.text.style.TextAlign.Center,color=MaterialTheme.colorScheme.onSurfaceVariant) }
        days.forEach { (date,records)->
            item(key="date-$date"){Text(date.format(DateTimeFormatter.ofPattern("yyyy年MM月dd日")),Modifier.padding(top=18.dp,bottom=4.dp),style=MaterialTheme.typography.labelMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)}
            items(records,key={it.id}) { LedgerRow(it,{onRecord(it)}) }
        }
        if(loading)item{Text("正在读取账单…",Modifier.padding(16.dp),color=MaterialTheme.colorScheme.onSurfaceVariant)}
        if(cursor!=null)item{SecondaryButton({scope.launch{load(true)}},Modifier.fillMaxWidth(),enabled=!loading){Text("加载更多账单")}}
    }
    if(picker)CalendarRangeSheet(start,end,{picker=false}){from,to->start=from;end=to;picker=false}
}
