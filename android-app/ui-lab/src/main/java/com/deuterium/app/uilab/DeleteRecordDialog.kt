package com.deuterium.app.uilab

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.DeleteOutline
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.launch

@Composable
fun DeleteRecordButton(onClick:()->Unit,modifier:Modifier=Modifier,enabled:Boolean=true,label:String="删除记录") {
    MotionButton(onClick,modifier,enabled,shape=RoundedCornerShape(14.dp),colors=ButtonDefaults.buttonColors(
        containerColor=MaterialTheme.colorScheme.errorContainer,contentColor=MaterialTheme.colorScheme.error,
        disabledContainerColor=MaterialTheme.colorScheme.errorContainer.copy(alpha=.5f),disabledContentColor=MaterialTheme.colorScheme.error.copy(alpha=.45f))) {
        Icon(Icons.Outlined.DeleteOutline,null,Modifier.size(18.dp));Spacer(Modifier.width(7.dp));Text(label)
    }
}

@Composable
fun DeleteRecordDialog(title: String, onDismiss: () -> Unit, onDelete: suspend () -> Boolean) {
    val scope = rememberCoroutineScope()
    var busy by remember { mutableStateOf(false) }
    var failed by remember { mutableStateOf(false) }
    IosDialog({ if (!busy) onDismiss() }, { Text("删除这条记录？") }, {
        Column {
            Text(title, style = MaterialTheme.typography.titleMedium)
            Text("删除后不再显示在你的记录列表中，交易凭证和对方的记录会保留。", Modifier.padding(top = 12.dp))
            if (failed) Text("暂时无法删除，记录可能仍需处理。请刷新后重试。", Modifier.padding(top = 10.dp), color = MaterialTheme.colorScheme.error)
        }
    }, {
        DeleteRecordButton({ if (!busy) scope.launch {
            busy = true
            try { if (onDelete()) onDismiss() else failed = true } finally { busy = false }
        } }, enabled = !busy, label=if (busy) "正在删除…" else "删除记录")
    }, { PlainButton(onDismiss, enabled = !busy) { Text("取消") } })
}
