package com.deuterium.app.uilab

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Computer
import androidx.compose.material.icons.outlined.Smartphone
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.launch
import org.json.JSONObject
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter

@Composable
fun DeviceSessionsPage(topInset: Dp) {
    val api = BackendApi.get(LocalContext.current)
    val scope = rememberCoroutineScope()
    var sessions by remember { mutableStateOf<List<JSONObject>?>(null) }
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var confirmation by remember { mutableStateOf<JSONObject?>(null) }
    var others by remember { mutableStateOf(false) }
    var message by remember { mutableStateOf<String?>(null) }
    suspend fun load() {
        busy = true; error = null
        try {
            val result = api.request("GET", "/account/sessions").getJSONArray("items")
            sessions = (0 until result.length()).map { result.getJSONObject(it) }
        } catch (cancelled: CancellationException) { throw cancelled }
        catch (e: Exception) { error = e.message }
        finally { busy = false }
    }
    LaunchedEffect(api.playerRef) { load() }
    fun date(value: String): String = runCatching {
        DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm").withZone(ZoneId.systemDefault()).format(Instant.parse(value))
    }.getOrDefault("暂未记录")
    LazyColumn(contentPadding = PaddingValues(start = 16.dp, end = 16.dp, top = topInset + 16.dp, bottom = 40.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        item { Text("查看登录中的设备。不再使用某台设备时，可以在这里退出。", color = MaterialTheme.colorScheme.onSurfaceVariant, style = MaterialTheme.typography.bodyMedium) }
        item { Row(horizontalArrangement = Arrangement.spacedBy(16.dp)) {
            PlainButton({ if (!busy) scope.launch { load() } }) { Text(if (busy) "正在读取…" else "刷新") }
            if (sessions?.any { !it.optBoolean("current") } == true) PlainButton({ if (!busy) others = true }) { Text("退出其他设备") }
        } }
        error?.let { item { Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodyMedium) } }
        message?.let { item { Text(it, color = MaterialTheme.colorScheme.onSurfaceVariant, style = MaterialTheme.typography.bodyMedium) } }
        items(sessions.orEmpty(), key = { it.getString("sessionId") }) { device ->
            SettingsGroup { Row(Modifier.fillMaxWidth().padding(18.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(14.dp)) {
                Icon(if (device.optString("clientKind") == "web") Icons.Outlined.Computer else Icons.Outlined.Smartphone, null, tint = MaterialTheme.colorScheme.primary)
                Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                    Text(device.optString("deviceName").ifBlank { if (device.optString("clientKind") == "web") "网页设备（旧版）" else "App 设备（旧版）" }, style = MaterialTheme.typography.titleSmall)
                    Text("最近活动 · ${date(device.optString("lastSeenAt"))}", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    Text(if (device.isNull("expiresAt")) "长期登录" else "有效至 ${date(device.optString("expiresAt"))}", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    if (device.optBoolean("current")) Text("当前设备", color = MaterialTheme.colorScheme.primary, style = MaterialTheme.typography.labelMedium)
                }
                if (!device.optBoolean("current")) PlainButton({ if (!busy) confirmation = device }) { Text("退出") }
            } }
        }
        if (sessions?.size == 200) item { Text("显示最近使用的 200 个登录记录；“退出其他设备”会处理全部其他登录。", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant) }
    }
    if (others || confirmation != null) IosDialog(
        { if (!busy) { others = false; confirmation = null } },
        { Text(if (others) "退出其他设备？" else "退出所选设备？") },
        { Text("退出的设备需要重新登录，当前设备仍可继续使用。") },
        { PlainButton({
            if (!busy) scope.launch {
                busy = true; error = null
                try {
                    if (others) api.request("POST", "/account/sessions/revoke-others", JSONObject())
                    else api.request("DELETE", "/account/sessions/${confirmation!!.getString("sessionId")}")
                    others = false; confirmation = null; message = "已退出所选设备"; load()
                } catch (cancelled: CancellationException) { throw cancelled }
                catch (e: Exception) { error = e.message; others = false; confirmation = null }
                finally { busy = false }
            }
        }) { Text(if (busy) "正在退出…" else "确认退出", color = MaterialTheme.colorScheme.error) } },
        { PlainButton({ if (!busy) { others = false; confirmation = null } }) { Text("取消") } }
    )
}
