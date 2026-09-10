package com.deuterium.app.uilab

import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.snap
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.drawscope.rotate
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.launch

private val StorageBlue=Color(0xFF007AFF)
private val StoragePurple=Color(0xFFAF52DE)
private val StorageTeal=Color(0xFF30B0C7)
private val StorageGray=Color(0xFF8E8E93)

@Composable
fun StoragePage(topInset:Dp){
    val context=LocalContext.current
    val updates=LocalAppUpdates.current
    val scope=rememberCoroutineScope()
    var stats by remember{mutableStateOf<StorageSnapshot?>(null)}
    var scanning by remember{mutableStateOf(true)}
    var clearing by remember{mutableStateOf(false)}
    var message by remember{mutableStateOf<String?>(null)}
    var selection by remember{mutableStateOf<CacheSelection?>(null)}
    fun refresh(){scope.launch{scanning=true;try{stats=AppStorage.measure(context)}catch(cancelled:CancellationException){throw cancelled}catch(_:Exception){message="暂时无法统计，请稍后重试"}finally{scanning=false}}}
    LaunchedEffect(Unit){try{stats=AppStorage.measure(context)}catch(cancelled:CancellationException){throw cancelled}catch(_:Exception){message="暂时无法统计，请稍后重试"}finally{scanning=false}}
    val updateProtected=updates?.downloadsProtected==true
    val estimate=when(selection){CacheSelection.Images->stats?.images;CacheSelection.Updates->stats?.updates;CacheSelection.Temporary->stats?.temporary;CacheSelection.All->stats?.let{it.cache-if(updateProtected)it.updates else 0};null->0} ?: 0
    StorageContent(topInset,stats,scanning,clearing,updateProtected,message,::refresh){selection=it}
    selection?.let{chosen->
        IosDialog({if(!clearing)selection=null},{Text("清理缓存？")},{Text("预计释放 ${StorageFiles.display(estimate)}。图片会在需要时重新下载，账号和交易记录会保留。")},
            {PlainButton({
                selection=null;clearing=true;message=null
                scope.launch{
                    try{val freed=AppStorage.clear(context,chosen,updates);stats=AppStorage.measure(context);message="已释放 ${StorageFiles.display(freed)}"}
                    catch(cancelled:CancellationException){throw cancelled}
                    catch(error:Exception){message=error.message ?: "部分缓存暂时无法清理";runCatching{stats=AppStorage.measure(context)}}
                    finally{clearing=false}
                }
            }){Text("清理",color=MaterialTheme.colorScheme.error)}},
            {PlainButton({selection=null}){Text("取消")}})
    }
}

