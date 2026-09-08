package com.deuterium.app.uilab

import android.graphics.drawable.ColorDrawable
import androidx.compose.runtime.*
import androidx.compose.ui.platform.LocalView
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import androidx.compose.ui.window.DialogWindowProvider

/** A transparent window and one shared dim/transition treatment for all floating surfaces. */
@Composable
fun IosOverlayHost(onDismissRequest:()->Unit,content:@Composable ()->Unit) {
    val motion=LocalMotion.current
    Dialog(onDismissRequest,properties=DialogProperties(usePlatformDefaultWidth=false,decorFitsSystemWindows=false)) {
        val view=LocalView.current
        SideEffect {
            (view.parent as? DialogWindowProvider)?.window?.apply {
                setBackgroundDrawable(ColorDrawable(android.graphics.Color.TRANSPARENT))
                setDimAmount(.18f)
                setWindowAnimations(if(motion)R.style.Animation_DeuteriumOverlay else 0)
            }
        }
        content()
    }
}
