package com.deuterium.app.uilab

import android.graphics.BitmapFactory
import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.pager.*
import androidx.compose.foundation.shape.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.*
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.unit.*
import kotlinx.coroutines.*

@Composable
fun ProductPhoto(resource:Int,description:String,modifier:Modifier=Modifier,scale:ContentScale=ContentScale.Fit) {
    val resources=LocalContext.current.resources
    val bitmap by produceState<ImageBitmap?>(null,resource){value=withContext(Dispatchers.IO){
        val bounds=BitmapFactory.Options().apply{inJustDecodeBounds=true};BitmapFactory.decodeResource(resources,resource,bounds)
        val options=BitmapFactory.Options().apply{inSampleSize=1;while(bounds.outWidth/inSampleSize>1500||bounds.outHeight/inSampleSize>1500)inSampleSize*=2}
        BitmapFactory.decodeResource(resources,resource,options)?.asImageBitmap()
    }}
    Box(modifier.background(Color.White),contentAlignment=Alignment.Center){bitmap?.let{Image(it,description,Modifier.fillMaxSize(),contentScale=scale)}}
}
@Composable
fun ServerAssetImage(source:String?,description:String,modifier:Modifier=Modifier,scale:ContentScale=ContentScale.Crop) {
    Box(modifier.background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha=.3f)),contentAlignment=Alignment.Center){
        source?.takeIf{it.isNotBlank()}?.let{CachedPhoto(it,description,Modifier.fillMaxSize(),scale)}
    }
}

@Composable
fun ShoppingBagFlight(product:ShopProduct,start:Offset,end:Offset,progress:Float) {
    ServerAssetImage(product.photos.firstOrNull(),product.name,Modifier.size(72.dp).graphicsLayer{
        val t=progress.coerceIn(0f,1f)
        val arc=kotlin.math.sin(t*Math.PI).toFloat()
        val current=start+(end-start)*t+Offset(-90f*arc,-100f*arc)
        translationX=current.x-size.width/2;translationY=current.y-size.height/2
        scaleX=1f-.78f*t;scaleY=scaleX;alpha=1f-.4f*t
    }.clip(RoundedCornerShape(16.dp)),scale=ContentScale.Crop)
}
@Composable
fun ProductGallery(product:ShopProduct,modifier:Modifier=Modifier) {
    val images=product.photos;val pager=rememberPagerState{images.size}
    Column(modifier,horizontalAlignment=Alignment.CenterHorizontally){
        HorizontalPager(pager,Modifier.fillMaxWidth().height(260.dp).clip(RoundedCornerShape(24.dp))){index->ServerAssetImage(images[index],"${product.name} 图片 ${index+1}",Modifier.fillMaxSize())}
        Row(Modifier.padding(top=14.dp),horizontalArrangement=Arrangement.spacedBy(7.dp)){images.indices.forEach{Box(Modifier.size(5.dp).background(if(it==pager.currentPage)MaterialTheme.colorScheme.onSurface else MaterialTheme.colorScheme.outlineVariant,CircleShape))}}
    }
}
@Composable
fun ListingImage(listing:MarketListing,modifier:Modifier=Modifier) {
    val uri=listing.photos.firstOrNull()
    if(uri!=null)ServerAssetImage(uri,listing.title,modifier) else ProductArt(listing.artKey,modifier.background(MaterialTheme.colorScheme.primaryContainer.copy(alpha=.35f),RoundedCornerShape(22.dp)))
}
@Composable
fun OrderThumbnail(line:OrderLine,modifier:Modifier=Modifier) {
    if(line.image!=0)Image(painterResource(line.image),line.title,modifier.clip(RoundedCornerShape(13.dp)),contentScale=ContentScale.Crop)
    else ListingImage(MarketListing(line.productId,line.title,line.subtitle,"","",line.unitPrice,1,"","",emptySet(),"",line.artKey,line.imageUri),modifier.clip(RoundedCornerShape(13.dp)))
}

@Composable
fun ListingGallery(listing:MarketListing) {
    val photos=listing.photos
    if(photos.size<=1){ListingImage(listing,Modifier.fillMaxWidth().height(270.dp).clip(RoundedCornerShape(24.dp)));return}
    val pager=rememberPagerState{photos.size}
    Column(horizontalAlignment=Alignment.CenterHorizontally){HorizontalPager(pager,Modifier.fillMaxWidth().height(280.dp).clip(RoundedCornerShape(24.dp))){index->ListingImage(listing.copy(imageUri=photos[index],imageUris=listOf(photos[index])),Modifier.fillMaxSize())};Row(Modifier.padding(top=12.dp),horizontalArrangement=Arrangement.spacedBy(6.dp)){photos.indices.forEach{Box(Modifier.size(5.dp).background(if(it==pager.currentPage)MaterialTheme.colorScheme.onSurface else MaterialTheme.colorScheme.outlineVariant,CircleShape))}}}
}
