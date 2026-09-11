package com.deuterium.app.uilab

import android.graphics.Bitmap
import android.graphics.Canvas
import androidx.compose.foundation.Image
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.platform.LocalContext
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.io.File

/** Render the installed launcher drawable, including its actual adaptive icon mask and background. */
@Composable
fun AppLauncherIcon(modifier:Modifier=Modifier) {
    val context=LocalContext.current
    val launcher=remember(context){LauncherIcons.get(context)}
    val selected=launcher.current
    val bitmap=remember(context,selected) {
        val resource=if(selected==LauncherIconChoice.Default)R.mipmap.ic_launcher else R.mipmap.ic_launcher_anniversary
        val drawable=checkNotNull(context.getDrawable(resource)).mutate()
        Bitmap.createBitmap(512,512,Bitmap.Config.ARGB_8888).also { drawable.setBounds(0,0,512,512);drawable.draw(Canvas(it)) }.asImageBitmap()
    }
    Image(bitmap,"Deuterium App 图标",modifier)
}

/** The version badge is shared by login and the launch transition; resource packs can update it. */
@Composable
fun LoginVersionBadge(modifier:Modifier=Modifier) {
    val context=LocalContext.current
    val directory=LocalAppUpdates.current?.resourceDirectory
    val images=remember(context){AppImages.get(context)}
    var failed by remember(directory){mutableStateOf(false)}
    val source by produceState<Any>(R.drawable.deuterium_brand,directory,failed) {
        value=if(failed)R.drawable.deuterium_brand else versionBadgeSource(directory)
    }
    val request=remember(context,source) {
        versionBadgeRequest(context,source)
    }
    // The launch badge scales up to 192dp. Decode off the UI thread and share
    // the bounded result between launch and login through the existing loader.
    coil3.compose.AsyncImage(request,"Deuterium 版本标",imageLoader=images.loader,modifier=modifier,onError={if(source is File)failed=true})
}

private suspend fun versionBadgeSource(directory:String?):Any=withContext(Dispatchers.IO) {
    directory?.let{File(it,"images/deuterium-brand.png")}?.takeIf{it.isFile} ?: R.drawable.deuterium_brand
}

private fun versionBadgeRequest(context:android.content.Context,source:Any)=
    coil3.request.ImageRequest.Builder(context).data(source).size(768,768).build()

/** Optional prefetch uses the identical request; it never changes the launch clock or ready state. */
internal suspend fun prewarmVersionBadge(context:android.content.Context,directory:String?) {
    val source=versionBadgeSource(directory);val loader=AppImages.get(context).loader
    val result=loader.execute(versionBadgeRequest(context,source))
    if(result is coil3.request.ErrorResult&&source is File)loader.execute(versionBadgeRequest(context,R.drawable.deuterium_brand))
}
