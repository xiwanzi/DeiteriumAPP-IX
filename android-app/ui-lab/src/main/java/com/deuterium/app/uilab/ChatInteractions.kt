package com.deuterium.app.uilab

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.TextFieldValue
import androidx.compose.ui.unit.dp

fun mentionRange(value: TextFieldValue): IntRange? {
    if(!value.selection.collapsed) return null
    val cursor = value.selection.start.coerceIn(0, value.text.length)
    val start = value.text.lastIndexOf('@', (cursor - 1).coerceAtLeast(0))
    if(start < 0 || start >= cursor) return null
    val query = value.text.substring(start + 1, cursor)
    if(query.any { it.isWhitespace() || it == '@' } || query.length > 24) return null
    return start until cursor
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PlayerDetails(player: PlayerProfile, state: LabState, onMention: () -> Unit, onTransfer: () -> Unit, onClose: () -> Unit) {
    IosSheet(onDismissRequest = onClose, containerColor = MaterialTheme.colorScheme.surface) {
        Column(Modifier.fillMaxWidth().verticalScroll(rememberScrollState()).padding(24.dp).navigationBarsPadding()) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                PlayerAvatar(player.name, Modifier.size(60.dp))
                Column(Modifier.padding(start = 16.dp)) {
                    Text(player.name, style = MaterialTheme.typography.headlineSmall)
                    Text(if(player.online) "服务器在线" else "暂时离线", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.primary)
                }
            }
            Text(player.bio, Modifier.padding(vertical = 18.dp), style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
            DetailRow("QQ", player.qq)
            Row(Modifier.fillMaxWidth().padding(vertical = 10.dp), verticalAlignment = Alignment.CenterVertically) {
                Icon(Icons.Outlined.StarOutline, null, tint = MaterialTheme.colorScheme.primary)
                Column(Modifier.weight(1f).padding(start = 12.dp)) {
                    Text("特别关心", style = MaterialTheme.typography.titleMedium)
                    Text("此玩家发言时显示本地提醒", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
                IosSwitch(player.name in state.followed, { state.toggleFollow(player.name) })
            }
            Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                SecondaryButton(onClick = onMention, modifier = Modifier.weight(1f).height(50.dp)) { Text("@ 提及") }
                MotionButton(onClick = onTransfer, modifier = Modifier.weight(1f).height(50.dp)) { Text("转账给 TA") }
            }
            Spacer(Modifier.height(18.dp))

        }
    }
}
