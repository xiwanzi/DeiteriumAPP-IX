package com.deuterium.app.uilab

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.ImageBitmap
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.IntSize
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlin.math.*

private data class ShopImage(val bitmap:ImageBitmap,val bounds:ShopImageBounds)

@Composable
internal fun ShopSubjectImage(source:String?,description:String,modifier:Modifier=Modifier) {
    val images=AppImages.get(LocalContext.current)
    var failed by remember(source,images.epoch){mutableStateOf(false)}
    val image by produceState<ShopImage?>(null,source,images.epoch){
        value=null
        if(source!=null)try {
            val bitmap=images.bitmap(source,512)
            value=withContext(Dispatchers.Default){val pixels=IntArray(bitmap.width*bitmap.height);bitmap.getPixels(pixels,0,bitmap.width,0,0,bitmap.width,bitmap.height);ShopImage(bitmap.asImageBitmap(),shopImageBounds(bitmap.width,bitmap.height,pixels))}
        }catch(c:kotlinx.coroutines.CancellationException){throw c}catch(_:Exception){failed=true}
    }
    Box(modifier.background(image?.let{Color(it.bounds.background)} ?: MaterialTheme.colorScheme.surfaceVariant.copy(alpha=.25f))) {
        image?.let{picture->Canvas(Modifier.fillMaxSize().semantics{contentDescription=description}) {
            val b=picture.bounds
            val scale=min(size.width*.88f/b.width,size.height*.88f/b.height)
            val width=(b.width*scale).roundToInt().coerceAtLeast(1);val height=(b.height*scale).roundToInt().coerceAtLeast(1)
            drawImage(picture.bitmap,srcOffset=IntOffset(b.x,b.y),srcSize=IntSize(b.width,b.height),dstOffset=IntOffset(((size.width-width)/2).roundToInt(),((size.height-height)/2).roundToInt()),dstSize=IntSize(width,height))
        }}
        if(failed)ServerAssetImage(source,description,Modifier.fillMaxSize(),scale=androidx.compose.ui.layout.ContentScale.Fit)
    }
}
