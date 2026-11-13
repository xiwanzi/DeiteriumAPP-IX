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
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.layout.*
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.layer.drawLayer
import androidx.compose.ui.graphics.rememberGraphicsLayer
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import com.kyant.backdrop.backdrops.rememberLayerBackdrop
import kotlin.math.abs

/** Exercises the real controls with a local backdrop and no production account. */
internal object CatalogGlassCheck {
    fun run(test: Instrumentation) {
        val result = Bundle()
        val host = test.startActivitySync(
            Intent(test.targetContext, DeuteriumActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        ) as DeuteriumActivity
        val out = test.targetContext.filesDir.resolve("catalog-glass").apply { mkdirs() }
        val theme = mutableIntStateOf(1)
        val toggle = mutableStateOf(false)
        val level = mutableFloatStateOf(.32f)
        val dialog = mutableStateOf(false)
        val sheet = mutableStateOf(false)
        val confirms = mutableIntStateOf(0)
        val dismissals = mutableIntStateOf(0)
        try {
            test.runOnMainSync {
                host.setContent {
                    LabTheme(theme.intValue, true) {
                        val light = theme.intValue == 1
                        SideEffect {
                            androidx.core.view.WindowCompat.getInsetsController(host.window, host.window.decorView)
                                .isAppearanceLightStatusBars = light
                        }
                        val backdrop = rememberGraphicsLayer()
                        val atmosphere = rememberGraphicsLayer()
                        val foreground = rememberGraphicsLayer()
                        val layer = rememberLayerBackdrop(graphicsLayer = backdrop) {
                            drawLayer(atmosphere)
                            drawLayer(foreground)
                        }
                        CompositionLocalProvider(LocalOverlayBackdrop provides backdrop) {
                            Box(Modifier.fillMaxSize()) {
                                GlassScene(atmosphere, backdrop, Modifier.fillMaxSize(), foreground, layer) {
                                    Canvas(Modifier.fillMaxSize()) {
                                        drawRect(Brush.verticalGradient(listOf(
                                            Color(0xFF61AFA6), Color(0xFF4D77A8),
                                            Color(0xFF171D2A), Color(0xFF10131A)
                                        )))
                                        drawCircle(Color(0xFF4185D9), size.width * .48f, Offset(size.width * .80f, size.height * .28f))
                                        drawCircle(Color(0xFF71CAB6), size.width * .44f, Offset(size.width * .10f, size.height * .53f))
                                    }
                                    Column(Modifier.align(Alignment.Center).padding(26.dp)) {
                                        LabCard {
                                            Text("Catalog controls", style = MaterialTheme.typography.titleMedium)
                                            Spacer(Modifier.height(10.dp))
                                            IosSwitch(toggle.value, { toggle.value = it },
                                                Modifier.semantics { contentDescription = "Catalog toggle" })
                                            IosSlider(level.floatValue, { level.floatValue = it }, 0f..1f,
                                                Modifier.fillMaxWidth().semantics { contentDescription = "Catalog slider" })
                                            MotionButton({ dialog.value = true }, Modifier.fillMaxWidth()) { Text("Open dialog") }
                                            Spacer(Modifier.height(8.dp))
                                            SecondaryButton({ sheet.value = true }, Modifier.fillMaxWidth()) { Text("Open sheet") }
                                        }
                                    }
                                }
                                GradientGlassHeader(backdrop, Modifier.fillMaxWidth().height(142.dp))
                                Text("市场", Modifier.align(Alignment.TopCenter).statusBarsPadding().padding(top = 24.dp),
                                    style = MaterialTheme.typography.titleLarge)
                                Box(Modifier.align(Alignment.BottomCenter).navigationBarsPadding()
                                    .padding(horizontal = 18.dp, vertical = 8.dp).fillMaxWidth().height(64.dp)) {
                                    MovingGlassNav(
                                        listOf("商城", "市场", "信息", "我的"),
                                        listOf(DeuteriumIcons.Shop, DeuteriumIcons.Market, DeuteriumIcons.Messages, DeuteriumIcons.Person),
                                        selected = 1, enabled = true, backdrop = layer
                                    ) {}
                                }
                            }
                            if (dialog.value) IosDialog(
                                { dialog.value = false }, { Text("Catalog dialog") },
                                { Text("The glass keeps the color of the page behind it.") },
                                { PlainButton({ confirms.intValue++; dialog.value = false }) { Text("Confirm") } },
                                { PlainButton({ dismissals.intValue++; dialog.value = false }) { Text("Cancel") } }
                            )
                            if (sheet.value) IosSheet({ sheet.value = false }) {
                                Column(Modifier.padding(24.dp)) {
                                    Text("Catalog sheet", style = MaterialTheme.typography.titleLarge)
                                    MotionButton({ sheet.value = false }, Modifier.fillMaxWidth().padding(top = 12.dp)) {
                                        Text("Close sheet")
                                    }
                                }
                            }
                        }
                    }
                }
            }

            fun node(label: String): AccessibilityNodeInfo {
                repeat(25) {
                    test.waitForIdleSync()
                    test.uiAutomation.clearCache()
                    find(test.uiAutomation.rootInActiveWindow, label)?.let { return it }
                    Thread.sleep(90)
                }
                error("Missing control: $label")
            }

            fun tap(label: String) {
                val bounds = Rect()
                node(label).getBoundsInScreen(bounds)
                val now = SystemClock.uptimeMillis()
                for (action in listOf(MotionEvent.ACTION_DOWN, MotionEvent.ACTION_UP)) {
                    val event = MotionEvent.obtain(now, SystemClock.uptimeMillis(), action,
                        bounds.exactCenterX(), bounds.exactCenterY(), 0)
                    test.sendPointerSync(event)
                    event.recycle()
                }
                test.waitForIdleSync()
            }

            fun screenshot(name: String) {
                test.waitForIdleSync()
                Thread.sleep(280)
                val bitmap = checkNotNull(test.uiAutomation.takeScreenshot())
                out.resolve("$name.png").outputStream().use { bitmap.compress(Bitmap.CompressFormat.PNG, 100, it) }
                bitmap.recycle()
            }

            for (mode in listOf(1, 2)) {
                test.runOnMainSync {
                    theme.intValue = mode
                    toggle.value = false
                    level.floatValue = .32f
                }
                val prefix = if (mode == 1) "light" else "dark"
                node("Catalog toggle")
                screenshot("$prefix-controls")
                tap("Catalog toggle")
                check(toggle.value) { "Touch toggle did not switch on in $prefix" }
                check(node("Catalog toggle").performAction(AccessibilityNodeInfo.ACTION_CLICK))
                test.waitForIdleSync()
                check(!toggle.value) { "Accessible toggle did not switch off in $prefix" }

                val slider = node("Catalog slider")
                check(slider.performAction(
                    AccessibilityNodeInfo.AccessibilityAction.ACTION_SET_PROGRESS.id,
                    Bundle().apply { putFloat(AccessibilityNodeInfo.ACTION_ARGUMENT_PROGRESS_VALUE, .71f) }
                ))
                test.waitForIdleSync()
                check(abs(level.floatValue - .71f) < .015f) { "Slider progress was not applied in $prefix" }
                val track = Rect()
                node("Catalog slider").getBoundsInScreen(track)
                val start = SystemClock.uptimeMillis()
                for (step in 0..12) {
                    val action = when (step) {
                        0 -> MotionEvent.ACTION_DOWN
                        12 -> MotionEvent.ACTION_UP
                        else -> MotionEvent.ACTION_MOVE
                    }
                    val event = MotionEvent.obtain(start, SystemClock.uptimeMillis(), action,
                        track.left + track.width() * (.71f - .43f * step / 12f), track.exactCenterY(), 0)
                    test.sendPointerSync(event)
                    event.recycle()
                    Thread.sleep(16)
                }
                test.waitForIdleSync()
                check(level.floatValue in .18f.. .40f) { "Touch drag did not move slider in $prefix" }

                tap("Open dialog")
                node("Catalog dialog")
                screenshot("$prefix-dialog")
                tap("Cancel")
                node("Open dialog")
                tap("Open dialog")
                tap("Confirm")
                node("Open sheet")
                tap("Open sheet")
                node("Catalog sheet")
                screenshot("$prefix-sheet")
                tap("Close sheet")
                node("Open sheet")
            }
            check(confirms.intValue == 2 && dismissals.intValue == 2) { "Dialog action fired more than once" }
            result.putString("catalogGlass", "PASS light/dark header, toggle, slider, dialog actions and sheet")
            test.finish(-1, result)
        } catch (error: Throwable) {
            result.putString("failure", error.stackTraceToString())
            test.finish(1, result)
        }
    }

    private fun find(root: AccessibilityNodeInfo?, label: String): AccessibilityNodeInfo? {
        if (root == null) return null
        if (root.text?.toString() == label || root.contentDescription?.toString() == label) return root
        repeat(root.childCount) { find(root.getChild(it), label)?.let { match -> return match } }
        return null
    }
}
