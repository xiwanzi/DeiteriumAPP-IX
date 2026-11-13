package com.deuterium.app.uilab

import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.graphics.Canvas as AndroidCanvas
import android.graphics.Color as AndroidColor
import android.graphics.ColorFilter
import android.graphics.Paint
import android.graphics.PixelFormat
import android.graphics.PorterDuff
import android.graphics.drawable.Drawable
import android.view.View
import android.widget.ImageView
import androidx.activity.compose.BackHandler
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.PickVisualMediaRequest
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.tween
import androidx.compose.foundation.*
import androidx.compose.foundation.gestures.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.clipToBounds
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Rect
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.input.pointer.*
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.layout.LayoutCoordinates
import androidx.compose.ui.layout.boundsInWindow
import androidx.compose.ui.layout.onGloballyPositioned
import androidx.compose.ui.platform.*
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.semantics.stateDescription
import androidx.compose.ui.semantics.clearAndSetSemantics
import androidx.compose.ui.unit.*
import androidx.compose.ui.viewinterop.AndroidView
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner
import coil3.compose.AsyncImage
import coil3.request.CachePolicy
import coil3.request.ImageRequest
import kotlinx.coroutines.*
import kotlinx.coroutines.sync.Semaphore
import kotlinx.coroutines.sync.withPermit
import java.io.File
import kotlin.math.*

internal data class ChatViewerRequest(val media:ChatMedia,val messageId:String?,val preview:File?,val origin:()->Rect?)
private val chatPlaybackSlots=Semaphore(3)

@Composable
private fun chatResumed():Boolean {
    val owner=LocalLifecycleOwner.current
    var resumed by remember(owner){mutableStateOf(owner.lifecycle.currentState.isAtLeast(Lifecycle.State.RESUMED))}
    DisposableEffect(owner){val observer=LifecycleEventObserver{_,_->resumed=owner.lifecycle.currentState.isAtLeast(Lifecycle.State.RESUMED)};owner.lifecycle.addObserver(observer);onDispose{owner.lifecycle.removeObserver(observer)}}
    return resumed
}

