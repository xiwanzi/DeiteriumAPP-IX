package com.deuterium.app.uilab

import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.graphics.Canvas
import androidx.compose.foundation.Image
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.io.File

/** Render the installed launcher drawable, including its actual adaptive icon mask and background. */
@Composable
fun AppLauncherIcon(modifier:Modifier=Modifier) {
    val context=LocalContext.current
    val bitmap=remember(context) {
        val drawable=context.applicationInfo.loadIcon(context.packageManager).mutate()
        Bitmap.createBitmap(256,256,Bitmap.Config.ARGB_8888).also { drawable.setBounds(0,0,256,256);drawable.draw(Canvas(it)) }.asImageBitmap()
    }
    Image(bitmap,"Deuterium App 图标",modifier)
}

/** The version badge is shared by login and the launch transition; resource packs can update it. */
@Composable
fun LoginVersionBadge(modifier:Modifier=Modifier) {
    val directory=LocalAppUpdates.current?.resourceDirectory
    val bitmap by produceState<Bitmap?>(null,directory) {
        value=withContext(Dispatchers.IO) {
            directory?.let { File(it,"images/deuterium-brand.png") }?.takeIf { it.isFile }?.let { file->
                val options=BitmapFactory.Options().apply { inJustDecodeBounds=true }
                BitmapFactory.decodeFile(file.path,options);options.inJustDecodeBounds=false;options.inSampleSize=1
                while(options.outWidth/options.inSampleSize>512||options.outHeight/options.inSampleSize>512)options.inSampleSize*=2
                BitmapFactory.decodeFile(file.path,options)
            }
        }
    }
    if(bitmap!=null)Image(bitmap!!.asImageBitmap(),"Deuterium 版本标",modifier)
    else Image(painterResource(R.drawable.deuterium_brand),"Deuterium 版本标",modifier)
}
