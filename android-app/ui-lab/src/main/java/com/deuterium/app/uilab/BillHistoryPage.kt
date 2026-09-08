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

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun BillHistoryPage(state:LabState,initial:String,topInset:Dp,onRecord:(LedgerEntry)->Unit) {
    val today=LocalDate.now()
    var type by rememberSaveable { mutableStateOf(initial) }
    var start by rememberSaveable { mutableLongStateOf(if(initial=="all")today.minusDays(29).toEpochDay() else today.toEpochDay()) }
    var end by rememberSaveable { mutableLongStateOf(today.toEpochDay()) }
    var picker by remember { mutableStateOf(false) }
    val rows=state.ledger.filter { it.at.toLocalDate().toEpochDay() in start..end && (type=="all" || if(type=="income")it.amount>0 else it.amount<0) }.sortedByDescending { it.at }
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
        item { LabCard {
            Text("${rows.size} 笔交易",style=MaterialTheme.typography.labelMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)
            Row(Modifier.fillMaxWidth().padding(top=12.dp),horizontalArrangement=Arrangement.SpaceBetween){
                Column { Text("收入",style=MaterialTheme.typography.bodySmall);Text("+${credit(rows.filter{it.amount>0}.sumOf{it.amount})}",style=MaterialTheme.typography.titleLarge,color=MaterialTheme.colorScheme.primary) }
                Column(horizontalAlignment=Alignment.End){Text("支出",style=MaterialTheme.typography.bodySmall);Text("−${credit(rows.filter{it.amount<0}.sumOf{-it.amount})}",style=MaterialTheme.typography.titleLarge)}
            }
        } }
        state.ledgerError?.let{error->item{Text(error,color=MaterialTheme.colorScheme.error)}}
        if(rows.isEmpty())item { Text("所选范围内暂无账单",Modifier.fillMaxWidth().padding(vertical=50.dp),textAlign=androidx.compose.ui.text.style.TextAlign.Center,color=MaterialTheme.colorScheme.onSurfaceVariant) }
        rows.groupBy{it.at.toLocalDate()}.forEach { (date,records)->
            item(key="date-$date"){Text(date.format(DateTimeFormatter.ofPattern("yyyy年MM月dd日")),Modifier.padding(top=18.dp,bottom=4.dp),style=MaterialTheme.typography.labelMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)}
            items(records,key={it.id}) { LedgerRow(it,{onRecord(it)}) }
        }
    }
    if(picker)CalendarRangeSheet(start,end,{picker=false}){from,to->start=from;end=to;picker=false}
}
