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
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.*
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.io.File
import java.time.*
import java.time.format.DateTimeFormatter
import java.util.Locale

fun fileSize(bytes:Long):String=if(bytes<1024*1024)String.format(Locale.US,"%.0f KB",bytes/1024.0) else String.format(Locale.US,"%.1f MB",bytes/1048576.0)

@Composable
fun UpdatePage(state:LabState,topInset:Dp) {
    val updates=LocalAppUpdates.current ?: return
    LaunchedEffect(updates){updates.check()}
    var reset by remember{mutableStateOf(false)}
    LazyColumn(contentPadding=PaddingValues(start=18.dp,end=18.dp,top=topInset,bottom=45.dp),verticalArrangement=Arrangement.spacedBy(22.dp)) {
        item{Column(Modifier.fillMaxWidth().padding(top=16.dp,bottom=8.dp),horizontalAlignment=Alignment.CenterHorizontally){
            LoginVersionBadge(Modifier.size(83.dp))
            Text("Deuterium APP",Modifier.padding(top=20.dp),style=MaterialTheme.typography.headlineSmall)
            Text(updates.currentVersionName,Modifier.padding(top=8.dp),style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)
            Text(if(updates.checking)"正在检查更新…" else if(updates.checkError!=null)"暂时无法检查更新" else if(updates.hasUpdates)"有更新可用" else if(updates.message.contains("无法")||updates.message.contains("失败")||updates.message.contains("不可用"))"暂时无法检查更新" else "当前已是最新版本",Modifier.padding(top=9.dp),style=MaterialTheme.typography.bodyMedium,color=if(updates.hasUpdates)MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant)
        }}
        updates.releases.forEach{release->item{UpdateReleaseCard(updates,release)}}
        if(updates.active!=null&&updates.releases.none{it.id==updates.active?.id}&&updates.stage in listOf(UpdateStage.Failed,UpdateStage.Ready,UpdateStage.Downloading,UpdateStage.Verifying,UpdateStage.Applying))item{UpdateReleaseCard(updates,updates.active!!)}
        item{SettingsGroup{
            Row(Modifier.fillMaxWidth().padding(18.dp),verticalAlignment=Alignment.CenterVertically){Icon(Icons.Outlined.Layers,null,Modifier.size(22.dp),tint=MaterialTheme.colorScheme.primary);Text("资源版本",Modifier.weight(1f).padding(start=12.dp),style=MaterialTheme.typography.bodyLarge);Text(if(updates.resourceVersion==0)"随包资源" else "${updates.resourceVersion}",style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)}
            SettingsDivider();PlainButton({updates.check()},Modifier.fillMaxWidth().height(53.dp),enabled=!updates.checking&&!updates.busy){Text(if(updates.checking)"正在检查…" else "检查更新")}
        }}
        if(updates.stage==UpdateStage.Applied)item{Row(Modifier.fillMaxWidth().padding(horizontal=12.dp),verticalAlignment=Alignment.CenterVertically){Icon(Icons.Outlined.CheckCircle,null,Modifier.size(19.dp),tint=MaterialTheme.colorScheme.primary);Text(updates.message,Modifier.padding(start=9.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)}}
        item{Column(Modifier.padding(horizontal=14.dp),verticalArrangement=Arrangement.spacedBy(9.dp)){
            Text("应用更新包含完整 APK；资源更新用于图片、文案和配置，无需重新安装。",style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
            updates.checkError?.let{Text(it,style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.error)}
            if(updates.lastChecked>0)Text("上次检查 "+Instant.ofEpochMilli(updates.lastChecked).atZone(ZoneId.systemDefault()).format(DateTimeFormatter.ofPattern("MM-dd HH:mm")),style=MaterialTheme.typography.labelSmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
            if(!updates.checking&&!updates.hasUpdates&&updates.stage!=UpdateStage.Applied&&updates.message!="当前已是最新版本")Text(updates.message,style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
        }}
        item{SettingsGroup{SettingsRow("版本说明",Icons.Outlined.Info,onClick={reset=true})}}
    }
    if(reset)IosSheet({reset=false}){Column(Modifier.verticalScroll(rememberScrollState()).padding(24.dp)){
        Text("Deuterium · ${BuildConfig.VERSION_NAME}",style=MaterialTheme.typography.titleLarge)
        Text("连接你的 Deuterium 社区、消息与游戏资产。支付确认画面不采集人脸，交易是否完成以服务器返回结果为准。",Modifier.padding(top=15.dp),style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)
        Text("账号、消息与交易记录由服务器保存，外观设置保存在当前设备。",Modifier.padding(top=12.dp),style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)
        state.storageMessage?.let{Text(it,Modifier.padding(top=12.dp),color=MaterialTheme.colorScheme.error)}

    }}
}

@Composable
private fun UpdateReleaseCard(updates:AppUpdates,release:AppRelease) {
    val active=updates.active?.id==release.id
    val stage=if(active)updates.stage else UpdateStage.Idle
    LabCard{
        Row(verticalAlignment=Alignment.CenterVertically){Box(Modifier.size(47.dp).background(MaterialTheme.colorScheme.primaryContainer,RoundedCornerShape(13.dp)),contentAlignment=Alignment.Center){Icon(if(release.kind==UpdateKind.Apk)Icons.Outlined.SystemUpdate else Icons.Outlined.Layers,null,Modifier.size(25.dp),tint=MaterialTheme.colorScheme.primary)}
            Column(Modifier.weight(1f).padding(start=13.dp)){Text(if(release.kind==UpdateKind.Apk)"Deuterium ${release.versionName}" else "内容资源更新",style=MaterialTheme.typography.titleMedium);Text("${release.versionName} · ${fileSize(release.sizeBytes)}",Modifier.padding(top=3.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)}}
        Text(release.notes,Modifier.padding(top=18.dp,bottom=20.dp),style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)
        if(active&&stage in listOf(UpdateStage.Downloading,UpdateStage.Verifying,UpdateStage.Applying)){
            val fraction=(updates.bytes.toFloat()/release.sizeBytes).coerceIn(0f,1f)
            Box(Modifier.fillMaxWidth().height(5.dp).background(MaterialTheme.colorScheme.outlineVariant.copy(alpha=.5f),CircleShape)){Box(Modifier.fillMaxWidth(fraction).fillMaxHeight().background(MaterialTheme.colorScheme.primary,CircleShape))}
            Row(Modifier.fillMaxWidth().padding(top=10.dp),horizontalArrangement=Arrangement.SpaceBetween){Text(if(stage==UpdateStage.Downloading)"${fileSize(updates.bytes)} / ${fileSize(release.sizeBytes)}" else updates.message,style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant);Text("${(fraction*100).toInt()}%",style=MaterialTheme.typography.bodySmall.copy(fontFeatureSettings="tnum"),color=MaterialTheme.colorScheme.primary)}
            if(stage==UpdateStage.Downloading)PlainButton({updates.cancel()},Modifier.align(Alignment.End)){Text("取消下载",style=MaterialTheme.typography.bodySmall)}
        }else{
            if(active&&stage==UpdateStage.Failed)Text(updates.message,Modifier.padding(bottom=13.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.error)
            if(active&&stage==UpdateStage.Ready)Text(updates.message,Modifier.padding(bottom=13.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
            MotionButton({if(active&&stage==UpdateStage.Ready){if(release.kind==UpdateKind.Apk)updates.install() else updates.applyReadyResources()}else updates.start(release)},Modifier.fillMaxWidth().height(50.dp),enabled=!updates.busy){Text(when{active&&stage==UpdateStage.Ready->if(release.kind==UpdateKind.Apk)"安装更新" else "完成资源更新";active&&stage==UpdateStage.Failed->"重新下载";release.kind==UpdateKind.Apk->"下载更新包";else->"下载并更新资源"})}
        }
    }
}
