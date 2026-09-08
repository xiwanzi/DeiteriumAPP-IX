package com.deuterium.app.uilab

import android.net.Uri
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.PickVisualMediaRequest
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.*
import androidx.compose.foundation.shape.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.*
import kotlinx.coroutines.*

@Composable
fun ListingPhotoEditor(photos:List<String>,onChange:(List<String>)->Unit,onBusy:(Boolean)->Unit={},maxImages:Int=5,evidence:Boolean=false,businessType:String=if(maxImages==1)"COMMISSION" else "MARKET_LISTING",businessRef:String?=null) {
    val context=LocalContext.current;val scope=rememberCoroutineScope()
    val list=rememberLazyListState()
    var busy by remember{mutableStateOf(false)};var error by remember{mutableStateOf<String?>(null)}
    val current by rememberUpdatedState(photos);val updated by rememberUpdatedState(onChange)
    fun copy(uris:List<Uri>){if(uris.isEmpty())return;busy=true;onBusy(true);scope.launch{
        val result=withContext(Dispatchers.IO){runCatching{uris.take(maxImages-current.size).mapIndexed{index,uri->
            val target=AppStorage.imageWorkFile(context,"listing-${System.currentTimeMillis()}-$index.img")
            try{
            context.contentResolver.openInputStream(uri)?.use{input->target.outputStream().use{output->val buffer=ByteArray(8192);var total=0;while(true){val count=input.read(buffer);if(count<0)break;total+=count;check(total<=20*1024*1024){"单张图片不能超过20MB"};output.write(buffer,0,count)}}} ?: error("无法读取图片")
            BackendAssets.upload(context,Uri.fromFile(target).toString(),if(evidence)"DISPUTE_EVIDENCE" else if(maxImages==1)"COMMISSION_COVER" else "MARKET_PHOTO",businessType,businessRef)
            } finally {target.delete()}
        }}}
        result.onSuccess{updated((current+it).take(maxImages))}.onFailure{error=it.message ?: "图片读取失败"};busy=false;onBusy(false)
    }}
    val single=rememberLauncherForActivityResult(ActivityResultContracts.PickVisualMedia()){uri->uri?.let{copy(listOf(it))}}
    val multiple=rememberLauncherForActivityResult(ActivityResultContracts.PickMultipleVisualMedia((maxImages-photos.size).coerceAtLeast(2))){copy(it)}
    Column {
        Row(Modifier.fillMaxWidth(),verticalAlignment=Alignment.CenterVertically){Text(if(evidence)"凭证图片（选填）" else if(maxImages==1)"委托封面" else "商品图片",Modifier.weight(1f),style=MaterialTheme.typography.titleMedium);Text("${photos.size}/$maxImages",style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)}
        Text(if(evidence)"可附上交付情况或沟通截图，最多 10 张" else if(maxImages==1)"清楚的封面可以让大家更快了解委托" else "第一张用于首页封面，还可添加 4 张详情图片",Modifier.padding(top=5.dp,bottom=12.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
        LazyRow(state=list,horizontalArrangement=Arrangement.spacedBy(10.dp)){
            itemsIndexed(photos,key={_,uri->uri}){index,uri->Column(Modifier.width(105.dp)){
                Box(Modifier.size(105.dp).clip(RoundedCornerShape(16.dp))){ListingImage(MarketListing("preview","商品图片","","","",1,1,"","",emptySet(),"",imageUri=uri),Modifier.fillMaxSize())
                    Box(Modifier.align(Alignment.TopStart).background(if(index==0)MaterialTheme.colorScheme.primary else Color.Black.copy(alpha=.5f)).padding(horizontal=6.dp,vertical=3.dp)){Text(if(evidence)"凭证 ${index+1}" else if(index==0)"首页封面" else "详情 ${index}",color=Color.White,fontSize=10.sp)}
                    IconButton({onChange(photos.filterIndexed{i,_->i!=index})},Modifier.align(Alignment.BottomEnd).size(32.dp).background(Color.Black.copy(alpha=.5f),CircleShape),enabled=!busy){Icon(Icons.Outlined.Close,"删除图片${index+1}",Modifier.size(17.dp),tint=Color.White)}
                }
                if(index>0&&!evidence)PlainButton({onChange(listOf(uri)+photos.filter{it!=uri});scope.launch{withFrameNanos{};list.animateScrollToItem(0)}},Modifier.fillMaxWidth(),enabled=!busy){Text("设为封面",fontSize=12.sp)}
            }}
            if(photos.size<maxImages)item{Box(Modifier.size(105.dp).clip(RoundedCornerShape(16.dp)).background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha=.55f)).clickable(enabled=!busy){if(maxImages-photos.size==1)single.launch(PickVisualMediaRequest(ActivityResultContracts.PickVisualMedia.ImageOnly))else multiple.launch(PickVisualMediaRequest(ActivityResultContracts.PickVisualMedia.ImageOnly))},contentAlignment=Alignment.Center){Column(horizontalAlignment=Alignment.CenterHorizontally){Icon(Icons.Outlined.AddPhotoAlternate,if(evidence)"添加凭证图片" else "添加商品图片",tint=MaterialTheme.colorScheme.primary);Text(if(busy)"上传中…" else "添加图片",Modifier.padding(top=7.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.primary)}}}
        }
        error?.let{Text(it,color=MaterialTheme.colorScheme.error,style=MaterialTheme.typography.bodySmall)}
    }
}
