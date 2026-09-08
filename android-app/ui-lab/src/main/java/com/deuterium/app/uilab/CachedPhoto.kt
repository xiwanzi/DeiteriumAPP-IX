package com.deuterium.app.uilab

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import coil3.compose.AsyncImage
import kotlinx.coroutines.delay
import java.time.Instant

@Composable
internal fun CachedPhoto(source:String,description:String,modifier:Modifier,scale:ContentScale,showError:Boolean=true,onLoaded:(Boolean)->Unit={}){
    val context=LocalContext.current
    val images=remember(context){AppImages.get(context)}
    val api=BackendApi.get(context)
    val scope=api.financialScope()
    val authorization=BackendAssets.imageKey(context,source)
    val access=RemoteImageUrls.access(scope,source)
    var expired by remember(source,scope,access?.retainUntil,access?.expired){mutableStateOf(access?.expired==true||access?.retainUntil?.isAfter(Instant.now())==false)}
    var error by remember(source,scope,authorization,images.epoch){mutableStateOf<String?>(null)}
    LaunchedEffect(source,scope,access?.retainUntil){
        access?.retainUntil?.let{expiry->delay((expiry.toEpochMilli()-System.currentTimeMillis()).coerceAtLeast(0));expired=true}
    }
    val request=remember(source,scope,authorization,images.epoch){images.request(source)}
    LaunchedEffect(expired){if(expired)onLoaded(false)}
    Box(modifier,contentAlignment=Alignment.Center){
        if(!expired)AsyncImage(request,description,images.loader,Modifier.fillMaxSize(),contentScale=scale,
            onError={result->onLoaded(false);error=if((result.result.throwable as? ApiFailure)?.code=="IMAGE_EXPIRED")"图片已过期" else "图片暂不可用"},
            onSuccess={error=null;onLoaded(true)})
        if(showError&&(expired||error!=null))Text(if(expired)"图片已过期" else error.orEmpty(),color=MaterialTheme.colorScheme.onSurfaceVariant,style=MaterialTheme.typography.labelSmall)
    }
}
