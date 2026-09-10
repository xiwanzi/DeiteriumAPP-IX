package com.deuterium.app.uilab

import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.*
import androidx.compose.foundation.shape.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.focus.*
import androidx.compose.ui.platform.LocalSoftwareKeyboardController
import androidx.compose.ui.unit.*
import kotlinx.coroutines.delay

@Composable
fun SearchPage(area:String,state:LabState,onClose:()->Unit,onOpen:(String)->Unit,active:Boolean=true) {
    var query by rememberSaveable{mutableStateOf("")};val focus=remember{FocusRequester()};val keyboard=LocalSoftwareKeyboardController.current
    var directoryMatches by remember(area,query){mutableStateOf(emptyList<PlayerProfile>())}
    var directorySearching by remember(area,query){mutableStateOf(area=="Info"&&query.isNotBlank())}
    var directoryError by remember(area,query){mutableStateOf<String?>(null)}
    LaunchedEffect(area,query,active){
        if(area!="Info"||query.isBlank()||!active)return@LaunchedEffect
        directorySearching=true
        try{delay(250);directoryMatches=state.searchDirectory(query)}
        catch(cancelled:kotlinx.coroutines.CancellationException){throw cancelled}
        catch(error:Exception){directoryError=error.message ?: "搜索暂时不可用"}
        finally{directorySearching=false}
    }
    LaunchedEffect(active){if(active){delay(180);focus.requestFocus();withFrameNanos{};delay(80);keyboard?.show()}}
    fun open(route:String){keyboard?.hide();onOpen(route)}
    Column(Modifier.fillMaxSize().background(MaterialTheme.colorScheme.background).statusBarsPadding().imePadding().navigationBarsPadding()) {
        Row(Modifier.fillMaxWidth().padding(start=16.dp,end=6.dp,top=10.dp,bottom=12.dp),verticalAlignment=Alignment.CenterVertically){
            RefinedField(query,{query=if(area=="Info")it.take(80) else it},Modifier.weight(1f).focusRequester(focus),placeholder={Text(when(area){"Shop"->"搜索商品或品牌";"Market"->"搜索物品、服务或玩家";"Info"->"搜索联系人";else->"搜索设置"},style=MaterialTheme.typography.bodyMedium)},singleLine=true,leadingIcon={Icon(Icons.Outlined.Search,null,Modifier.size(21.dp))},trailingIcon=if(query.isNotBlank()){{IconButton({query=""},Modifier.size(30.dp)){Icon(Icons.Outlined.Cancel,"清除搜索",Modifier.size(18.dp))}}}else null)
            PlainButton({keyboard?.hide();onClose()}){Text("取消")}
        }
        LazyColumn(Modifier.fillMaxSize(),contentPadding=PaddingValues(horizontal=16.dp,vertical=8.dp),verticalArrangement=Arrangement.spacedBy(12.dp)) {
            if(query.isBlank()) {
                item{Text("搜索${when(area){"Shop"->"商城";"Market"->"市场";"Info"->"联系人";else->"设置"}}",style=MaterialTheme.typography.headlineSmall)}
                item{Text(if(area=="Info")"输入玩家 ID 或 QQ，搜索全部玩家" else if(area=="Shop"&&ShopCatalog.isEmpty())"输入商品名称或分类" else "试试这些关键词",Modifier.padding(top=10.dp),style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)}
                val suggestions=when(area){"Shop"->shopSearchSuggestions(ShopCatalog);"Market"->listOf("建筑服务","石材","工具","附魔");"Info"->recentContacts(Players,state.followed.toSet(),state.directChats).take(3).map{it.person.name};else->listOf("外观","个人简介","订单","通知")}
                item{FlowRow(horizontalArrangement=Arrangement.spacedBy(8.dp),verticalArrangement=Arrangement.spacedBy(10.dp)){suggestions.forEach{ChoiceChip(false,{query=it},{Text(it)})}}}
            } else when(area) {
                "Shop"->{val products=ShopCatalog.filter{it.name.contains(query,true)||it.category.contains(query,true)||it.brand.contains(query,true)}
                    if(products.isEmpty())item{SearchEmpty()}
                    items(products,key={it.id}){product->SearchResult(product.name,"${product.brand} · ${credit(product.price)} 信用点",{OrderThumbnail(OrderLine(product.id,product.name,product.subtitle,product.price,1,product.image,imageUri=product.photos.firstOrNull(),imageUris=product.photos),Modifier.size(63.dp,73.dp))}){open("product:${product.id}")}}
                }
                "Market"->{val listings=state.commerce.listings.filter{it.active&&(it.title.contains(query,true)||it.category.contains(query,true)||it.seller.contains(query,true))}
                    if(listings.isEmpty())item{SearchEmpty()}
                    items(listings,key={it.id}){listing->SearchResult(listing.title,"${listing.seller} · ${credit(listing.price)} 信用点",{ListingImage(listing,Modifier.size(68.dp).clip(RoundedCornerShape(15.dp)))}){open("market-product:${listing.id}")}}
                }
                "Info"->{
                    val people=recentContacts(directoryMatches+searchPlayers(query),state.followed.toSet(),state.directChats,searchMode=true).map{it.person}
                    if(people.isEmpty()){item{if(directorySearching)Text("正在搜索玩家…",Modifier.padding(20.dp),color=MaterialTheme.colorScheme.onSurfaceVariant) else if(directoryError!=null)Text(directoryError!!,Modifier.padding(20.dp),color=MaterialTheme.colorScheme.onSurfaceVariant) else SearchEmpty()}}
                    items(people,key={it.playerRef.ifBlank{it.name}}){person->SearchResult(person.name,"QQ ${person.qq}",{PlayerAvatar(person.name,Modifier.size(47.dp))}){open("player:${person.name}")}}
                    if(people.isNotEmpty()&&directoryError!=null)item{Text("暂时无法联网搜索，已显示本机已有结果",Modifier.padding(12.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)}
                }
                else->{val options=listOf(Triple("外观","浅色 深色 柔光玻璃 灵动视效 动态追光 材质细节","appearance"),Triple("个人简介","资料 签名 自我介绍","bio"),Triple("我的订单","退款 购买 收货 验收","orders"),Triple("我的委托","发布 接取 任务 报酬","my-commissions"),Triple("软件更新","关于 版本 APK 资源 下载","about"),Triple("存储空间","缓存 清理 图片 更新 文件 占用","storage"),Triple("账号与安全","头像 密码 账号","account"),Triple("通知","提醒 消息","notifications"),Triple("我的优惠","优惠券 折扣 满减","coupons"),Triple("我的钱包","账单 收支 历史 余额 转账","wallet"))
                    val results=options.filter{it.first.contains(query,true)||it.second.contains(query,true)};if(results.isEmpty())item{SearchEmpty()}
                    items(results){entry->SettingsGroup{SettingsRow(entry.first,Icons.Outlined.ChevronRight){open(entry.third)}}}
                }
            }
        }
    }
}
@Composable
private fun SearchEmpty(){Text("没有找到相关结果",Modifier.fillMaxWidth().padding(vertical=40.dp),textAlign=androidx.compose.ui.text.style.TextAlign.Center,color=MaterialTheme.colorScheme.onSurfaceVariant)}
@Composable
private fun SearchResult(title:String,subtitle:String,art:@Composable ()->Unit,onClick:()->Unit){Surface(onClick=onClick,shape=RoundedCornerShape(20.dp),color=MaterialTheme.colorScheme.surface){Row(Modifier.fillMaxWidth().padding(14.dp),verticalAlignment=Alignment.CenterVertically){art();Column(Modifier.weight(1f).padding(start=14.dp)){Text(title,style=MaterialTheme.typography.titleMedium);Text(subtitle,Modifier.padding(top=5.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)};Icon(Icons.Outlined.ChevronRight,null,Modifier.size(18.dp),tint=MaterialTheme.colorScheme.onSurfaceVariant)}}}
