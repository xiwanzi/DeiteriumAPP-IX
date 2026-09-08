package com.deuterium.app.uilab

import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.*
import androidx.compose.foundation.shape.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.draw.clip
import androidx.compose.ui.Modifier
import androidx.compose.ui.Alignment
import androidx.compose.ui.unit.dp

val MarketCategories=listOf("建材","装备","补给","装饰","建筑服务","其他")
fun marketCategoryLabel(category:String):String=when(val root=category.substringBefore(" · ")){"建筑"->"建材";"工具"->"装备";"外观"->"装饰";else->if(root in MarketCategories)root else "其他"}
fun marketCategoryMatches(category:String,selected:String)=selected=="全部"||marketCategoryLabel(category)==marketCategoryLabel(selected)
@Composable
fun MarketCategoryBar(selected:String,onSelect:(String)->Unit) {
    var show by remember{mutableStateOf(false)}
    val normalized=if(selected=="全部")selected else marketCategoryLabel(selected)
    Row(Modifier.fillMaxWidth(),verticalAlignment=Alignment.CenterVertically){
        LazyRow(Modifier.weight(1f),horizontalArrangement=Arrangement.spacedBy(7.dp)){
            items(listOf("全部","建材","装备","建筑服务")){label->ChoiceChip(normalized==label,{onSelect(if(normalized==label)"全部" else label)},{Text(label)})}
        }
        IconButton({show=true}){Icon(Icons.Outlined.Tune,"筛选分类")}
    }
    if(show)CategorySheet(normalized,{show=false},allowClear=true){onSelect(it);show=false}
}
@Composable
fun CategorySheet(selected:String,onClose:()->Unit,allowClear:Boolean=false,onSelect:(String)->Unit) {
    val explanations=listOf("木石、玻璃与各类建筑材料","工具、武器、防具与附魔","矿物、食物、红石与日常补给","家具、地图、唱片与收藏","代建、装修、园林与红石工程","其他物品与玩家服务")
    IosSheet(onClose){LazyColumn(contentPadding=PaddingValues(22.dp),verticalArrangement=Arrangement.spacedBy(6.dp)){
        item{Row(Modifier.fillMaxWidth().padding(bottom=8.dp),verticalAlignment=Alignment.CenterVertically){Text("选择分类",Modifier.weight(1f),style=MaterialTheme.typography.titleLarge);PlainButton(onClose){Text("完成")}}}
        itemsIndexed(MarketCategories){index,category->
            val active=selected!="全部"&&marketCategoryLabel(selected)==category
            Row(Modifier.fillMaxWidth().clip(RoundedCornerShape(16.dp)).background(if(active)MaterialTheme.colorScheme.primaryContainer.copy(alpha=.6f) else MaterialTheme.colorScheme.surface).clickable{onSelect(if(allowClear&&active)"全部" else category)}.padding(14.dp),verticalAlignment=Alignment.CenterVertically){
                Column(Modifier.weight(1f)){Text(category,style=MaterialTheme.typography.titleMedium);Text(explanations[index],Modifier.padding(top=3.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)}
                if(active)Icon(Icons.Outlined.Check,null,tint=MaterialTheme.colorScheme.primary)
            }
        }
        if(allowClear)item{PlainButton({onSelect("全部")},Modifier.fillMaxWidth().padding(top=8.dp)){Text("清除筛选")}}
    }}
}
