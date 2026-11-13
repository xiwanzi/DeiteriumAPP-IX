package com.deuterium.app.uilab

import android.app.Instrumentation
import android.content.Intent
import android.graphics.Bitmap
import android.os.Bundle
import android.view.ViewTreeObserver
import androidx.activity.compose.setContent
import androidx.compose.foundation.background
import androidx.compose.foundation.gestures.scrollBy
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyListState
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.drawWithContent
import androidx.compose.ui.geometry.Rect
import androidx.compose.ui.layout.LayoutCoordinates
import androidx.compose.ui.layout.onGloballyPositioned
import androidx.compose.ui.layout.positionInRoot
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.withContext
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.atomic.AtomicInteger
import kotlin.math.abs

/** Compare actual placed row/portrait bounds at draw time, not a copy of the renderer's formula. */
internal object AvatarFrameCheck {
    fun run(test: Instrumentation) {
        check(test.targetContext.packageName.endsWith(".chatmediaqa"))
        val report = Bundle()
        val activity = test.startActivitySync(Intent(test.targetContext, DeuteriumActivity::class.java)
            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)) as DeuteriumActivity
        val lines = mutableStateListOf<ChatLine>()
        val rowBounds = ConcurrentHashMap<Long, LayoutCoordinates>()
        val avatarBounds = ConcurrentHashMap<Long, LayoutCoordinates>()
        var rootTop = 0f
        var rootHeight = 0f
        var checksEnabled = false
        val frames = AtomicInteger()
        var maxError = 0f
        var crossedDockFrames = 0
        var partialExitFrames = 0
        val failures = mutableListOf<String>()
        lateinit var list: LazyListState
        val density = test.targetContext.resources.displayMetrics.density
        var inset = (80 * density).toInt().toFloat()
        val portraitHeight = (42 * density + .5f).toInt().toFloat()
        val listener = ViewTreeObserver.OnDrawListener {
            if (checksEnabled && rootHeight > 0 && avatarBounds.isNotEmpty()) {
                frames.incrementAndGet()
                val groups = groupChatMessages(lines.toList())
                // Lazy prefetch may keep attached but unplaced coordinates after a jump.
                // Only currently placed list items count as visible message evidence.
                val placedIds = list.layoutInfo.visibleItemsInfo.mapTo(hashSetOf()) { it.key }
                val bottom = rootTop + rootHeight - inset
                val drawingBottom = rootTop + rootHeight
                fun current(coordinates: LayoutCoordinates?): Rect? {
                    if (coordinates == null || !coordinates.isAttached) return null
                    val p = coordinates.positionInRoot()
                    return Rect(p.x, p.y, p.x + coordinates.size.width, p.y + coordinates.size.height)
                }
                for ((id, coordinates) in avatarBounds) {
                    val portrait = current(coordinates) ?: continue
                    if (portrait.right <= 0 || portrait.bottom <= rootTop) continue
                    if (portrait.top >= bottom && portrait.top < drawingBottom) crossedDockFrames++
                    if (portrait.top < drawingBottom && portrait.bottom > drawingBottom) partialExitFrames++
                    val group = groups.firstOrNull { it.last.id == id } ?: continue
                    val first = current(rowBounds[group.first.id])?.top ?: Float.NEGATIVE_INFINITY
                    val last = current(rowBounds[group.last.id])?.bottom ?: Float.POSITIVE_INFINITY
                    // Independent constraints on actual placed rows, including their current scroll.
                    val expected = maxOf(first, minOf(last, bottom) - portraitHeight).coerceAtMost(last - portraitHeight)
                    val error = abs(expected - portrait.top)
                    maxError = maxOf(maxError, error)
                    if (error > 1.1f && failures.size < 8) failures += "id=$id error=$error expected=$expected actual=${portrait.top}"
                }
                val active = avatarBounds.mapNotNull { (id, coordinates) ->
                    current(coordinates)?.takeIf { it.right > 0 && it.bottom > rootTop && it.top < drawingBottom }?.let { id to it }
                }
                groups.filter { group -> group.ids.any { id ->
                    id in placedIds && current(rowBounds[id])?.let { it.bottom > rootTop && it.top < drawingBottom } == true
                } }.forEach { group ->
                    if (active.none { it.first == group.last.id } && failures.size < 8) failures += "Missing visible group ${group.first.id}..${group.last.id}; placed=$placedIds"
                }
                active.sortedBy { it.second.top }.zipWithNext().forEach { (a, b) ->
                    if (a.second.bottom > b.second.top + 1 && failures.size < 8) failures += "Overlapping groups ${a.first}/${b.first}"
                }
            }
        }
        try {
            test.runOnMainSync {
                repeat(90) { i ->
                    val who = "Player${i / 6}"
                    lines += ChatLine(i.toLong() + 1, who, "消息 $i\n" + "可变高度的连续消息。".repeat(i % 4 + 1),
                        false, serverAt = 1794542400000 + i * 1000L, playerRef = who)
                }
                activity.setContent {
                    LabTheme(1, true) {
                        list = rememberLazyListState()
                        val snapshot = lines.toList()
                        val groups = remember(snapshot) { groupChatMessages(snapshot) }
                        Box(Modifier.fillMaxSize().background(MaterialTheme.colorScheme.background)
                            .drawWithContent {
                                // Android's OnDraw fires BEFORE Compose's pending placement.
                                // Inspect after child drawing instead, when these are rendered positions.
                                val frame = list.layoutInfo
                                drawContent()
                                if (frame.totalItemsCount > 0) listener.onDraw()
                            }
                            .onGloballyPositioned { rootTop = it.positionInRoot().y; rootHeight = it.size.height.toFloat() }) {
                            ChatGlassLayout(body = { dockInset ->
                            inset = with(androidx.compose.ui.platform.LocalDensity.current) { (dockInset + 8.dp).toPx() }
                            Box(Modifier.fillMaxSize()) {
                            LazyColumn(Modifier.fillMaxSize(), state = list, reverseLayout = true,
                                contentPadding = PaddingValues(start = 20.dp, end = 20.dp, top = 72.dp, bottom = dockInset + 8.dp),
                                verticalArrangement = Arrangement.spacedBy(10.dp)) {
                                items(snapshot.asReversed(), key = { it.id }) { line ->
                                    DisposableEffect(line.id) { onDispose { rowBounds.remove(line.id) } }
                                    Row(Modifier.fillMaxWidth().heightIn(min = 42.dp).onGloballyPositioned {
                                        rowBounds[line.id] = it
                                    }) {
                                        Spacer(Modifier.width(51.dp))
                                        Text(line.text, Modifier.width(260.dp).background(MaterialTheme.colorScheme.surface).padding(12.dp))
                                    }
                                }
                            }
                            ChatGroupAvatars(list, groups) { line, modifier ->
                                DisposableEffect(line.id) { onDispose { avatarBounds.remove(line.id) } }
                                PlayerAvatar(line.name, modifier.onGloballyPositioned {
                                    avatarBounds[line.id] = it
                                })
                            }
                            }
                            }, composer = {
                                Box(Modifier.navigationBarsPadding()) {
                                    MessageComposer(androidx.compose.ui.text.input.TextFieldValue(), {}, {}, "输入消息")
                                }
                            })
                        }
                    }
                }
            }
            test.waitForIdleSync(); Thread.sleep(500)
            runBlocking { withContext(androidx.compose.ui.platform.AndroidUiDispatcher.Main) {
                checksEnabled = true
                suspend fun scroll(frames: Int, pixels: Float) {
                    repeat(frames) { withFrameNanos { }; list.scrollBy(pixels) }
                }
                scroll(100, 19f)
                scroll(100, -19f)
                scroll(45, 137f)
                scroll(45, -137f)
                // Cross recycled item boundaries while replacing groups and prepending history.
                list.scrollToItem(20)
                withFrameNanos { }
                lines.removeAll { it.id in 72L..77L }
                delay(100)
                scroll(45, 81f)
                scroll(45, -81f)
                lines.addAll(0, (1L..12L).map { ChatLine(-it, "Older", "更早的消息 $it", false,
                    serverAt = 1794542300000 + it * 1000, playerRef = "Older") })
                delay(100)
                scroll(30, 63f)
                scroll(30, -63f)
                checksEnabled = false
            } }
            check(frames.get() > 150) { "Too few observed draw frames: ${frames.get()}" }
            check(crossedDockFrames > 8 && partialExitFrames > 3) { "Avatar still disappears at input boundary: crossed=$crossedDockFrames clipped=$partialExitFrames" }
            check(failures.isEmpty()) { "Portrait lag/jump: ${failures.joinToString()}" }
            val image = test.uiAutomation.takeScreenshot()!!
            test.targetContext.filesDir.resolve("avatar-frames-15.png").outputStream().use { image.compress(Bitmap.CompressFormat.PNG,100,it) }
            image.recycle()
            report.putString("avatarFrames", "PASS ${frames.get()} draw frames, max error $maxError px; slow/fast/reverse/group deletion/history prepend; $crossedDockFrames frames beneath dock, $partialExitFrames partial bottom exits")
            test.finish(-1, report)
        } catch (failure: Throwable) {
            report.putString("failure", failure.stackTraceToString()); test.finish(1, report)
        } finally {
            test.runOnMainSync { if (activity.window.decorView.viewTreeObserver.isAlive) activity.window.decorView.viewTreeObserver.removeOnDrawListener(listener) }
        }
    }
}
