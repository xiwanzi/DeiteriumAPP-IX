package com.deuterium.app.uilab

import android.graphics.drawable.ColorDrawable
import androidx.activity.compose.BackHandler
import androidx.compose.animation.core.*
import androidx.compose.foundation.*
import androidx.compose.foundation.gestures.detectVerticalDragGestures
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalView
import androidx.compose.ui.semantics.*
import androidx.compose.ui.unit.*
import androidx.compose.ui.window.*
import coil3.compose.AsyncImage
import coil3.request.ImageRequest
import kotlinx.coroutines.launch
import kotlin.math.abs
import kotlin.math.sqrt

/** Single selected image, with the same half-height / expanded gesture as the system photo picker. */
@Composable
internal fun ChatAttachmentPreview(uri:String,importing:Boolean,busy:Boolean,error:String?,onChooseAgain:()->Unit,onSend:()->Unit,onDismiss:()->Unit){
    val motion=LocalMotion.current
    val scope=rememberCoroutineScope()
    val density=LocalDensity.current
    val context=LocalContext.current
    val height=remember{Animatable(0f)}
    var dragging by remember{mutableStateOf<Float?>(null)}
    var closing by remember{mutableStateOf(false)}
    var imageRatio by remember(uri){mutableFloatStateOf(1f)}
    val registry=LocalIosOverlayRegistry.current
    DisposableEffect(registry){registry?.let{it.count++};onDispose{registry?.let{it.count--}}}
    fun close(){if(busy||closing)return;closing=true;scope.launch{height.snapTo(dragging ?: height.value);dragging=null;if(motion)height.animateTo(0f,tween(220, easing=FastOutSlowInEasing));onDismiss()}}
    Dialog(::close,properties=DialogProperties(usePlatformDefaultWidth=false,decorFitsSystemWindows=false)){
        val window=(LocalView.current.parent as? DialogWindowProvider)?.window
        DisposableEffect(window){window?.apply{setBackgroundDrawable(ColorDrawable(android.graphics.Color.TRANSPARENT));setDimAmount(0f);setWindowAnimations(0)};onDispose{}}
        BackHandler{close()}
        BoxWithConstraints(Modifier.fillMaxSize()){
            val top=WindowInsets.statusBars.asPaddingValues().calculateTopPadding()
            val full=with(density){(maxHeight-top-6.dp).toPx()}
            val half=(full*.64f).coerceAtLeast(with(density){320.dp.toPx()}).coerceAtMost(full)
            LaunchedEffect(full){if(!closing){if(motion)height.animateTo(half,tween(280, easing=FastOutSlowInEasing))else height.snapTo(half)}}
            val current=(dragging ?: height.value).coerceIn(0f,full)
            val expanded=current>(full+half)/2
            val coverage=(current/half).coerceIn(0f,1f)
            Box(Modifier.fillMaxSize().background(Color.Black.copy(alpha=(.16f+.15f*(current/full))*coverage)).clickable(interactionSource=remember{MutableInteractionSource()},indication=null,onClick=::close))
            val resizeGesture=Modifier.pointerInput(full,half,busy){
                    detectVerticalDragGestures(onDragStart={if(!busy&&!closing){dragging=height.value;scope.launch{height.stop()}}},onVerticalDrag={change,amount->
                        if(!busy&&!closing){change.consume();dragging=((dragging ?: height.value)-amount).coerceIn(0f,full)}
                    },onDragCancel={if(!closing){val from=dragging ?: height.value;scope.launch{height.snapTo(from);dragging=null;if(motion)height.animateTo(if(from>(half+full)/2)full else half,tween(240))else height.snapTo(if(from>(half+full)/2)full else half)}}},onDragEnd={
                        if(!closing){val from=dragging ?: height.value
                            scope.launch{height.snapTo(from);dragging=null;if(from<half*.64f)close()else{val target=if(from>(half+full)/2)full else half;if(motion)height.animateTo(target,tween(260, easing=FastOutSlowInEasing))else height.snapTo(target)}}}
                    })
                }
            SoftGlassSurface(Modifier.align(Alignment.BottomCenter).fillMaxWidth().height(with(density){current.toDp()})
                .semantics{contentDescription="图片预览抽屉";stateDescription=if(expanded)"已展开" else "半屏"},radius=28.dp){
                Column(Modifier.fillMaxSize().navigationBarsPadding()){
                  Column(resizeGesture){
                    Box(Modifier.fillMaxWidth().height(25.dp).semantics{contentDescription=if(expanded)"收起预览" else "展开预览"}.clickable(enabled=!busy&&!closing){scope.launch{val target=if(expanded)half else full;if(motion)height.animateTo(target,tween(260, easing=FastOutSlowInEasing))else height.snapTo(target)}},contentAlignment=Alignment.Center){Box(Modifier.size(34.dp,4.dp).background(MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha=.35f),CircleShape))}
                    Row(Modifier.fillMaxWidth().padding(horizontal=10.dp),verticalAlignment=Alignment.CenterVertically){
                        IconButton(::close,enabled=!busy&&!closing){Icon(Icons.Outlined.Close,"取消")}
                        Text(if(importing)"添加贴纸" else "发送图片",Modifier.weight(1f),style=MaterialTheme.typography.titleMedium)
                        PlainButton(onChooseAgain,enabled=!busy&&!closing){Text("重新选择")}
                    }
                  }
                    BoxWithConstraints(Modifier.weight(1f).fillMaxWidth().padding(horizontal=10.dp,vertical=8.dp)){
                        val long=imageRatio>2.5f&&!importing
                        val width=with(density){maxWidth.roundToPx()}.coerceIn(1,1440)
                        val height=(width*(if(long)imageRatio else 2.5f)).toInt().coerceIn(1,8192)
                        val scale=sqrt(8_000_000.0/(width.toDouble()*height)).coerceAtMost(1.0)
                        val targetWidth=(width*scale).toInt().coerceAtLeast(1);val targetHeight=(height*scale).toInt().coerceAtLeast(1)
                        // Switching from a fitted thumbnail to a scrollable long image must
                        // request enough pixels; reusing the initial tiny decode blurs its text.
                        val request=remember(uri,targetWidth,targetHeight){ImageRequest.Builder(context).data(uri).size(targetWidth,targetHeight).memoryCacheKey("attachment:$uri:$targetWidth:$targetHeight").build()}
                        val imageModifier=if(importing)Modifier.aspectRatio(1f/imageRatio,matchHeightConstraintsFirst=imageRatio>maxHeight/maxWidth).clip(RoundedCornerShape(14)) else if(long)Modifier.fillMaxWidth().height(maxWidth*imageRatio) else Modifier.fillMaxSize()
                        Box(if(long)Modifier.fillMaxSize().verticalScroll(rememberScrollState()) else Modifier.fillMaxSize(),contentAlignment=if(importing)Alignment.Center else Alignment.TopStart){
                            AsyncImage(request,"所选图片",imageModifier,contentScale=ContentScale.Fit,onSuccess={result->val image=result.result.image;val ratio=image.height.toFloat()/image.width.coerceAtLeast(1);if(abs(imageRatio-ratio)>.02f)imageRatio=ratio})
                        }
                    }
                    error?.let{Text(it,Modifier.padding(horizontal=20.dp,vertical=6.dp),color=MaterialTheme.colorScheme.error,style=MaterialTheme.typography.bodySmall)}
                    Row(Modifier.fillMaxWidth().padding(start=20.dp,end=12.dp,top=6.dp,bottom=10.dp),verticalAlignment=Alignment.CenterVertically){
                        Icon(Icons.Outlined.Photo,null,Modifier.size(19.dp),tint=MaterialTheme.colorScheme.onSurfaceVariant)
                        Text("1 张图片",Modifier.weight(1f).padding(start=8.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
                        MotionButton(onSend,enabled=!busy&&!closing,shape=CircleShape){if(busy)CircularProgressIndicator(Modifier.size(16.dp),strokeWidth=2.dp);Text(if(importing)"添加" else "发送",Modifier.padding(horizontal=10.dp))}
                    }
                }
            }
        }
    }
}
