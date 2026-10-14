package com.deuterium.app.uilab

import android.app.Instrumentation
import android.content.ContextWrapper
import android.content.Intent
import android.graphics.Bitmap
import android.os.Bundle
import android.view.KeyEvent
import android.view.accessibility.AccessibilityNodeInfo
import androidx.activity.compose.setContent
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.runBlocking
import okhttp3.Protocol
import okhttp3.Response
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.ResponseBody.Companion.toResponseBody
import okio.Buffer
import org.json.JSONArray
import org.json.JSONObject
import java.io.File
import java.time.Instant
import java.util.concurrent.atomic.AtomicBoolean
import java.util.concurrent.atomic.AtomicInteger

/** QA APK only: intercepted HTTP and separate preferences; never real accounts. */
object AccountSessionsCheck {
    fun run(test: Instrumentation) {
        val result = Bundle()
        val instance = BackendApi::class.java.getDeclaredField("instance").apply { isAccessible = true }
        val previous = instance.get(null)
        val context = object : ContextWrapper(test.targetContext) {
            override fun getSharedPreferences(name: String, mode: Int) = super.getSharedPreferences("qa-account-sessions-$name", mode)
        }
        val remoteRevoked = AtomicBoolean(false)
        val revokes = AtomicInteger()
        val registrations = AtomicInteger()
        val user = JSONObject().put("userId", "qa-account").put("playerRef", "qa-player").put("gameId", "FixturePlayer").put("displayName", "测试玩家").put("email", "qa@example.com").put("gameVerified", false)
        try {
            context.getSharedPreferences("backend-v2", 0).edit().clear().commit()
            val api = BackendApi(context)
            val client = api.http.newBuilder().addInterceptor { chain ->
                val request = chain.request()
                val path = request.url.encodedPath
                val now = Instant.now().toString()
                val data = when {
                    path.endsWith("/account/login") -> JSONObject().put("token", "qa-session-token").put("user", user)
                    path.endsWith("/account/me") -> JSONObject().put("user", user)
                    path.endsWith("/account/sessions") -> {
                        val items = JSONArray().put(JSONObject().put("sessionId", "qa-current").put("clientKind", "app").put("deviceName", "Android · 当前测试设备").put("createdAt", now).put("lastSeenAt", now).put("expiresAt", JSONObject.NULL).put("current", true))
                        if (!remoteRevoked.get()) items.put(JSONObject().put("sessionId", "qa-other").put("clientKind", "web").put("deviceName", "Windows · Chrome").put("createdAt", now).put("lastSeenAt", now).put("expiresAt", Instant.now().plusSeconds(30L * 86400).toString()).put("current", false))
                        JSONObject().put("items", items)
                    }
                    path.endsWith("/account/sessions/qa-other") -> { check(request.method == "DELETE"); revokes.incrementAndGet(); remoteRevoked.set(true); JSONObject().put("revoked", true) }
                    path.endsWith("/account/game-profile") -> {
                        val buffer = Buffer(); request.body!!.writeTo(buffer)
                        val input = JSONObject(buffer.readUtf8())
                        check(input.getString("clientRequestId").isNotBlank())
                        check(input.getString("gameId") == "OwnPlayerUi")
                        registrations.incrementAndGet(); user.put("gameId", "OwnPlayerUi")
                        JSONObject().put("registered", true).put("gameVerified", false)
                    }
                    else -> JSONObject()
                }
                Response.Builder().request(request).protocol(Protocol.HTTP_1_1).code(200).message("fixture")
                    .body(JSONObject().put("data", data).toString().toResponseBody("application/json".toMediaType())).build()
            }.build()
            BackendApi::class.java.getDeclaredField("http").apply { isAccessible = true }.set(api, client)
            instance.set(null, api)
            runBlocking { api.login("qa@example.com", "test-only-password") }
            val activity = test.startActivitySync(Intent(test.targetContext, DeuteriumActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)) as DeuteriumActivity
            fun find(node: AccessibilityNodeInfo?, predicate: (AccessibilityNodeInfo) -> Boolean): AccessibilityNodeInfo? {
                if (node == null) return null
                if (predicate(node)) return node
                for (i in 0 until node.childCount) find(node.getChild(i), predicate)?.let { return it }
                return null
            }
            fun waitText(text: String): AccessibilityNodeInfo {
                repeat(70) { find(test.uiAutomation.rootInActiveWindow) { it.text?.toString() == text }?.let { return it }; Thread.sleep(100) }
                error("Missing native control: $text")
            }
            fun click(node: AccessibilityNodeInfo) {
                val label = node.text?.toString()
                repeat(70) {
                    var target: AccessibilityNodeInfo? = find(test.uiAutomation.rootInActiveWindow) { it.text?.toString() == label }
                    while (target != null && !target.isClickable) target = target.parent
                    if (target?.isEnabled == true && target?.performAction(AccessibilityNodeInfo.ACTION_CLICK) == true) return
                    Thread.sleep(50)
                }
                error("Native control did not become clickable: $label")
            }
            fun screenshot(name: String) {
                test.waitForIdleSync(); Thread.sleep(300)
                test.uiAutomation.takeScreenshot()!!.let { image ->
                    check((1..9).any { (image.getPixel(image.width/2, image.height*it/10) and 0xffffff) != 0 }) { "Emulator rendering is disabled; screenshot is black" }
                    File(test.targetContext.filesDir, "qa-account-$name.png").outputStream().use { image.compress(Bitmap.CompressFormat.PNG, 100, it) }; image.recycle()
                }
            }
            fun devices(theme: Int) = test.runOnMainSync {
                activity.setContent { LabTheme(theme, false) { CompositionLocalProvider(LocalContentColor provides MaterialTheme.colorScheme.onSurface) {
                    Box(Modifier.fillMaxSize().background(MaterialTheme.colorScheme.background)) {
                        DeviceSessionsPage(100.dp)
                        Text("登录设备", Modifier.statusBarsPadding().padding(24.dp), style = MaterialTheme.typography.titleLarge)
                    }
                } } }
            }
            devices(1); waitText("Windows · Chrome"); waitText("长期登录"); screenshot("devices-light")
            click(waitText("退出")); waitText("退出所选设备？"); screenshot("device-confirm")
            click(waitText("确认退出")); waitText("已退出所选设备")
            check(revokes.get() == 1); check(api.signedIn)
            screenshot("device-revoked")
            remoteRevoked.set(false); devices(2); click(waitText("刷新")); waitText("Windows · Chrome"); screenshot("devices-dark")
            user.put("gameId", ""); runBlocking { api.verifySession() }
            test.runOnMainSync { activity.setContent { LabTheme(1, false) { CompositionLocalProvider(LocalContentColor provides MaterialTheme.colorScheme.onSurface) { GameVerificationPage(false, {}, {}) } } } }
            waitText("登记游戏身份"); waitText("待登记游戏身份"); screenshot("profile-released")
            val edit = find(test.uiAutomation.rootInActiveWindow) { it.isEditable } ?: error("Missing game ID input")
            check(edit.performAction(AccessibilityNodeInfo.ACTION_SET_TEXT, Bundle().apply { putCharSequence(AccessibilityNodeInfo.ACTION_ARGUMENT_SET_TEXT_CHARSEQUENCE, "OwnPlayerUi") }))
            click(waitText("登记游戏身份")); waitText("QQ 号"); check(registrations.get() == 1); check(api.playerRef == "qa-player" && !api.gameVerified)
            screenshot("profile-registered")
            click(waitText("登录设备")); waitText("返回身份验证"); waitText("Windows · Chrome")
            test.sendKeyDownUpSync(KeyEvent.KEYCODE_BACK); waitText("QQ 号")
            result.putString("device_list_light_dark", "PASS")
            result.putString("device_revoke_and_current_session", "PASS")
            result.putString("released_profile_register_and_back", "PASS")
            result.putString("status", "PASS")
            test.finish(-1, result)
        } catch (error: Throwable) { result.putString("error", error.stackTraceToString()); test.finish(1, result) }
        finally { instance.set(null, previous) }
    }
}
