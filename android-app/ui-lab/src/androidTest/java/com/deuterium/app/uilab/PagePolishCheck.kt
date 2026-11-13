package com.deuterium.app.uilab

import android.app.Instrumentation
import android.content.Intent
import android.graphics.Bitmap
import android.graphics.Rect
import android.os.Bundle
import android.view.accessibility.AccessibilityNodeInfo
import androidx.activity.compose.setContent
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Add
import androidx.compose.material.icons.outlined.Assignment
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.rememberGraphicsLayer
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import java.time.LocalDateTime
import java.util.concurrent.atomic.AtomicInteger

/** Real page layouts on a local fixture. The pending transfer never reaches a backend. */
internal object PagePolishCheck {
    fun run(test: Instrumentation) {
        check(test.targetContext.packageName.endsWith(".chatmediaqa"))
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main.immediate)
        val report = Bundle()
        val output = test.targetContext.filesDir.resolve("page-polish-9").apply { mkdirs() }
        val activity = test.startActivitySync(
            Intent(test.targetContext, DeuteriumActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        ) as DeuteriumActivity
        val theme = mutableIntStateOf(1)
        val scene = mutableStateOf("hall")
        val waiting = CompletableDeferred<Boolean>()
        val calls = AtomicInteger()
        try {
            test.runOnMainSync {
                val state = LabState(scope, userName = "FixtureSelf")
                state.balance = 12345678
                state.balanceKnown = true
                state.recentTransfers.addAll(listOf("BuilderOak", "GardenKeeper", "RiverWalker"))
                Players.clear()
                Players.addAll(state.recentTransfers.map { PlayerProfile(it, playerRef = "fixture_$it") })
                state.recentTransfers.forEachIndexed { index, name ->
                    state.conversation(name).add(ChatLine(index.toLong() + 1, name, "本机验收会话", false,
                        remoteId = "fixture_$index", serverAt = index.toLong() + 1))
                }
                state.conversation("AI 助手").addAll(listOf(
                    ChatLine(11, "你", "怎样在市场查看商品？", true),
                    ChatLine(12, "AI 助手", "打开市场后，可以按分类查看玩家发布的物品。", false),
                    ChatLine(13, "你", "可以查看交付说明吗？", true),
                    ChatLine(14, "AI 助手", "打开商品详情即可查看交付方式、价格和库存。", false)
                ))
                val time = LocalDateTime.of(2026, 1, 1, 12, 0)
                state.commissions.entries.add(Commission("active-fixture", "active", "BuilderOak",
                    CommissionDraft("搭建公共工坊", "小型建筑委托", "主城", 1234500, 24, Urgency.Soon, null), time))
                state.commissions.entries.add(Commission("mine-fixture", "mine", "FixtureSelf",
                    CommissionDraft("整理材料仓库", "完成归类与标记", "主城", 1230600, 24, Urgency.Soon, null),
                    time, stage = CommissionStage.Cancelled, worker = "GardenKeeper",
                    refund = RefundState.Approved, canHideRecord = true))
                activity.setContent {
                    val page = scene.value
                    val mode = theme.intValue
                    key(page, mode) {
                        LabTheme(mode, true) {
                            SideEffect {
                                androidx.core.view.WindowCompat.getInsetsController(activity.window, activity.window.decorView)
                                    .isAppearanceLightStatusBars = mode == 1
                            }
                            val layer = rememberGraphicsLayer()
                            val atmosphere = rememberGraphicsLayer()
                            val status = WindowInsets.statusBars.asPaddingValues().calculateTopPadding()
                            val top = status + 84.dp
                            CompositionLocalProvider(LocalOverlayBackdrop provides layer) {
                                Box(Modifier.fillMaxSize().background(MaterialTheme.colorScheme.background)) {
                                    GlassScene(atmosphere, layer, Modifier.fillMaxSize()) {
                                        when (page) {
                                            "hall" -> CommissionHallPage(state, top, {})
                                            "mine" -> CommissionHallPage(state, top, {}, mine = true)
                                            "ai" -> Box(Modifier.fillMaxSize().navigationBarsPadding()) {
                                                DirectChatPage(state, "AI 助手", top, onProfile = {})
                                            }
                                            "transfer", "waiting" -> WalletPage(state, {}, {}, top)
                                            "search" -> SearchPage("Info", state, {}, {}, active = true)
                                        }
                                    }
                                    if (page in setOf("hall", "mine", "ai", "transfer", "waiting")) {
                                        GradientGlassHeader(layer, Modifier.fillMaxWidth().height(status + 72.dp))
                                        val title = when (page) {
                                            "hall" -> "委托大厅"
                                            "mine" -> "我的委托"
                                            "ai" -> "小祥 AI"
                                            else -> "我的钱包"
                                        }
                                        CompactPageToolbar(title, status, {}, actionCount = if (page == "hall" || page == "mine") 2 else 0) {
                                            if (page == "hall" || page == "mine") GlassIconButton({}, Modifier.size(48.dp)) {
                                                Icon(Icons.Outlined.Assignment, "我的委托")
                                            }
                                            if (page == "hall" || page == "mine") GlassIconButton({}, Modifier.size(48.dp)) {
                                                Icon(Icons.Outlined.Add, "发布委托")
                                            }
                                        }
                                    }
                                }
                                if (page == "transfer") TransferSheet(state, onClose = {})
                                if (page == "waiting") IosSheet(
                                    onDismissRequest = {},
                                    surfaceParameters = TransactionGlass,
                                    dimAmount = .14f,
                                    replacementContent = {
                                        PaymentExperience(100, "转给 BuilderOak", "转账成功", onCommit = {
                                            calls.incrementAndGet()
                                            waiting.await()
                                        }, embedded = true, onClose = {})
                                    }
                                ) {}
                            }
                        }
                    }
                }
            }
            fun nodes(): List<AccessibilityNodeInfo> {
                val list = mutableListOf<AccessibilityNodeInfo>()
                fun walk(node: AccessibilityNodeInfo?) {
                    if (node == null) return
                    list.add(node)
                    repeat(node.childCount) { walk(node.getChild(it)) }
                }
                test.uiAutomation.clearCache()
                walk(test.uiAutomation.rootInActiveWindow)
                return list
            }
            fun matches(label: String) = nodes().filter {
                it.text?.toString() == label || it.contentDescription?.toString() == label
            }
            fun waitFor(label: String): AccessibilityNodeInfo {
                repeat(38) {
                    test.waitForIdleSync()
                    matches(label).firstOrNull()?.let { return it }
                    Thread.sleep(70)
                }
                error("Missing page control: $label")
            }
            val expected = linkedMapOf(
                "hall" to "互相搭把手", "mine" to "我发布的", "ai" to "向 AI 提问",
                "transfer" to "确认转账", "waiting" to "请稍候…", "search" to "输入玩家 ID 或 QQ，搜索全部玩家"
            )
            for (mode in listOf(1, 2)) {
                for ((page, label) in expected) {
                    test.runOnMainSync { theme.intValue = mode; scene.value = page }
                    val marker = waitFor(label)
                    when (page) {
                        "hall" -> {
                            val mine = Rect()
                            val add = Rect()
                            waitFor("我的委托").getBoundsInScreen(mine)
                            waitFor("发布委托").getBoundsInScreen(add)
                            check(mine.right <= add.left && mine.centerY() == add.centerY())
                        }
                        "mine" -> {
                            val first = Rect()
                            waitFor("全部").getBoundsInScreen(first)
                            val display = test.targetContext.resources.displayMetrics
                            check(first.left >= 0 && first.width() >= 22f * display.density) {
                                "The initial filter is clipped: $first"
                            }
                            check(matches("我的委托").count { it.text?.toString() == "我的委托" } >= 1)
                        }
                        "ai" -> check(matches("发送消息").size == 1)
                        "waiting" -> check(!marker.isClickable) { "Waiting is still exposed as a button" }
                    }
                    Thread.sleep(if (page == "search") 650 else 380)
                    val screenshot = checkNotNull(test.uiAutomation.takeScreenshot())
                    output.resolve("${if (mode == 1) "light" else "dark"}-$page.png").outputStream().use {
                        screenshot.compress(Bitmap.CompressFormat.PNG, 100, it)
                    }
                    screenshot.recycle()
                    if (page == "search") {
                        test.runOnMainSync { scene.value = "idle" }
                        test.waitForIdleSync()
                        Thread.sleep(300)
                    }
                }
            }
            check(calls.get() == 2) { "Expected one fixture payment request per theme" }
            report.putString("pagePolish", "PASS search, AI composer, commission filters, transfer form and waiting in both themes")
            test.finish(-1, report)
        } catch (failure: Throwable) {
            report.putString("failure", failure.stackTraceToString())
            test.finish(1, report)
        } finally {
            scope.cancel()
        }
    }
}
