package com.deuterium.app.uilab

import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.*

internal val demoPriceColors=listOf("蓝色" to Color(0xFF007AFF),"粉色" to Color(0xFFED6697),"黑色" to Color(0xFF161618),"红色" to Color(0xFFE53935),"橙色" to Color(0xFFEE7B22))

@Composable
internal fun DemoPricePalette(selected:Int,onSelect:(Int)->Unit,modifier:Modifier=Modifier) {
    var expanded by rememberSaveable{mutableStateOf(false)}
    Column(modifier,horizontalAlignment=Alignment.End,verticalArrangement=Arrangement.spacedBy(6.dp)) {
        if(expanded)Surface(shape=RoundedCornerShape(18.dp),color=MaterialTheme.colorScheme.surface,shadowElevation=5.dp){
            Row(Modifier.padding(6.dp)){demoPriceColors.forEachIndexed{index,(label,color)->
                Column(Modifier.width(52.dp).clickable{onSelect(index)}.padding(vertical=8.dp),horizontalAlignment=Alignment.CenterHorizontally,verticalArrangement=Arrangement.spacedBy(5.dp)){
                    Box(Modifier.size(30.dp).border(if(selected==index)3.dp else 1.dp,if(selected==index)MaterialTheme.colorScheme.onSurface else MaterialTheme.colorScheme.outlineVariant,CircleShape).padding(5.dp).background(color,CircleShape))
                    Text(label,fontSize=11.sp)
                }
            }}
        }
        Surface(onClick={expanded=!expanded},shape=CircleShape,color=MaterialTheme.colorScheme.surface,shadowElevation=4.dp){
            Row(Modifier.height(40.dp).padding(horizontal=12.dp),verticalAlignment=Alignment.CenterVertically,horizontalArrangement=Arrangement.spacedBy(7.dp)){
                Box(Modifier.size(12.dp).background(demoPriceColors[selected].second,CircleShape))
                Text(if(expanded)"收起配色"else"价格配色",fontSize=12.sp)
            }
        }
    }
}
