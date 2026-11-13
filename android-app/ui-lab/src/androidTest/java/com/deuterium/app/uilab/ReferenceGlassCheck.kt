package com.deuterium.app.uilab

import android.app.Instrumentation
import android.content.Intent
import android.graphics.Bitmap
import android.graphics.Rect
import android.os.Bundle
import android.os.SystemClock
import android.view.MotionEvent
import android.view.accessibility.AccessibilityNodeInfo
import androidx.activity.compose.setContent
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import com.kyant.backdrop.backdrops.layerBackdrop
import com.kyant.backdrop.backdrops.rememberLayerBackdrop
import kotlinx.coroutines.*
import java.time.Instant
import java.util.concurrent.atomic.AtomicInteger

/** Real components, synthetic local content. No account or remote write is used. */
internal object ReferenceGlassCheck {
    fun run(test: Instrumentation) {
        check(test.targetContext.packageName.endsWith(".chatmediaqa"))
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main.immediate)
        val report = Bundle()
        val output = test.targetContext.filesDir.resolve("reference-glass-12").apply { mkdirs() }
        val activity = test.startActivitySync(Intent(test.targetContext, DeuteriumActivity::class.java)
            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)) as DeuteriumActivity
        val theme = mutableIntStateOf(1)
        val scene = mutableStateOf("nav")
        val selected = mutableIntStateOf(0)
        val profiles = AtomicInteger()
        lateinit var state: LabState
        try {
            test.runOnMainSync {
                state = LabState(scope, userName = "FixtureSelf")
                val at = Instant.parse("2026-11-13T02:00:00Z").toEpochMilli()
                Players.clear()
                Players.addAll(listOf(PlayerProfile("Alice", playerRef = "fixture_alice"), PlayerProfile("Bob", playerRef = "fixture_bob")))
                state.chat.clear()
                state.chat.addAll((1L..20L).map { id ->
                    val who = if (id <= 18) "Alice" else "Bob"
                    ChatLine(id, who, "同一消息组 $id\n连续消息保持原来的内容宽度。", false,
                        serverAt = at + id * 1000, playerRef = "fixture_${who.lowercase()}")
                })
                activity.setContent {
                    LabTheme(theme.intValue, true) {
                        Box(Modifier.fillMaxSize().background(MaterialTheme.colorScheme.background)) {
                            if (scene.value == "nav") {
                                val backdrop = rememberLayerBackdrop()
                                Column(Modifier.fillMaxSize().layerBackdrop(backdrop)) {
                                    repeat(24) { row ->
                                        Text("页面内容 $row  ·  背景从玻璃后方连续经过",
                                            Modifier.fillMaxWidth().height(42.dp).background(
                                                if (row % 2 == 0) Color(0xFF83BFEA) else Color(0xFFEAA492)),
                                            color = Color(0xFF203040))
                                    }
                                }
                                Box(Modifier.align(Alignment.BottomCenter).navigationBarsPadding()
                                    .padding(horizontal = 18.dp, vertical = 12.dp).height(64.dp)) {
                                    MovingGlassNav(listOf("商城", "市场", "信息", "我的"),
                                        listOf(Icons.Outlined.Storefront, Icons.Outlined.GridView, Icons.Outlined.ChatBubbleOutline, Icons.Outlined.PersonOutline),
                                        selected.intValue, true, backdrop = backdrop) { selected.intValue = it }
                                }
                            } else {
                                ChatPage(state, 72.dp, { profiles.incrementAndGet() }, {})
                            }
                        }
                    }
                }
            }
            fun nodes(): List<AccessibilityNodeInfo> {
                test.uiAutomation.clearCache()
                val found = mutableListOf<AccessibilityNodeInfo>()
                fun walk(node: AccessibilityNodeInfo?) { if (node == null) return; found += node; repeat(node.childCount) { walk(node.getChild(it)) } }
                walk(test.uiAutomation.rootInActiveWindow)
                return found
            }
            fun find(label: String) = nodes().firstOrNull { it.text?.toString() == label || it.contentDescription?.toString() == label }
            fun waitFor(label: String): AccessibilityNodeInfo {
                repeat(45) { find(label)?.let { return it }; Thread.sleep(80) }
                error("Missing $label")
            }
            fun bounds(node: AccessibilityNodeInfo) = Rect().also { node.getBoundsInScreen(it) }
            fun shot(name: String): Bitmap {
                test.waitForIdleSync(); Thread.sleep(240)
                return checkNotNull(test.uiAutomation.takeScreenshot()).also { image ->
                    output.resolve("$name.png").outputStream().use { image.compress(Bitmap.CompressFormat.PNG, 100, it) }
                }
            }
            fun event(action: Int, x: Float, y: Float, downAt: Long) {
                val e = MotionEvent.obtain(downAt, SystemClock.uptimeMillis(), action, x, y, 0)
                test.sendPointerSync(e); e.recycle()
            }
            for (mode in listOf(1, 2)) {
                test.runOnMainSync { theme.intValue = mode; selected.intValue = 0 }
                val tab = bounds(waitFor("商城"))
                val next = bounds(waitFor("市场"))
                val rest = shot("nav-$mode-rest")
                val now = SystemClock.uptimeMillis()
                event(MotionEvent.ACTION_DOWN, tab.exactCenterX(), tab.exactCenterY(), now)
                Thread.sleep(450)
                val pressed = shot("nav-$mode-pressed")
                var outsideChanged = 0
                val density = test.targetContext.resources.displayMetrics.density
                val barTop = (tab.bottom - 48 * density).toInt()
                for (y in (barTop - 24 * density).toInt().coerceAtLeast(0) until (barTop - 1 * density).toInt()) {
                    for (x in (tab.centerX() - 40 * density).toInt().coerceAtLeast(0) until (tab.centerX() + 40 * density).toInt().coerceAtMost(rest.width)) {
                        val a = rest.getPixel(x,y); val b = pressed.getPixel(x,y)
                        if (kotlin.math.abs(android.graphics.Color.red(a) - android.graphics.Color.red(b)) > 10) outsideChanged++
                    }
                }
                check(outsideChanged > 30) { "Pressed capsule did not render outside its shell: $outsideChanged" }
                rest.recycle(); pressed.recycle()
                repeat(14) { step ->
                    event(MotionEvent.ACTION_MOVE, tab.exactCenterX() + (next.exactCenterX() - tab.exactCenterX()) * (step + 1) / 14, tab.exactCenterY(), now)
                    Thread.sleep(20)
                }
                shot("nav-$mode-drag").recycle()
                event(MotionEvent.ACTION_UP, next.exactCenterX(), tab.exactCenterY(), now)
                Thread.sleep(650)
                check(selected.intValue == 1) { "Dragged tab was not selected" }
                shot("nav-$mode-released").recycle()
                val cancelAt = SystemClock.uptimeMillis()
                event(MotionEvent.ACTION_DOWN, next.exactCenterX(), next.exactCenterY(), cancelAt)
                event(MotionEvent.ACTION_MOVE, tab.exactCenterX(), next.exactCenterY(), cancelAt)
                event(MotionEvent.ACTION_CANCEL, tab.exactCenterX(), next.exactCenterY(), cancelAt)
                Thread.sleep(600)
                check(selected.intValue == 1) { "Cancelled drag changed tabs" }
            }
            report.putString("navigation", "PASS light/dark press overflow, drag selection, release and cancellation")
            test.runOnMainSync { scene.value = "chat"; theme.intValue = 1 }
            waitFor("玩家Alice头像")
            shot("avatars-light-start").recycle()
            check(nodes().count { it.contentDescription?.toString() == "玩家Alice头像" } == 1)
            check(nodes().count { it.contentDescription?.toString() == "玩家Bob头像" } == 1)
            val display = test.targetContext.resources.displayMetrics
            val at = SystemClock.uptimeMillis()
            val x = display.widthPixels * .75f
            val y = display.heightPixels * .40f
            event(MotionEvent.ACTION_DOWN, x, y, at)
            repeat(20) { step -> event(MotionEvent.ACTION_MOVE, x, y + display.density * 190f * (step + 1) / 20, at); Thread.sleep(25) }
            event(MotionEvent.ACTION_UP, x, y + display.density * 190f, at)
            Thread.sleep(400)
            val avatar = bounds(waitFor("玩家Alice头像"))
            val composer = bounds(waitFor("发消息，输入 @ 提及玩家"))
            check(avatar.bottom < composer.top && composer.top - avatar.bottom < display.density * 40) {
                "Long-group avatar did not pin above the composer: $avatar / $composer"
            }
            check(nodes().count { it.contentDescription?.toString() == "玩家Alice头像" } == 1)
            shot("avatars-light-pinned").recycle()
            test.runOnMainSync { theme.intValue = 2 }
            shot("avatars-dark-pinned").recycle()
            var node = waitFor("玩家Alice头像")
            while (!node.isClickable) node = node.parent
            check(node.performAction(AccessibilityNodeInfo.ACTION_CLICK))
            check(profiles.get() == 1)
            node = waitFor("玩家Alice头像")
            while (!node.isLongClickable) node = node.parent
            check(node.performAction(AccessibilityNodeInfo.ACTION_LONG_CLICK))
            Thread.sleep(250)
            check(state.chatDraft.text.contains("@Alice"))
            report.putString("avatars", "PASS one avatar per group, sticky viewport boundary, profile click and long-press mention")
            test.finish(-1, report)
        } catch (failure: Throwable) {
            report.putString("failure", failure.stackTraceToString())
            test.finish(1, report)
        } finally { scope.cancel() }
    }
}
