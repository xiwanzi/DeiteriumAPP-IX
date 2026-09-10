package com.deuterium.app.uilab

import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Close
import androidx.compose.material.icons.outlined.NotificationsNone
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.layer.GraphicsLayer
import androidx.compose.ui.semantics.*
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp

/** Shared, non-modal notification surface; touches outside it stay with the page. */
@Composable
fun InAppNoticeCard(title:String,body:String,onOpen:()->Unit,onDismiss:()->Unit,modifier:Modifier=Modifier,backdrop:GraphicsLayer?=LocalOverlayBackdrop.current) {
    LiquidGlass(modifier.fillMaxWidth().semantics{liveRegion=LiveRegionMode.Polite},backdrop=backdrop,onClick=onOpen) {
        Row(Modifier.padding(start=15.dp,top=12.dp,end=8.dp,bottom=12.dp),verticalAlignment=Alignment.CenterVertically) {
            Icon(Icons.Outlined.NotificationsNone,null,tint=MaterialTheme.colorScheme.primary)
            Column(Modifier.weight(1f).padding(horizontal=10.dp)) {
                Text(title,style=MaterialTheme.typography.titleMedium)
                Text(body,style=MaterialTheme.typography.bodySmall,maxLines=2,overflow=TextOverflow.Ellipsis)
            }
            IconButton(onDismiss,Modifier.size(44.dp)){Icon(Icons.Outlined.Close,"关闭提醒",Modifier.size(18.dp))}
        }
    }
}
