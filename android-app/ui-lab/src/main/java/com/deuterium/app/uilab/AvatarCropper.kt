package com.deuterium.app.uilab

import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.graphics.ImageDecoder
import android.net.Uri
import android.os.Build
import androidx.compose.foundation.*
import androidx.compose.foundation.gestures.detectTransformGestures
import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.*
import androidx.compose.ui.graphics.*
import androidx.compose.ui.graphics.drawscope.*
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.layout.onSizeChanged
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalView
import androidx.core.view.WindowCompat
import androidx.compose.ui.unit.*
import androidx.compose.ui.window.*
import kotlinx.coroutines.*

@Composable
fun AvatarCropper(source:String,onCancel:()->Unit,onSaved:(String)->Unit) {
    val context=LocalContext.current;val scope=rememberCoroutineScope()
    var bitmap by remember { mutableStateOf<Bitmap?>(null) };var error by remember{mutableStateOf<String?>(null)}
    var zoom by remember { mutableFloatStateOf(1f) };var pan by remember { mutableStateOf(Offset.Zero) };var side by remember{mutableFloatStateOf(1f)}
    var saving by remember{mutableStateOf(false)}
    LaunchedEffect(source) {
        val result=withContext(Dispatchers.IO){runCatching{decodeLocalPhoto(context,Uri.parse(source),1600)}}
        result.onSuccess{bitmap=it}.onFailure{error="无法读取这张图片，请重新选择"}
    }
    fun bounded(value:Offset,scale:Float):Offset {
        val image=bitmap ?: return Offset.Zero
        val base=maxOf(side/image.width,side/image.height)*scale
        return Offset(value.x.coerceIn(-(image.width*base-side)/2,(image.width*base-side)/2),value.y.coerceIn(-(image.height*base-side)/2,(image.height*base-side)/2))
    }
    Dialog(onDismissRequest={if(!saving)onCancel()},properties=DialogProperties(usePlatformDefaultWidth=false,decorFitsSystemWindows=false)) {
        val view=LocalView.current
        SideEffect { (view.parent as? DialogWindowProvider)?.window?.let { window->WindowCompat.getInsetsController(window,window.decorView).apply{isAppearanceLightStatusBars=false;isAppearanceLightNavigationBars=false} } }
        Column(Modifier.fillMaxSize().background(Color(0xFF08080A)).statusBarsPadding().navigationBarsPadding()) {
            Row(Modifier.fillMaxWidth().padding(10.dp),verticalAlignment=Alignment.CenterVertically){PlainButton(onCancel,enabled=!saving){Text("取消")};Text("调整头像",Modifier.weight(1f),color=Color.White,textAlign=androidx.compose.ui.text.style.TextAlign.Center,style=MaterialTheme.typography.titleMedium)
                PlainButton({
                    val input=bitmap ?: return@PlainButton
                    saving=true
                    val cropZoom=zoom;val cropPan=pan;val cropSide=side
                    scope.launch {
                        val result=withContext(Dispatchers.IO){runCatching{
                            val out=Bitmap.createBitmap(640,640,Bitmap.Config.ARGB_8888)
                            val canvas=android.graphics.Canvas(out)
                            val scale=maxOf(cropSide/input.width,cropSide/input.height)*cropZoom*640f/cropSide
                            canvas.translate(320f+cropPan.x*640f/cropSide,320f+cropPan.y*640f/cropSide);canvas.scale(scale,scale)
                            canvas.drawBitmap(input,-input.width/2f,-input.height/2f,android.graphics.Paint(android.graphics.Paint.ANTI_ALIAS_FLAG or android.graphics.Paint.FILTER_BITMAP_FLAG))
                            val target=AppStorage.imageWorkFile(context,"avatar-crop-${System.currentTimeMillis()}.png")
                            target.outputStream().use { check(out.compress(Bitmap.CompressFormat.PNG,100,it)) };out.recycle();Uri.fromFile(target).toString()
                        }}
                        result.onSuccess{localUri->
                            try{
                            runCatching{
                                val api=BackendApi.get(context)
                                val asset=BackendAssets.upload(context,localUri,"AVATAR","PROFILE")
                                val profile=api.request("GET","/players/${api.playerRef}")
                                api.request("PATCH","/account/me/profile",org.json.JSONObject().put("clientRequestId",java.util.UUID.randomUUID().toString()).put("expectedVersion",profile.getLong("version")).put("avatarAssetId",asset.removePrefix("asset:")))
                                asset
                            }.onSuccess(onSaved).onFailure{error=it.message ?: "头像上传失败，请重试";saving=false}
                            } finally {withContext(kotlinx.coroutines.NonCancellable+Dispatchers.IO){AppStorage.removeImageWorkFile(context,localUri)}}
                        }.onFailure{error="保存失败，请重试";saving=false}
                    }
                },enabled=bitmap!=null&&!saving){Text(if(saving)"保存中" else "完成")}
            }
            Box(Modifier.weight(1f).fillMaxWidth(),contentAlignment=Alignment.Center){
                Canvas(Modifier.padding(horizontal=20.dp).fillMaxWidth().aspectRatio(1f).onSizeChanged{side=it.width.toFloat()}
                    .pointerInput(bitmap,side){detectTransformGestures { centroid,move,scale,_->
                        val previous=zoom;zoom=(zoom*scale).coerceIn(1f,4f)
                        val center=Offset(side/2,side/2)
                        pan=bounded((pan+center-centroid)*(zoom/previous)+centroid-center+move,zoom)
                    }}) {
                    bitmap?.let{photo->val factor=maxOf(size.width/photo.width,size.height/photo.height)*zoom
                        clipRect { withTransform({translate(center.x+pan.x,center.y+pan.y);scale(factor,factor,Offset.Zero)}){drawImage(photo.asImageBitmap(),Offset(-photo.width/2f,-photo.height/2f))} }
                    }
                    val outside=Path().apply{fillType=PathFillType.EvenOdd;addRect(Rect(Offset.Zero,size));addOval(Rect(Offset.Zero,size))}
                    drawPath(outside,Color.Black.copy(alpha=.6f));drawCircle(Color.White.copy(alpha=.85f),size.minDimension/2,style=Stroke(1.dp.toPx()))
                }
            }
            Column(Modifier.padding(horizontal=34.dp,vertical=28.dp),horizontalAlignment=Alignment.CenterHorizontally){
                Text(error ?: "移动和缩放，使头像位于圆形区域内",color=if(error==null)Color(0xFFAAAAAF) else Color(0xFFFF6961),style=MaterialTheme.typography.bodyMedium)
                IosSlider(zoom,{zoom=it;pan=bounded(pan,zoom)},1f..4f,Modifier.fillMaxWidth().padding(top=16.dp))
            }
        }
    }
}