/** Also used by the instrumentation layout check with actual measured cache files. */
@Composable
internal fun StorageContent(topInset:Dp,stats:StorageSnapshot?,scanning:Boolean,clearing:Boolean,updateProtected:Boolean,message:String?,refresh:()->Unit,onClear:(CacheSelection)->Unit){
    val available=stats?.let{it.cache-if(updateProtected)it.updates else 0} ?: 0
    val muted=MaterialTheme.colorScheme.onSurfaceVariant
    LazyColumn(contentPadding=PaddingValues(start=16.dp,end=16.dp,top=topInset,bottom=42.dp),verticalArrangement=Arrangement.spacedBy(22.dp)){
        item{
            SettingsGroup{
                Column(Modifier.padding(20.dp),verticalArrangement=Arrangement.spacedBy(18.dp)){
                    Row(Modifier.fillMaxWidth(),verticalAlignment=Alignment.CenterVertically){
                        Box(Modifier.size(42.dp).background(StorageGray.copy(alpha=.14f),RoundedCornerShape(11.dp)),contentAlignment=Alignment.Center){Icon(Icons.Outlined.Storage,null,tint=muted,modifier=Modifier.size(26.dp))}
                        Column(Modifier.weight(1f).padding(start=12.dp)){
                            Text("Deuterium APP",style=MaterialTheme.typography.titleMedium)
                            Text("本机存储空间",style=MaterialTheme.typography.bodySmall,color=muted)
                        }
                        if(scanning||clearing)StorageActivityIndicator()
                    }
                    Row(Modifier.fillMaxWidth(),verticalAlignment=Alignment.Bottom){
                        val total=stats?.let{StorageFiles.display(it.total)} ?: "—"
                        Text(total,Modifier.weight(1f),fontSize=36.sp,lineHeight=42.sp,fontWeight=FontWeight.Bold,letterSpacing=(-1).sp)
                        Text("当前占用",Modifier.padding(bottom=5.dp),style=MaterialTheme.typography.bodySmall,color=muted)
                    }
                    StorageBar(stats)
                    Column(verticalArrangement=Arrangement.spacedBy(8.dp)){
                        Row(Modifier.fillMaxWidth()){StorageLegend("应用与数据",StorageGray,Modifier.weight(1f));StorageLegend("图片缓存",StorageBlue,Modifier.weight(1f))}
                        Row(Modifier.fillMaxWidth()){StorageLegend("更新文件",StoragePurple,Modifier.weight(1f));StorageLegend("临时缓存",StorageTeal,Modifier.weight(1f))}
                    }
                    HorizontalDivider(color=MaterialTheme.colorScheme.outlineVariant.copy(alpha=.5f),thickness=.5.dp)
                    Text(stats?.let{"手机剩余 ${StorageFiles.display(it.free)}"} ?: "正在统计存储空间…",style=MaterialTheme.typography.bodySmall,color=muted)
                }
            }
        }
        item{
            Column(verticalArrangement=Arrangement.spacedBy(8.dp)){
                StorageCaption("可清理缓存")
                SettingsGroup{
                    StorageRow("图片缓存","头像、商品和委托图片",Icons.Outlined.PhotoLibrary,StorageBlue,stats?.images,!scanning&&!clearing&&(stats?.images ?: 0)>0){onClear(CacheSelection.Images)}
                    SettingsDivider()
                    StorageRow("更新文件",if(updateProtected)"正在使用，暂不可清理" else "下载的安装包和资源包",Icons.Outlined.SystemUpdate,StoragePurple,stats?.updates,!scanning&&!clearing&&!updateProtected&&(stats?.updates ?: 0)>0){onClear(CacheSelection.Updates)}
                    SettingsDivider()
                    StorageRow("临时缓存","使用过程中产生的临时文件",Icons.Outlined.FolderOpen,StorageTeal,stats?.temporary,!scanning&&!clearing&&(stats?.temporary ?: 0)>0){onClear(CacheSelection.Temporary)}
                }
                Text("清理后，浏览图片时会按需重新下载。图片缓存会自动管理，最多使用 ${StorageFiles.display(stats?.imageBudget ?: AppImages.DISK_LIMIT)}。",Modifier.padding(horizontal=16.dp),style=MaterialTheme.typography.bodySmall,color=muted)
            }
        }
        item{
            SettingsGroup{
                PlainButton({onClear(CacheSelection.All)},Modifier.fillMaxWidth().heightIn(min=56.dp),enabled=!scanning&&!clearing&&available>0){
                    Text(if(clearing)"正在清理…" else "清理所有缓存",color=if(available>0)MaterialTheme.colorScheme.primary else muted)
                }
            }
            if(message!=null)Row(Modifier.fillMaxWidth().padding(top=12.dp),horizontalArrangement=Arrangement.Center,verticalAlignment=Alignment.CenterVertically){
                if(message.startsWith("已释放"))Icon(Icons.Outlined.CheckCircle,null,Modifier.size(16.dp),tint=Color(0xFF34C759))
                Text(message,Modifier.padding(start=6.dp),style=MaterialTheme.typography.bodySmall,color=muted)
            }
        }
        item{
            Column(verticalArrangement=Arrangement.spacedBy(8.dp)){
                StorageCaption("应用与数据")
                SettingsGroup{StorageRow("应用与数据","应用本体、设置与本地记录",Icons.Outlined.Apps,StorageGray,stats?.application,false,null)}
                Text("账号、聊天、订单记录及正在使用的资源不会被清理。",Modifier.padding(horizontal=16.dp),style=MaterialTheme.typography.bodySmall,color=muted)
                PlainButton(refresh,Modifier.align(Alignment.CenterHorizontally),enabled=!scanning&&!clearing){Text(if(scanning)"正在统计…" else "重新统计",style=MaterialTheme.typography.bodyMedium)}
            }
        }
    }
}

