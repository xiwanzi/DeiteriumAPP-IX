package com.deuterium.app.uilab

import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.rememberTextMeasurer
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.*

@Composable
fun AdaptiveMoney(value:String,modifier:Modifier=Modifier,color:Color=MaterialTheme.colorScheme.onSurface,size:TextUnit=28.sp){
    val measurer=rememberTextMeasurer();val density=LocalDensity.current
    val style=MaterialTheme.typography.titleLarge.copy(fontSize=size,fontWeight=FontWeight.SemiBold,fontFeatureSettings="tnum",letterSpacing=(-0.4).sp)
    BoxWithConstraints(modifier){
        val width=with(density){maxWidth.toPx()};val measured=measurer.measure(AnnotatedString(value),style,maxLines=1,softWrap=false).size.width
        val fitted=if(measured>width&&measured>0)size*(width/measured) else size
        Text(value,Modifier.fillMaxWidth(),style=style.copy(fontSize=fitted),color=color,maxLines=1,softWrap=false)
    }
}

@Composable
fun LedgerTotals(income:Long,expense:Long){
    val creditText="+${credit(income)}";val debitText="−${credit(expense)}"
    val measure=rememberTextMeasurer();val density=LocalDensity.current
    val style=MaterialTheme.typography.titleLarge.copy(fontSize=24.sp,fontWeight=FontWeight.SemiBold)
    BoxWithConstraints(Modifier.fillMaxWidth().padding(top=16.dp)){
        val available=with(density){(maxWidth-24.dp).toPx()}/2
        val stacked=listOf(creditText,debitText).any{measure.measure(AnnotatedString(it),style,maxLines=1,softWrap=false).size.width>available}
        val cell:@Composable (String,String,Boolean,Modifier)->Unit={label,text,incoming,modifier->Column(modifier){
            Text(label,style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
            AdaptiveMoney(text,Modifier.padding(top=6.dp).testTag(if(incoming)"ledger-income" else "ledger-expense"),if(incoming)MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurface,if(stacked)28.sp else 24.sp)
        }}
        if(stacked)Column(verticalArrangement=Arrangement.spacedBy(18.dp)){
            cell("收入",creditText,true,Modifier.fillMaxWidth())
            HorizontalDivider(color=MaterialTheme.colorScheme.outlineVariant.copy(alpha=.45f))
            cell("支出",debitText,false,Modifier.fillMaxWidth())
        }else Row(horizontalArrangement=Arrangement.spacedBy(24.dp)){
            cell("收入",creditText,true,Modifier.weight(1f));cell("支出",debitText,false,Modifier.weight(1f))
        }
    }
}
