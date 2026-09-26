package com.deuterium.app.uilab

import androidx.activity.compose.BackHandler
import androidx.compose.runtime.Composable
import androidx.compose.ui.platform.LocalSoftwareKeyboardController
import androidx.compose.ui.platform.LocalView
import androidx.core.view.ViewCompat
import androidx.core.view.WindowInsetsCompat

@Composable
internal fun NavigationBackHandler(enabled:Boolean,onBack:()->Unit){
    val view=LocalView.current
    val keyboard=LocalSoftwareKeyboardController.current
    // IME normally consumes the first Back. Keep our callback registered while
    // its closing animation still has height, so the next Back cannot exit the task.
    BackHandler(enabled){
        if(ViewCompat.getRootWindowInsets(view)?.isVisible(WindowInsetsCompat.Type.ime())==true)keyboard?.hide()
        else onBack()
    }
}