@OptIn(ExperimentalFoundationApi::class)
@Composable
internal fun ChatMediaTile(service:BackendChatMedia,media:ChatMedia,messageId:String?=null,local:File?=null,panel:Boolean=false,playRequested:Boolean=true,onOpen:(ChatViewerRequest)->Unit,onLong:(()->Unit)?=null){
    val context=LocalContext.current;val density=LocalDensity.current;val view=LocalView.current
    val motion=LocalMotion.current;val resumed=chatResumed();val epoch=service.cacheEpoch
    var coordinates by remember{mutableStateOf<LayoutCoordinates?>(null)}
    var artworkCoordinates by remember{mutableStateOf<LayoutCoordinates?>(null)}
    var visible by remember{mutableStateOf(false)}
    var preview by remember(media.id,epoch){mutableStateOf<File?>(local)}
    var full by remember(media.id,epoch){mutableStateOf<File?>(null)}
    var error by remember(media.id,epoch){mutableStateOf<String?>(null)}
    var retry by remember{mutableIntStateOf(0)}
    var playing by remember{mutableStateOf(false)}
    LaunchedEffect(media.id,local,epoch,retry){
        if(local?.isFile==true){preview=local;return@LaunchedEffect}
        try{
            val cached=withContext(Dispatchers.IO){service.cache.file(media.sha)}
            if(cached!=null){if(media.animated){full=cached;preview=withContext(Dispatchers.IO){service.cache.file(media.thumbnailSha)}}else preview=cached}
            else preview=service.file(media,messageId,true)
            error=null
        }catch(e:CancellationException){throw e}catch(e:Exception){error=if(e is ApiFailure&&e.code=="MEDIA_EXPIRED")"图片已过期" else "点击重试"}
    }
    LaunchedEffect(playRequested,visible,motion,resumed,media.id,panel){
        if(!panel&&media.animated&&playRequested&&visible&&motion&&resumed)chatPlaybackSlots.withPermit{
            try{if(full==null)full=service.file(media,messageId);playing=true;awaitCancellation()}
            catch(e:CancellationException){throw e}catch(_:Exception){}finally{playing=false}
        }
    }
    val sticker=media.type=="sticker"||panel
    val width=if(panel)76f else if(sticker)148f else min(240f,media.width.coerceAtLeast(1).toFloat())
    val height=if(panel)76f else (width*media.height/media.width.coerceAtLeast(1)).coerceIn(32f,if(sticker)160f else 300f)
    val ratio=media.width.coerceAtLeast(1).toFloat()/media.height.coerceAtLeast(1)
    val artwork=if(sticker)Modifier.sizeIn(maxWidth=width.dp,maxHeight=height.dp).aspectRatio(ratio,matchHeightConstraintsFirst=ratio<width/height).clip(RoundedCornerShape(14)) else Modifier.fillMaxSize()
    fun open(){onOpen(ChatViewerRequest(media,messageId,preview ?: full){(if(sticker)artworkCoordinates else coordinates)?.takeIf{it.isAttached}?.boundsInWindow()?.takeIf{it.bottom>0&&it.top<view.height}})}
    Box(Modifier.size(width.dp,height.dp).onGloballyPositioned{coordinates=it;val bounds=it.boundsInWindow();visible=bounds.bottom>0&&bounds.top<view.height&&bounds.width>0}
        .semantics{contentDescription=if(sticker)"查看贴纸" else "查看图片"}
        .then(if(sticker)Modifier else Modifier.clip(RoundedCornerShape(15.dp))).background(if(sticker)Color.Transparent else MaterialTheme.colorScheme.surfaceVariant)
        .combinedClickable(onClick={if(error!=null&&preview==null&&full==null){retry++;error=null}else open()},onLongClick=onLong ?: {open()}),contentAlignment=Alignment.Center){
        if(media.animated&&full!=null)ChatAnimationImage(full!!,playing,artwork.onGloballyPositioned{artworkCoordinates=it},fit=sticker,rounded=sticker)
        else if(preview!=null)AsyncImage(ImageRequest.Builder(context).data(preview).memoryCacheKey("${preview!!.path}:$epoch").diskCachePolicy(CachePolicy.DISABLED).build(),"聊天图片",imageLoader=AppImages.get(context).loader,contentScale=if(sticker)ContentScale.Fit else ContentScale.Crop,alignment=Alignment.TopCenter,modifier=artwork.onGloballyPositioned{artworkCoordinates=it})
        else if(error!=null)Text(error!!,style=MaterialTheme.typography.labelSmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
        else CircularProgressIndicator(Modifier.size(18.dp),strokeWidth=1.5.dp)
        if(!panel&&(media.animated&&!playing||media.height>media.width*2.5))Text(if(media.animated)"GIF" else "长图",Modifier.align(Alignment.BottomEnd).padding(6.dp).background(Color.Black.copy(alpha=.5f),RoundedCornerShape(5.dp)).padding(horizontal=5.dp,vertical=2.dp),color=Color.White,fontSize=10.sp)
    }
}

/** Single software canvas, one decoded frame at a time; also supports API 26. */
private class ChatAnimationDrawable(private val animation:ChatWebp.Animation):Drawable(){
    private val scope=CoroutineScope(SupervisorJob()+Dispatchers.Main.immediate)
    private val bitmap=Bitmap.createBitmap(animation.width,animation.height,Bitmap.Config.ARGB_8888)
    private val canvas=AndroidCanvas(bitmap)
    private val paint=Paint(Paint.FILTER_BITMAP_FLAG)
    private val corners=android.graphics.Path()
    var rounded=false
        set(value){if(field!=value){field=value;invalidateSelf()}}
    private var job:Job?=null
    private var enabled=false
    private suspend fun render(index:Int){
        val f=animation.frames[index];val decoded=withContext(Dispatchers.Default){BitmapFactory.decodeByteArray(f.bytes,0,f.bytes.size)} ?: return
        try{
            if(index==0)bitmap.eraseColor(AndroidColor.TRANSPARENT)
            else animation.frames[index-1].takeIf{it.flags and 1!=0}?.let{canvas.save();canvas.clipRect(it.x,it.y,it.x+it.width,it.y+it.height);canvas.drawColor(AndroidColor.TRANSPARENT,PorterDuff.Mode.CLEAR);canvas.restore()}
            if(f.flags and 2!=0){canvas.save();canvas.clipRect(f.x,f.y,f.x+f.width,f.y+f.height);canvas.drawColor(AndroidColor.TRANSPARENT,PorterDuff.Mode.CLEAR);canvas.restore()}
            canvas.drawBitmap(decoded,f.x.toFloat(),f.y.toFloat(),paint);invalidateSelf()
        }finally{decoded.recycle()}
    }
    fun play(value:Boolean){if(job!=null&&enabled==value)return;enabled=value;job?.cancel();job=scope.launch{
        var loop=0;var deadline=android.os.SystemClock.uptimeMillis()
        do{for(i in animation.frames.indices){render(i);if(!enabled)return@launch;deadline+=animation.frames[i].duration;delay((deadline-android.os.SystemClock.uptimeMillis()).coerceAtLeast(0))};loop++}while(enabled&&(animation.loops==0||loop<animation.loops))
    }}
    override fun draw(canvas:AndroidCanvas){
        if(bitmap.isRecycled)return
        canvas.save()
        if(rounded){val radius=min(bounds.width(),bounds.height())*.14f;corners.reset();corners.addRoundRect(bounds.left.toFloat(),bounds.top.toFloat(),bounds.right.toFloat(),bounds.bottom.toFloat(),radius,radius,android.graphics.Path.Direction.CW);canvas.clipPath(corners)}
        canvas.drawBitmap(bitmap,null,bounds,paint);canvas.restore()
    }
    override fun getIntrinsicWidth()=animation.width
    override fun getIntrinsicHeight()=animation.height
    override fun setAlpha(alpha:Int){paint.alpha=alpha}
    override fun setColorFilter(filter:ColorFilter?){paint.colorFilter=filter}
    @Deprecated("Deprecated in Java") override fun getOpacity()=PixelFormat.TRANSLUCENT
    fun close(){scope.cancel();bitmap.recycle()}
}

@Composable
private fun ChatAnimationImage(file:File,playing:Boolean,modifier:Modifier=Modifier,fit:Boolean=true,rounded:Boolean=false){
    val drawable by produceState<ChatAnimationDrawable?>(null,file.path){
        val parsed=withContext(Dispatchers.Default){runCatching{ChatWebp.animation(file.readBytes())}.getOrNull()}
        parsed?.let{value=ChatAnimationDrawable(it)}
    }
    DisposableEffect(drawable){val owned=drawable;onDispose{owned?.close()}}
    AndroidView(factory={context->ImageView(context).apply{setLayerType(View.LAYER_TYPE_SOFTWARE,null)}},modifier=modifier,update={image->image.scaleType=if(fit)ImageView.ScaleType.FIT_CENTER else ImageView.ScaleType.CENTER_CROP;if(image.drawable!==drawable)image.setImageDrawable(drawable);drawable?.rounded=rounded;drawable?.play(playing)})
}

@Composable
internal fun ChatMediaComposer(
    state:LabState,focus:androidx.compose.ui.focus.FocusRequester,
    value:androidx.compose.ui.text.input.TextFieldValue=state.chatDraft,
    onValue:(androidx.compose.ui.text.input.TextFieldValue)->Unit={state.chatDraft=it},
    onSend:()->Unit=state::sendChat,reply:ChatReply?=state.chatReplyTo,
    onCancelReply:()->Unit={state.chatReplyTo=null},recipient:String?=null,
    onView:(ChatViewerRequest)->Unit
){
    val service=state.chatMedia ?: return
    suspend fun enqueue(uri:String?,sticker:ChatMedia?=null,importing:Boolean=false){
        val target=if(importing||recipient==null)null else state.directMediaTarget(recipient)
        service.enqueue(uri,if(importing)null else reply,sticker,importing,target)
        if(!importing)onCancelReply()
    }
    val scope=rememberCoroutineScope()
    val keyboard=LocalSoftwareKeyboardController.current
    val focusManager=LocalFocusManager.current
    val density=LocalDensity.current
    val motion=LocalMotion.current
    val panelState=androidx.compose.runtime.saveable.rememberSaveableStateHolder()
    var panel by rememberSaveable{mutableStateOf(false)}
    var switchingToKeyboard by remember{mutableStateOf(false)}
    var importing by remember{mutableStateOf(false)}
    var selection by rememberSaveable{mutableStateOf<String?>(null)}
    var selectionBusy by remember{mutableStateOf(false)}
    var selectionError by remember{mutableStateOf<String?>(null)}
    val ime=WindowInsets.ime.getBottom(density)
    val navigation=WindowInsets.navigationBars.getBottom(density)
    var keyboardHeight by rememberSaveable{mutableIntStateOf(with(density){292.dp.roundToPx()})}
    val maxHeight=with(density){(LocalConfiguration.current.screenHeightDp-150).coerceAtLeast(150).dp.toPx()}
    val panelHeight=keyboardHeight.toFloat().coerceAtMost(maxHeight)
    val dock=remember{Animatable(0f)}
    // Learn only a settled keyboard height, never the intermediate frames of its dismissal.
    LaunchedEffect(ime,panel,switchingToKeyboard){
        if(switchingToKeyboard&&ime>=dock.value-2&&ime>0){switchingToKeyboard=false}
        else if(!panel&&!switchingToKeyboard&&ime>with(density){128.dp.toPx()}){delay(180);keyboardHeight=ime}
    }
    LaunchedEffect(switchingToKeyboard){if(switchingToKeyboard){delay(800);switchingToKeyboard=false}}
    LaunchedEffect(panel,switchingToKeyboard,panelHeight,motion){
        if(panel){
            if(ime>0||!motion)dock.snapTo(panelHeight)
            else dock.animateTo(panelHeight,tween(280,easing=androidx.compose.animation.core.FastOutSlowInEasing))
        }else if(!switchingToKeyboard){
            if(ime>=dock.value||!motion)dock.snapTo(0f)
            else dock.animateTo(0f,tween(240,easing=androidx.compose.animation.core.FastOutSlowInEasing))
        }
    }
    val picker=rememberLauncherForActivityResult(ActivityResultContracts.PickVisualMedia()){uri->if(uri!=null){selectionError=null;selection=uri.toString()}}
    fun choose(sticker:Boolean){
        if(sticker&&service.stickers.size>=10){state.storageMessage="贴纸已满（10/10），删除已有贴纸后可继续添加";return}
        importing=sticker;focusManager.clearFocus();keyboard?.hide()
        picker.launch(PickVisualMediaRequest(ActivityResultContracts.PickVisualMedia.ImageOnly))
    }
    fun showKeyboard(){if(panel){panel=false;switchingToKeyboard=true};focus.requestFocus();keyboard?.show()}
    fun toggle(){
        if(panel)showKeyboard()
        else{panel=true;switchingToKeyboard=false;focusManager.clearFocus();keyboard?.hide();scope.launch{service.refreshStickers()}}
    }
    BackHandler(panel){panel=false;switchingToKeyboard=false}
    Column {
        MessageComposer(value,{if(it.text.length<=256)onValue(it)},onSend,"输入消息",focus,reply,onCancelReply,onEmoji=::toggle,onImage={choose(false)},onFocus={if(panel)showKeyboard()},emojiOpen=panel)
        // Navigation padding belongs inside the panel material, so it reaches the screen edge.
        // With the panel closed, this same space keeps the input clear of system gestures.
        val reserved=(max(dock.value,navigation.toFloat())-ime).coerceAtLeast(0f)
        Box(Modifier.fillMaxWidth().height(with(density){reserved.toDp()}).clipToBounds().graphicsLayer{alpha=if(dock.value>0f)1f else 0f}.then(if(dock.value<=0f||reserved<=0f)Modifier.clearAndSetSemantics{} else Modifier)){
            // Keep the first page measured before the animation starts, so lazy-grid startup
            // cannot consume the short entrance animation's first frames.
            panelState.SaveableStateProvider("expressions"){
                Box(Modifier.fillMaxWidth().wrapContentHeight(Alignment.Top,unbounded=true).requiredHeight(with(density){panelHeight.toDp()}),contentAlignment=Alignment.TopStart){
                    ChatGlassSurface(Modifier.fillMaxSize().semantics{contentDescription="表情面板"}.pointerInput(Unit){detectTapGestures{}},panel=true){
                        Box(Modifier.fillMaxSize().navigationBarsPadding()){ChatExpressionPanel(state,{choose(true)},onView,value,onValue){enqueue(null,it)}}
                    }
                }
            }
        }
    }
    selection?.let{uri->ChatAttachmentPreview(uri,importing,selectionBusy,selectionError,onChooseAgain={choose(importing)},onSend={
        scope.launch{selectionBusy=true;selectionError=null
            try{enqueue(uri,importing=importing);selection=null}
            catch(e:CancellationException){throw e}catch(e:Exception){selectionError=e.message ?: "图片暂时无法处理，请重试"}
            finally{selectionBusy=false}
        }
    },onDismiss={selection=null;selectionError=null})}
}

@Composable
internal fun ChatPendingMediaRow(service:BackendChatMedia,job:PendingChatMedia,onView:(ChatViewerRequest)->Unit){
    val scope=rememberCoroutineScope()
    val media=job.media ?: ChatMedia("", "", "",0,0,job.width,job.height,job.animated,job.type)
    Column(Modifier.fillMaxWidth(),horizontalAlignment=Alignment.End){
        Box{
            job.reply?.let{QuotedMessage(it,false)}
            ChatMediaTile(service,media,local=job.localFile.takeIf{it.isFile},playRequested=false,onOpen={if(job.media!=null)onView(it)})
            if(job.status=="failed")GlassIconButton({service.retry(job)},Modifier.align(Alignment.BottomEnd).padding(6.dp).size(30.dp), tint = MaterialTheme.colorScheme.error){Icon(Icons.Outlined.Refresh,"重试发送",Modifier.size(18.dp),tint=Color.White)}
            else CircularProgressIndicator(Modifier.align(Alignment.BottomEnd).padding(10.dp).size(18.dp),color=MaterialTheme.colorScheme.primary,strokeWidth=2.dp)
        }
        if(job.status=="failed")Row(verticalAlignment=Alignment.CenterVertically){Text(job.error ?: "发送失败",Modifier.widthIn(max=230.dp),style=MaterialTheme.typography.labelSmall,color=MaterialTheme.colorScheme.error);PlainButton({scope.launch{service.dismiss(job)}}){Text("移除")}}
    }
}

@Composable
internal fun ChatMediaViewer(service:BackendChatMedia,request:ChatViewerRequest,onClose:()->Unit){
    val scope=rememberCoroutineScope();val density=LocalDensity.current;val motion=LocalMotion.current;val resumed=chatResumed();val media=request.media
    val progress=remember{Animatable(if(motion)0f else 1f)};var closing by remember{mutableStateOf(false)}
    var file by remember{mutableStateOf(request.preview)};var loaded by remember{mutableStateOf(false)};var notice by remember{mutableStateOf<String?>(null)}
    var zoom by remember{mutableFloatStateOf(1f)};var offset by remember{mutableStateOf(Offset.Zero)};var restingOffset by remember{mutableStateOf(Offset.Zero)};var showControls by remember{mutableStateOf(true)}
    val initial=request.origin()
    fun close(){if(closing)return;closing=true;scope.launch{if(motion)progress.animateTo(0f,tween(200));onClose()}}
    LaunchedEffect(media.id){launch{if(motion)progress.animateTo(1f,tween(250))};try{file=service.file(media,request.messageId);loaded=true}catch(e:CancellationException){throw e}catch(e:Exception){notice=if(e is ApiFailure&&e.code=="MEDIA_EXPIRED")"清晰图片已过期，当前显示本机预览" else "清晰图片暂不可用"}}
    Dialog(::close,properties=DialogProperties(usePlatformDefaultWidth=false,decorFitsSystemWindows=false)){
        BoxWithConstraints(Modifier.fillMaxSize().background(Color.Black.copy(alpha=.96f*progress.value))){
            val w=constraints.maxWidth.toFloat();val h=constraints.maxHeight.toFloat();val long=media.height>media.width*2.5
            val safeTop=WindowInsets.statusBars.getTop(density)+with(density){60.dp.toPx()};val safeBottom=WindowInsets.navigationBars.getBottom(density)+with(density){48.dp.toPx()}
            val normalY=(safeTop-safeBottom)/2
            val base=minOf(1f,(w-with(density){24.dp.toPx()})/media.width,if(long)Float.POSITIVE_INFINITY else (h-safeTop-safeBottom)/media.height)
            val width=media.width*base;val height=media.height*base
            fun bounded(value:Offset,scale:Float,allowDismiss:Boolean=false):Offset {
                val x=max(0f,(width*scale-w+with(density){24.dp.toPx()})/2)
                val minY=h-safeBottom-h/2-height*scale/2;val maxY=safeTop-h/2+height*scale/2
                return Offset(value.x.coerceIn(-x,x),if(allowDismiss)value.y else if(minY<=maxY)value.y.coerceIn(minY,maxY) else normalY)
            }
            LaunchedEffect(w,h){offset=bounded(Offset(0f,if(long)(height-h)/2+safeTop else normalY),1f);restingOffset=offset}
            val origin=(if(closing)request.origin() else initial)?.takeIf{it.bottom>0&&it.top<h}
            val p=progress.value;val shownW=if(origin!=null)origin.width+(width-origin.width)*p else width;val shownH=if(origin!=null)origin.height+(height-origin.height)*p else height
            val left=if(origin!=null)origin.left+((w-width)/2-origin.left)*p else (w-width)/2
            val top=if(origin!=null)origin.top+((h-height)/2-origin.top)*p else (h-height)/2
            Box(Modifier.fillMaxSize().semantics{stateDescription="缩放 ${(zoom*100).roundToInt()}%"}.pointerInput(media.id,w,h){
                detectTapGestures(onTap={showControls=!showControls},onDoubleTap={point->if(zoom>1.01f){zoom=1f;offset=bounded(restingOffset,1f)}else{restingOffset=offset;zoom=3f;val center=point-Offset(w/2,h/2);offset=bounded(center-(center-offset)*zoom,zoom)}})
            }.pointerInput(media.id,w,h){awaitEachGesture{
                awaitFirstDown(requireUnconsumed=false);var multi=false;var moved=false
                do{
                    val event=awaitPointerEvent();val count=event.changes.count{it.pressed};if(count>1)multi=true
                    val change=event.calculateZoom();val pan=event.calculatePan()
                    if(count>1||pan.getDistance()>0f){
                        val previous=zoom;val next=(zoom*change).coerceIn(1f,4f);val center=event.calculateCentroid(useCurrent=false)-Offset(w/2,h/2)
                        offset=bounded(center-(center-offset)*(next/previous)+pan,next,!long&&!multi&&next<=1.01f);zoom=next
                        if(pan.getDistance()>viewConfiguration.touchSlop||moved||change!=1f){moved=true;event.changes.forEach{it.consume()}}
                    }
                }while(event.changes.any{it.pressed})
                if(!long&&!multi&&zoom<=1.01f&&abs(offset.y-normalY)>with(density){100.dp.toPx()})close()else offset=bounded(offset,zoom)
            }}){
                Box(Modifier.offset{IntOffset(left.roundToInt(),top.roundToInt())}.size(with(density){shownW.toDp()},with(density){shownH.toDp()}).graphicsLayer{scaleX=if(closing)1+(zoom-1)*p else zoom;scaleY=scaleX;translationX=offset.x*p;translationY=offset.y*p;alpha=if(origin==null)p else 1f}.clip(if(media.type=="sticker")RoundedCornerShape(14) else RoundedCornerShape((15*(1-p)).dp)),contentAlignment=Alignment.Center){
                    if(file!=null){if(media.animated&&loaded)ChatAnimationImage(file!!,resumed,Modifier.fillMaxSize(),rounded=media.type=="sticker")else AsyncImage(file,"聊天图片",Modifier.fillMaxSize(),contentScale=ContentScale.Fit)}else CircularProgressIndicator(color=Color.White,modifier=Modifier.size(24.dp))
                }
            }
            if(showControls)Row(Modifier.align(Alignment.TopCenter).fillMaxWidth().statusBarsPadding().padding(12.dp),verticalAlignment=Alignment.CenterVertically){
                GlassIconButton(::close, surface = Color.Black.copy(alpha = .65f)){Icon(Icons.Outlined.Close,"关闭图片",tint=Color.White)};Spacer(Modifier.weight(1f));Text(if(media.type=="sticker")"贴纸" else "图片",color=Color.White);Spacer(Modifier.weight(1f))
                if(request.messageId!=null)GlassIconButton({scope.launch{try{service.collect(media,request.messageId);notice="已加入贴纸添加队列"}catch(e:Exception){notice=e.message}}}, surface = Color.Black.copy(alpha = .65f)){Icon(Icons.Outlined.AddPhotoAlternate,"添加到贴纸",tint=Color.White)}else Spacer(Modifier.size(48.dp))
            }
            if(showControls||notice!=null)Text(notice ?: "双指缩放 · 双击放大",Modifier.align(Alignment.BottomCenter).navigationBarsPadding().padding(20.dp),color=Color.White.copy(alpha=.7f),style=MaterialTheme.typography.labelMedium)
        }
    }
}
