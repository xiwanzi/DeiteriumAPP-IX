package com.deuterium.app.uilab

import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.semantics.stateDescription
import androidx.compose.ui.unit.*

@Composable
fun SettingsRow(title:String,icon:ImageVector,color:Color=MaterialTheme.colorScheme.primary,detail:String="",badge:Boolean=false,onClick:()->Unit) {
    Row(Modifier.fillMaxWidth().semantics{if(badge)stateDescription="有新内容"}.clickable(onClick=onClick).padding(horizontal=18.dp,vertical=15.dp),verticalAlignment=Alignment.CenterVertically) {
        Box(Modifier.size(31.dp).background(color,RoundedCornerShape(8.dp)),contentAlignment=Alignment.Center){Icon(icon,null,Modifier.size(21.dp),tint=Color.White)}
        Text(title,Modifier.weight(1f).padding(horizontal=13.dp),style=MaterialTheme.typography.bodyLarge)
        if(badge)Box(Modifier.padding(end=8.dp).size(7.dp).background(MaterialTheme.colorScheme.error,CircleShape))
        if(detail.isNotBlank())Text(detail,style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)
        Icon(Icons.Outlined.ChevronRight,null,Modifier.padding(start=6.dp).size(20.dp),tint=MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha=.55f))
    }
}
@Composable
fun SettingsGroup(content:@Composable ColumnScope.()->Unit){Surface(shape=RoundedCornerShape(24.dp),color=MaterialTheme.colorScheme.surface,modifier=Modifier.fillMaxWidth()){Column(content=content)}}
@Composable
fun SettingsDivider(){HorizontalDivider(Modifier.padding(start=62.dp),color=MaterialTheme.colorScheme.outlineVariant.copy(alpha=.5f),thickness=.5.dp)}

@Composable
fun ProfilePage(state:LabState,avatar:String?,userName:String,topInset:Dp,query:String,onOpen:(String)->Unit,onAvatar:()->Unit) {
    fun match(vararg words:String)=query.isBlank()||words.any{it.contains(query,true)}
    LazyColumn(contentPadding=PaddingValues(start=16.dp,end=16.dp,top=topInset,bottom=115.dp),verticalArrangement=Arrangement.spacedBy(22.dp)) {
        if(!match("账号","头像","个人资料",userName,"钱包","余额","转账","订单","商城订单","市场订单","退款","发布","出售","上架","账单","收支","历史","优惠","优惠券","折扣","满减","外观","浅色","深色","自动","玻璃","动效","通知","消息","提醒","安全","密码","存储","空间","缓存","清理","关于","版本"))item{Text("没有找到相关设置",Modifier.fillMaxWidth().padding(vertical=36.dp),textAlign=androidx.compose.ui.text.style.TextAlign.Center,color=MaterialTheme.colorScheme.onSurfaceVariant)}
        if(match("账号","头像","个人资料",userName))item { SettingsGroup {
            Row(Modifier.fillMaxWidth().clickable{onOpen("account")}.padding(20.dp),verticalAlignment=Alignment.CenterVertically){LocalAvatar(avatar,userName,Modifier.size(63.dp).clickable(onClick=onAvatar));Column(Modifier.weight(1f).padding(start=16.dp)){Text(userName,style=MaterialTheme.typography.titleLarge);Text(state.profileBio.ifBlank{"添加个人简介"},Modifier.padding(top=5.dp).clickable{onOpen("bio")},maxLines=1,overflow=androidx.compose.ui.text.style.TextOverflow.Ellipsis,style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)};Icon(Icons.Outlined.ChevronRight,null,tint=MaterialTheme.colorScheme.onSurfaceVariant)}
        } }
        item { SettingsGroup {
            if(match("钱包","余额","转账","账单","收支","历史"))SettingsRow("我的钱包",Icons.Outlined.AccountBalanceWallet,Color(0xFF222226),if(state.balanceKnown)credit(state.balance) else "等待同步"){onOpen("wallet")}
            if(query.isBlank())SettingsDivider()
            if(match("订单","商城订单","市场订单","退款"))SettingsRow("我的订单",Icons.Outlined.ShoppingBag,Color(0xFF007AFF)){onOpen("orders")}
            if(query.isBlank())SettingsDivider()
            if(match("委托","接取"))SettingsRow("我的委托",DeuteriumIcons.Commission,Color(0xFF7B73BC)){onOpen("my-commissions")}
            if(query.isBlank())SettingsDivider()
            if(match("发布","出售","上架"))SettingsRow("我发布的",Icons.Outlined.Inventory2,Color(0xFFFF9500)){onOpen("listings")}
            if(query.isBlank())SettingsDivider()
            if(match("优惠","优惠券","折扣","满减"))SettingsRow("我的优惠",Icons.Outlined.ConfirmationNumber,Color(0xFF34AADC),badge=state.commerce.network?.couponAttention?.hasUnread==true){onOpen("coupons")}
        } }
        item { SettingsGroup {
            if(match("外观","浅色","深色","自动","玻璃","动效"))SettingsRow("外观",Icons.Outlined.Contrast,Color(0xFF8E6BE8)){onOpen("appearance")}
            if(query.isBlank())SettingsDivider()
            if(match("通知","消息","提醒"))SettingsRow("通知",Icons.Outlined.NotificationsNone,Color(0xFFFF453A)){onOpen("notifications")}
            if(query.isBlank())SettingsDivider()
            if(match("存储","空间","缓存","清理"))SettingsRow("存储空间",Icons.Outlined.Storage,Color(0xFF8E8E93)){onOpen("storage")}
            if(query.isBlank())SettingsDivider()
            if(match("账号","安全","密码"))SettingsRow("账号与安全",Icons.Outlined.Lock,Color(0xFF34C759)){onOpen("account")}
        } }
        if(match("关于","版本"))item { SettingsGroup { SettingsRow("关于 Deuterium APP",Icons.Outlined.Info,Color(0xFF8E8E93),badge=LocalAppUpdates.current?.hasUpdates==true){onOpen("about")} } }
    }
}