@Composable
private fun StorageCaption(text:String){Text(text,Modifier.padding(start=16.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)}

@Composable
private fun StorageActivityIndicator(){
    val phase=if(LocalMotion.current){
        val transition=androidx.compose.animation.core.rememberInfiniteTransition(label="storage-progress")
        val rotation = transition.animateFloat(0f,12f,androidx.compose.animation.core.infiniteRepeatable(tween(900,easing=androidx.compose.animation.core.LinearEasing)),label="storage-spokes")
        rotation
    } else null
    val color=MaterialTheme.colorScheme.onSurfaceVariant
    Canvas(Modifier.size(18.dp).semantics{contentDescription="正在处理"}){
        repeat(12){index->rotate(index*30f){drawLine(color.copy(alpha=.2f+.8f*((index-(phase?.value?.toInt() ?: 0)+12)%12)/12f),Offset(center.x,size.height*.06f),Offset(center.x,size.height*.24f),strokeWidth=1.5.dp.toPx(),cap=StrokeCap.Round)}}
    }
}

@Composable
private fun StorageRow(title:String,subtitle:String,icon:ImageVector,color:Color,bytes:Long?,enabled:Boolean,onClick:(()->Unit)?){
    Row(Modifier.fillMaxWidth().then(if(onClick!=null)Modifier.clickable(enabled=enabled,onClick=onClick) else Modifier).padding(horizontal=18.dp,vertical=16.dp),verticalAlignment=Alignment.CenterVertically){
        Box(Modifier.size(34.dp).background(color,RoundedCornerShape(9.dp)),contentAlignment=Alignment.Center){Icon(icon,null,Modifier.size(22.dp),tint=Color.White)}
        Column(Modifier.weight(1f).padding(horizontal=12.dp)){
            Text(title,style=MaterialTheme.typography.bodyLarge)
            Text(subtitle,Modifier.padding(top=2.dp),style=MaterialTheme.typography.labelSmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
        }
        Text(bytes?.let(StorageFiles::display) ?: "—",style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)
        if(onClick!=null)Icon(Icons.Outlined.ChevronRight,null,Modifier.padding(start=4.dp).size(17.dp),tint=MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha=if(enabled).5f else .2f))
    }
}

@Composable
private fun StorageLegend(title:String,color:Color,modifier:Modifier){Row(modifier,verticalAlignment=Alignment.CenterVertically){Box(Modifier.size(7.dp).background(color,CircleShape));Text(title,Modifier.padding(start=6.dp),style=MaterialTheme.typography.labelSmall,color=MaterialTheme.colorScheme.onSurfaceVariant)}}

@Composable
private fun StorageBar(stats:StorageSnapshot?){
    val values=listOf((stats?.application ?: 0) to StorageGray,(stats?.images ?: 0) to StorageBlue,(stats?.updates ?: 0) to StoragePurple,(stats?.temporary ?: 0) to StorageTeal)
    Row(Modifier.fillMaxWidth().height(17.dp).clip(RoundedCornerShape(5.dp)).background(MaterialTheme.colorScheme.surfaceVariant)
        .semantics{contentDescription="存储占用分类：应用与数据、图片缓存、更新文件、临时缓存"}){
        values.forEach{(bytes,color)->
            val fraction=if(stats?.total==null||stats.total==0L)0f else bytes.toFloat()/stats.total
            val animated by animateFloatAsState(fraction,if(LocalMotion.current)tween(350) else snap(),label="storage-category")
            if(animated>0)Box(Modifier.weight(animated).fillMaxHeight().background(color))
        }
    }
}
