package com.deuterium.app.uilab

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.Alignment
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.launch

@Composable
fun DeleteRecordButton(onClick:()->Unit,modifier:Modifier=Modifier,enabled:Boolean=true,label:String="删除记录") {
    PlainButton(onClick,modifier,enabled) {
        Text(label,style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.error.copy(alpha=if(enabled)1f else .35f))
    }
}

@Composable
fun DeleteRecordDialog(title: String, onDismiss: () -> Unit, onDelete: suspend () -> Boolean) {
    val scope = rememberCoroutineScope()
    var busy by remember { mutableStateOf(false) }
    var failed by remember { mutableStateOf(false) }
    IosDialog({ if (!busy) onDismiss() }, { Text("删除这条记录？") }, {
        Column(Modifier.fillMaxWidth(),horizontalAlignment=Alignment.CenterHorizontally) {
            Text(title, style = MaterialTheme.typography.titleMedium)
            Text("删除后不再显示在你的记录列表中，交易凭证和对方的记录会保留。", Modifier.padding(top = 12.dp))
            if (failed) Text("暂时无法删除，记录可能仍需处理。请刷新后重试。", Modifier.padding(top = 10.dp), color = MaterialTheme.colorScheme.error)
        }
    }, {
        PlainButton({ if (!busy) scope.launch {
            busy = true
            try { if (onDelete()) onDismiss() else failed = true } finally { busy = false }
        } }, Modifier.fillMaxWidth().heightIn(min=51.dp), enabled = !busy) {
            Text(if(busy) "处理中…" else "确认",color=MaterialTheme.colorScheme.error.copy(alpha=if(busy).35f else 1f))
        }
    }, { PlainButton(onDismiss, Modifier.fillMaxWidth().heightIn(min=51.dp), enabled = !busy) { Text("取消") } })
}
