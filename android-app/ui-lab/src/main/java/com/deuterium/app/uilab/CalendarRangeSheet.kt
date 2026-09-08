package com.deuterium.app.uilab

import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.*
import java.time.*

@Composable
fun CalendarRangeSheet(initialStart:Long,initialEnd:Long,onClose:()->Unit,onApply:(Long,Long)->Unit) {
    var start by remember{mutableStateOf<LocalDate?>(LocalDate.ofEpochDay(initialStart))}
    var end by remember{mutableStateOf<LocalDate?>(LocalDate.ofEpochDay(initialEnd))}
    var month by remember{mutableStateOf(YearMonth.from(start))}
    var selectingEnd by remember{mutableStateOf(false)}
    IosSheet(onClose) { Column(Modifier.fillMaxWidth().padding(horizontal=20.dp).padding(bottom=24.dp)) {
        Row(Modifier.fillMaxWidth(),verticalAlignment=Alignment.CenterVertically){Text("选择日期范围",Modifier.weight(1f),style=MaterialTheme.typography.titleLarge);PlainButton(onClose){Text("取消")}}
        Row(Modifier.fillMaxWidth().padding(vertical=18.dp),horizontalArrangement=Arrangement.spacedBy(12.dp)) {
            listOf("开始日期" to start,"结束日期" to end).forEachIndexed{index,(label,date)->
                Surface(onClick={selectingEnd=index==1},modifier=Modifier.weight(1f),shape=MaterialTheme.shapes.medium,color=if(selectingEnd==(index==1))MaterialTheme.colorScheme.primaryContainer else MaterialTheme.colorScheme.surfaceVariant.copy(alpha=.45f)){Column(Modifier.padding(14.dp)){Text(label,style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant);Text(date?.toString() ?: "请选择",Modifier.padding(top=5.dp),style=MaterialTheme.typography.titleMedium)}}
            }
        }
        Row(Modifier.fillMaxWidth(),verticalAlignment=Alignment.CenterVertically){IconButton({month=month.minusMonths(1)}){Icon(Icons.Outlined.ChevronLeft,"上个月")};Text("${month.year}年 ${month.monthValue}月",Modifier.weight(1f),textAlign=androidx.compose.ui.text.style.TextAlign.Center,style=MaterialTheme.typography.titleMedium);IconButton({month=month.plusMonths(1)}){Icon(Icons.Outlined.ChevronRight,"下个月")}}
        Row(Modifier.fillMaxWidth()){listOf("一","二","三","四","五","六","日").forEach{Box(Modifier.weight(1f).height(36.dp),contentAlignment=Alignment.Center){Text(it,style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)}}}
        val pad=month.atDay(1).dayOfWeek.value-1
        repeat((pad+month.lengthOfMonth()+6)/7){week->Row(Modifier.fillMaxWidth()) {
            repeat(7){day->val number=week*7+day-pad+1
                Box(Modifier.weight(1f).height(46.dp),contentAlignment=Alignment.Center){if(number in 1..month.lengthOfMonth()) {
                    val date=month.atDay(number);val selected=date==start||date==end
                    val inRange=start!=null&&end!=null&&date>=start&&date<=end
                    if(inRange)Box(Modifier.fillMaxWidth().height(38.dp).background(MaterialTheme.colorScheme.primaryContainer))
                    Box(Modifier.size(39.dp).background(if(selected)MaterialTheme.colorScheme.primary else Color.Transparent,CircleShape).clickable{
                        if(!selectingEnd||start==null||date<start){start=date;end=null;selectingEnd=true}else{end=date;selectingEnd=false}
                    },contentAlignment=Alignment.Center){Text(number.toString(),color=if(selected)Color.White else MaterialTheme.colorScheme.onSurface,style=MaterialTheme.typography.bodyLarge)}
                }}
            }
        }}
        MotionButton({onApply(start!!.toEpochDay(),end!!.toEpochDay())},Modifier.fillMaxWidth().padding(top=24.dp).height(50.dp),enabled=start!=null&&end!=null){Text("应用范围")}
    } }
}
