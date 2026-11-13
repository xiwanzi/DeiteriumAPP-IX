package com.deuterium.app.uilab

import android.app.Instrumentation
import android.content.Intent
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.os.Bundle
import android.os.ParcelFileDescriptor
import androidx.activity.compose.setContent
import androidx.compose.foundation.Image
import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.unit.dp
import com.kyant.backdrop.backdrops.layerBackdrop
import com.kyant.backdrop.backdrops.rememberLayerBackdrop

/** Local study image only: supplied over adb into the isolated emulator, never packaged in App. */
internal object GlassCalibrationCheck {
    fun run(test: Instrumentation) {
        val result = Bundle()
        check(test.targetContext.packageName.endsWith(".chatmediaqa"))
        try {
            val input = test.uiAutomation.executeShellCommand("cat /data/local/tmp/deuterium-glass-calibration.png")
            val bitmap = ParcelFileDescriptor.AutoCloseInputStream(input).use { BitmapFactory.decodeStream(it) }!!
            val activity = test.startActivitySync(Intent(test.targetContext, DeuteriumActivity::class.java)
                .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)) as DeuteriumActivity
            test.runOnMainSync {
                activity.setContent {
                    LabTheme(1, true) {
                        val backdrop = rememberLayerBackdrop()
                        Box(Modifier.fillMaxSize()) {
                            Image(bitmap.asImageBitmap(), null, Modifier.fillMaxSize().layerBackdrop(backdrop), contentScale = ContentScale.FillBounds)
                            BottomGlassFade(Modifier.align(Alignment.BottomCenter))
                            // Xiaomi reference has a measured 68 px system gesture inset at density 3.25.
                            Box(Modifier.align(Alignment.BottomCenter).padding(start=18.dp,end=18.dp,bottom=(68f/3.25f+8f).dp).height(64.dp)) {
                                MovingGlassNav(listOf("商城","市场","信息","我的"),
                                    listOf(Icons.Outlined.Storefront,Icons.Outlined.GridView,Icons.Outlined.ChatBubbleOutline,Icons.Outlined.PersonOutline),
                                    0,true,backdrop=backdrop,
                                    parameters=GlassMaterials().bottomBar) {}
                            }
                        }
                    }
                }
            }
            fun shot(name: String) {
                test.waitForIdleSync();Thread.sleep(700)
                repeat(30) {
                    Thread.sleep(150)
                    val b = test.uiAutomation.takeScreenshot()!!
                    val ready = listOf(100,600,1000).all { x ->
                        val expected=bitmap.getPixel(x,2300);val actual=b.getPixel(x,2300)
                        kotlin.math.abs(android.graphics.Color.red(expected)-android.graphics.Color.red(actual)) < 8 &&
                            kotlin.math.abs(android.graphics.Color.blue(expected)-android.graphics.Color.blue(actual)) < 8
                    }
                    if (ready) {
                        test.targetContext.filesDir.resolve(name).outputStream().use { b.compress(Bitmap.CompressFormat.PNG,100,it) }
                        b.recycle();return
                    }
                    b.recycle()
                }
                error("Calibration backdrop did not render")
            }
            shot("calibration-current.png")
            result.putString("calibration", "PASS calibrated material on the reconstructed identical backdrop")
            test.finish(-1,result)
        } catch (failure:Throwable) { result.putString("failure",failure.stackTraceToString());test.finish(1,result) }
    }
}