fun decodeLocalPhoto(context:android.content.Context,uri:Uri,target:Int):Bitmap {
    val resolver=context.contentResolver
    if(Build.VERSION.SDK_INT>=28)return ImageDecoder.decodeBitmap(ImageDecoder.createSource(resolver,uri)){decoder,info,_->
        val factor=(target.toFloat()/maxOf(info.size.width,info.size.height)).coerceAtMost(1f)
        decoder.setTargetSize((info.size.width*factor).toInt().coerceAtLeast(1),(info.size.height*factor).toInt().coerceAtLeast(1));decoder.allocator=ImageDecoder.ALLOCATOR_SOFTWARE
    }
    val bounds=BitmapFactory.Options().apply{inJustDecodeBounds=true};resolver.openInputStream(uri)?.use{BitmapFactory.decodeStream(it,null,bounds)}
    val options=BitmapFactory.Options().apply{inSampleSize=1;while(bounds.outWidth/inSampleSize>target||bounds.outHeight/inSampleSize>target)inSampleSize*=2}
    val bitmap=resolver.openInputStream(uri)?.use{BitmapFactory.decodeStream(it,null,options)} ?: error("No image")
    val orientation=resolver.openInputStream(uri)?.use{android.media.ExifInterface(it).getAttributeInt(android.media.ExifInterface.TAG_ORIENTATION,1)} ?: 1
    val matrix=android.graphics.Matrix().apply { when(orientation){2->setScale(-1f,1f);3->setRotate(180f);4->setScale(1f,-1f);5->{setRotate(90f);postScale(-1f,1f)};6->setRotate(90f);7->{setRotate(-90f);postScale(-1f,1f)};8->setRotate(-90f)} }
    return if(orientation==1)bitmap else Bitmap.createBitmap(bitmap,0,0,bitmap.width,bitmap.height,matrix,true)
}
