package com.deuterium.app.uilab

import android.graphics.drawable.ColorDrawable
import androidx.compose.runtime.*
import androidx.compose.ui.platform.LocalView
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import androidx.compose.ui.window.DialogWindowProvider

class IosOverlayRegistry { var count by mutableIntStateOf(0);internal set }
val LocalIosOverlayRegistry=staticCompositionLocalOf<IosOverlayRegistry?>{null}

/** A transparent window and one shared dim/transition treatment for all floating surfaces. */
@Composable
fun IosOverlayHost(onDismissRequest:()->Unit,content:@Composable ()->Unit) {
    val motion=LocalMotion.current
    val density=LocalDensity.current
    val registry=LocalIosOverlayRegistry.current
    DisposableEffect(registry){registry?.let{it.count++};onDispose{registry?.let{it.count--}}}
    Dialog(onDismissRequest,properties=DialogProperties(usePlatformDefaultWidth=false,decorFitsSystemWindows=false)) {
        val view=LocalView.current
        val window=(view.parent as? DialogWindowProvider)?.window
        DisposableEffect(window,motion) {
            window?.apply {
                setBackgroundDrawable(ColorDrawable(android.graphics.Color.TRANSPARENT))
                setDimAmount(.18f)
                setWindowAnimations(if(motion)R.style.Animation_DeuteriumOverlay else 0)
            }
            onDispose{}
        }
        CompositionLocalProvider(LocalDensity provides density){content()}
    }
}