@Composable
fun AppearancePage(theme:Int,motion:Boolean,glass:Boolean,tilt:Boolean,topInset:Dp,onTheme:(Int)->Unit,onMotion:(Boolean)->Unit,onGlass:(Boolean)->Unit,onTilt:(Boolean)->Unit,onTune:()->Unit,sensorAvailable:Boolean,overlayGlass:Boolean,onOverlayGlass:(Boolean)->Unit) {
    LazyColumn(contentPadding=PaddingValues(start=16.dp,end=16.dp,top=topInset,bottom=40.dp),verticalArrangement=Arrangement.spacedBy(22.dp)) {
        item { SettingsGroup {
            Row(Modifier.fillMaxWidth().padding(22.dp),horizontalArrangement=Arrangement.spacedBy(28.dp)) {
                listOf("浅色","深色").forEachIndexed { i,label->val dark=i==1
                    Column(Modifier.weight(1f).clickable{onTheme(i+1)},horizontalAlignment=Alignment.CenterHorizontally) {
                        Column(Modifier.size(86.dp,146.dp).clip(RoundedCornerShape(15.dp)).background(if(dark)Color(0xFF080809) else Color(0xFFF2F2F7)).border(1.dp,MaterialTheme.colorScheme.outlineVariant,RoundedCornerShape(15.dp)).padding(9.dp),verticalArrangement=Arrangement.spacedBy(7.dp)) {
                            Box(Modifier.align(Alignment.CenterHorizontally).width(25.dp).height(5.dp).background(if(dark)Color(0xFF333336) else Color(0xFFBFC0C5),CircleShape))
                            Spacer(Modifier.height(9.dp));repeat(3){Box(Modifier.fillMaxWidth().height(20.dp).background(if(dark)Color(0xFF2C2C2E) else Color.White,RoundedCornerShape(5.dp)))}
                        }
                        Text(label,Modifier.padding(top=13.dp),style=MaterialTheme.typography.bodyLarge)
                        Box(Modifier.padding(top=10.dp).size(23.dp).background(if(theme==i+1)MaterialTheme.colorScheme.primary else Color.Transparent,CircleShape).border(1.dp,if(theme==i+1)Color.Transparent else MaterialTheme.colorScheme.outlineVariant,CircleShape),contentAlignment=Alignment.Center){if(theme==i+1)Icon(Icons.Outlined.Check,null,Modifier.size(16.dp),tint=Color.White)}
                    }
                }
            }
            SettingsDivider();ToggleRow("自动","跟随系统的浅色与深色外观",theme==0,{onTheme(if(it)0 else 1)})
        } }
        item { SettingsGroup {
            ToggleRow("底栏柔光玻璃","底部导航的模糊、折射与高光",glass,onGlass)
            SettingsDivider();ToggleRow("浮层柔光玻璃","转账、付款、改密等浮层卡片",overlayGlass,onOverlayGlass)
            SettingsDivider();ToggleRow("动态追光",if(sensorAvailable)"轻轻转动手机即可看到变化" else "当前设备无可用传感器",tilt,onTilt,sensorAvailable)
            SettingsDivider();ToggleRow("灵动视效","导航、购物袋和页面过渡",motion,onMotion)
            SettingsDivider();SettingsRow("材质细节",Icons.Outlined.BlurOn,Color(0xFF8E8E93),onClick=onTune)
        } }
    }
}
@Composable
fun ToggleRow(title:String,subtitle:String,checked:Boolean,onChange:(Boolean)->Unit,enabled:Boolean=true) {
    Row(Modifier.fillMaxWidth().padding(horizontal=18.dp,vertical=13.dp),verticalAlignment=Alignment.CenterVertically){Column(Modifier.weight(1f).padding(end=12.dp)){Text(title,style=MaterialTheme.typography.bodyLarge);if(subtitle.isNotBlank())Text(subtitle,Modifier.padding(top=4.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)};IosSwitch(checked,onChange,enabled=enabled)}
}
@Composable
fun AccountPage(avatar:String?,userName:String,topInset:Dp,onAvatar:()->Unit,onPassword:()->Unit,onLogout:()->Unit,onBio:()->Unit={}) {
    var confirm by remember{mutableStateOf(false)}
    LazyColumn(contentPadding=PaddingValues(start=16.dp,end=16.dp,top=topInset,bottom=50.dp),verticalArrangement=Arrangement.spacedBy(22.dp)) {
        item { Column(Modifier.fillMaxWidth().padding(vertical=20.dp),horizontalAlignment=Alignment.CenterHorizontally){LocalAvatar(avatar,userName,Modifier.size(92.dp).clickable(onClick=onAvatar));PlainButton(onAvatar){Text("更换头像")}} }
        item { SettingsGroup { Box(Modifier.padding(horizontal=18.dp,vertical=7.dp)){DetailRow("玩家 ID",userName)};SettingsDivider();SettingsRow("修改密码",Icons.Outlined.Key,Color(0xFF007AFF),onClick=onPassword);SettingsDivider();SettingsRow("个人简介",Icons.Outlined.Edit,Color(0xFF8E8E93),onClick=onBio) } }
        item { SettingsGroup { PlainButton({confirm=true},Modifier.fillMaxWidth().height(54.dp)){Text("退出登录",color=MaterialTheme.colorScheme.error)} } }
    }
    if(confirm)IosDialog({confirm=false},{Text("退出登录？")},{Text("头像和外观设置会保留。")},{PlainButton(onLogout){Text("退出",color=MaterialTheme.colorScheme.error)}},{PlainButton({confirm=false}){Text("取消")}})
}
@Composable
fun AboutPage(state:LabState,topInset:Dp) { UpdatePage(state,topInset) }
