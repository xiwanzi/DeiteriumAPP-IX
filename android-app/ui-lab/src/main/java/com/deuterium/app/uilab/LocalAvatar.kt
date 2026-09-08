package com.deuterium.app.uilab

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.layout.ContentScale

val LocalAccountAvatar = staticCompositionLocalOf<PlayerProfile?> { null }

@Composable
fun LocalAvatar(uri: String?, name: String, modifier: Modifier = Modifier) {
    var loaded by remember(uri,name){mutableStateOf(false)}
    Box(modifier.clip(CircleShape).background(MaterialTheme.colorScheme.primaryContainer), contentAlignment = Alignment.Center) {
        Text(name.take(1).uppercase(),modifier=Modifier.graphicsLayer{alpha=if(loaded)0f else 1f},color = MaterialTheme.colorScheme.primary, style = MaterialTheme.typography.headlineSmall)
        uri?.takeIf{it.isNotBlank()}?.let{CachedPhoto(it,"$name 的头像",Modifier.fillMaxSize(),ContentScale.Crop,showError=false,onLoaded={value->loaded=value})}
    }
}
